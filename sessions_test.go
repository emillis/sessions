package sessions

import (
	"testing"
)

func initializeSessionStore(n int, r *Requirements) *SessionStore[string] {
	s := New[string](r)

	for ; n > 0; n-- {
		s.New(string(rune(n)))
	}

	return s
}

//===========[TESTING]====================================================================================================

func TestNew(t *testing.T) {
	storeNoReq := New[string](nil)
	storeWithReq := New[string](&Requirements{})

	if storeNoReq == nil {
		t.Errorf("Function New() with nil supplied for Requirements was expected to return a *SessionStore, got nil")
	}

	if storeWithReq == nil {
		t.Errorf("Function New() with Requirements supplied was expected to return a *SessionStore, got nil")
	}
}

func TestSessionStore_New(t *testing.T) {
	ss := initializeSessionStore(10, nil)

	s := ss.New("1")

	if s == nil {
		t.Errorf("Expecten method New to return a Session, got nil")
	}
}
