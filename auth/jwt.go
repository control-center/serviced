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
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	// Verify JWTIdentity implements the Identity interface
	_ Identity   = &jwtIdentity{}
	_ jwt.Claims = &jwtIdentity{}
)

// At sets a fake time, executes the provided
// function, then restores the default time getter,
// making it possible to test time-sensitive stuff
func At(t time.Time, f func()) {
	defer func() {
		jwt.TimeFunc = time.Now
	}()

	jwt.TimeFunc = func() time.Time {
		return t
	}

	f()
}

// jwtIdentity is an implementation of the Identity interface based on a JSON
// web token.
type jwtIdentity struct {
	Host        string `json:"hid,omitempty"`
	Pool        string `json:"pid,omitempty"`
	ExpiresAt   int64  `json:"exp,omitempty"`
	IssuedAt    int64  `json:"iat,omitempty"`
	AdminAccess bool   `json:"adm,omitempty"`
	DFSAccess   bool   `json:"dfs,omitempty"`
	PubKey      string `json:"key,omitempty"`
}

// ParseJWTIdentity parses a JSON Web Token string, verifying that it was signed by the master.
func ParseJWTIdentity(token string) (Identity, error) {
	claims := &jwtIdentity{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm matches the key
		if _, ok := token.Method.(*jwt.SigningMethodRSAPSS); !ok {
			return nil, ErrInvalidSigningMethod
		}
		return GetMasterPublicKey()
	})
	if err != nil {
		if verr, ok := err.(*jwt.ValidationError); ok {
			if verr.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, ErrIdentityTokenExpired
			}
			if verr.Errors&(jwt.ValidationErrorNotValidYet|jwt.ValidationErrorIssuedAt) != 0 {
				return nil, ErrIdentityTokenNotValidYet
			}
			if verr.Errors&(jwt.ValidationErrorSignatureInvalid|jwt.ValidationErrorUnverifiable) != 0 {
				return nil, ErrIdentityTokenBadSig
			}
			return nil, verr.Inner
		}
		return nil, err
	}
	if claims, ok := parsed.Claims.(*jwtIdentity); ok && parsed.Valid {
		return claims, nil
	}
	return nil, ErrIdentityTokenInvalid
}

// CreateJWTIdentity returns a signed string
func CreateJWTIdentity(hostID, poolID string, admin, dfs bool, pubKeyPEM []byte, expiration time.Duration) (string, int64, error) {
	now := jwt.TimeFunc().UTC()
	claims := &jwtIdentity{
		Host:        hostID,
		Pool:        poolID,
		ExpiresAt:   now.Add(expiration).Unix(),
		IssuedAt:    now.Unix(),
		AdminAccess: admin,
		DFSAccess:   dfs,
		PubKey:      string(pubKeyPEM),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
	masterPrivKey, err := getMasterPrivateKey()
	if err != nil {
		return "", 0, err
	}
	signed, err := token.SignedString(masterPrivKey)
	return signed, claims.ExpiresAt, err
}

func (id *jwtIdentity) Valid() error {

	if id.Expired() {
		return ErrIdentityTokenExpired
	}

	now := jwt.TimeFunc().UTC().Unix()
	if now < id.IssuedAt {
		return ErrIdentityTokenNotValidYet
	}

	return nil
}

func (id *jwtIdentity) Expired() bool {
	now := jwt.TimeFunc().UTC().Unix()
	return now >= id.ExpiresAt
}

func (id *jwtIdentity) HostID() string {
	return id.Host

}

func (id *jwtIdentity) PoolID() string {
	return id.Pool

}

func (id *jwtIdentity) HasAdminAccess() bool {
	return id.AdminAccess
}

func (id *jwtIdentity) HasDFSAccess() bool {
	return id.DFSAccess
}

func (id *jwtIdentity) Verifier() (Verifier, error) {
	return RSAVerifierFromPEM([]byte(id.PubKey))
}
