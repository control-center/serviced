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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package node

import (
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/zenoss/glog"

	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// validOwnerSpec returns true if the owner is specified in owner:group format and the
// identifiers are valid POSIX.1-2008 username and group strings, respectively.
func validOwnerSpec(owner string) bool {
	var pattern = regexp.MustCompile(`^[a-zA-Z]+[a-zA-Z0-9.-]*:[a-zA-Z]+[a-zA-Z0-9.-]*$`)
	return pattern.MatchString(owner)
}

// GetInterfaceIPAddress attempts to find the IP address based on interface name
func GetInterfaceIPAddress(_interface string) (string, error) {
	output, err := exec.Command("/sbin/ip", "-4", "-o", "addr").Output()
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		if strings.HasPrefix(fields[1], _interface) {
			return strings.Split(fields[3], "/")[0], nil
		}
	}

	return "", fmt.Errorf("Unable to find ip for interface: %s", _interface)
}

// getIPAddrFromOutGoingConnection get the IP bound to the interface which
// handles the default route traffic.
func getIPAddrFromOutGoingConnection() (ip string, err error) {
	addr, err := net.ResolveUDPAddr("udp4", "8.8.8.8:53")
	if err != nil {
		return "", err
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return "", err
	}

	localAddr := conn.LocalAddr()
	parts := strings.Split(localAddr.String(), ":")
	return parts[0], nil
}

// ExecPath returns the path to the currently running executable.
func ExecPath() (string, string, error) {
	path, err := getExecPath()
	if err != nil {
		return "", "", err
	}
	return filepath.Dir(path), filepath.Base(path), nil
}

// GetDockerVersion returns docker version number.
func GetDockerVersion() ([]int, error) {
	dc, err := docker.NewClient()
	if err != nil {
		return nil, err
	}
	env, err := dc.Version()
	if err != nil {
		return nil, err
	}
	versionString := env.Get("Version")
	versionSplit := strings.Split(versionString, ".")
	version := make([]int, len(versionSplit))
	for i, v := range versionSplit {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		version[i] = n
	}
	return version, nil
}

// CreateDirectory creates a directory using the given username as the owner and the
// given perm as the directory permission.
func CreateDirectory(path, username string, perm os.FileMode) error {
	user, err := user.Lookup(username)
	if err == nil {
		err = os.MkdirAll(path, perm)
		if err == nil || err == os.ErrExist {
			uid, _ := strconv.Atoi(user.Uid)
			gid, _ := strconv.Atoi(user.Gid)
			err = os.Chown(path, uid, gid)
		}
	}
	return err
}

