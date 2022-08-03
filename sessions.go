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
	UidExist:   func(uid string) bool { return false },
}

//===========[INTERFACES]===============================================================================================

type Cookie interface {
	Cookie(string) (*http.Cookie, error)
}

type ISession[TValue any] interface {
	Uid() string
	Value() TValue
	Key() string
	SetValue(v TValue)
	LastModified() time.Time
	UpdateLastModified()
}

//===========[STRUCTURES]===============================================================================================

//Requirements outline the base setup of a SessionStore
type Requirements struct {
	//Sessions are usually "key":"value" pairs and so, this would be the default "key" in the "key":"value" pair
	DefaultKey string `json:"default_key" bson:"default_key"`

	//Timout defines amount of time after which the session gets automatically removed if UpdateLastModified() not called
	Timeout time.Duration `json:"timeout" bson:"timeout"`

	//Here you can define a function that would check for existence of the UID other than locally within SessionStore.
	//For example, check for existence in the Database or other caches
	UidExist func(string) bool
}

//Unexported session store where all the related sessions will be cached
type sessionStore[TValue any] struct {
	//Every pointer to a Session structure will be stored here
	_sessions cacheMachine.Cache[string, *Session[TValue]]

	//Only purpose of this cache is to store pointers to Sessions that were modified. This cache is going to be used only
	//for updating the database where instead of saving the entire cache, only the modified ones will be updated
	_modifiedSessions cacheMachine.Cache[string, *Session[TValue]]

	//When checking for UID existence, possible unique ID will be stored here until determined that it's indeed unique
	_tmpUidStore cacheMachine.Cache[string, struct{}]

	//DefaultKey is the default key used in key:value pairs such as cookie.Name
	Requirements Requirements

	mx sync.RWMutex
}

//SessionStore is exported access point to all the cached sessions
type SessionStore[TValue any] struct {
	sessionStore[TValue]
}

//New creates new session in this store with the Value supplied and returns pointer to it
func (ss *SessionStore[TValue]) New(data TValue) ISession[TValue] {
	uid := generateUid(ss)

	s := &Session[TValue]{session[TValue]{
		Uid:   uid,
		mx:    sync.RWMutex{},
		store: ss,
		Value: data,
	}}

	ss._sessions.AddWithTimeout(uid, s, ss.Requirements.Timeout)
	ss._modifiedSessions.Add(uid, s)

	return s
}

//Get returns Session based on the UID provided
func (ss *SessionStore[TValue]) Get(uid string) ISession[TValue] {
	//TODO: Once the bug with cacheMachine having to do with Cache.Get() is fixed, use that instead
	if !ss._sessions.Exist(uid) {
		return nil
	}

	return ss._sessions.GetEntry(uid).Value()
}

//GetFromCookie returns session if UID was specified in the http.Request cookies
func (ss *SessionStore[TValue]) GetFromCookie(c Cookie) ISession[TValue] {
	if c == nil {
		return nil
	}

	cookie, err := c.Cookie(ss.Requirements.DefaultKey)
	if err != nil {
		return nil
	}

	s, exist := ss._sessions.Get(cookie.Value)
	if !exist {
		return nil
	}

	return s
}

//Remove removes session based on the uid supplied
func (ss *SessionStore[TValue]) Remove(uid string) {
	ss._sessions.Remove(uid)
	ss._modifiedSessions.Remove(uid)
}

//Exist checks whether supplied uid exist in the cache
func (ss *SessionStore[TValue]) Exist(uid string) bool {
	return ss._sessions.Exist(uid)
}

//Unexported session definition. Kept private to disable direct access to the session
type session[TValue any] struct {
	//This is the unique identifier of the session. It is by default, 99 alphanumeric chars + some special symbols
	Uid string `json:"uid" bson:"uid"`

	//Key is used in key-value pairs. E.g. It is assigned to cookie.Name
	Key string `json:"key" bson:"key"`

	//This is where some Value is stored for the session
	Value TValue `json:"value" bson:"value"`

	//Holds the time when this session was modified last
	LastModified time.Time `json:"last_modified" bson:"last_modified"`

	store *SessionStore[TValue]

	mx sync.RWMutex
}

//Updates last modified field in this session, but this method is not protected by a mutex
func (s *session[TValue]) updateLastModified() {
	s.LastModified = time.Now()
}

//Session structure that defines an individual session
type Session[TValue any] struct {
	session[TValue]
}

//Uid returns unique ID of the session
func (s *Session[TValue]) Uid() string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Uid
}

//Value returns value stored under this uid
func (s *Session[TValue]) Value() TValue {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Value
}

//SetValue assigns new value for the session
func (s *Session[TValue]) SetValue(v TValue) {
	s.mx.Lock()
	s.session.Value = v
	s.session.updateLastModified()
	s.mx.Unlock()
}

//Key returns session key that can be used as cookie name, etc..
func (s *Session[TValue]) Key() string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Key
}

//SetHttpCookie sets cookie for the session in the ResponseWriter. The second cookie argument is optional and is used
//to have some default values set by the client. In essence, this function would override the Name and Value fields of
//the supplied cookie with the session values
func (s *Session[TValue]) SetHttpCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if cookie == nil {
		cookie = &http.Cookie{}
	}

	cookie.Name = s.Key()
	cookie.Value = s.Uid()

	http.SetCookie(w, cookie)
}

//LastModified returns time when this session was modified the last
func (s *Session[TValue]) LastModified() time.Time {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.LastModified
}

//UpdateLastModified Sets LastModified field to the time when this function gets invoked
func (s *Session[TValue]) UpdateLastModified() {
	s.mx.Lock()
	s.session.updateLastModified()
	s.mx.Unlock()
	s.store._modifiedSessions.Add(s.Uid(), s)
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

	if r.UidExist == nil {
		r.UidExist = defaultRequirements.UidExist
	}

	return r
}

//New initiates and returns a pointer to SessionStore
func New[TValue any](r *Requirements) *SessionStore[TValue] {
	if r == nil {
		r = &defaultRequirements
	} else {
		r = makeRequirementsReasonable(r)
	}

	s := &SessionStore[TValue]{sessionStore[TValue]{
		_sessions:         cacheMachine.New[string, *Session[TValue]](nil),
		_modifiedSessions: cacheMachine.New[string, *Session[TValue]](nil),
		_tmpUidStore:      cacheMachine.New[string, struct{}](nil),
		Requirements:      *r,
		mx:                sync.RWMutex{},
	}}

	return s
}

//doesUidExist checks the cache and db whether the uid already exist
func doesUidExist[TValue any](ss *SessionStore[TValue], uid string) bool {
	return ss._sessions.Exist(uid) || ss._tmpUidStore.Exist(uid) || ss.Requirements.UidExist(uid)
}

//Generates and returns new unique UID
func generateUid[TValue any](ss *SessionStore[TValue]) string {
	for {
		newUid := idGen.Random(&idGen.Config{Length: 99})

		if doesUidExist(ss, newUid) {
			continue
		}

		return newUid
	}
}
