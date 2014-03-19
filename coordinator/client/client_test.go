package client

import (
	"reflect"
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client/retry"
)

type mockDriver struct{}

var callTimes = 0

func newMockDriver(machines []string, timeout time.Duration) (driver Driver, err error) {
	driver = mockDriver{}
	return driver, err
}

func (driver mockDriver) ValidateMachineList(machines []string) error {
	return nil
}

func (driver mockDriver) Create(path string, data []byte) error {
	callTimes++
	if callTimes > 30 {
		return nil
	}
	return ErrNodeExists
}

func (driver mockDriver) CreateDir(path string) error {
	callTimes++
	if callTimes > 30 {
		return nil
	}
	return ErrNodeExists
}

func (driver mockDriver) Exists(path string) (bool, error) {
	return false, nil
}

func (driver mockDriver) Delete(path string) error {
	return nil
}

func TestRegisteredDrivers(t *testing.T) {

	if drivers := RegisteredDrivers(); !reflect.DeepEqual(drivers, []string{}) {
		t.Logf("Expected no drivers, got %v", drivers)
		t.FailNow()
	}

	if err := RegisterDriver("mock", newMockDriver); err != nil {
		t.Logf("Expected no error when registering mock driver: %s", err)
		t.FailNow()
	}

	if drivers := RegisteredDrivers(); !reflect.DeepEqual(drivers, []string{"mock"}) {
		t.Logf("Expected only 'mock' driver, got %v", drivers)
		t.FailNow()
	}

	if err := RegisterDriver("mock", newMockDriver); err != ErrDriverAlreadyRegistered {
		t.Logf("Expected ErrDriverAlreadyRegistered, got %s", err)
		t.FailNow()
	}
}

func TestNew(t *testing.T) {
	if _, err := New([]string{}, time.Second, "", nil); err != ErrInvalidMachines {
		t.Logf("Expected ErrInvalidMachines got : %s", err)
		t.FailNow()
	}
	if _, err := New([]string{"foo", ""}, time.Second, "", nil); err != ErrInvalidMachines {
		t.Logf("Expected ErrInvalidMachines got : %s", err)
		t.FailNow()
	}

	client, err := New([]string{"foo"}, time.Second, "mock",
		retry.BoundedExponentialBackoff(time.Millisecond*10, time.Second*10, 10))
	if err != nil {
		t.Fatalf("could not create client :%s", err)
	}
	client.NewRetryLoop(
		func(cancelChan chan chan error) chan error {
			t.Logf("running callable")
			errc := make(chan error)
			go func() {
				t.Logf("getting connection")
				conn, err := client.GetConnection()
				if err != nil {
					errc <- err
					return
				}
				errc <- conn.CreateDir("/foo")
			}()
			return errc
		}).Wait()
}
