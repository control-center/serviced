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

package nfs

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/control-center/serviced/commons/atomicfile"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

// Server manages exporting an NFS mount.
type Server struct {
	sync.Mutex
	basePath      string
	exportedName  string
	exportOptions string
	network       string
	clients       map[string]struct{}
	volumes       map[string]int32
	exported      map[string]struct{}
}

var (
	// ErrInvalidExportedName is returned when an exported name is not a valid single directory name
	ErrInvalidExportedName = errors.New("nfs server: invalid exported name")
	// ErrInvalidBasePath is returned when the local path to export is invalid
	ErrInvalidBasePath = errors.New("nfs server: invalid base path")
	// ErrBasePathNotDir is returned when the base path is not a directory
	ErrBasePathNotDir = errors.New("nfs server: base path not a directory")
	// ErrInvalidNetwork is returned when the network specifier does not parse in CIDR format
	ErrInvalidNetwork = errors.New("nfs server: the network value is not CIDR")
)

var (
	mp = utils.GetDefaultMountProc()
)

var (
	osMkdirAll = os.MkdirAll
	osChmod    = os.Chmod
	fsidIdx    int32
)

const defaultDirectoryPerm = 0755

const hostDenyMarker = "# serviced, do not remove past this line"
const hostDenyDefaults = "\n# serviced, do not remove past this line\nrpcbind mountd nfsd statd lockd rquotad : ALL\n\n"

const hostAllowMarker = "# serviced, do not remove past this line"
const hostAllowDefaults = "\n# serviced, do not remove past this line\nrpcbind mountd nfsd statd lockd rquotad : 127.0.0.1"

const etcExportsStartMarker = "\n# --- SERVICED EXPORTS BEGIN ---\n# --- Do not edit this section\n"
const etcExportsEndMarker = "\n# --- SERVICED EXPORTS END ---\n"
const etcExportsRemoveComment = "# serviced removed: "

func verifyExportsDir(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		//handle does not exist
		return osMkdirAll(path, defaultDirectoryPerm)
	}
	if !stat.IsDir() {
		return ErrBasePathNotDir
	}
	if (stat.Mode() & defaultDirectoryPerm) != defaultDirectoryPerm {
		err = osChmod(path, defaultDirectoryPerm)
	}
	return err
}

// NewServer returns a nfs.Server object that manages the given nfs mounts to
// configured clients;  basePath is the path for volumes, exportedName is the container dir to hold exported volumes
func NewServer(basePath, exportedName, network string) (*Server, error) {

	if len(exportedName) < 2 || strings.Contains(exportedName, "/") {
		return nil, ErrInvalidExportedName
	}
	if len(basePath) < 2 {
		return nil, ErrInvalidBasePath
	}
	if err := verifyExportsDir(basePath); err != nil {
		return nil, err
	}
	exportedNamePath := filepath.Join(exportsDir, exportedName)
	if err := verifyExportsDir(exportedNamePath); err != nil {
		return nil, err
	}
	if err := mp.Unmount(exportedNamePath); err != nil {
		glog.Errorf("Could not unmount export directory %s: %s", exportedNamePath, err)
		return nil, err
	}
	if _, _, err := net.ParseCIDR(network); err != nil {
		return nil, ErrInvalidNetwork
	}
	if err := start(); err != nil {
		return nil, err
	}
	return &Server{
		basePath:      basePath,
		exportedName:  exportedName,
		exportOptions: "rw,insecure,no_subtree_check,async",
		clients:       make(map[string]struct{}),
		network:       network,
		volumes:       make(map[string]int32),
		exported:      make(map[string]struct{}),
	}, nil
}

// ExportPath returns the external export name; foo for nfs export /exports/foo
func (c *Server) ExportPath() string {
	return filepath.Join("/", c.exportedName)
}

// Clients returns the IP Addresses of the current clients
func (c *Server) Clients() []string {
	clients := make([]string, len(c.clients))
	i := 0
	for key := range c.clients {
		clients[i] = key
	}
	return clients
}

// SetClients replaces the existing clients with the new clients
func (c *Server) SetClients(clients ...string) {
	c.Lock()
	defer c.Unlock()
	c.clients = make(map[string]struct{})

	for _, client := range clients {
		c.clients[client] = struct{}{}
	}
}

// VolumeCreated set that path of a volume that should be exported
func (c *Server) AddVolume(volumePath string) error {
	c.Lock()
	defer c.Unlock()
	fsid := atomic.AddInt32(&fsidIdx, 1)
	c.volumes[volumePath] = fsid
	return nil
}

