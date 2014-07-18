// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package subprocess

import (
	"testing"
	"time"
)

func TestSubprocess(t *testing.T) {
	s, exited, err := New(time.Second*5, nil, "sleep", "1")
	if err != nil {
		t.Fatalf("expected subprocess to start: %s", err)
	}
	select {
	case <-time.After(time.Millisecond * 1200):
		t.Fatal("expected sleep to finish")
	case <-exited:

	}

	timeout := time.AfterFunc(time.Millisecond*500, func() {
		t.Fatal("Should have closed subprocess already!")
	})
	s.Close()
	timeout.Stop()
}
