package web

import (
	"github.com/ant0ine/go-json-rest"
	"encoding/base64"
	"net/http"
	"crypto/rand"
	"log"
	"errors"
	"time"
)

const SessionCookie = "ZCPToken"
const UsernameCookie = "ZUsername"

type Session struct {
	Id string
	User string
	creation time.Time
	access time.Time
}

var sessions map[string]*Session

func init() {
	sessions = make(map[string]*Session)
}

/*
 * This function should be called by any secure REST resource
 */
func LoginOk(r *rest.Request) bool {
	cookie, err := r.Request.Cookie(SessionCookie)
	if err != nil {
		return false
	}
	session, err := findSession(cookie.Value)
	if err != nil {
		log.Println("Unable to find session", cookie.Value)
		return false
	}
	log.Println("Found session", session.Id)
	session.access = time.Now()
	return true
}

/*
 * Perform logout, return JSON
 */
func RestLogout(w *rest.ResponseWriter, r *rest.Request) {
	cookie, err := r.Request.Cookie(SessionCookie)
	if err != nil {
		log.Println("Unable to read session cookie")
	} else {
		delete(sessions, cookie.Value)
		log.Println("Deleted session", cookie.Value)
	}
	
	http.SetCookie(
		w.ResponseWriter,
		&http.Cookie {
			Name:SessionCookie,
			Value:"",
			Path:"/",
			MaxAge: -1,
		})

	w.WriteJson(&SimpleResponse{"Logged out", loginLink()})
}

/*
 * Get some data about the currently logged in user
 */
func RestUser(w *rest.ResponseWriter, r *rest.Request) {
	noCache(w)
	if LoginOk(r) {
		// TODO: Get real user data
		w.WriteJson(&UserData{"admin","Administrator","support@zenoss.com"})
	} else {
		RedirectLogin(w, r)
	}
}

/*
 * Perform login, return JSON
 */
func RestLogin(w *rest.ResponseWriter, r *rest.Request) {
	creds := Login{}
	err := r.DecodeJsonPayload(&creds)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// TODO: Fix hardcoded credentials
	if validateLogin(&creds) {
		session, err := createSession(creds.Username)
		sessions[session.Id] = session
		if err != nil {
			WriteJson(w, &SimpleResponse{"Session could not be created", loginLink()}, http.StatusInternalServerError)
			return
		}
		log.Println("Session ID:", session.Id)
		http.SetCookie(
			w.ResponseWriter,
			&http.Cookie {
				Name:SessionCookie,
				Value:session.Id,
				Path:"/",
				MaxAge: 0,
			})
		http.SetCookie(
			w.ResponseWriter,
			&http.Cookie {
				Name:UsernameCookie,
				Value:creds.Username,
				Path:"/",
				MaxAge: 0,
			})

		w.WriteJson(&SimpleResponse{"Accepted", homeLink()})
	} else {
		WriteJson(w, &SimpleResponse{"Login failed", loginLink()}, http.StatusUnauthorized)
	}
}

func validateLogin(creds *Login) bool {
	return creds.Username == "admin" && creds.Password == "zenoss"
}

func createSession(user string) (*Session, error){
	sid, err := randomSessionId()
	if err != nil {
		return nil, err
	}
	return &Session{ sid, user, time.Now(), time.Now() }, nil
}

func findSession(sid string) (*Session, error) {
	session := sessions[sid]
	if session == nil {
		return nil, errors.New("Session not found")
	}
	return session, nil
}

func randomSessionId() (string, error) {
	s, err := randomStr()
	if err != nil {
		return "", err
	}
	if sessions[s] != nil {
		return "", errors.New("Session ID collided")
	}
	return s, nil
}

func randomStr() (string, error) {
	sid := make([]byte, 32)
	n, err := rand.Read(sid)
	if n != len(sid) || err != nil {
		log.Println(err)
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sid), nil
}