// VolumeCreated set that path of a volume that should be exported
func (c *Server) RemoveVolume(volumePath string) error {
	c.Lock()
	defer c.Unlock()
	delete(c.volumes, volumePath)
	return nil
}

// Sync ensures that the nfs exports are visible to all clients
func (c *Server) Sync() error {
	c.Lock()
	defer c.Unlock()
	if err := c.hostsDeny(); err != nil {
		glog.Errorf("error writing host deny %v", err)
		return err
	}
	if err := c.hostsAllow(); err != nil {
		glog.Errorf("error writing host allow %v", err)
		return err
	}
	if err := c.writeExports(); err != nil {
		glog.Errorf("error writing exports %v", err)
		return err
	}
	if err := start(); err != nil {
		glog.Errorf("error running start %v", err)
		return err
	}
	if err := reload(); err != nil {
		glog.Errorf("error running reload %v", err)
		return err
	}
	c.cleanupBindMounts()
	return nil
}

// Restart restarts the nfs subsystem
func (c *Server) Restart() error {
	c.Lock()
	defer c.Unlock()
	if err := c.hostsDeny(); err != nil {
		return err
	}
	if err := c.hostsAllow(); err != nil {
		return err
	}
	if err := c.writeExports(); err != nil {
		return err
	}
	if err := restart(); err != nil {
		return err
	}
	c.cleanupBindMounts()
	return nil
}

// Stop stops the nfs subsystem
func (c *Server) Stop() error {
	c.Lock()
	defer c.Unlock()
	if err := stop(); err != nil {
		glog.Errorf("err running stop %v", err)
		return err
	}
	c.cleanupBindMounts()
	return nil
}

func (c *Server) hostsDeny() error {

	s, err := readFileIfExists(etcHostsDeny)
	if err != nil {
		return err
	}
	if strings.Contains(s, hostDenyDefaults) {
		return nil
	}

	if index := strings.Index(s, hostDenyMarker); index > 0 {
		s = s[:index-1]
	}
	s = s + hostDenyDefaults
	return atomicfile.WriteFile(etcHostsDeny, []byte(s), 0664)
}

func readFileIfExists(path string) (s string, err error) {
	var exists bool
	if exists, err = doesExists(path); err != nil {
		return s, err
	}
	if exists {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return s, err
		}
		s = string(bytes)
	}
	return s, nil
}

func (c *Server) hostsAllow() error {
	s, err := readFileIfExists(etcHostsAllow)
	if err != nil {
		return err
	}

	if index := strings.Index(s, hostAllowMarker); index > 0 {
		s = s[:index-1]
	}

	hosts := make([]string, len(c.clients))
	i := 0
	for key := range c.clients {
		hosts[i] = key
		i++
	}
	sort.Strings(hosts)
	s = s + hostAllowDefaults + " " + strings.Join(hosts, " ") + "\n\n"

	return atomicfile.WriteFile(etcHostsAllow, []byte(s), 0664)
}

func (c *Server) writeExports() error {
	network := c.network
	if network == "0.0.0.0/0" {
		network = "*" // turn this in to nfs 'allow all hosts' syntax
	}

	if err := os.MkdirAll(exportsDir, 0775); err != nil {
		return err
	}

	edir := filepath.Join(exportsDir, c.exportedName)
	if err := os.MkdirAll(edir, 0775); err != nil {
		return err
	}
	exports := make(map[string]struct{})
	serviced_exports := fmt.Sprintf("%s\t%s(rw,fsid=0,no_root_squash,insecure,no_subtree_check,async,crossmnt)\n",
		exportsDir, network)
	for volume, fsid := range c.volumes {
		volume = filepath.Clean(volume)
		_, volName := filepath.Split(volume)
		exports[volName] = struct{}{}
		exported := filepath.Join(edir, volName)
		if err := bindMount(volume, exported); err != nil {
			return err
		}
		serviced_exports += fmt.Sprintf("%s\t%s(rw,fsid=%d,no_root_squash,insecure,no_subtree_check,async)\n",
			exported, network, fsid)
	}
	c.exported = exports

	glog.Infof("serviced exports:\n %s", serviced_exports)
	originalContents, err := readFileIfExists(etcExports)
	if err != nil {
		return err
	}

	// comment out lines that conflicts with serviced exported mountpoints
	mountpaths := map[string]bool{exportsDir: true, filepath.Join(exportsDir, c.exportedName): true}
	filteredContent := ""
	scanner := bufio.NewScanner(strings.NewReader(originalContents))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				mountpoint := fields[0]
				if _, ok := mountpaths[mountpoint]; ok {
					filteredContent += etcExportsRemoveComment + line + "\n"
					continue
				}
			}
		}

		filteredContent += line + "\n"
	}

	// create file content
	preamble, postamble := filteredContent, ""
	if index := strings.Index(filteredContent, etcExportsStartMarker); index >= 0 {
		preamble = filteredContent[:index]
		remainder := filteredContent[index:]
		if index := strings.Index(remainder, etcExportsEndMarker); index >= 0 {
			postamble = remainder[index+len(etcExportsEndMarker):]
		}
	}
	fileContents := preamble + etcExportsStartMarker + serviced_exports + etcExportsEndMarker + postamble

	return atomicfile.WriteFile(etcExports, []byte(fileContents), 0664)
}

