package sessions

import (
	"github.com/emillis/cacheMachine"
	"github.com/emillis/idGen"
	"net/http"
	"sync"
	"time"
)

//===========[CACHE/STATIC]=============================================================================================

//If requirements are not supplied, this will be used as default fallback
var defaultRequirements = Requirements{
	DefaultKey: "_ssid",
	Timeout:    0,
}

//===========[STRUCTURES]===============================================================================================

//Requirements outline the base setup of a SessionStore
type Requirements struct {
	DefaultKey string        `json:"default_key" bson:"default_key"`
	Timeout    time.Duration `json:"timeout" bson:"timeout"`
}

//Unexported session store where all the related sessions will be cached
type sessionStore[TData any] struct {
	//Every pointer to a Session structure will be stored here
	_sessions cacheMachine.Cache[string, *Session[TData]]

	//Only purpose of this cache is to store pointers to Sessions that were modified. This cache is going to be used only
	//for updating the database where instead of saving the entire cache, only the modified ones will be updated
	_modifiedSessions cacheMachine.Cache[string, *Session[TData]]

	//When checking for UID existence, possible unique ID will be stored here until determined that it's indeed unique
	_tmpUidStore cacheMachine.Cache[string, struct{}]

	//DefaultKey is the default key used in key:value pairs such as cookie.Name
	Requirements Requirements

	mx sync.RWMutex
}

//SessionStore is exported access point to all the cached sessions
type SessionStore[TData any] struct {
	sessionStore[TData]
}

//New creates new session in this store with the data supplied and returns pointer to it
func (ss *SessionStore[TData]) New(data TData) *Session[TData] {
	uid := generateUid()

	s := &Session[TData]{session[TData]{
		Uid:  uid,
		mx:   sync.RWMutex{},
		data: data,
	}}

	ss._sessions.AddWithTimeout(uid, s, ss.Requirements.Timeout)
	ss._modifiedSessions.Add(uid, s)

	return s
}

//Unexported session definition. Kept private to disable direct access to the session
type session[TData any] struct {
	//This is the unique identifier of the session. It is by default, 99 alphanumeric chars + some special symbols
	Uid string `json:"uid" bson:"uid"`

	//Key is used in key-value pairs. E.g. It is assigned to cookie.Name
	Key string `json:"key" bson:"key"`

	//This is where some data is stored for the session
	data TData

	//Defines a period of time in which the session becomes invalid if not used.
	//Default value for this is set to 8 hours
	TimeoutDuration time.Duration `json:"timeout_duration" bson:"timeout_duration"`

	//Time when this session should be invalidated (Valid() returns false)
	ValidUntil time.Time `json:"valid_until" bson:"valid_until"`

	//Holds the time when this session was modified last
	LastModified time.Time `json:"last_modified" bson:"last_modified"`

	mx sync.RWMutex
}

//Session structure that defines an individual session
type Session[TData any] struct {
	session[TData]
}

//Uid returns unique ID of the session
func (s *Session[TData]) Uid() string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Uid
}

//SetUid custom defined Uid for the session
func (s *Session[TData]) SetUid(uid string) {
	s.mx.Lock()
	s.session.Uid = uid
	s.mx.Unlock()
	s.UpdateLastModified()
}

//Key returns session key that can be used as cookie name, etc..
func (s *Session[TData]) Key() string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Key
}

//SetKey assigns new key for the session
func (s *Session[TData]) SetKey(key string) {
	s.mx.Lock()
	s.session.Key = key
	s.mx.Unlock()
	s.UpdateLastModified()
}

//TimeoutDuration returns duration in which, if the session is inactive, it goes invalid. Returns duration in seconds
func (s *Session[TData]) TimeoutDuration() time.Duration {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.TimeoutDuration
}

//SetTimeoutDuration sets the duration of time until the session becomes invalid if not used.
func (s *Session[TData]) SetTimeoutDuration(t time.Duration) {
	s.mx.Lock()
	s.session.TimeoutDuration = t
	s.mx.Unlock()
	s.RefreshTimeout()
	s.UpdateLastModified()
}

