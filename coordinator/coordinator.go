package coordinator

import (
	"errors"
	"strings"
)

type Coordinator struct {
	driver   Driver
	basePath string
}

var (
	ErrConnectionClosed        = errors.New("coordinator: connection closed")
	ErrUnknown                 = errors.New("coordinator: unknown error")
	ErrAPIError                = errors.New("coordinator: api error")
	ErrNoNode                  = errors.New("coordinator: node does not exist")
	ErrNoAuth                  = errors.New("coordinator: not authenticated")
	ErrBadVersion              = errors.New("coordinator: version conflict")
	ErrNoChildrenForEphemerals = errors.New("coordinator: ephemeral nodes may not have children")
	ErrNodeExists              = errors.New("coordinator: node already exists")
	ErrNotEmpty                = errors.New("coordinator: node has children")
	ErrSessionExpired          = errors.New("coordinator: session has been expired by the server")
	ErrInvalidACL              = errors.New("coordinator: invalid ACL specified")
	ErrAuthFailed              = errors.New("coordinator: client authentication failed")
	ErrClosing                 = errors.New("coordinator: zookeeper is closing")
	ErrNothing                 = errors.New("coordinator: no server responsees to process")
	ErrSessionMoved            = errors.New("coordinator: session moved to another server, so operation is ignored")
	// Returned when an invalid path is used
	ErrInvalidPath = errors.New("coordinator: invalid basepath")
)

// New creates a new Coordinator with the given driver and uses the basePath
// in the underlying coordinator.
func New(driver Driver, basePath string) (coordinator *Coordinator, err error) {

	if strings.HasSuffix(basePath, "/") {
		return nil, ErrInvalidPath
	}

	coordinator = &Coordinator{
		basePath: basePath,
		driver:   driver,
	}

	return coordinator, nil
}
