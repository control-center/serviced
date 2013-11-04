package dao

import "testing"


// Test newUuid()
func TestNewUuid(t *testing.T) {
	urandomFilename = "../testfiles/urandom_bytes"

	uuid, err := NewUuid()
	if err != nil {
		t.Errorf("Did not expect error: %s", err)
		t.Fail()
	}
	expectedUuid := "1102c395-e94b-0a08-d1e9-307e31a5213e"
	if uuid != expectedUuid {
		t.Errorf("uuid: expected %s, got %s", expectedUuid, uuid)
		t.Fail()
	}
}

