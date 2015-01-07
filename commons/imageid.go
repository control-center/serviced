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

package commons

import (
	"bufio"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/docker/docker/pkg/parsers"
)

// states that the parser can be in as it scans
const (
	scanningHostOrRepoName = iota
	scanningHost
	scanningPortOrTag
	scanningPort
	scanningRepoNameOrRepo
	scanningRepoName
	scanningRepo
	scanningTag
)

// runes that we have to check for as we scan
var (
	colon      rune
	dash       rune
	period     rune
	slash      rune
	underscore rune
)

// ImageID represents a Docker Image identifier.
type ImageID struct {
	Host string
	Port int
	User string
	Repo string
	Tag  string
}

func init() {
	// setup the required runes
	colon, _ = utf8.DecodeRune([]byte(":"))
	dash, _ = utf8.DecodeRune([]byte("-"))
	period, _ = utf8.DecodeRune([]byte("."))
	slash, _ = utf8.DecodeRune([]byte("/"))
	underscore, _ = utf8.DecodeRune([]byte("_"))
}

func RenameImageID(dockerRegistry, tenantId string, imgID string, tag string) (*ImageID, error) {
	repo, _ := parsers.ParseRepositoryTag(imgID)
	re := regexp.MustCompile("/?([^/]+)\\z")
	matches := re.FindStringSubmatch(repo)
	if matches == nil {
		return nil, errors.New("malformed imageid")
	}
	name := matches[1]
	newImageID := fmt.Sprintf("%s/%s/%s:%s", dockerRegistry, tenantId, name, tag)
	return ParseImageID(newImageID)
}

// ParseImageID parses the string representation of a Docker image ID into an ImageID structure.
// The grammar used by the parser is:
// image id = [host(':'port|'/')]reponame[':'tag]
// host     = {alpha|digit|'.'|'-'}+
// port     = {digit}+
// reponame = [user'/']repo
// user     = {alpha|digit|'-'|'_'}+
// repo     = {alpha|digit|'-'|'_'}+
// tag      = {alpha|digit|'-'|'_'|'.'}+
// The grammar is ambiguous so the parser is a little messy in places.
func ParseImageID(iid string) (*ImageID, error) {
	scanner := bufio.NewScanner(strings.NewReader(iid))
	scanner.Split(bufio.ScanRunes)
	result := &ImageID{}

	scanned := []string{}
	tokbuf := []byte{}

	state := scanningHostOrRepoName

	for scanner.Scan() {
		rune, _ := utf8.DecodeRune([]byte(scanner.Text()))
		switch state {
		case scanningHostOrRepoName:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash:
				tokbuf = append(tokbuf, byte(rune))
			case rune == period:
				tokbuf = append(tokbuf, byte(rune))
				state = scanningHost
			case rune == underscore:
				tokbuf = append(tokbuf, byte(rune))
				state = scanningRepoName
			case rune == slash:
				scanned = append(scanned, string(tokbuf))
				tokbuf = []byte{}
				state = scanningRepoNameOrRepo
			case rune == colon:
				scanned = append(scanned, string(tokbuf))
				tokbuf = []byte{}
				state = scanningPortOrTag
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad host or name", iid)
			}
		case scanningHost:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == period, rune == dash:
				tokbuf = append(tokbuf, byte(rune))
			case rune == colon:
				result.Host = string(tokbuf)
				tokbuf = []byte{}
				state = scanningPort
			case rune == slash:
				result.Host = string(tokbuf)
				tokbuf = []byte{}
				state = scanningRepoName
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad hostname", iid)
			}
		case scanningRepoNameOrRepo:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash, rune == underscore:
				tokbuf = append(tokbuf, byte(rune))
			case rune == colon:
				result.User = scanned[0]
				scanned = []string{}
				result.Repo = string(tokbuf)
				tokbuf = []byte{}
				state = scanningTag
			case rune == slash:
				result.Host = scanned[0]
				scanned = []string{}
				result.User = string(tokbuf)
				tokbuf = []byte{}
				state = scanningRepo
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad host or repo name", iid)
			}
		case scanningPort:
			switch {
			case unicode.IsDigit(rune):
				tokbuf = append(tokbuf, byte(rune))
			case rune == slash:
				portno, err := strconv.Atoi(string(tokbuf))
				if err != nil {
					return nil, fmt.Errorf("invalid ImageID %s: %v", iid, err)
				}
				result.Port = portno
				tokbuf = []byte{}
				state = scanningRepoName
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad port number", iid)
			}
		case scanningRepoName:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash, rune == underscore:
				tokbuf = append(tokbuf, byte(rune))
			case rune == slash:
				result.User = string(tokbuf)
				tokbuf = []byte{}
				state = scanningRepo
			case rune == colon:
				result.Repo = string(tokbuf)
				tokbuf = []byte{}
				state = scanningTag
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad repo name", iid)
			}
		case scanningRepo:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash, rune == underscore:
				tokbuf = append(tokbuf, byte(rune))
			case rune == colon:
				result.Repo = string(tokbuf)
				tokbuf = []byte{}
				state = scanningTag
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad repo", iid)
			}
		case scanningTag:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash, rune == underscore, rune == period:
				tokbuf = append(tokbuf, byte(rune))
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad tag (rune:'%c')", iid, rune)
			}
		case scanningPortOrTag:
			switch {
			case unicode.IsDigit(rune):
				tokbuf = append(tokbuf, byte(rune))
			case unicode.IsLetter(rune), rune == dash, rune == period:
				tokbuf = append(tokbuf, byte(rune))
				result.Repo = scanned[0]
				scanned = []string{}
				state = scanningTag
			case rune == slash:
				result.Host = scanned[0]
				scanned = []string{}

				portno, err := strconv.Atoi(string(tokbuf))
				if err != nil {
					return nil, fmt.Errorf("invalid ImageID %s: %v", iid, err)
				}
				result.Port = portno

				tokbuf = []byte{}
				state = scanningRepoName
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad port or tag", iid)
			}
		}
	}

	switch state {
	case scanningHostOrRepoName, scanningRepoName, scanningRepo:
		result.Repo = string(tokbuf)
	case scanningRepoNameOrRepo:
		result.User = scanned[0]
		result.Repo = string(tokbuf)
	case scanningPort, scanningHost:
		return nil, fmt.Errorf("incomplete ImageID %s", iid)
	case scanningPortOrTag:
		result.Repo = scanned[0]
		result.Tag = string(tokbuf)
	case scanningTag:
		result.Tag = string(tokbuf)
	}

	return result, nil
}

