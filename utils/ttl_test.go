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

// +build unit

package utils

import (
	"errors"
	"testing"
	"time"
)

type TestTTL struct {
	errch <-chan error
}

func (ttl *TestTTL) Purge(t time.Duration) (time.Duration, error) {
	if err := <-ttl.errch; err != nil {
		return 0, err
	}

	return t, nil
}

func TestRunTTL(t *testing.T) {
	cancel := make(chan interface{})
	errch := make(chan error)

	ttl := &TestTTL{errch}

	// verify cancel
	t.Logf("verify cancel")
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunTTL(ttl, cancel, 100*time.Millisecond, 1*time.Second)
	}()
	close(cancel)
	close(errch)
	select {
	case <-done:
		t.Logf("cancel verified")
	case <-time.After(5 * time.Second):
		t.Fatalf("Cancel did not occur within the time limit")
		return
	}

	// verify success
	t.Logf("verify success")
	cancel = make(chan interface{})
	errch = make(chan error)
	ttl.errch = errch
	go RunTTL(ttl, cancel, 100*time.Millisecond, 1*time.Second)

	errch <- nil
	select {
	case errch <- nil:
		t.Errorf("expecting long signal")
	case <-time.After(500 * time.Millisecond):
		t.Logf("success verified")
	}
	close(cancel)
	close(errch)

	// verify error
	t.Logf("verify error")
	cancel = make(chan interface{})
	errch = make(chan error)
	ttl.errch = errch
	go RunTTL(ttl, cancel, 100*time.Millisecond, 1*time.Second)

	errch <- errors.New("error")
	select {
	case errch <- nil:
		t.Logf("error verified")
	case <-time.After(500 * time.Millisecond):
		t.Errorf("expecting short signal")
	}
	close(cancel)
	close(errch)
}
