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
	"errors"
	"net/http"
	"strings"
)

var (
	ErrBadRestToken = errors.New("Could not extract auth token from REST request header")
)

func ExtractRestToken(header string) (string, error) {
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

func ExtractRestTokenFromHeaders(h http.Header) (string, error) {
	rawHeader := h.Get("Authorization")
	if rawHeader != "" {
		return ExtractRestToken(rawHeader)
	}
	return rawHeader, nil //Token not present
}

func ValidateRestToken(token string) (bool, error) {
	identity, err := ParseJWTIdentity(token)
	if err != nil {
		return false, err
	}
	valid := !identity.Expired() && identity.HasAdminAccess()
	return valid, nil
}
