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

package rsync

import (
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"

	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	// DriverName is the name of this rsync volume driver implementation
	DriverName = "rsync"
)

// RsyncDriver is a driver for the rsync volume
type RsyncDriver struct {
	sync.Mutex
}

// RsyncConn is a connection to a rsync volume
type RsyncConn struct {
	timeout time.Duration
	name    string
	root    string
	sync.Mutex
}

func init() {
	rsyncdriver, err := New()
	if err != nil {
		glog.Errorf("Can't create rsync driver", err)
		return
	}

	volume.Register(DriverName, rsyncdriver)
}

// New creates a new RsyncDriver
func New() (*RsyncDriver, error) {
	return &RsyncDriver{}, nil
}

// Mount creates a new subvolume at given root dir
func (d *RsyncDriver) Mount(volumeName, rootDir string) (volume.Conn, error) {
	d.Lock()
	defer d.Unlock()
	conn := &RsyncConn{timeout: 30 * time.Second, name: volumeName, root: rootDir}
	if err := os.MkdirAll(conn.Path(), 0775); err != nil {
		return nil, err
	}
	return conn, nil
}

// List lists all of the folders at given root dir
func (d *RsyncDriver) List(rootDir string) (result []string) {
	if files, err := ioutil.ReadDir(rootDir); err != nil {
		glog.Errorf("Error trying to read from root directory: %s", rootDir)
	} else {
		for _, fi := range files {
			if fi.IsDir() {
				result = append(result, fi.Name())
			}
		}
	}

	return
}

// Name provides the name of the subvolume
func (c *RsyncConn) Name() string {
	return c.name
}

// Path provides the full path to the subvolume
func (c *RsyncConn) Path() string {
	return path.Join(c.root, c.name)
}

func (c *RsyncConn) SnapshotPath(label string) string {
	return path.Join(c.root, label)
}

// Snapshot performs a writable snapshot on the subvolume
func (c *RsyncConn) Snapshot(label string) (err error) {
	c.Lock()
	defer c.Unlock()
	dest := c.SnapshotPath(label)
	if exists, err := volume.IsDir(dest); exists || err != nil {
		if exists {
			return fmt.Errorf("snapshot %s already exists", label)
		}
		return err
	}

	exe, err := exec.LookPath("rsync")
	if err != nil {
		return err
	}
	argv := []string{"-a", c.Path() + "/", dest + "/"}
	glog.Infof("Performing snapshot rsync command: %s %s", exe, argv)
	rsync := exec.Command(exe, argv...)

	done, output, err := make(chan interface{}), []byte{}, nil
	go func() {
		defer close(done)
		output, err = rsync.CombinedOutput()
	}()

	select {
	case <-time.After(c.timeout):
		glog.V(2).Infof("Received signal to kill rsync")
		rsync.Process.Kill()
	case <-done:
	}

	<-done
	if err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}

// Snapshots returns the current snapshots on the volume
func (c *RsyncConn) Snapshots() (labels []string, err error) {
	c.Lock()
	defer c.Unlock()
	var infos []os.FileInfo
	infos, err = ioutil.ReadDir(c.root)
	if err != nil {
		return nil, err
	}
	labels = make([]string, 0)
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		if strings.HasPrefix(info.Name(), c.name+"_") {
			labels = append(labels, info.Name())
		}
	}
	return labels, nil
}

// RemoveSnapshot removes the snapshot with the given label
func (c *RsyncConn) RemoveSnapshot(label string) error {
	c.Lock()
	defer c.Unlock()
	parts := strings.Split(label, "_")
	if len(parts) != 2 {
		return fmt.Errorf("malformed label: %s", label)
	}
	if parts[0] != c.name {
		return fmt.Errorf("label %s refers to some other volume", label)
	}
	sh := exec.Command("rm", "-Rf", c.SnapshotPath(label))
	glog.V(4).Infof("About to execute: %s", sh)
	output, err := sh.CombinedOutput()
	if err != nil {
		glog.Errorf("could not remove snapshot: %s", string(output))
		return fmt.Errorf("could not remove snapshot: %s", label)
	}
	return nil
}

// Unmount deletes the volume and snapshots
func (c *RsyncConn) Unmount() error {

	// Delete all of the snapshots
	snapshots, err := c.Snapshots()
	if err != nil {
		return err
	}

	for _, snapshot := range snapshots {
		if err := c.RemoveSnapshot(snapshot); err != nil {
			return err
		}
	}

	// Delete the volume
	c.Lock()
	defer c.Unlock()
	sh := exec.Command("rm", "-Rf", c.Path())
	glog.V(4).Infof("About to execute: %s", sh)
	output, err := sh.CombinedOutput()
	if err != nil {
		glog.Errorf("could not delete subvolume: %s", string(output))
		return fmt.Errorf("could not delete subvolume: %s", c.Path())
	}
	return nil
}

// Rollback rolls back the volume to the given snapshot
func (c *RsyncConn) Rollback(label string) (err error) {
	c.Lock()
	defer c.Unlock()
	src := c.SnapshotPath(label)
	if exists, err := volume.IsDir(src); !exists || err != nil {
		if !exists {
			return fmt.Errorf("snapshot %s does not exist", label)
		}
		return err
	}
	rsync := exec.Command("rsync", "-a", "--del", "--force", src+"/", c.Path()+"/")
	glog.V(4).Infof("About to execute: %s", rsync)
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}
