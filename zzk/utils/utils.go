package utils

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
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