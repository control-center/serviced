package web

import (
	"github.com/ant0ine/go-json-rest"
	"github.com/zenoss/glog"

	"encoding/base64"
	"net/http"
	"crypto/rand"
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
	go purgeOldSessions()
}


func purgeOldSessions() {
	for {
		time.Sleep(time.Second * 60)
		if len(sessions) == 0 {
			continue;
		}
		cutoff := time.Now().UTC().Unix() - int64((30 * time.Minute).Seconds())
		toDel := []string{}
		for key, value := range sessions {
			if value.access.UTC().Unix() < cutoff {
				toDel = append(toDel, key)
			}
		}
		for _, key := range toDel {
			glog.Infof("Deleting session %s (exceeded max age)", key)
			delete(sessions, key)
		}
	}
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
		glog.Infof("Unable to find session %s", cookie.Value)
		return false
	}
	session.access = time.Now()
	return true
}

/*
 * Perform logout, return JSON
 */
func RestLogout(w *rest.ResponseWriter, r *rest.Request) {
	cookie, err := r.Request.Cookie(SessionCookie)
	if err != nil {
		glog.Infoln("Unable to read session cookie")
	} else {
		delete(sessions, cookie.Value)
		glog.Infof("Deleted session %s", cookie.Value)
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
 * Perform login, return JSON
 */
func RestLogin(w *rest.ResponseWriter, r *rest.Request) {
	creds := Login{}
	err := r.DecodeJsonPayload(&creds)
	if err != nil {
		RestBadRequest(w)
		return
	}
	
	// TODO: Fix hardcoded credentials
	if validateLogin(&creds) {
		session, err := createSession(creds.Username)
		if err != nil {
			WriteJson(w, &SimpleResponse{"Session could not be created", loginLink()}, http.StatusInternalServerError)
			return
		}
		sessions[session.Id] = session
		glog.Infof("Session ID: %s", session.Id)
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

func createSession(user string) (*Session, error){
	sid, err := randomSessionId()
	if err != nil {
		return nil, err
	}
	return &Session{ sid, user, time.Now(), time.Now() }, nil
}

func findSession(sid string) (*Session, error) {
	session, ok := sessions[sid]
	if !ok {
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
	if n != len(sid) {
		return "", errors.New("Not enough random bytes")
	}
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sid), nil
}

