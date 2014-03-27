package client

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/zenoss/serviced/coordinator/client/retry"
)

var (
	ErrDriverDoesNotExist = errors.New("coord-client: driver does not exist")
	ErrNodeExists         = errors.New("coord-client: node exists")
	ErrInvalidMachines    = errors.New("coord-client: invalid servers list")
	ErrInvalidMachine     = errors.New("coord-client: invalid machine")
	ErrInvalidRetryPolicy = errors.New("coord-client: invalid retry policy")
	ErrConnectionNotFound = errors.New("coord-client: connection not found")
)

type opClientRequestType int

const (
	opClientRequestConnection opClientRequestType = iota
	opClientCloseConnection
	opClientClose
)

var registeredDrivers = make(map[string]Driver)

func RegisterDriver(name string, driver Driver) {
	if _, exists := registeredDrivers[name]; exists {
		panic(name + " driver is already registered")
	}
	registeredDrivers[name] = driver
}

func newOpClientRequest(reqType opClientRequestType, args interface{}) opClientRequest {
	return opClientRequest{
		op:       reqType,
		args:     args,
		response: make(chan interface{}),
	}
}

type opClientRequest struct {
	op       opClientRequestType
	args     interface{}
	response chan interface{}
}

type Client struct {
	driver      Driver
	connectionString string
	done        chan struct{}
	retryPolicy retry.Policy
	*sync.RWMutex
	opRequests        chan opClientRequest
	connectionFactory Driver
}

func DefaultRetryPolicy() retry.Policy {
	return retry.NTimes(30, time.Millisecond*50)
}

func New(driverName, connectionString string, retryPolicy retry.Policy) (client *Client, err error) {

	var driver Driver
	var exists bool
	if driver, exists = registeredDrivers[driverName]; !exists {
		return nil, ErrDriverDoesNotExist
	}

	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}
	client = &Client{
		driver:            driver,
		connectionString:  connectionString,
		done:              make(chan struct{}),
		retryPolicy:       retryPolicy,
		connectionFactory: driver,
		opRequests:        make(chan opClientRequest),
	}
	go client.loop()
	return client, nil
}

func EnsurePath(client *Client, path string, makeLastNode bool) error {
	return client.NewRetryLoop(
		func(cancelChan chan chan error) chan error {
			errc := make(chan error)
			go func() {
				conn, err := client.GetConnection()
				if err != nil {
					errc <- err
					return
				}

				parts := strings.Split(path, "/")
				lastPartId := len(parts) - 1
				currentPath := ""
				for i, part := range parts {
					if lastPartId == i || i == 0 {
						continue
					}
					currentPath += "/" + part

					err = conn.CreateDir(currentPath)
					if err == ErrNodeExists {
						continue
					}
					errc <- err
					return
				}
				errc <- nil
			}()
			return errc
		}).Wait()
}

func (client *Client) loop() {
	connections := make(map[int]*Connection)
	connectionId := 0

	for {
		select {
		case req := <-client.opRequests:
			switch req.op {
			case opClientCloseConnection:
				connectionId := req.args.(int)
				if connection, found := connections[connectionId]; found {
					(*connection).Close()
					delete(connections, connectionId)
					req.response <- nil
				} else {
					req.response <- ErrConnectionNotFound
				}
			case opClientRequestConnection:
				c, err := client.connectionFactory.GetConnection(client.connectionString)
				// setting up a callback to close the connection in this client
				// if someone calls Close() on the driver reference
				c.SetOnClose(func() {
					client.CloseConnection(connectionId)
				})
				if err == nil {
					connections[connectionId] = &c
					connectionId++
					req.response <- c
				} else {
					req.response <- err
				}
			}

		case <-client.done:
			for _, c := range connections {
				(*c).Close()
			}
			return
		}
	}
}

func (client *Client) CloseConnection(connectionId int) error {
	request := newOpClientRequest(opClientCloseConnection, connectionId)
	client.opRequests <- request
	response := <-request.response
	return response.(error)
}

func (client *Client) NewRetryLoop(cancelable func(chan chan error) chan error) retry.Loop {
	return retry.NewLoop(client.retryPolicy, cancelable)
}

func (client *Client) GetConnection() (Connection, error) {
	request := newOpClientRequest(opClientRequestConnection, nil)
	client.opRequests <- request
	response := <-request.response
	switch response.(type) {
	case error:
		return nil, response.(error)
	case Connection:
		return response.(Connection), nil
	}
	panic("unreachable")
}

func (client *Client) Close() {
	client.done <- struct{}{}
}

func (client *Client) Unregister(id int) {

}

func (client *Client) SetRetryPolicy(policy retry.Policy) error {
	if policy == nil {
		return ErrInvalidRetryPolicy
	}
	client.retryPolicy = policy
	return nil
}
