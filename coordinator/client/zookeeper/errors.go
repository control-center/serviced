// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package zookeeper

import (
	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/serviced/coordinator/client"
)

func xlateError(err error) error {

	switch err {
	case zklib.ErrNodeExists:
		return client.ErrNodeExists
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
	case zklib.ErrNoChildrenForEphemerals:
		return client.ErrNoChildrenForEphemerals
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
