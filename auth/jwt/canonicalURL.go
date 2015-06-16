// Copyright 2015 The Serviced Authors.
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

// Package jwt implements Zenoss-specific JWT facilities
package jwt

import (
	"fmt"
	"net/url"
	"strings"

	purell "github.com/PuerkitoBio/purell"
)

// CanonicalURL returns a canonical URL per Zenoss-conventions
func CanonicalURL(method, urlString, uriPrefix string, body []byte) ([]byte, error) {
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input string %q : %v", urlString, err)
	}

	// The following are categorized as "safe" rules in the general case
	normalizationRules := purell.FlagUppercaseEscapes |
		purell.FlagDecodeUnnecessaryEscapes |
		purell.FlagEncodeNecessaryEscapes |
		purell.FlagRemoveEmptyQuerySeparator

	// The following are categorized as "usually safe" in the most general case
	normalizationRules |= purell.FlagRemoveTrailingSlash | purell.FlagRemoveDotSegments

	// The following may change semantics in the most general case, but
	// 		should be safe for CC REST URLs:
	normalizationRules |= purell.FlagSortQuery | purell.FlagRemoveDuplicateSlashes

	normalizedURL, err := url.Parse(purell.NormalizeURL(parsedURL, normalizationRules))
	if err != nil {
		return nil, fmt.Errorf("failed to normalized URL: %v", err)
	}

	canonicalURI := getCanonicalURI(normalizedURL, uriPrefix)
	canonicalQuery := getCanonicalQuery(normalizedURL)

	canonicalURL := make([]byte, 0, len(method)+len(canonicalURI)+len(canonicalQuery)+len(body)+3)
	canonicalURL = append(canonicalURL, []byte(method)...)
	canonicalURL = append(canonicalURL, ' ')
	canonicalURL = append(canonicalURL, []byte(canonicalURI)...)
	canonicalURL = append(canonicalURL, ' ')
	canonicalURL = append(canonicalURL, []byte(canonicalQuery)...)
	canonicalURL = append(canonicalURL, ' ')
	canonicalURL = append(canonicalURL, body...)
	return canonicalURL, nil
}

func getCanonicalURI(url *url.URL, uriPrefix string) string {
	path := strings.TrimRight(url.Path, "/")
	uriPrefix = strings.TrimRight(uriPrefix, "/")
	prefixLen := len(uriPrefix)
	if prefixLen > 0 && prefixLen <= len(path) && uriPrefix == path[0:prefixLen] {
		path = path[prefixLen:]
	}

	if len(path) == 0 {
		path = "/"
	}
	canonicalURI := path
	return canonicalURI
}

func getCanonicalQuery(url *url.URL) string {
	return url.RawQuery
}
