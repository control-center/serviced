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

// +build unit

package utils

import (
	"testing"

	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/zenoss/glog"
)

// TestFileLock
func TestFileLock(t *testing.T) {
	t.Skip("filelock with flock/fcntl fails - unused until go sys/unix subrepo is fixed")

	var err error
	hostid, err := HostID()
	if err != nil {
		t.Fatalf("unable to get hostid: %s", err)
	}

	lockdir := os.Getenv("FILELOCK_TESTPATH")
	if len(lockdir) == 0 {
		// create temporary proc dir
		if lockdir, err = ioutil.TempDir("", "file_lock"); err != nil {
			t.Fatalf("could not create tempdir %+v: %s", lockdir, err)
		}
		defer os.RemoveAll(lockdir)
	}

	if err := os.MkdirAll(lockdir, 0755); err != nil {
		t.Fatalf("unable to mkdir %+v: %s", lockdir, err)
	}

	filelock := path.Join(lockdir, "file.lck")
	lockuser := func(id int, events chan string) {
		glog.V(0).Infof("%s:%d locking  %s", hostid, id, lockdir)
		fp, err := LockFile(filelock)
		if err != nil {
			t.Fatalf("unable to lock %+v: %s", filelock, err)
		}
		defer fp.Close()
		defer os.Remove(filelock) // uncomment to test Open(O_EXCL)
		fp.WriteString(fmt.Sprintf("%s:%d\n", hostid, id))
		events <- "locked"
		duration := 5 * time.Second
		glog.V(0).Infof("%s:%d locked    %s - sleeping %s", hostid, id, lockdir, duration)
		time.Sleep(duration)
		events <- "done"
		glog.V(0).Infof("%s:%d unlocking %s", hostid, id, lockdir)
	}

	events := make(chan string)
	expected := []string{"locked", "done", "locked", "done", "locked", "done", "locked", "done"}
	numClients := len(expected) / 2
	for id := 0; id < numClients; id++ {
		go lockuser(id, events)
	}

	reported := []string{}
WAIT_FOR_REPORTERS:
	for {
		glog.V(2).Infof("waiting for reporters - reported: %+v", reported)
		select {
		case <-time.After(time.Duration(numClients*5+numClients) * time.Second):
			break WAIT_FOR_REPORTERS

		case ev := <-events:
			reported = append(reported, ev)
			glog.V(2).Infof("reported: %+v", reported)
		}
	}

	if !reflect.DeepEqual(expected, reported) {
		t.Fatalf("expected: %+v != actual: %+v", expected, reported)
	}
}
