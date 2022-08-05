package sessions

import (
	"net/http"
	"sync"
	"time"
)

//===========[STRUCTS]====================================================================================================

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

//SetUid sets new uid for this session
func (s *Session[TValue]) SetUid(uid string) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.session.updateLastModified()
	s.session.Uid = uid
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

//SetKey sets new key for this session
func (s *Session[TValue]) SetKey(k string) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.session.updateLastModified()
	s.session.Key = k
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