// singleJoiningSlash joins a and b ensuring there is only a single /
// character between them.
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// NewReverseProxy differs from httputil.NewSingleHostReverseProxy in that it rewrites
// the path so that it does /not/ include the incoming path. e.g. request for
// "/mysvc/thing" when proxy is served from "/mysvc" means target is
// targeturl.Path + "/thing"; vs. httputil.NewSingleHostReverseProxy, in which
// it would be targeturl.Path + "/mysvc/thing".
func NewReverseProxy(path string, targeturl *url.URL) *httputil.ReverseProxy {
	targetQuery := targeturl.RawQuery
	director := func(r *http.Request) {
		r.URL.Scheme = targeturl.Scheme
		r.URL.Host = targeturl.Host
		newpath := strings.TrimPrefix(r.URL.Path, path)
		r.URL.Path = singleJoiningSlash(targeturl.Path, newpath)
		if targetQuery == "" || r.URL.RawQuery == "" {
			r.URL.RawQuery = targetQuery + r.URL.RawQuery
		} else {
			r.URL.RawQuery = targetQuery + "&" + r.URL.RawQuery
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

// Assumes that the local docker image (imageSpec) exists and has been sync'd
// with the registry.
var dockerRun = func(imageSpec string, args ...string) (output string, err error) {
	targs := []string{"run", imageSpec}
	for _, s := range args {
		targs = append(targs, s)
	}
	docker := exec.Command("docker", targs...)
	var outputBytes []byte
	outputBytes, err = docker.Output()
	if err != nil {
		return
	}
	output = string(outputBytes)
	return
}

type uidgid struct {
	uid int
	gid int
}

var userSpecCache struct {
	lookup map[string]uidgid
	sync.Mutex
}

func init() {
	userSpecCache.lookup = make(map[string]uidgid)
}

// Assumes that the local docker image (imageSpec) exists and has been sync'd
// with the registry.
func getInternalImageIDs(userSpec, imageSpec string) (uid, gid int, err error) {

	userSpecCache.Lock()
	defer userSpecCache.Unlock()

	key := userSpec + "!" + imageSpec
	if val, found := userSpecCache.lookup[key]; found {
		return val.uid, val.gid, nil
	}

	var output string
	// explicitly ignoring errors because of -rm under load
	output, _ = dockerRun(imageSpec, "/bin/sh", "-c",
		fmt.Sprintf(`touch test.txt && chown %s test.txt && ls -ln test.txt | awk '{ print $3, $4 }'`,
			userSpec))

	s := strings.TrimSpace(string(output))
	pattern := regexp.MustCompile(`^\d+ \d+$`)

	if !pattern.MatchString(s) {
		err = fmt.Errorf("unexpected output from getInternalImageIDs: %s", s)
		return
	}
	fields := strings.Fields(s)
	if len(fields) != 2 {
		err = fmt.Errorf("unexpected number of fields from container spec: %s", fields)
		return
	}
	uid, err = strconv.Atoi(fields[0])
	if err != nil {
		return
	}
	gid, err = strconv.Atoi(fields[1])
	if err != nil {
		return
	}
	// cache the results
	userSpecCache.lookup[key] = uidgid{uid: uid, gid: gid}
	time.Sleep(time.Second)
	return
}

// createVolumeDir() creates a directory on the running host using the user ids
// found within the specified image. For example, it can create a directory owned
// by the mysql user (as seen by the container) despite there being no mysql user
// on the host system.
// Assumes that the local docker image (imageSpec) exists and has been sync'd
// with the registry.
func createVolumeDir(conn client.Connection, hostPath, containerSpec, imageSpec, userSpec, permissionSpec string) error {

	// use zookeeper lock of basename of hostPath (volume name)
	zkVolumeInitLock := path.Join("/locks/volumeinit", filepath.Base(hostPath))
	lock, err := conn.NewLock(zkVolumeInitLock)
	if err != nil {
		glog.Errorf("Could not initialize lock for %s: %s", zkVolumeInitLock, err)
		return err
	}
	if err := lock.Lock(); err != nil {
		glog.Errorf("Could not acquire lock for %s: %s", zkVolumeInitLock, err)
		return err
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			glog.Errorf("Could not unlock %s: %s", zkVolumeInitLock, err)
		}
	}()

	// return if service volume has been initialized
	dotfileCompatibility := path.Join(hostPath, ".serviced.initialized") // for compatibility with previous versions of serviced
	dotfileHostPath := path.Join(filepath.Dir(hostPath), fmt.Sprintf(".%s.serviced.initialized", filepath.Base(hostPath)))
	dotfiles := []string{dotfileCompatibility, dotfileHostPath}
	for _, dotfileHostPath := range dotfiles {
		_, err := os.Stat(dotfileHostPath)
		if err == nil {
			glog.V(2).Infof("DFS volume initialized earlier for src:%s dst:%s image:%s user:%s perm:%s", hostPath, containerSpec, imageSpec, userSpec, permissionSpec)
			return nil
		}
	}

	// start initializing dfs volume dir with dir in image
	starttime := time.Now()

	var output []byte
	command := [...]string{
		"docker", "run",
		"--rm", "--user=root", "--workdir=/",
		"-v", hostPath + ":/mnt/dfs",
		imageSpec,
		"/bin/bash", "-c",
		fmt.Sprintf(`
set -e
if [ ! -d "%s" ]; then
	echo "WARNING: DFS mount %s does not exist in image %s"
else
	cp -rp %s/. /mnt/dfs/
fi
chown %s /mnt/dfs
chmod %s /mnt/dfs
sync
`, containerSpec, containerSpec, imageSpec, containerSpec, userSpec, permissionSpec),
	}

	for i := 0; i < 2; i++ {
		docker := exec.Command(command[0], command[1:]...)
		output, err = docker.CombinedOutput()
		if err == nil {
			duration := time.Now().Sub(starttime)
			if strings.Contains(string(output), "WARNING:") {
				glog.Warning(string(output))
			} else {
				glog.Info(string(output))
			}
			glog.Infof("DFS volume init #%d took %s for src:%s dst:%s image:%s user:%s perm:%s", i, duration, hostPath, containerSpec, imageSpec, userSpec, permissionSpec)

			if e := ioutil.WriteFile(dotfileHostPath, []byte(""), 0664); e != nil {
				glog.Errorf("unable to create DFS volume initialized dotfile %s: %s", dotfileHostPath, e)
				return e
			}
			return nil
		}
		time.Sleep(time.Second)
		glog.Warningf("retrying due to error creating DFS volume %+v: %s", hostPath, string(output))
	}

	glog.Errorf("could not create DFS volume %+v: %s", hostPath, string(output))
	return err
}

// In the container
func AddToEtcHosts(host, ip string) error {
	// First make sure /etc/hosts is writeable
	command := []string{
		"/bin/bash", "-c", fmt.Sprintf(`
if [ -n "$(mount | grep /etc/hosts)" ]; then \
	cat /etc/hosts > /tmp/etchosts; \
	umount /etc/hosts; \
	mv /tmp/etchosts /etc/hosts; \
fi; \
echo "%s %s" >> /etc/hosts`, ip, host)}
	return exec.Command(command[0], command[1:]...).Run()
}
