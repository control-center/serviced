package commons

import (
	"bufio"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// states that the parser can be in as it scans
const (
	reading = iota
	readingPort
	readingPortOrTag
	readingRepo
	readingRepoName
	readingTag
)

// runes that we have to check for as we scan
var (
	colon      rune
	dash       rune
	period     rune
	slash      rune
	underscore rune
)

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

// ParseImageID parses the string representation of a Docker image ID into an ImageID structure.
// The grammar used by the parser is:
// image id = [host]repo[tag]
// host     = {alpha|digit|'.'|'-'}+[port]
// port     = ':'{digit}+
// repo     = [user]name
// user     = {alpha}+'/'
// name     = {alpha|digit|'-'|'_'}+
// tag      = {alpha|digit}+
// The grammar is ambiguous so the parser is a little messy in places.
func ParseImageID(iid string) (*ImageID, error) {
	scanner := bufio.NewScanner(strings.NewReader(iid))
	scanner.Split(bufio.ScanRunes)
	result := &ImageID{}

	scanned := []string{}
	scanbuf := []byte{}

	readingHost := false
	state := reading

	for scanner.Scan() {
		rune, _ := utf8.DecodeRune([]byte(scanner.Text()))
		switch state {
		case reading:
			switch {
			case unicode.IsLetter(rune):
				scanbuf = append(scanbuf, byte(rune))
			case unicode.IsDigit(rune):
				scanbuf = append(scanbuf, byte(rune))
			case rune == period:
				scanbuf = append(scanbuf, byte(rune))
				readingHost = true
			case rune == dash:
				scanbuf = append(scanbuf, byte(rune))
				readingHost = true
			case rune == colon:
				if readingHost {
					result.Host = string(scanbuf)
					state = readingPort
				} else {
					scanned = append(scanned, string(scanbuf))
				}
				scanbuf = []byte{}
				state = readingPortOrTag
			case rune == slash:
				if readingHost {
					result.Host = string(scanbuf)
					state = readingRepo
				} else {
					scanned = append(scanned, string(scanbuf))
				}
				scanbuf = []byte{}
			default:
				return nil, fmt.Errorf("Invalid ImageID %s: bad hostname", iid)
			}
		case readingPort:
			switch {
			case unicode.IsDigit(rune):
				scanbuf = append(scanbuf, byte(rune))
			case rune == slash:
				portno, err := strconv.Atoi(string(scanbuf))
				if err != nil {
					return nil, fmt.Errorf("Invalid ImageID %s: %v", iid, err)
				}
				result.Port = portno
				scanbuf = []byte{}
				state = readingRepo
			default:
				return nil, fmt.Errorf("Invalid ImageID %s: bad port number", iid)
			}
		case readingRepo:
			switch {
			case unicode.IsLetter(rune):
				scanbuf = append(scanbuf, byte(rune))
			case unicode.IsDigit(rune), rune == dash, rune == underscore:
				scanbuf = append(scanbuf, byte(rune))
				state = readingRepoName
			case rune == slash:
				result.User = string(scanbuf)
				scanbuf = []byte{}
				state = readingRepoName
			case rune == colon:
				result.Repo = string(scanbuf)
				scanbuf = []byte{}
				state = readingTag
			default:
				return nil, fmt.Errorf("Invalid ImageID %s: bad repo", iid)
			}
		case readingRepoName:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash, rune == underscore:
				scanbuf = append(scanbuf, byte(rune))
			case rune == colon:
				result.Repo = string(scanbuf)
				scanbuf = []byte{}
				state = readingTag
			default:
				return nil, fmt.Errorf("Invalid ImageID %s: bad reponame", iid)
			}
		case readingTag:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune):
				scanbuf = append(scanbuf, byte(rune))
			default:
				return nil, fmt.Errorf("Invalid ImageID %s: bad tag", iid)
			}
		case readingPortOrTag:
			switch {
			case unicode.IsDigit(rune):
				scanbuf = append(scanbuf, byte(rune))
			case unicode.IsLetter(rune):
				scanbuf = append(scanbuf, byte(rune))
				state = readingTag
			case rune == slash:
				portno, err := strconv.Atoi(string(scanbuf))
				if err != nil {
					return nil, fmt.Errorf("Invalid ImageID %s: %v", iid, err)
				}
				result.Port = portno

				if len(scanned) > 0 {
					result.Host = scanned[0]
					scanned = []string{}
				}

				scanbuf = []byte{}
				state = readingRepo
			default:
				return nil, fmt.Errorf("Invalid ImageID %s: bad port or tag", iid)
			}
		}
	}

	switch state {
	case reading:
		if len(scanned) == 1 {
			result.User = scanned[0]
			result.Repo = string(scanbuf)
		}
		result.Repo = string(scanbuf)
	case readingPort:
		return nil, fmt.Errorf("Incomplete ImageID %s", iid)
	case readingPortOrTag:
		switch len(scanned) {
		case 1:
			result.Repo = scanned[0]
			result.Tag = string(scanbuf)
		case 2:
			result.User = scanned[0]
			result.Repo = scanned[1]
			result.Tag = string(scanbuf)
		}
	case readingRepo, readingRepoName:
		result.Repo = string(scanbuf)
	case readingTag:
		if len(scanned) == 1 {
			result.Repo = scanned[0]
		}
		result.Tag = string(scanbuf)
	}

	return result, nil
}

// String returns a string representation of the ImageID structure
func (iid ImageID) String() string {
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

	if iid.Tag != "" {
		s = append(s, ":", iid.Tag)
	}

	return strings.Join(s, "")
}

// Validate returns true if the ImageID structure is valid.
func (iid *ImageID) Validate() bool {
	piid, err := ParseImageID(iid.String())
	if err != nil {
		return false
	}

	return reflect.DeepEqual(piid, iid)
}
