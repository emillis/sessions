package sessions

import "time"

//===========[CACHE/STATIC]=============================================================================================

//If requirements are not supplied, this will be used as default fallback
var defaultRequirements = Requirements{
	DefaultKey: "_ssid",
	Timeout:    0,
	UidExist:   func(uid string) bool { return false },
}

//===========[STRUCTS]====================================================================================================

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
