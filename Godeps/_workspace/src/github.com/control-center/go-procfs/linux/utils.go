package linux

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func parseFlagsLine(line string, flags *map[string]bool) (err error) {
	fl := make(map[string]bool)
	for _, flag := range strings.Fields(line) {
		fl[flag] = true
	}
	*flags = fl
	return
}

func parseIntLine(valString string, val *uint) (err error) {
	var uval uint64
	uval, err = strconv.ParseUint(strings.TrimSpace(valString), 10, 64)
	*val = uint(uval)
	return
}

func parseInt64Line(valString string, val *uint64) (err error) {
	var uval uint64
	uval, err = strconv.ParseUint(strings.TrimSpace(valString), 10, 64)
	*val = uval
	return
}

func parseFloat32Line(valString string, val *float32) (err error) {
	var f float64
	f, err = strconv.ParseFloat(strings.TrimSpace(valString), 32)
	*val = float32(f)
	return
}

func parseBoolLine(valString string, val *bool) (err error) {
	switch strings.TrimSpace(valString) {
	case "yes":
		*val = true
	case "no":
		*val = false
	}
	return
}

func parseBytesLine(valString string, val *uint64) (err error) {
	fields := strings.Fields(valString)
	var uval uint64
	if len(fields) != 2 {
		return fmt.Errorf("expected 2 fields for bytes line")
	}
	uval, err = strconv.ParseUint(fields[0], 10, 64)
	switch strings.ToLower(fields[1]) {
	case "kb":
		uval = uval * 1024
	}
	*val = uval
	return
}

func parseAddressSizesLine(line string) (physical, virtual uint8, err error) {
	pattern := regexp.MustCompile(`\d+ bits physical, \d+ bits virtual`)
	if !pattern.MatchString(line) {
		err = fmt.Errorf("address size pattern mismatch: %s", line)
		return
	}
	fields := strings.Fields(line)
	var uval uint64
	uval, err = strconv.ParseUint(fields[0], 10, 8)
	if err != nil {
		return
	}
	physical = uint8(uval)
	uval, err = strconv.ParseUint(fields[3], 10, 8)
	virtual = uint8(uval)
	return
}
