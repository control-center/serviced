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

	if err := os.Chmod(name, perm); err != nil {
		return err
	}
	return os.Rename(name, filename)
}
