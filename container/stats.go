// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package container

import (
	"time"
	"io/ioutil"
	"path"
)

func statReporter(interval time.Duration, closing chan struct{}) {

	tick := time.Tick(interval)
	for {
		switch {
		case <-tick:
			collect()
		case <-closing:
			break
	}
}


func readInt64Stats(dir string) (results map[string]int64, err error) {
	finfos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	results = make(map[string]int64)
	for _, i := range finfos {
		if finfos[i].IsDir() {
			continue
		}
		fname := path.Join(dir, finfos[i].Name())
		data, err := ReadFile(fname)
		if err != nil {
			return err
		}
		i, err := strconv.ParseInt(string.Trim(string(data)), 10, 64)
		if err != nil {
			return err
		}
		results[finfos[i].Name()] = i
	}
	return results, nil
}

