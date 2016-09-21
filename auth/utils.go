// Copyright 2016 The Serviced Authors.
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

package auth

import (
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// NotifyOnChange watches a file for changes that match the given operation set
// and notifies on a channel when they occur.
func NotifyOnChange(filename string, ops fsnotify.Op, cancel <-chan interface{}) (<-chan struct{}, error) {
	filename = filepath.Clean(filename)
	dir := filepath.Dir(filename)
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(dir); err != nil {
		return nil, err
	}
	outchan := make(chan struct{})
	go func() {
		defer w.Close()
		defer close(outchan)
		for {
			select {
			case e := <-w.Events:
				if filepath.Clean(e.Name) == filename && e.Op&ops != 0 {
					outchan <- struct{}{}
				}
			case <-w.Errors:
			case <-cancel:
				return
			}
		}
	}()
	return outchan, nil
}
