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
	scanning = iota
	scanningHost
	scanningHostOrName
	scanningName
	scanningPort
	scanningPortOrTag
	scanningRepo
	scanningRepoName
	scanningRepoNameOrName
	scanningTag
	scanningUUID
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
	UUID string
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
// image id = [host(':'port|'/')]reponame[':'tag]
// host     = {alpha|digit|'.'|'-'}+
// port     = {digit}+
// reponame = [user'/']name
// user     = {alpha}+
// name     = [uuid'_']repo
// uuid     = {alpha|digit|'-'}+
// repo     = {alpha|digit|'-'}+
// tag      = {alpha|digit}+
// The grammar is ambiguous so the parser is a little messy in places.
func ParseImageID(iid string) (*ImageID, error) {
	scanner := bufio.NewScanner(strings.NewReader(iid))
	scanner.Split(bufio.ScanRunes)
	result := &ImageID{}

	scanned := []string{}
	tokbuf := []byte{}

	state := scanning

	for scanner.Scan() {
		rune, _ := utf8.DecodeRune([]byte(scanner.Text()))
		switch state {
		case scanning:
			switch {
			case unicode.IsLetter(rune):
				tokbuf = append(tokbuf, byte(rune))
			case unicode.IsDigit(rune), rune == dash:
				tokbuf = append(tokbuf, byte(rune))
				state = scanningHostOrName
			case rune == period:
				tokbuf = append(tokbuf, byte(rune))
				state = scanningHost
			case rune == colon:
				scanned = append(scanned, string(tokbuf))
				tokbuf = []byte{}
				state = scanningPortOrTag
			case rune == slash:
				scanned = append(scanned, string(tokbuf))
				tokbuf = []byte{}
				state = scanningRepoNameOrName
			default:
				return nil, fmt.Errorf("invalid ImageID %s", iid)
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
		case scanningHostOrName:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash:
				tokbuf = append(tokbuf, byte(rune))
			case rune == period:
				tokbuf = append(tokbuf, byte(rune))
				state = scanningHost
			case rune == colon:
				scanned = append(scanned, string(tokbuf))
				tokbuf = []byte{}
				state = scanningPortOrTag
			case rune == underscore:
				result.UUID = string(tokbuf)
				tokbuf = []byte{}
				state = scanningName
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad host or name", iid)
			}
		case scanningRepoNameOrName:
			switch {
			case unicode.IsLetter(rune):
				tokbuf = append(tokbuf, byte(rune))
			case unicode.IsDigit(rune), rune == dash:
				tokbuf = append(tokbuf, byte(rune))
				state = scanningName
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
				state = scanningName
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
			case unicode.IsLetter(rune):
				tokbuf = append(tokbuf, byte(rune))
			case unicode.IsDigit(rune), rune == dash:
				tokbuf = append(tokbuf, byte(rune))
				state = scanningName
			case rune == slash:
				result.User = string(tokbuf)
				tokbuf = []byte{}
				state = scanningName
			case rune == colon:
				result.Repo = string(tokbuf)
				tokbuf = []byte{}
				state = scanningTag
			case rune == underscore:
				result.UUID = string(tokbuf)
				tokbuf = []byte{}
				state = scanningRepo
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad repo name", iid)
			}
		case scanningName:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash:
				tokbuf = append(tokbuf, byte(rune))
			case rune == colon:
				result.Repo = string(tokbuf)
				tokbuf = []byte{}
				state = scanningTag
			case rune == underscore:
				result.UUID = string(tokbuf)
				tokbuf = []byte{}
				state = scanningRepo
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad name", iid)
			}
		case scanningRepo:
			switch {
			case unicode.IsLetter(rune), unicode.IsDigit(rune), rune == dash:
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
			case unicode.IsLetter(rune), unicode.IsDigit(rune):
				tokbuf = append(tokbuf, byte(rune))
			default:
				return nil, fmt.Errorf("invalid ImageID %s: bad tag", iid)
			}
		case scanningPortOrTag:
			switch {
			case unicode.IsDigit(rune):
				tokbuf = append(tokbuf, byte(rune))
			case unicode.IsLetter(rune):
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
	case scanning, scanningRepoName, scanningName, scanningRepo:
		result.Repo = string(tokbuf)
	case scanningRepoNameOrName:
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

	if iid.UUID != "" {
		s = append(s, iid.UUID, "_")
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
