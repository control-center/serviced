package subprocess

import (
	"testing"
	"time"
)

func TestSubprocess(t *testing.T) {
	s, err := New(time.Millisecond, time.Second*5, "sleep", "1")
	if err != nil {
		t.Fatalf("expected subprocess to start: %s", err)
	}
	<-time.After(time.Millisecond * 3500)
	if s.restarts != 3 {
		t.Fatalf("Expected 3 restarts, got %d", s.restarts)
	}

	timeout := time.AfterFunc(time.Millisecond*500, func() {
		t.Fatal("Should have closed subprocess already!")
	})
	s.Close()
	timeout.Stop()
}
