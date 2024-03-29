package sessions

import (
	"github.com/emillis/cacheMachine"
	"github.com/emillis/idGen"
	"net/http"
	"sync"
	"time"
)

//===========[INTERFACES]====================================================================================================

type Cookie interface {
	Cookie(string) (*http.Cookie, error)
}

type ISession[TValue any] interface {
	Uid() string
	SetUid(uid string)
	Value() TValue
	Key() string
	SetKey(k string)
	SetValue(v TValue)
	LastModified() time.Time
	UpdateLastModified()
}

//===========[STRUCTURES]===============================================================================================

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
	if e := ss._sessions.GetEntry(uid); e == nil {
		return nil
	} else {
		return e.Value()
	}
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

//===========[FUNCTIONALITY]====================================================================================================

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

//doesUidExist checks the cache and db whether the uid already exist
func doesUidExist[TValue any](ss *SessionStore[TValue], uid string) bool {
	return ss._sessions.Exist(uid) || ss._tmpUidStore.Exist(uid) || ss.Requirements.UidExist(uid)
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
