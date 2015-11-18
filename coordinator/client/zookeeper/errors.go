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

package zookeeper

import (
	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/control-center/serviced/coordinator/client"
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
	case zklib.ErrNoServer:
		return client.ErrNoServer
	}
	return err
}