// JoinRepoTag joins an image repo with the tag
func JoinRepoTag(repo, tag string) string {
	return fmt.Sprintf("%s:%s", repo, tag)
}

// Equals compares to ImageID objects to verify they are the same
func (iid ImageID) Equals(iid2 ImageID) bool {
	if iid.BaseName() != iid2.BaseName() {
		return false
	}

	return iid.Tag == iid2.Tag || (iid.IsLatest() && iid2.IsLatest())
}

// String returns a string representation of the ImageID structure
func (iid ImageID) String() string {
	name := iid.BaseName()

	if iid.Tag != "" {
		name = name + ":" + iid.Tag
	}

	return name
}

// BaseName returns a string representation of the ImageID structure sans tag
func (iid ImageID) BaseName() string {
	s := []string{}

	if iid.Host != "" {
		s = append(s, iid.Host)
		if iid.Port != 0 {
			s = append(s, ":", strconv.Itoa(iid.Port))
		}
		s = append(s, "/")
	}

	if iid.User != "" {
		s = append(s, iid.User, "/")
	}

	s = append(s, iid.Repo)

	return strings.Join(s, "")
}

// Registry returns registry component of the ImageID as a string with the form: hostname:port
func (iid ImageID) Registry() string {
	s := []string{}

	if len(iid.Host) == 0 {
		return ""
	}

	s = append(s, iid.Host)

	if iid.Port != 0 {
		s = append(s, ":", strconv.Itoa(iid.Port))
	}

	return strings.Join(s, "")
}

// IsLatest returns a boolean that indicates that the image ID is the latest
func (iid ImageID) IsLatest() bool {
	switch iid.Tag {
	case "", "latest":
		return true
	}
	return false
}

// Validate returns true if the ImageID structure is valid.
func (iid *ImageID) Validate() bool {
	piid, err := ParseImageID(iid.String())
	if err != nil {
		return false
	}

	return reflect.DeepEqual(piid, iid)
}
