package sessions

import (
	"github.com/emillis/cacheMachine"
	"github.com/emillis/idGen"
	"sync"
)

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
