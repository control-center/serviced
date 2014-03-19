package retry

import (
	"errors"
	"testing"
	"time"
)

var (
	mockErr = errors.New("mockerr")
)

func TestNTimes(t *testing.T) {

	n := NTimes(0, time.Millisecond*10)
	if retry, _ := n.AllowRetry(0, time.Second); retry {
		t.Logf("expected 0 retries")
		t.FailNow()
	}
	n = NTimes(1, time.Millisecond*10)
	if retry, _ := n.AllowRetry(0, time.Second); !retry {
		t.Logf("expected 1 retries")
		t.FailNow()
	}

	// check if elapsed means anything
	if retry, _ := n.AllowRetry(0, time.Second*10000); !retry {
		t.Logf("expected 1 retries")
		t.FailNow()
	}
}
