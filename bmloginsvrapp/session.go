package main

import (
	"github.com/gorilla/sessions"
)

const cookieKey = "session-store"

var store = sessions.NewCookieStore([]byte(cookieKey))
