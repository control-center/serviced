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

package storage

import (
	"github.com/zenoss/glog"

	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"
)

// MockStorageDriver is an interface that mock the storage subsystem
type MockStorageDriver struct {
	exportPath string
}

func (m *MockStorageDriver) ExportPath() string {
	return m.exportPath
}

func (m *MockStorageDriver) SetClients(clients ...string) {
}

func (m *MockStorageDriver) Sync() error {
	return nil
}

func (m *MockStorageDriver) Restart() error {
	return nil
}

func TestMonitorVolume(t *testing.T) {
	// create temporary proc dir
	tmpPath, err := ioutil.TempDir("", "storage")
	if err != nil {
		t.Fatalf("could not create tempdir %+v: %s", tmpPath, err)
	}
	defer os.RemoveAll(tmpPath)

	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		t.Fatalf("unable to mkdir %+v: %s", tmpPath, err)
	}

	// make shutdown channel
	shutdown := make(chan interface{})
	defer close(shutdown)

	// ---- create mock driver
	driver := &MockStorageDriver{
		exportPath: tmpPath,
	}

	// ---- start monitor
	monitor, err := NewMonitor(driver, time.Duration(3*time.Second))
	if err != nil {
		t.Fatalf("unable to create new monitor %s", err)
	}

	updatedCount := 0
	updatedCountLock := sync.RWMutex{}
	pDFSVolumeMonitorPollUpdateFunc := func(mountpoint, remoteIP string, isUpdated bool) {
		glog.Infof("==== received remoteIP:%v isUpdated:%+v", remoteIP, isUpdated)
		if isUpdated {
			updatedCountLock.Lock()
			updatedCount++
			updatedCountLock.Unlock()
		}
	}
	
	exportTime := strconv.FormatInt(time.Now().UnixNano(), 16)

	go monitor.MonitorDFSVolume(tmpPath, "1.2.3.4", exportTime, shutdown, pDFSVolumeMonitorPollUpdateFunc)

	time.Sleep(time.Second * 2)

	// ---- start writer; wait some time; check that the update count increased
	glog.Infof("==== starting writer")
	remoteIP := "127.0.0.1"
	monitor.SetMonitorRemoteHosts(remoteIP)

	remoteShutdown := make(chan interface{})
	defer close(remoteShutdown)
	writeInterval := time.Duration(1 * time.Second)
	go UpdateRemoteMonitorFile(tmpPath, writeInterval, remoteIP, remoteShutdown)

	// wait some time
	waitTime := time.Second * 12
	time.Sleep(waitTime)

	monitorPath := path.Join(tmpPath, monitorSubDir)
	updatedCountLock.RLock()
	if updatedCount < 3 {
		t.Fatalf("have not seen any updates to dir %s", monitorPath)
	}
	updatedCountLock.RUnlock()

	// ---- stop writing; wait some time; check count has not increased
	glog.Infof("==== stopping writer")
	remoteShutdown <- true

	// wait some time for shutdown
	time.Sleep(2 * writeInterval)
	updatedCountLock.Lock()
	updatedCount = 0
	updatedCountLock.Unlock()

	// wait some time
	time.Sleep(waitTime)

	updatedCountLock.RLock()
	if updatedCount != 0 {
		t.Fatalf("updates (updatedCount: %d) should not have occurred to dir %s", updatedCount, monitorPath)
	}

	glog.Infof("==== current updatedCount:%+v", updatedCount)
	updatedCountLock.RUnlock()
}
