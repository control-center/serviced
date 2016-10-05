// Copyright 2016 The Serviced Authors.
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

package auth

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type AuthTokenRetriever func() (string, error)

var (
	RequestDelimiter                       = " "
	RestTokenExpiration                    = 10 * time.Second
	AuthTokenGetter     AuthTokenRetriever = AuthTokenNonBlocking
)

type RestToken interface {
	Valid() error
	Expired() bool
	AuthToken() string
	RestToken() string
	RequestHash() []byte
	HasAdminAccess() bool
}

type jwtRestClaims struct {
	IssuedAt      int64  `json:"iat,omitempty"`
	ExpiresAt     int64  `json:"exp,omitempty"`
	DelegateToken string `json:"tkn,omitempty"`
	ReqHash       []byte `json:"req,omitempty"`
}

func (t *jwtRestClaims) Valid() error {
	if t.Expired() {
		return ErrRestTokenExpired
	}
	return nil
}

func (t *jwtRestClaims) Expired() bool {
	now := jwt.TimeFunc().UTC().Unix()
	return now >= t.ExpiresAt
}

type jwtRestToken struct {
	*jwtRestClaims
	authIdentity Identity
	restToken    string
}

func (t *jwtRestToken) AuthToken() string {
	return t.DelegateToken
}

func (t *jwtRestToken) RequestHash() []byte {
	return t.ReqHash
}

func (t *jwtRestToken) HasAdminAccess() bool {
	return t.authIdentity.HasAdminAccess()
}

func (t *jwtRestToken) RestToken() string {
	return t.restToken
}

func GetRequestHash(r *http.Request) []byte {
	// Simplified request hash for now
	req := strings.ToUpper(r.Method) + RequestDelimiter + strings.ToLower(r.RequestURI)
	hashedReq := sha256.Sum256([]byte(req))
	return hashedReq[:]
}

func BuildRestToken(r *http.Request) (string, error) {
	now := jwt.TimeFunc().UTC()
	requestHash := GetRequestHash(r)
	iat := now.Unix()
	exp := now.Add(RestTokenExpiration).Unix()
	authToken, err := AuthTokenGetter()
	if err != nil {
		return "", err
	}
	claims := &jwtRestClaims{iat, exp, authToken, requestHash}
	restToken := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
	delegatePrivKey, err := getDelegatePrivateKey()
	if err != nil {
		return "", err
	}
	signedToken, err := restToken.SignedString(delegatePrivKey)
	return signedToken, err
}

func AddRestTokenToRequest(r *http.Request, token string) {
	r.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
}

func ExtractRestToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", nil //Token not present
	} else {
		// The expected header value is "Bearer JWT_ToKen"
		header = strings.TrimSpace(header)
		splitted := strings.Split(header, " ")
		if len(splitted) >= 2 {
			bearer := splitted[0]
			token := splitted[len(splitted)-1]
			if strings.ToLower(bearer) == "bearer" && len(token) > 0 {
				return token, nil
			}
		}
		return "", ErrBadRestToken
	}
}

func ParseRestToken(token string) (RestToken, error) {
	claims := &jwtRestClaims{}
	identity := &jwtIdentity{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm matches the key
		if _, ok := token.Method.(*jwt.SigningMethodRSAPSS); !ok {
			return nil, ErrInvalidSigningMethod
		}
		// Get the delegate token and extract the host delegate key
		id, err := ParseJWTIdentity(claims.DelegateToken)
		if err != nil {
			return nil, err
		}
		if ji, ok := id.(*jwtIdentity); ok {
			identity = ji
			return RSAPublicKeyFromPEM([]byte(ji.PubKey))
		}
		return nil, ErrIdentityTokenBadSig
	})
	if err != nil {
		if verr, ok := err.(*jwt.ValidationError); ok {
			if verr.Inner != nil && (verr.Inner == ErrIdentityTokenExpired || verr.Inner == ErrIdentityTokenBadSig) {
				return nil, verr.Inner
			}
			if verr.Errors&jwt.ValidationErrorExpired != 0 || verr.Inner != nil && verr.Inner == ErrRestTokenExpired {
				return nil, ErrRestTokenExpired
			}
			if verr.Errors&(jwt.ValidationErrorSignatureInvalid|jwt.ValidationErrorUnverifiable) != 0 {
				return nil, ErrRestTokenBadSig
			}
			if verr.Errors&(jwt.ValidationErrorMalformed) != 0 {
				return nil, ErrBadRestToken
			}
			if verr.Inner != nil {
				return nil, verr.Inner
			}
		}
		return nil, err
	}
	if claims, ok := parsed.Claims.(*jwtRestClaims); ok && parsed.Valid {
		restToken := &jwtRestToken{}
		restToken.jwtRestClaims = claims
		restToken.authIdentity = identity
		restToken.restToken = token
		return restToken, nil
	}
	return nil, ErrIdentityTokenInvalid
}
