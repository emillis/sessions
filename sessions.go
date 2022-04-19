package sessions

import (
	"github.com/emillis/idGen"
	"net/http"
)

//===========[CACHE/STATIC]====================================================================================================

var uidToSession map[string]*Session

const CookieSessionName = "_ssid"

//===========[STRUCTURES]====================================================================================================

type session struct {
	Uid string `json:"uid" bson:"uid"`
}

//Session structure that defines a session
type Session struct {
	session
}

//Uid returns unique ID of the session. Not to be confused with WebSessId, the Uid is used only to reference
//the session on the back end
func (s *Session) Uid() string {
	return s.session.Uid
}

//GenerateNewUid generates new Uid and replaces the current one
func (s *Session) GenerateNewUid() {
	for {
		newUid := idGen.Random(&idGen.Config{Length: 64})

		if doesUidExist(newUid) {
			continue
		}

		s.session.Uid = newUid
		break
	}
}

//===========[FUNCTIONALITY]====================================================================================================

//New creates a new session with default values already pre-filled
func New() *Session {
	s := &Session{session{}}

	s.GenerateNewUid()

	return s
}

//TODO: Implement uid existence check
//doesUidExist checks the cache and db whether the uid already exist
func doesUidExist(uid string) bool {

	return false
}

//Get returns pointer to a session or nil if not found
func Get(uid string) *Session {
	return uidToSession[uid]
}

//GetFromRequest finds session id defined in the request's cookies. Custom key can be defined. If key left empty,
//constant CookieSessionName is used as the key
func GetFromRequest(r *http.Request, key string) *Session {
	if key == "" {
		key = CookieSessionName
	}

	cookie, err := r.Cookie(key)
	if err != nil {
		return nil
	}

	return Get(cookie.Value)
}

func SetSessionCookie(w http.ResponseWriter) {
	//TODO: Finish this
	//http.SetCookie(w, http.Cookie{})
}

//===========[INITIALIZATION]====================================================================================================

func init() {
	uidToSession = make(map[string]*Session)
}
