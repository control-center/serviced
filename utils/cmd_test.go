package utils

import (
	"errors"
	"os/exec"
	"testing"
)

func TestGetExitStatus(t *testing.T) {
	t.Log("verifying nil error")
	if exitstatus, ok := GetExitStatus(nil); !ok {
		t.Errorf("nil exit status should be considered a valid exit status")
	} else if exitstatus != 0 {
		t.Errorf("Expected 0 exit code; actual %d", exitstatus)
	}

	t.Log("verifying invalid error type")
	invalidError := errors.New("not an ExitError")
	if _, ok := GetExitStatus(invalidError); ok {
		t.Errorf("error `%s` is not an ExitError", invalidError)
	}

	t.Log("verifying valid errors")
	exiterrors := map[int]error{
		0: exec.Command("/bin/bash", "-c", "exit 0").Run(),
		1: exec.Command("/bin/bash", "-c", "exit 1").Run(),
	}

	for expected, err := range exiterrors {
		if actual, ok := GetExitStatus(err); !ok {
			t.Errorf("error '%s' should be considered a valid exit error", err)
		} else if expected != actual {
			t.Errorf("Expected %d; Actual: %d", expected, actual)
		}
	}
}