// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package atomicfile

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// WriteFile will write the given data to the filename in an atomic manner so that
// partial writes are not possible.
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	// find the dirname of the filename

	d := filepath.Dir(filename)
	if err := os.MkdirAll(d, 0755); err != nil {
		return err
	}

	tempfile, err := ioutil.TempFile(d, filepath.Base(filename))
	if err != nil {
		return err
	}
	name := tempfile.Name()
	defer os.Remove(name)

	if err := tempfile.Close(); err != nil {
		return err
	}

	if err := ioutil.WriteFile(name, data, perm); err != nil {
		return err
	}
	return os.Rename(name, filename)
}
