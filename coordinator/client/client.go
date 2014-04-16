// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package client

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client/retry"
)

type EventType int32

type Event struct {
	Type EventType
	Path string // For non-session events, the path of the watched node.
	Err  error
}

var (
	EventNodeCreated         = EventType(1)
	EventNodeDeleted         = EventType(2)
	EventNodeDataChanged     = EventType(3)
	EventNodeChildrenChanged = EventType(4)
	EventSession             = EventType(-1)
	EventNotWatching         = EventType(-2)
)

var (
	ErrInvalidNodeType    = errors.New("coord-client: invalid node type")
	ErrSerialization      = errors.New("coord-client: serialization error")
	ErrInvalidDSN         = errors.New("coord-client: invalid DSN")
	ErrDriverDoesNotExist = errors.New("coord-client: driver does not exist")
	ErrNodeExists         = errors.New("coord-client: node exists")
	ErrInvalidMachines    = errors.New("coord-client: invalid servers list")
	ErrInvalidMachine     = errors.New("coord-client: invalid machine")
	ErrInvalidRetryPolicy = errors.New("coord-client: invalid retry policy")
	ErrConnectionNotFound = errors.New("coord-client: connection not found")
)

var (
	ErrConnectionClosed        = errors.New("coord-client: connection closed")
	ErrUnknown                 = errors.New("coord-client: unknown error")
	ErrAPIError                = errors.New("coord-client: api error")
	ErrNoNode                  = errors.New("coord-client: node does not exist")
	ErrNoAuth                  = errors.New("coord-client: not authenticated")
	ErrBadVersion              = errors.New("coord-client: version conflict")
	ErrNoChildrenForEphemerals = errors.New("coord-client: ephemeral nodes may not have children")
	ErrNotEmpty                = errors.New("coord-client: node has children")
	ErrSessionExpired          = errors.New("coord-client: session has been expired by the server")
	ErrInvalidACL              = errors.New("coord-client: invalid ACL specified")
	ErrAuthFailed              = errors.New("coord-client: client authentication failed")
	ErrClosing                 = errors.New("coord-client: zookeeper is closing")
	ErrNothing                 = errors.New("coord-client: no server responsees to process")
	ErrSessionMoved            = errors.New("coord-client: session moved to another server, so operation is ignored")
)

type opClientRequestType int

const (
	opClientRequestConnection opClientRequestType = iota
	opClientCloseConnection
	opClientClose
)

var registeredDrivers = make(map[string]Driver)

// RegisterDriver registers driver under name. This function should only be called
// in a func init().
func RegisterDriver(name string, driver Driver) {

	if _, exists := registeredDrivers[name]; exists {
		panic(name + " driver is already registered")
	}
	registeredDrivers[name] = driver
}

// newOpClientRequest create a client request object.
func newOpClientRequest(reqType opClientRequestType, args interface{}) opClientRequest {
	return opClientRequest{
		op:       reqType,
		args:     args,
		response: make(chan interface{}),
	}
}

//opClientRequest is a client request object; it specifies the request type, its
// args and a response channel.
type opClientRequest struct {
	op       opClientRequestType
	args     interface{}
	response chan interface{}
}

// Client is a coordination client that abstracts using services like etcd or
// zookeeper.
type Client struct {
	basePath          string               // the base path for every connection
	connectionString  string               // the driver specific connection string
	done              chan chan struct{}   // a shutdown channel
	retryPolicy       retry.Policy         // the default retry policy to use
	mutext            sync.RWMutex         // a sync to prevent some race conditions
	opRequests        chan opClientRequest // the request channel the main loop uses
	connectionFactory Driver               // the current driver under use
}

// DefaultRetryPolicy return a retry.Policy of retry.NTimes(30, time.Millisecond*50)
func DefaultRetryPolicy() retry.Policy {
	return retry.NTimes(30, time.Millisecond*50)
}

// New returns a client that will create connections using the given driver and
// connection string. Any retryable operations will use the given retry policy.
func New(driverName, connectionString, basePath string, retryPolicy retry.Policy) (client *Client, err error) {

	var driver Driver
	var exists bool
	if driver, exists = registeredDrivers[driverName]; !exists {
		return nil, ErrDriverDoesNotExist
	}

	if retryPolicy == nil {
		retryPolicy = DefaultRetryPolicy()
	}
	client = &Client{
		connectionString:  connectionString,
		done:              make(chan chan struct{}),
		retryPolicy:       retryPolicy,
		connectionFactory: driver,
		opRequests:        make(chan opClientRequest),
	}
	go client.loop() // start the main loop that listens for requests
	return client, nil
}

// EnsurePath creates the given path with empty nodes if they don't exist; the
// last node in the path is only created if makeLastNode is true.
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

// loop() is the  clients main entrypoint; it responds for requests for connections
// and closing.
func (client *Client) loop() {

	// keep track of outstanding connections
	connections := make(map[int]*Connection)
	// connectionIds are for local identificatin
	var connectionId int

	for {
		select {
		case req := <-client.opRequests:
			switch req.op {
			case opClientCloseConnection:
				id := req.args.(int)
				delete(connections, id)
				if _, found := connections[id]; found {

					req.response <- nil
				} else {
					req.response <- ErrConnectionNotFound
				}
			case opClientRequestConnection:
				c, err := client.connectionFactory.GetConnection(client.connectionString, client.basePath)
				if err == nil {
					// save a reference to the connection locally
					connections[connectionId] = &c
					c.SetId(connectionId)
					c.SetOnClose(func(id int) {
						client.closeConnection(id)
					})
					connectionId++
					// setting up a callback to close the connection in this client
					// if someone calls Close() on the driver reference
					req.response <- c
				} else {
					req.response <- err
				}
			}

		case req := <-client.done:
			// during a shutdown request, close all outstanding connections
			for id, _ := range connections {
				func() {
					defer func() {
						if r := recover(); r != nil {
							glog.Errorf("recovered from: %s", r)
						}
					}()
					(*connections[id]).Close()
				}()
			}
			req <- struct{}{}
			return
		}
	}
}

// closeConnection will request that the main loop close the given connection.
func (client *Client) closeConnection(connectionId int) error {
	request := newOpClientRequest(opClientCloseConnection, connectionId)
	client.opRequests <- request
	response := <-request.response
	return response.(error)
}

// NewRetryLoop returns a retry loop that will call the given cancelable function.
// If the client.Close() method is called, the retry loop will stop. The cancelable
// function must be able to accept a 'chan chan error' which will have a value if
// client.Close() was called. The function must respond to the request or risk
// making the users of this client non-responsive.
func (client *Client) NewRetryLoop(cancelable func(chan chan error) chan error) retry.Loop {
	return retry.NewLoop(client.retryPolicy, cancelable)
}

// GetConnection returns a client.Connection that will be managed by the client.
// Callers should call close() on thier connections when done. The client will also
// close connections if close is called on the client.
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

// Close is a shutdown request. It will shutdown all outstanding connections.
func (client *Client) Close() {
	response := make(chan struct{})
	client.done <- (response)
	<-response
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
