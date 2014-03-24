package client

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/zenoss/serviced/coordinator/client/retry"
)

var (
	ErrDriverAlreadyRegistered = errors.New("coord-client: driver already registered")
	ErrDriverNotFound          = errors.New("coord-client: flavor not found")
	ErrNodeExists              = errors.New("coord-client: node exists")
	ErrInvalidMachines         = errors.New("coord-client: invalid servers list")
	ErrInvalidMachine          = errors.New("coord-client: invalid machine")
	ErrInvalidRetryPolicy      = errors.New("coord-client: invalid retry policy")
	ErrConnectionNotFound      = errors.New("coord-client: connection not found")
)

type regDriversType struct {
	driverMap map[string]func([]string, time.Duration) (Driver, error)
	sync.Mutex
}

var (
	registeredDrivers = regDriversType{
		driverMap: make(map[string]func([]string, time.Duration) (Driver, error)),
	}
)

func RegisterDriver(name string, driver func([]string, time.Duration) (Driver, error)) error {
	registeredDrivers.Lock()
	defer registeredDrivers.Unlock()
	if _, found := registeredDrivers.driverMap[name]; !found {
		registeredDrivers.driverMap[name] = driver
		return nil
	}
	return ErrDriverAlreadyRegistered
}

func RegisteredDrivers() []string {
	names := make([]string, len(registeredDrivers.driverMap))
	i := 0
	for key, _ := range registeredDrivers.driverMap {
		names[i] = key
		i++
	}
	return names
}

type opClientRequestType int

const (
	opClientRequestConnection opClientRequestType = iota
	opClientCloseConnection
	opClientClose
)

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
	machines    []string
	timeout     time.Duration
	done        chan struct{}
	retryPolicy retry.Policy
	*sync.RWMutex
	opRequests    chan opClientRequest
	driverFactory func([]string, time.Duration) (Driver, error)
}

func DefaultRetryPolicy() retry.Policy {
	return retry.NTimes(30, time.Millisecond*50)
}

func New(machines []string, timeout time.Duration, flavor string, retryPolicy retry.Policy) (client *Client, err error) {
	if len(machines) == 0 {
		return nil, ErrInvalidMachines
	}
	for _, machine := range machines {
		if len(machine) == 0 {
			return nil, ErrInvalidMachines
		}
	}
	drv, found := registeredDrivers.driverMap[flavor]
	if !found {
		return nil, ErrDriverNotFound
	}
	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}
	client = &Client{
		machines:      machines,
		timeout:       timeout,
		done:          make(chan struct{}),
		retryPolicy:   retryPolicy,
		driverFactory: drv,
		opRequests:    make(chan opClientRequest),
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

					log.Printf("CreateDir(%s) ", currentPath)
					err = conn.CreateDir(currentPath)
					if err == ErrNodeExists {
						log.Printf("%s exists", currentPath)
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
	connections := make(map[int]*Driver)
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
				c, err := client.driverFactory(client.machines, client.timeout)
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
				log.Printf("# of connections: %d", len(connections))
			}

		case <-client.done:
			log.Printf("Closing client")
			for i, c := range connections {
				log.Printf("closing connection %d", i)
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

func (client *Client) GetConnection() (Driver, error) {
	request := newOpClientRequest(opClientRequestConnection, nil)
	client.opRequests <- request
	response := <-request.response
	switch response.(type) {
	case error:
		return nil, response.(error)
	case Driver:
		return response.(Driver), nil
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

func (client *Client) SetTimeout(timeout time.Duration) {
	client.timeout = timeout
}