// umnount any bind mounts in exported directory if not exported
func (c *Server) cleanupBindMounts() {
	edir := filepath.Join(exportsDir, c.exportedName)
	//umount any directories not exported
	if dirContents, err := ioutil.ReadDir(edir); err != nil {
		glog.Warningf("could not read contents of %s; %v", edir, err)
	} else {
		for _, file := range dirContents {
			if _, found := c.exported[file.Name()]; !found && file.IsDir() {
				dir := filepath.Join(edir, file.Name())
				if err := mp.Unmount(dir); err != nil {
					glog.Warningf("Could not unmount exported directory %s: %s", dir, err)
					continue
				}
				//remove the directory
				glog.V(1).Infof("deleting dir %s as it is no longer exported", dir)
				if err := os.RemoveAll(dir); err != nil {
					glog.Warningf("Error removing exported directory %s: %v", dir, err)
				}
			}
		}
	}
	// CC-1816: clean up remnants of old /exports/serviced_var_volumes directory
	c.removeDeprecated("/exports/serviced_var_volumes")
}

// removeDeprecated will clean up any old exports path
func (c *Server) removeDeprecated(dirpath string) {
	if dirContents, err := ioutil.ReadDir(dirpath); !os.IsNotExist(err) {
		if err != nil {
			glog.Warningf("Could not look up deprecated exports path %s: %s", dirpath, err)
			return
		}

		// Unmount path
		if err := mp.Unmount(dirpath); err != nil {
			glog.Warningf("Could not unmount deprecated path %s: %s", dirpath, err)
		}

		// Guarantee the path is now empty
		if l := len(dirContents); l > 0 {
			glog.Warningf("Path %s is not empty.", dirpath)
		} else {
			// Remove the path only if it is empty
			if err := os.Remove(dirpath); err != nil {
				glog.Warningf("Could not remove deprecated path %s: %s", dirpath, err)
				return
			}
			glog.Infof("Deleted deprecated path %s", dirpath)
		}
	}
}

type bindMountF func(string, string) error

var bindMount = bindMountImp

// bindMountImp performs a bind mount of src to dst.
func bindMountImp(src, dst string) error {
	glog.Infof("bindMount %s at %s", src, dst)
	if mounted, err := mp.IsMounted(dst); err != nil {
		return err
	} else if mounted {
		return nil
	}
	if err := os.MkdirAll(dst, 0775); err != nil {
		return err
	}
	runMountCommand := func(options ...string) ([]byte, error) {
		cmd, args := mntArgs(src, dst, "", options...)
		mount := exec.Command(cmd, args...)
		glog.Infof("running mount: %s %s", cmd, strings.Join(args, " "))
		return mount.CombinedOutput()
	}
	out, returnErr := runMountCommand("bind")
	if returnErr != nil {
		// If the mount fails, it could be due to a stale NFS handle, signalled
		// by a return code of 32. Stale handle can occur if e.g., the source
		// directory has been deleted and restored (a common occurrence in the
		// dev workflow) Try again, with remount option.
		if exitcode, ok := utils.GetExitStatus(returnErr); ok && (exitcode&32) != 0 {
			out, returnErr = runMountCommand("bind", "remount")
		}
	}
	if returnErr != nil {
		return fmt.Errorf("%s: %s", out, returnErr)
	}
	return nil
}

func doesExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// mntArgs computes the required arguments for the mount command given
// fs (src fs), dst (mount point), fsType (ext3, xfs, etc), options (parameters
// passed to the -o flag).
func mntArgs(fs, dst, fsType string, options ...string) (cmd string, args []string) {
	args = make([]string, 0)
	if syscall.Getuid() != 0 {
		args = append(args, "sudo")
	}
	args = append(args, "mount")
	for _, option := range options {
		args = append(args, "-o")
		args = append(args, option)
	}
	if len(fsType) > 0 {
		args = append(args, "-t")
		args = append(args, fsType)
	}
	if len(fs) > 0 {
		args = append(args, fs)
	}
	args = append(args, dst)
	return args[0], args[1:]
}
