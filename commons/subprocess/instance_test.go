package subprocess

import (
	"testing"
	"time"
)

func TestSubprocess(t *testing.T) {
	s, exited, err := New(time.Second*5, "sleep", "1")
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
