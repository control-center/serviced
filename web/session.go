// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	userdomain "github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"

	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const sessionCookie = "ZCPToken"
const usernameCookie = "ZUsername"

var adminGroup = "sudo"

type sessionT struct {
	ID       string
	User     string
	creation time.Time
	access   time.Time
}

var sessions map[string]*sessionT
var sessionsLock = &sync.RWMutex{}

var allowRootLogin bool = true

func init() {
	falses := []string{"0", "false", "f", "no"}
	if v := strings.ToLower(os.Getenv("SERVICED_ALLOW_ROOT_LOGIN")); v != "" {
		for _, t := range falses {
			if v == t {
				allowRootLogin = false
			}
		}
	}

	if utils.Platform == utils.Rhel {
		adminGroup = "wheel"
	}

	sessions = make(map[string]*sessionT)
	go purgeOldsessionTs()
}

func purgeOldsessionTs() {

	// use a closure to facilitate safe locking regardless of when the purge function returns
	doPurge := func() {
		sessionsLock.Lock()
		defer sessionsLock.Unlock()

		if len(sessions) == 0 {
			return
		}

		glog.V(1).Info("Searching for expired sessions")
		cutoff := time.Now().UTC().Unix() - int64((30 * time.Minute).Seconds())
		toDel := []string{}
		for key, value := range sessions {
			if value.access.UTC().Unix() < cutoff {
				toDel = append(toDel, key)
			}
		}
		for _, key := range toDel {
			glog.V(0).Infof("Deleting session %s (exceeded max age)", key)
			delete(sessions, key)
		}
	}

	for {
		time.Sleep(time.Second * 60)

		doPurge()
	}
}

/*
 * This function should be called by any secure REST resource
 */
func loginOK(r *rest.Request) bool {
	cookie, err := r.Request.Cookie(sessionCookie)
	if err != nil {
		glog.V(1).Info("Error getting cookie ", err)
		return false
	}

	sessionsLock.Lock()
	defer sessionsLock.Unlock()
	value, err := url.QueryUnescape(strings.Replace(cookie.Value, "+", url.QueryEscape("+"), -1))
	if err != nil {
		glog.Warning("Unable to decode session ", cookie.Value)
		return false
	}
	session, err := findsessionT(value)
	if err != nil {
		glog.Info("Unable to find session ", value)
		return false
	}
	session.access = time.Now()
	glog.V(2).Infof("sessionT %s used", session.ID)
	return true
}

/*
 * Perform logout, return JSON
 */
func restLogout(w *rest.ResponseWriter, r *rest.Request) {
	cookie, err := r.Request.Cookie(sessionCookie)
	if err != nil {
		glog.V(1).Info("Unable to read session cookie")
	} else {
		deleteSessionT(cookie.Value)
		glog.V(1).Infof("Deleted session %s for explicit logout", cookie.Value)
	}

	http.SetCookie(
		w.ResponseWriter,
		&http.Cookie{
			Name:   sessionCookie,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})

	w.WriteJson(&simpleResponse{"Logged out", loginLink()})
}

/*
 * Perform login, return JSON
 */
func restLogin(w *rest.ResponseWriter, r *rest.Request, client *node.ControlClient) {
	creds := login{}
	err := r.DecodeJsonPayload(&creds)
	if err != nil {
		glog.V(1).Info("Unable to decode login payload ", err)
		restBadRequest(w, err)
		return
	}

	if creds.Username == "root" && !allowRootLogin {
		glog.V(1).Info("root login disabled")
		writeJSON(w, &simpleResponse{"Root login disabled", loginLink()}, http.StatusUnauthorized)
		return
	}

	if pamValidateLogin(&creds, adminGroup) || cpValidateLogin(&creds, client) {
		sessionsLock.Lock()
		defer sessionsLock.Unlock()

		session, err := createsessionT(creds.Username)
		if err != nil {
			writeJSON(w, &simpleResponse{"sessionT could not be created", loginLink()}, http.StatusInternalServerError)
			return
		}
		sessions[session.ID] = session

		glog.V(1).Info("Created authenticated session: ", session.ID)
		http.SetCookie(
			w.ResponseWriter,
			&http.Cookie{
				Name:   sessionCookie,
				Value:  session.ID,
				Path:   "/",
				MaxAge: 0,
			})
		http.SetCookie(
			w.ResponseWriter,
			&http.Cookie{
				Name:   usernameCookie,
				Value:  creds.Username,
				Path:   "/",
				MaxAge: 0,
			})

		w.WriteJson(&simpleResponse{"Accepted", homeLink()})
	} else {
		writeJSON(w, &simpleResponse{"Login failed", loginLink()}, http.StatusUnauthorized)
	}
}

func cpValidateLogin(creds *login, client *node.ControlClient) bool {
	glog.V(0).Infof("Attempting to validate user %v against the control center api", creds.Username)
	// create a client
	user := userdomain.User{
		Name:     creds.Username,
		Password: creds.Password,
	}
	// call validate token on it
	var result bool
	err := client.ValidateCredentials(user, &result)

	if err != nil {
		glog.Errorf("Unable to validate credentials %s", err)
	}

	return result
}

func createsessionT(user string) (*sessionT, error) {
	sid, err := randomsessionTId()
	if err != nil {
		return nil, err
	}
	return &sessionT{sid, user, time.Now(), time.Now()}, nil
}

func findsessionT(sid string) (*sessionT, error) {
	session, ok := sessions[sid]
	if !ok {
		return nil, errors.New("sessionT not found")
	}
	return session, nil
}

func randomsessionTId() (string, error) {
	s, err := randomStr()
	if err != nil {
		return "", err
	}
	if sessions[s] != nil {
		return "", errors.New("sessionT ID collided")
	}
	return s, nil
}

func randomStr() (string, error) {
	sid := make([]byte, 32)
	n, err := rand.Read(sid)
	if n != len(sid) {
		return "", errors.New("not enough random bytes")
	}
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sid), nil
}

func deleteSessionT(sid string) {
	sessionsLock.Lock()
	defer sessionsLock.Unlock()

	delete(sessions, sid)
}
