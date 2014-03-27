package client

import (
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/zenoss/serviced/coordinator/client/retry"
)

var callTimes int

type mockDriver struct {
	machines []string
	timeout  time.Duration
}

type mockConnection struct {
	onClose **func()
}

func newMockDriver(machines []string, timeout time.Duration) (driver Driver, err error) {
	driver = &mockDriver{
		machines: machines,
		timeout:  timeout,
	}
	return driver, err
}

func (driver *mockDriver) GetConnection() (Connection, error) {
	return &mockConnection{
		onClose: new(*func()),
	}, nil
}

func (conn *mockConnection) SetOnClose(f func()) {
	log.Printf("calling set on close")
	*conn.onClose = &f
}

func (conn *mockConnection) Close() {
	log.Printf("in driver.Close()")
	if *conn.onClose != nil {
		log.Printf("calling onClose pointer")
		(*(*conn.onClose))()
	}
}

func (conn *mockConnection) Create(path string, data []byte) error {
	callTimes++
	if callTimes > 30 {
		return nil
	}
	return ErrNodeExists
}

func (conn *mockConnection) CreateDir(path string) error {
	callTimes++
	if callTimes > 30 {
		return nil
	}
	return ErrNodeExists
}

func (conn *mockConnection) Exists(path string) (bool, error) {
	return false, nil
}

func (conn *mockConnection) Delete(path string) error {
	return nil
}

func (conn *mockConnection) Unlock(path, lockId string) error {
	return nil
}

func (conn *mockConnection) Lock(path string) (lockId string, err error) {
	return "", nil
}

func TestRegisteredDrivers(t *testing.T) {

	if drivers := RegisteredDrivers(); !reflect.DeepEqual(drivers, []string{}) {
		t.Logf("Expected no drivers, got %v", drivers)
		t.FailNow()
	}

	driver, _ := newMockDriver([]string{}, time.Second)

	if err := RegisterDriver("mock", driver); err != nil {
		t.Logf("Expected no error when registering mock driver: %s", err)
		t.FailNow()
	}

	if drivers := RegisteredDrivers(); !reflect.DeepEqual(drivers, []string{"mock"}) {
		t.Logf("Expected only 'mock' driver, got %v", drivers)
		t.FailNow()
	}

	if err := RegisterDriver("mock", driver); err != ErrDriverAlreadyRegistered {
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
	connection, _ := client.GetConnection()
	connection.Close()
	client.NewRetryLoop(
		func(cancelChan chan chan error) chan error {
			t.Logf("running callable")
			errc := make(chan error)
			go func() {
				t.Logf("getting connection")
				var conn Connection
				var err error
				result := make(chan bool)
				go func() {
					conn, err = client.GetConnection()
					result <- true
				}()
				select {
				case <-result:
				case canit := <-cancelChan:
					canit <- err
					return
				}
				if err != nil {
					errc <- err
					return
				}
				errc <- conn.CreateDir("/foo")
			}()
			return errc
		}).Wait()
	defer client.Close()

}
