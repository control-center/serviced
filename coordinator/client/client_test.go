package client

import (
	"reflect"
	"sort"
	"testing"
)

func TestRegisteredDrivers(t *testing.T) {

	expectedDrivers := []string{"etcd", "zookeeper"}
	registered := RegisteredDrivers()
	sort.Strings(registered)
	if !reflect.DeepEqual(expectedDrivers, registered) {
		t.Logf("Expected: %v, got %v", expectedDrivers, registered)
		t.FailNow()
	}
}
