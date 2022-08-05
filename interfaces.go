package sessions

import (
	"net/http"
	"time"
)

//===========[INTERFACES]===============================================================================================

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