//Valid checks whether the session is still valid
func (s *Session[TData]) Valid() bool {
	return time.Now().Before(s.ValidUntil())
}

//RefreshTimeout refreshes time left until session goes invalid. It gives the session amount of time defined in
//SetValidUntil method
func (s *Session[TData]) RefreshTimeout() {
	s.SetValidUntil(time.Now().Add(s.TimeoutDuration()))
}

//SetSessionCookie sets cookie for the session in the ResponseWriter. The second cookie argument is optional and is used
//to have some default values set by the client. In essence, this function would override the Name and Value fields of
//the supplied cookie with the session values
func (s *Session[TData]) SetSessionCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if cookie == nil {
		cookie = &http.Cookie{}
	}

	cookie.Name = s.Key()
	cookie.Value = s.Uid()

	http.SetCookie(w, cookie)
}

//ValidUntil returns timestamp in seconds when this session will become invalid
func (s *Session[TData]) ValidUntil() time.Time {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.ValidUntil
}

//SetValidUntil updates timestamp in seconds when this session will become invalid. It essentially does the calculation
//(ValidUntil = (current time in seconds) + TimeoutDuration())
func (s *Session[TData]) SetValidUntil(t time.Time) {
	s.mx.Lock()
	s.session.ValidUntil = t
	s.mx.Unlock()
	s.UpdateLastModified()
}

//LastModified returns time when this session was modified the last
func (s *Session[TData]) LastModified() time.Time {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.LastModified
}

//UpdateLastModified Sets LastModified field to the time when this function gets invoked
func (s *Session[TData]) UpdateLastModified() {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.session.LastModified = time.Now()
	_modifiedSessions.Add(s.Uid(), s)
}

//saveToCache saves this session to local cache
func (s *Session[TData]) saveToCache() {
	_sessionCache.Add(s.Uid(), s)
}

//removeFromCache deletes the session from local cache
func (s *Session[TData]) removeFromCache() {
	_sessionCache.Remove(s.Uid())
	_modifiedSessions.Remove(s.Uid())
}

//===========[FUNCTIONALITY]============================================================================================

//Checks whether Requirements don't have problematic values
func makeRequirementsReasonable(r *Requirements) *Requirements {
	if r.DefaultKey == "" {
		r.DefaultKey = defaultRequirements.DefaultKey
	}

	if r.Timeout == 0 {
		r.Timeout = defaultRequirements.Timeout
	}

	return r
}

//New initiates and returns a pointer to SessionStore
func New[TData any](r *Requirements) *SessionStore[TData] {
	if r == nil {
		r = &defaultRequirements
	} else {
		r = makeRequirementsReasonable(r)
	}

	s := &SessionStore[TData]{sessionStore[TData]{
		_sessions:         cacheMachine.New[string, *Session[TData]](nil),
		_modifiedSessions: cacheMachine.New[string, *Session[TData]](nil),
		_tmpUidStore:      cacheMachine.New[string, struct{}](nil),
		Requirements:      *r,
		mx:                sync.RWMutex{},
	}}

	return s
}

//Get returns pointer to a session or nil if not found
func Get(uid string) *Session {
	s, _ := _sessionCache.Get(uid)
	return s
}

//GetFromRequest finds session id defined in the request's cookies. Custom key can be defined. If key left empty,
//constant DefaultKey is used as the key
func GetFromRequest(r *http.Request, key string) *Session {
	if key == "" {
		key = DefaultKey
	}

	cookie, err := r.Cookie(key)
	if err != nil {
		return nil
	}

	return Get(cookie.Value)
}

//TODO: Implement uid existence check
//doesUidExist checks the cache and db whether the uid already exist
func doesUidExist(uid string) bool {

	return false
}

//TODO: get this to take in SessionStore as an argument so it could find all the existing UIDs
//Generates and returns new unique UID
func generateUid() string {
	for {
		newUid := idGen.Random(&idGen.Config{Length: 99})

		if doesUidExist(newUid) {
			continue
		}

		return newUid
	}
}
