// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client/retry"
	"github.com/zenoss/glog"
)

// EventType is a numerical type to identify event types.
type EventType int32

// Event describes a coordination client event
type Event struct {
	Type EventType
	Path string // For non-session events, the path of the watched node.
	Err  error
}

var (
	// EventNodeCreated is emmited when a node is created
	EventNodeCreated = EventType(1)
	// EventNodeDeleted is emitted when a node is deleted
	EventNodeDeleted = EventType(2)
	// EventNodeDataChanged is emitted when a node's data is changed
	EventNodeDataChanged = EventType(3)
	// EventNodeChildrenChanged is emitted when any of the currently watched node's children is changed
	EventNodeChildrenChanged = EventType(4)
	// EventSession is emitted when a disconnect or reconnect happens
	EventSession = EventType(-1)
	// EventNotWatching is emitted when the current watch expired
	EventNotWatching = EventType(-2)
)

var (
	// ErrEmptyNode is returned when a get is performed and the node does not contain data
	ErrEmptyNode = errors.New("coord-client: empty node")
	// ErrInvalidVersionObj is returned when a node's Version() value is not understood by the underlying driver
	ErrInvalidVersionObj = errors.New("coord-client: invalid version object")
	// ErrInvalidPath is returned when the path to store a node is malformed or illegal
	ErrInvalidPath = errors.New("coord-client: invalid path")
	// ErrSerialization is returned when there is an error serializing a node
	ErrSerialization = errors.New("coord-client: serialization error")
	// ErrInvalidDSN is returned when the given DSN can not use interpreted by driver
	ErrInvalidDSN = errors.New("coord-client: invalid DSN")
	// ErrDriverDoesNotExist is returned when the specified driver is not registered
	ErrDriverDoesNotExist = errors.New("coord-client: driver does not exist")
	// ErrNodeExists is returned when an attempt at creating a node at a path where a node already exists
	ErrNodeExists = errors.New("coord-client: node exists")
	// ErrInvalidRetryPolicy is returned when a nil value is for a client policy
	ErrInvalidRetryPolicy = errors.New("coord-client: invalid retry policy")
	// ErrConnectionNotFound is returned when Close(id) is attemped on a connection id that does not exist
	ErrConnectionNotFound = errors.New("coord-client: connection not found")
	// ErrConnectionClosed is returned when an operation is attemped on a closed connection
	ErrConnectionClosed        = errors.New("coord-client: connection is closed")
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
	ErrNoServer                = errors.New("coord-client: could not connect to a server")
)

type opClientRequestType int

const (
	opClientRequestConnection opClientRequestType = iota
	opClientRequestCustomConnection
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
		basePath:          basePath,
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
	pp := strings.Split(path, "/")
	last := len(pp)
	if last == 0 {
		return ErrInvalidPath
	}
	if makeLastNode {
		pp = pp[1:last]
	} else {
		pp = pp[1 : last-1]
	}

	return client.NewRetryLoop(
		func(cancelChan chan chan error) chan error {
			errc := make(chan error)
			go func() {
				conn, err := client.GetConnection()
				if err != nil {
					errc <- err
					return
				}

				p := ""
				for _, part := range pp {
					p += "/" + part
					if err := conn.CreateDir(p); !(err == nil || err == ErrNodeExists) {
						errc <- err
						return
					}
				}
				errc <- nil
			}()
			return errc
		}).Wait()
}

// loop() is the client's main entrypoint; it responds for requests for connections
// and closing.
func (client *Client) loop() {
	// keep track of outstanding connections
	connections := make(map[int]*Connection)
	// connectionIDs are for local identification
	var connectionID int

	for {
		select {
		case req := <-client.opRequests:
			switch req.op {
			case opClientCloseConnection:
				id := req.args.(int)
				if _, found := connections[id]; found {
					delete(connections, id)
					req.response <- nil
				} else {
					req.response <- ErrConnectionNotFound
				}
			case opClientRequestConnection:
				c, err := client.connectionFactory.GetConnection(client.connectionString, client.basePath)
				if err == nil {
					// save a reference to the connection locally
					connections[connectionID] = &c
					c.SetID(connectionID)
					c.SetOnClose(func(id int) {
						client.closeConnection(id)
					})
					connectionID++
					// setting up a callback to close the connection in this client
					// if someone calls Close() on the driver reference
					req.response <- c
				} else {
					req.response <- err
				}
			case opClientRequestCustomConnection:
				myBasePath := req.args.(string)
				c, err := client.connectionFactory.GetConnection(client.connectionString, myBasePath)
				if err == nil {
					// save a reference to the connection locally
					connections[connectionID] = &c
					c.SetID(connectionID)
					c.SetOnClose(func(id int) {
						client.closeConnection(id)
					})
					connectionID++
					// setting up a callback to close the connection in this client
					// if someone calls Close() on the driver reference
					req.response <- c
				} else {
					req.response <- err
				}
			}

		case req := <-client.done:
			// during a shutdown request, close all outstanding connections
			for id := range connections {
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
func (client *Client) closeConnection(connectionID int) error {
	request := newOpClientRequest(opClientCloseConnection, connectionID)
	client.opRequests <- request
	response := <-request.response
	if val, ok := response.(error); ok {
		return val
	}
	return nil
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

func (client *Client) GetCustomConnection(basePath string) (Connection, error) {
	request := newOpClientRequest(opClientRequestCustomConnection, basePath)
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

// SetRetryPolicy sets the given policy on the client
func (client *Client) SetRetryPolicy(policy retry.Policy) error {
	if policy == nil {
		return ErrInvalidRetryPolicy
	}
	client.retryPolicy = policy
	return nil
}

// ConnectionString returns the connection String
func (client *Client) ConnectionString() string {
	return client.connectionString
}
