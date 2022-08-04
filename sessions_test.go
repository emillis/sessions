package sessions

import (
	"net/http"
	"testing"
)

func initializeSessionStore(n int, r *Requirements) *SessionStore[string] {
	s := New[string](r)

	for ; n > 0; n-- {
		s.New(string(rune(n)))
	}

	return s
}

type testHttpRequest struct {
	cookie *http.Cookie
}

func (t *testHttpRequest) Cookie(s string) (*http.Cookie, error) {
	return t.cookie, nil
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

func TestSessionStore_Get(t *testing.T) {
	ss := initializeSessionStore(0, nil)

	s1Uid := ss.New("1").Uid()

	v := ss.Get(s1Uid)
	v2 := ss.Get("test")

	if v == nil {
		t.Errorf("Expected v to have ISession interface returned, got nil")
	}

	if v2 != nil {
		t.Errorf("Expected v2 to be nil, got %T", v2)
	}
}

func TestSessionStore_Exist(t *testing.T) {
	ss := initializeSessionStore(0, nil)

	randomUid := "this_should_not_work"

	s1Uid := ss.New("1").Uid()

	if ss.Exist(randomUid) {
		t.Errorf("Session with UID \"%s\" shouldn't be present in the SessionStore, but it is", randomUid)
	}

	if !ss.Exist(s1Uid) {
		t.Errorf("Session with UID \"%s\" should be in the cache, but it is not", s1Uid)
	}
}

func TestSessionStore_Remove(t *testing.T) {
	ss := initializeSessionStore(0, nil)

	s1Uid := ss.New("1").Uid()

	if ss.Get(s1Uid) == nil {
		t.Errorf("Session with UID \"%s\" should exist in the SessionStore, but it does not", s1Uid)
	}

	ss.Remove(s1Uid)

	if ss.Get(s1Uid) != nil {
		t.Errorf("Session with UID \"%s\" shouldn't exist in the SessionStore, but it does", s1Uid)
	}
}

func TestSessionStore_GetFromCookie(t *testing.T) {
	testVal := "hi mom!"
	ss := initializeSessionStore(0, nil)
	s := ss.New(testVal)

	testRequest := testHttpRequest{&http.Cookie{}}
	testRequest.cookie.Value = s.Uid()
	testRequest.cookie.Name = s.Key()

	nSess := ss.GetFromCookie(&testRequest)

	if nSess == nil {
		t.Errorf("There was suppoed to be a Session returned from cookie, but got nil")
	}

	if nSess.Value() != testVal {
		t.Errorf("Expected to receive value \"%s\", got \"%s\"", testVal, nSess.Value())
	}
}
