package client

import (
	"errors"
	"time"

	"github.com/zenoss/serviced/coordinator/client/retry"
)

var (
	ErrNodeExists         = errors.New("coord-client: node exists")
	ErrInvalidMachines    = errors.New("coord-client: invalid servers list")
	ErrInvalidRetryPolicy = errors.New("coord-client: invalid retry policy")
)

type Client struct {
	machines    []string
	timeout     time.Duration
	done        chan struct{}
	retryPolicy retry.Policy
}

func New(machines []string, timeout time.Duration, retryPolicy retry.Policy) (client *Client, err error) {
	if len(machines) == 0 {
		return nil, ErrInvalidMachines
	}
	for _, machine := range machines {
		if len(machine) == 0 {
			return nil, ErrInvalidMachines
		}
	}
	client = &Client{
		machines:    machines,
		timeout:     timeout,
		done:        make(chan struct{}),
		retryPolicy: retryPolicy,
	}
	return client, nil
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
