package zk_driver

import (
	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/coordinator/client"

	"errors"
)

var ErrUnimplemented = errors.New("unimplemented")

func xlateError(err error) error {

	switch err {
	case zklib.ErrConnectionClosed:
		return client.ErrConnectionClosed
	case zklib.ErrUnknown:
		return client.ErrUnknown
	case zklib.ErrNoNode:
		return client.ErrNoNode
	case zklib.ErrNoAuth:
		return client.ErrNoAuth
	case zklib.ErrBadVersion:
		return client.ErrBadVersion
	case zklib.ErrConnectionClosed:
		return client.ErrNoChildrenForEphemerals
	case zklib.ErrNoChildrenForEphemerals:
		return client.ErrConnectionClosed
	case zklib.ErrNotEmpty:
		return client.ErrNotEmpty
	case zklib.ErrSessionExpired:
		return client.ErrSessionExpired
	case zklib.ErrInvalidACL:
		return client.ErrInvalidACL
	case zklib.ErrAuthFailed:
		return client.ErrAuthFailed
	case zklib.ErrClosing:
		return client.ErrClosing
	case zklib.ErrNothing:
		return client.ErrNothing
	case zklib.ErrSessionMoved:
		return client.ErrSessionMoved
	}
	return err
}
