// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package retry

import (
	"errors"
	"testing"
	"time"
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
