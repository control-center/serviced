// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package volume

import (
	"fmt"
	"os"
)

// IsDir() checks if the given dir is a directory. If any error is encoutered
// it is returned and directory is set to false.
func IsDir(dirName string) (dir bool, err error) {
	if lstat, err := os.Lstat(dirName); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	} else {
		if !lstat.IsDir() {
			return false, fmt.Errorf("%s is not a directory", dirName)
		}
	}
	return true, nil
}
