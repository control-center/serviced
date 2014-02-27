// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package rsync

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/volume"

	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

const (
	DriverName = "rsync"
)

type RsyncDriver struct {
}

type RsyncConn struct {
	name string
	root string
}

func init() {
	rsyncdriver, err := New()
	if err != nil {
		glog.Errorf("Can't create rsync driver", err)
		return
	}

	volume.Register(DriverName, rsyncdriver)
}

func New() (*RsyncDriver, error) {
	return &RsyncDriver{}, nil
}

func (d *RsyncDriver) Mount(volumeName, rootDir string) (volume.Conn, error) {
	conn := &RsyncConn{name: volumeName, root: rootDir}
	if err := os.MkdirAll(conn.Path(), 0775); err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *RsyncConn) Name() string {
	return c.name
}

func (c *RsyncConn) Path() string {
	return path.Join(c.root, c.name)
}

func (c *RsyncConn) Snapshot(label string) (err error) {
	dest := path.Join(c.root, label)
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
	glog.V(0).Infof("Performing snapshot rsync command: %s %s", exe, argv)
	rsync := exec.Command(exe, argv...)
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}

func (c *RsyncConn) Snapshots() (labels []string, err error) {
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

func (c *RsyncConn) RemoveSnapshot(label string) error {
	parts := strings.Split(label, "_")
	if len(parts) != 2 {
		return fmt.Errorf("malformed label: %s", label)
	}
	if parts[0] != c.name {
		return fmt.Errorf("label %s refers to some other volume", label)
	}
	sh := exec.Command("rm", "-Rf", path.Join(c.root, label))
	glog.V(4).Infof("About to execute: %s", sh)
	output, err := sh.CombinedOutput()
	if err != nil {
		glog.Errorf("could not remove snapshot: %s", string(output))
		return fmt.Errorf("could not remove snapshot: %s", label)
	}
	return nil
}

func (c *RsyncConn) Rollback(label string) (err error) {
	src := path.Join(c.root, label)
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
