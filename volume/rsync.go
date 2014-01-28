// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package volume

import (
	"github.com/zenoss/glog"

	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

type RsyncVolume struct {
	baseDir string
	name    string
}

func (v *RsyncVolume) New(baseDir, name string) (Volume, error) {

	volume := &RsyncVolume{
		baseDir: baseDir,
		name:    name,
	}
	if err := os.MkdirAll(volume.Dir(), 0775); err != nil {
		return nil, err
	}
	return volume, nil
}

func (v *RsyncVolume) Dir() string {
	return path.Join(v.baseDir, v.name)
}

func (v *RsyncVolume) Name() (name string) {
	return v.name
}

func (v *RsyncVolume) Snapshot(label string) (err error) {

	dest := path.Join(v.baseDir, label)
	if exists, err := isDir(dest); exists || err != nil {
		if exists {
			return errors.New("snapshot already exists")
		}
		return err
	}
	rsync := exec.Command("rsync", "-a", v.Dir()+"/", dest+"/")
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}

func (v *RsyncVolume) Snapshots() (labels []string, err error) {

	var infos []os.FileInfo
	infos, err = ioutil.ReadDir(v.Dir())
	if err != nil {
		return nil, err
	}
	labels = make([]string, 0)
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		if strings.HasPrefix(info.Name(), v.name+"_") {
			labels = append(labels, info.Name())
		}
	}
	return labels, nil
}

func (v *RsyncVolume) RemoveSnapshot(label string) error {
	sh := exec.Command("rm", "-Rf", path.Join(v.BaseDir(), label))
	output, err := sh.CombinedOutput()
	if err != nil {
		glog.Errorf("could not remove snapshot: %s", string(output))
		return errors.New("could not remove snapshot")
	}
	return nil
}

func (v *RsyncVolume) Rollback(label string) (err error) {
	src := path.Join(v.baseDir, label)
	if exists, err := isDir(src); !exists || err != nil {
		if !exists {
			return errors.New("snapshot does not exists")
		}
		return err
	}
	rsync := exec.Command("rsync", "-a", "--del", "--force", src+"/", v.Dir()+"/")
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}

func (v *RsyncVolume) BaseDir() string {
	return v.baseDir
}
