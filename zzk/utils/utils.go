// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package utils

import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/coordinator/client"
)

// PathExists ensures the error from the coordinator is not an ErrNoNode
// USE THIS INSTEAD OF conn.Exists!!!!
func PathExists(conn client.Connection, path string) (bool, error) {
	exists, err := conn.Exists(path)
	if err != nil {
		if err != client.ErrNoNode {
			glog.Errorf("Error with pathExists.Exists(%s) %+v", path, err)
			return false, err
		}
		exists = false
	}
	return exists, nil
}