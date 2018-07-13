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
	"github.com/control-center/serviced/auth"
	userdomain "github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/rpc/master"
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
const auth0TokenCookie = "auth0AccessToken"

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
func loginWithBasicAuthOK(r *rest.Request) bool {
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

func loginWithTokenOK(r *rest.Request, token string) bool {
	restToken, err := auth.ParseRestToken(token)
	if err != nil {
		msg := "Unable to parse rest token"
		plog.WithError(err).WithField("url", r.URL.String()).Debug(msg)
		return false
	} else {
		if !restToken.ValidateRequestHash(r.Request) {
			msg := "Could not login with rest token. Request signature does not match token."
			plog.WithField("url", r.URL.String()).Debug(msg)
			return false
		} else if !restToken.HasAdminAccess() {
			msg := "Could not login with rest token. Insufficient permissions."
			plog.WithField("url", r.URL.String()).Debug(msg)
			return false
		} else {
			return true
		}
	}
}

func loginWithAuth0TokenOK(r *rest.Request, token string) (auth.Auth0Token, bool) {
	auth0Token, err := auth.ParseAuth0Token(token)
	if err != nil {
		msg := "Unable to parse auth0 rest token"
		plog.WithError(err).WithField("url", r.URL.String()).Debug(msg)
		return nil, false
	} else {
		if !auth0Token.HasAdminAccess() {
			msg := "Could not login with auth0 rest token. Insufficient permissions."
			plog.WithField("url", r.URL.String()).Debug(msg)
			return nil, false
		} else {
			return auth0Token, true
		}
	}
}

func loginWithAuth0CookieOk(r *rest.Request) bool {
	cookie, err := r.Request.Cookie(auth0TokenCookie)
	if err != nil {
		glog.V(1).Info("Error getting cookie ", err)
		return false
	}
	token := cookie.Value
	_, result := loginWithAuth0TokenOK(r, token)
	return result
}

func loginOK(w *rest.ResponseWriter, r *rest.Request) bool {
	token, tErr := auth.ExtractRestToken(r.Request)
	if tErr != nil { // There is a token in the header but we could not extract it
		msg := "Unable to extract auth token from header"
		plog.WithError(tErr).WithField("url", r.URL.String()).Debug(msg)
		return false
	}
	if auth.Auth0IsConfigured() {
		if auth0LoginOK(w, r, token) {
			return true
		}
		// CC-4109: even with auth0 configured, we still need token authentication for REST calls.
		return loginWithTokenOK(r, token)
	}
	return basicAuthLoginOK(w, r, token)
}

func auth0LoginOK(w *rest.ResponseWriter, r *rest.Request, token string) bool {
	if token != "null" && token != "" {
		if parsed, ok := loginWithAuth0TokenOK(r, token); ok {
			// Set cookie with token, so api calls can work.
			// Secure and HttpOnly flags are important to mitigate CSRF/XSRF attack risk.
			exp := parsed.Expiration()
			expireTime := time.Unix(exp, 0)

			http.SetCookie(
				w.ResponseWriter,
				&http.Cookie{
					Name:     auth0TokenCookie,
					Value:    token,
					Path:     "/",
					Expires:  expireTime,
					Secure:   true,
					HttpOnly: true,
				})

			// not setting secure, httponly on name cookie - this should be for display only
			http.SetCookie(
				w.ResponseWriter,
				&http.Cookie{
					Name:     usernameCookie,
					Value:    parsed.User(),
					Path:     "/",
					Expires:  expireTime,
					Secure:   false,
					HttpOnly: false,
				})
			return true
		}
		return false
	} else {
		return loginWithAuth0CookieOk(r)
	}
}

func basicAuthLoginOK(w *rest.ResponseWriter, r *rest.Request, token string) bool {
	if token != "null" && token != "" {
		return loginWithTokenOK(r, token)
	} else {
		return loginWithBasicAuthOK(r)
	}
}

/*
 * Perform logout, return JSON
 */
func restLogout(w *rest.ResponseWriter, r *rest.Request) {
	glog.V(2).Info("restLogout() called.")
	// Read session cookie and delete session
	cookie, err := r.Request.Cookie(sessionCookie)
	if err != nil {
		glog.V(2).Info("Unable to read session cookie")
	} else {
		deleteSessionT(cookie.Value)
		glog.V(2).Infof("Deleted session %s for explicit logout", cookie.Value)
	}

	// Blank out all login cookies
	writeBlankCookie(w, r, auth0TokenCookie)
	writeBlankCookie(w, r, sessionCookie)
	writeBlankCookie(w, r, usernameCookie)
	w.WriteJson(&simpleResponse{"Logged out", loginLink()})
}

func writeBlankCookie(w *rest.ResponseWriter, r *rest.Request, cname string) {
	http.SetCookie(
		w.ResponseWriter,
		&http.Cookie{
			Name:   cname,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
}

func restLoginWithBasicAuth(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	creds := login{}
	err := r.DecodeJsonPayload(&creds)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	if creds.Username == "root" && !allowRootLogin {
		glog.V(1).Info("root login disabled")
		writeJSON(w, &simpleResponse{"Root login disabled", loginLink()}, http.StatusUnauthorized)
		return
	}

	client, err := ctx.getMasterClient()
	if err != nil {
		restServerError(w, err)
		return
	}

	if validateLogin(&creds, client) {
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

/*
 * Perform login, return JSON
 */
func restLogin(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	token, tErr := auth.ExtractRestToken(r.Request)
	glog.V(0).Info("restLogin()")
	if tErr != nil { // There is a token in the header but we could not extract it
		msg := "Unable to extract auth token from header"
		plog.WithError(tErr).Warning(msg)
		writeJSON(w, &simpleResponse{msg, loginLink()}, http.StatusUnauthorized)
	} else if token != "" {
		if _, ok := loginWithAuth0TokenOK(r, token); ok {
			w.WriteJson(&simpleResponse{"Accepted", homeLink()})
			return
		} else if loginWithTokenOK(r, token) {
			w.WriteJson(&simpleResponse{"Accepted", homeLink()})
			return
		}
		writeJSON(w, &simpleResponse{"Login failed", loginLink()}, http.StatusUnauthorized)
	} else {
		restLoginWithBasicAuth(w, r, ctx)
	}
}

func validateLogin(creds *login, client master.ClientInterface) bool {
	glog.V(1).Info("validateLogin()")
	systemUser, err := client.GetSystemUser()
	if err == nil && creds.Username == systemUser.Name {
		validated := cpValidateLogin(creds, client)
		if validated {
			return validated
		}
	}
	return pamValidateLogin(creds, adminGroup)
}

func cpValidateLogin(creds *login, client master.ClientInterface) bool {
	glog.V(0).Infof("Attempting to validate user %v against the control center api", creds.Username)
	// create a client
	user := userdomain.User{
		Name:     creds.Username,
		Password: creds.Password,
	}
	// call validate token on it
	var result bool
	result, err := client.ValidateCredentials(user)
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

func getUser(r *rest.Request) (string, error) {
	for _, cookie := range r.Cookies() {
		if cookie.Name == usernameCookie {
			return cookie.Value, nil
		}
	}
	return "", errors.New("Unable to retriever user name")
}
