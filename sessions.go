package sessions

import (
	"github.com/emillis/cacheMachine"
	"github.com/emillis/idGen"
	"net/http"
	"sync"
	"time"
)

//===========[CACHE/STATIC]=============================================================================================

var _sessionCache cacheMachine.Cache[string, *Session]
var _modifiedSessions cacheMachine.Cache[string, *Session]

//DefaultKey is the default key used in key:value pairs such as cookie.Name
const DefaultKey = "_ssid"

//===========[STRUCTURES]===============================================================================================

//Unexported session definition. Kept private to disable direct access to the session
type session struct {
	//This is the unique identifier of the session. It is by default, 99 alphanumeric chars + some special symbols
	Uid string `json:"uid" bson:"uid"`

	//Key is used in key-value pairs. E.g. It is assigned to cookie.Name
	Key string `json:"key" bson:"key"`

	//Defines a period of time in which the session becomes invalid if not used.
	//Default value for this is set to 8 hours
	TimeoutDuration time.Duration `json:"timeout_duration" bson:"timeout_duration"`

	//Time when this session should be invalidated (Valid() returns false)
	ValidUntil time.Time `json:"valid_until" bson:"valid_until"`

	//Holds the time when this session was modified last
	LastModified time.Time `json:"last_modified" bson:"last_modified"`

	mx sync.RWMutex
}

//Session structure that defines a session
type Session struct {
	session
}

//Uid returns unique ID of the session
func (s *Session) Uid() string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Uid
}

//SetUid custom defined Uid for the session
func (s *Session) SetUid(uid string) {
	s.mx.Lock()
	s.session.Uid = uid
	s.mx.Unlock()
	s.UpdateLastModified()
}

//GenerateNewUid generates new Uid and replaces the current one
func (s *Session) GenerateNewUid() {
	for {
		newUid := idGen.Random(&idGen.Config{Length: 99})

		if doesUidExist(newUid) {
			continue
		}

		s.SetUid(newUid)
		break
	}
}

//Key returns session key that can be used as cookie name, etc..
func (s *Session) Key() string {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Key
}

//SetKey assigns new key for the session
func (s *Session) SetKey(key string) {
	s.mx.Lock()
	s.session.Key = key
	s.mx.Unlock()
	s.UpdateLastModified()
}

//TimeoutDuration returns duration in which, if the session is inactive, it goes invalid. Returns duration in seconds
func (s *Session) TimeoutDuration() time.Duration {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.TimeoutDuration
}

//SetTimeoutDuration sets the duration of time until the session becomes invalid if not used.
func (s *Session) SetTimeoutDuration(t time.Duration) {
	s.mx.Lock()
	s.session.TimeoutDuration = t
	s.mx.Unlock()
	s.RefreshTimeout()
	s.UpdateLastModified()
}

//Valid checks whether the session is still valid
func (s *Session) Valid() bool {
	return time.Now().Before(s.ValidUntil())
}

//RefreshTimeout refreshes time left until session goes invalid. It gives the session amount of time defined in
//SetValidUntil method
func (s *Session) RefreshTimeout() {
	s.SetValidUntil(time.Now().Add(s.TimeoutDuration()))
}

//SetSessionCookie sets cookie for the session in the ResponseWriter. The second cookie argument is optional and is used
//to have some default values set by the client. In essence, this function would override the Name and Value fields of
//the supplied cookie with the session values
func (s *Session) SetSessionCookie(w http.ResponseWriter, cookie *http.Cookie) {
	if cookie == nil {
		cookie = &http.Cookie{}
	}

	cookie.Name = s.Key()
	cookie.Value = s.Uid()

	http.SetCookie(w, cookie)
}

//ValidUntil returns timestamp in seconds when this session will become invalid
func (s *Session) ValidUntil() time.Time {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.ValidUntil
}

//SetValidUntil updates timestamp in seconds when this session will become invalid. It essentially does the calculation
//(ValidUntil = (current time in seconds) + TimeoutDuration())
func (s *Session) SetValidUntil(t time.Time) {
	s.mx.Lock()
	s.session.ValidUntil = t
	s.mx.Unlock()
	s.UpdateLastModified()
}

//LastModified returns time when this session was modified the last
func (s *Session) LastModified() time.Time {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.LastModified
}

//UpdateLastModified Sets LastModified field to the time when this function gets invoked
func (s *Session) UpdateLastModified() {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.session.LastModified = time.Now()
	_modifiedSessions.Add(s.Uid(), s)
}

//saveToCache saves this session to local cache
func (s *Session) saveToCache() {
	_sessionCache.Add(s.Uid(), s)
}

//removeFromCache deletes the session from local cache
func (s *Session) removeFromCache() {
	_sessionCache.Remove(s.Uid())
	_modifiedSessions.Remove(s.Uid())
}

//===========[FUNCTIONALITY]============================================================================================

//New creates a new session with default values already pre-filled
func New() *Session {
	s := &Session{session{
		mx: sync.RWMutex{},
	}}

	s.GenerateNewUid()
	s.SetKey("")
	s.SetTimeoutDuration(time.Second * 28800)
	s.saveToCache()

	modifiedSessions.Add(s)

	return s
}

//Get returns pointer to a session or nil if not found
func Get(uid string) *Session {
	return _sessionCache.Get(uid)
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

//===========[INITIALIZATION]===========================================================================================

//Initializing sessions
func init() {
	_sessionCache = cacheMachine.New[string, *Session](nil)
	_modifiedSessions = cacheMachine.New[string, *Session](nil)
}
