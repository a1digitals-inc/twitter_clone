package session

import sessions "github.com/gorilla/sessions"

var Store = sessions.NewCookieStore([]byte("SECRET"))