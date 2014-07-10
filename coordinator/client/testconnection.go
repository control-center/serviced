package client

import (
	"bytes"
	"encoding/json"
	"path"
	"sort"
	"strings"
)

// TestConnection is a test connection type
type TestConnection struct {
	id      int
	nodes   map[string][]byte
	watches map[string]chan<- Event
	Err     error // Connection Error set by the user
}

// NewTestConnection initializes a new test connection
func NewTestConnection() *TestConnection {
	return new(TestConnection).init()
}

func (conn *TestConnection) init() *TestConnection {
	conn = &TestConnection{
		nodes:   map[string][]byte{"/": nil},
		watches: make(map[string]chan<- Event),
	}
	return conn
}

func (conn *TestConnection) checkpath(p *string) error {
	*p = path.Clean(*p)
	return conn.Err
}

func (conn *TestConnection) updatewatch(p string, eventtype EventType) {
	if watch := conn.watches[p]; watch != nil {
		delete(conn.watches, p)
		watch <- Event{eventtype, p, nil}
	}

	parent := path.Dir(p) + "/"
	if watch := conn.watches[parent]; watch != nil {
		delete(conn.watches, parent)
		watch <- Event{EventNodeChildrenChanged, parent, nil}
	}
}

func (conn *TestConnection) addwatch(p string) <-chan Event {
	eventC := make(chan Event, 1)
	watch := conn.watches[p]
	conn.watches[p] = eventC
	if watch != nil {
		watch <- Event{EventNotWatching, p, nil}
	}
	return eventC
}

// Close implements Connection.Close
func (conn *TestConnection) Close() {
	var dirW, nodeW []string

	// Organize by child watch vs node watch
	for p := range conn.watches {
		if strings.HasSuffix(p, "/") {
			dirW = append(dirW, p)
		} else {
			nodeW = append(nodeW, p)
		}
	}

	// Signal the top-level path first and continue down the node tree
	sort.Strings(dirW)
	for _, d := range dirW {
		conn.updatewatch(d, EventSession)
	}

	// Signal all of the nodes
	for _, n := range nodeW {
		conn.updatewatch(n, EventSession)
	}
}

// SetID implements Connection.SetID
func (conn *TestConnection) SetID(id int) { conn.id = id }

// ID implements Connection.ID
func (conn *TestConnection) ID() int { return conn.id }

// SetOnClose implements Connection.SetOnClose
func (conn *TestConnection) SetOnClose(f func(int)) {}

// Create implements Connection.Create
func (conn *TestConnection) Create(p string, node Node) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	if node := conn.nodes[p]; node != nil {
		return ErrNodeExists
	} else if err := conn.CreateDir(p); err != nil {
		return err
	}

	data, err := json.Marshal(node)
	if err != nil {
		conn.nodes[p] = data
	}
	conn.nodes[p] = data
	return nil
}

// CreateDir implements Connection.CreateDir
func (conn *TestConnection) CreateDir(p string) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	if _, exists := conn.nodes[p]; exists {
		return nil
	} else if err := conn.CreateDir(path.Dir(p)); err != nil {
		return err
	}
	conn.nodes[p] = nil
	conn.updatewatch(p, EventNodeCreated)
	return nil
}

// Exists implements Connection.Exists
func (conn *TestConnection) Exists(p string) (bool, error) {
	if err := conn.checkpath(&p); err != nil {
		return false, err
	}

	_, exists := conn.nodes[p]
	return exists, nil
}

// Delete implements Connection.Delete
func (conn *TestConnection) Delete(p string) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	children, _ := conn.Children(p)
	for _, c := range children {
		if err := conn.Delete(path.Join(p, c)); err != nil {
			return err
		}
	}
	if _, ok := conn.nodes[p]; ok {
		delete(conn.nodes, p)
	}
	conn.updatewatch(p, EventNodeDeleted)
	return nil
}

// ChildrenW implements Connection.ChildrenW
func (conn *TestConnection) ChildrenW(p string) ([]string, <-chan Event, error) {
	if err := conn.checkpath(&p); err != nil {
		return nil, nil, err
	}

	children, err := conn.Children(p)
	if err != nil {
		return nil, nil, err
	}

	return children, conn.addwatch(p + "/"), nil
}

// Children implements Connection.Children
func (conn *TestConnection) Children(p string) (children []string, err error) {
	if err := conn.checkpath(&p); err != nil {
		return nil, err
	}

	pattern := path.Join(p, "*")
	for nodepath, _ := range conn.nodes {
		if match, _ := path.Match(pattern, nodepath); match {
			children = append(children, strings.TrimPrefix(nodepath, p+"/"))
		}
	}

	return children, nil
}

// GetW implements Connection.GetW
func (conn *TestConnection) GetW(p string, node Node) (<-chan Event, error) {
	if err := conn.checkpath(&p); err != nil {
		return nil, err
	}

	if err := conn.Get(p, node); err != nil {
		return nil, err
	}

	return conn.addwatch(p), nil
}

// Get implements Connection.Get
func (conn *TestConnection) Get(p string, node Node) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	if data, ok := conn.nodes[p]; !ok {
		return ErrNoNode
	} else if data == nil {
		return ErrEmptyNode
	} else if err := json.Unmarshal(data, node); err != nil {
		return err
	}
	return nil
}

// Set implements Connection.Set
func (conn *TestConnection) Set(p string, node Node) error {
	if err := conn.checkpath(&p); err != nil {
		return err
	}

	if _, ok := conn.nodes[p]; !ok {
		return ErrNoNode
	}

	data, err := json.Marshal(node)
	if err != nil {
		return err
	}

	// only update if something actually changed
	if bytes.Compare(conn.nodes[p], data) != 0 {
		conn.nodes[p] = data
		conn.updatewatch(p, EventNodeDataChanged)
	}
	return nil
}

// NewLock implements Connection.NewLock
func (conn *TestConnection) NewLock(path string) Lock {
	return nil
}

// NewLeader implements Connection.NewLeader
func (conn *TestConnection) NewLeader(path string, data Node) Leader {
	return nil
}

// CreateEphemeral implements Connection.CreateEphemeral
func (conn *TestConnection) CreateEphemeral(path string, node Node) (string, error) {
	if err := conn.Create(path, node); err != nil {
		return "", err
	}
	return path, nil
}