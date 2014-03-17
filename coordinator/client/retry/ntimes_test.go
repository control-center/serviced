package retry

import (
	"testing"
	"time"
	"errors"
)

var(

	mockErr = errors.New("mockerr")
)

func TestNTimes(t *testing.T) {

	n := NTimes(0, time.Millisecond * 10)
	if n.AllowRetry(0, time.Second) {
		t.Logf("expected 0 retries")
		t.FailNow()
	}
	n = NTimes(1, time.Millisecond * 10)
	if !n.AllowRetry(0, time.Second) {
		t.Logf("expected 1 retries")
		t.FailNow()
	}

	// check if elapsed means anything
	if !n.AllowRetry(0, time.Second * 10000) {
		t.Logf("expected 1 retries")
		t.FailNow()
	}
}

