package client

import (
	"errors"
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

type Client struct {
	machines      []string
	timeout       time.Duration
	done          chan struct{}
	retryPolicy   retry.Policy
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
	}
	return client, nil
}

func (client *Client) NewRetryLoop(cancelable retry.Cancelable) retry.Loop {
	return retry.NewLoop(client.retryPolicy, cancelable)
}

func (client *Client) GetConnection() (Driver, error) {
	return client.driverFactory(client.machines, client.timeout)
}

func (client *Client) Close() {
	client.done <- struct{}{}
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
