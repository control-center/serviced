package client

import (
	"errors"
	"time"
	"github.com/zenoss/serviced/coordinator/client/retry"
)


var (
	ErrInvalidZkServers = errors.New("coord-client: invalid zk servers list")
	ErrInvalidRetryPolicy = errors.New("coord-client: invalid retry policy")
)

type Client struct {
	zkServers []string
	timeout time.Duration
	done chan struct{}
	retryPolicy retry.Policy
}

func New(zkServers []string, timeout time.Duration, retryPolicy retry.Policy) (client *Client, err error) {
	if len(zkServers) == 0 {
		return nil, ErrInvalidZkServers
	}
	for _, server := range zkServers {
		if len(server) == 0 {
			return nil, ErrInvalidZkServers
		}
	}
	client = &Client{
		zkServers: zkServers,
		timeout: timeout,
		done: make(chan struct{}),
		retryPolicy: retryPolicy,
	}
	return client, nil
}

func (client *Client) Close() {
	client.done <- struct{}{}
}

func (client *Client) SetRetryPolicy (policy retry.Policy) error {
	if policy == nil {
		return ErrInvalidRetryPolicy
	}
	client.retryPolicy = policy
	return nil
}

func (client *Client) SetTimeout(timeout time.Duration) {
	client.timeout = timeout
}

