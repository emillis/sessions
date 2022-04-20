package sessions

import (
	"github.com/emillis/idGen"
	"net/http"
	"sync"
	"time"
)

//===========[CACHE/STATIC]====================================================================================================

//Maps Uid to session's address
var uidToSession map[string]*Session

const DefaultKey = "_ssid"

//===========[STRUCTURES]====================================================================================================

//Unexported session definition. Kept private to disable direct access to the session
type session struct {
	//This is the unique identifier of the session. It is by default, 99 alphanumeric chars + some special symbols
	Uid string `json:"uid" bson:"uid"`

	//Key is used in key-value pairs. E.g. It is assigned to cookie.Name
	Key string `json:"key" bson:"key"`

	//Defines a period of time in which the session becomes invalid if not used.
	//Default value for this is set to 8 hours
	Timeout int `json:"timeout" bson:"timeout"`

	//Marks session valid/invalid
	Valid bool `json:"valid" bson:"valid"`

	//TODO: Implement this
	//Time when this session should be invalidated (Valid changed to false)
	invalidateTime int

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
	defer s.mx.Unlock()
	s.session.Uid = uid
}

//GenerateNewUid generates new Uid and replaces the current one
func (s *Session) GenerateNewUid() {
	for {
		newUid := idGen.Random(&idGen.Config{Length: 99})

		if doesUidExist(newUid) {
			continue
		}

		s.mx.Lock()
		s.session.Uid = newUid
		s.mx.Unlock()
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
	defer s.mx.Unlock()

	s.session.Key = key
}

//Timeout returns currently set timeout to invalidate this session if it hasn't been accessed in the defined period
func (s *Session) Timeout() int {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Timeout
}

//SetTimeout sets number of seconds before this session becomes invalid if not used.
func (s *Session) SetTimeout(t time.Duration) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.session.Timeout = timeInSeconds() + int(t.Seconds())
}

//Valid checks whether the session is still valid
func (s *Session) Valid() bool {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return s.session.Valid
}

//SetValid sets the state of the session valid/invalid
func (s *Session) SetValid(status bool) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.session.Valid = status
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

//===========[FUNCTIONALITY]====================================================================================================

//New creates a new session with default values already pre-filled
func New() *Session {
	s := &Session{session{
		mx: sync.RWMutex{},
	}}

	s.GenerateNewUid()
	s.SetKey("")
	s.SetTimeout(time.Second * 28800)

	return s
}

//Get returns pointer to a session or nil if not found
func Get(uid string) *Session {
	return uidToSession[uid]
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

//Returns current time in seconds
func timeInSeconds() int {
	return int(time.Now().UnixNano() / 1000000000)
}

//===========[INITIALIZATION]====================================================================================================

//Initializing sessions
func init() {
	uidToSession = make(map[string]*Session)
}
