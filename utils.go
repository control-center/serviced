// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package serviced

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const TIMEFMT = "20060102-150405"

func GetLabel(name string) string {
	localtime := time.Now()
	utc := localtime.UTC()
	return fmt.Sprintf("%s_%s", name, utc.Format(TIMEFMT))
}

var hostIDCmdString = "/usr/bin/hostid"

// HostID retreives the system's unique id, on linux this maps
// to /usr/bin/hostid.
func HostID() (hostid string, err error) {
	cmd := exec.Command(hostIDCmdString)
	stdout, err := cmd.Output()
	if err != nil {
		return hostid, err
	}
	return strings.TrimSpace(string(stdout)), err
}

// Path to meminfo file. Placed here so getMemorySize() is testable.
var meminfoFile = "/proc/meminfo"

// validOwnerSpec returns true if the owner is specified in owner:group format and the
// identifiers are valid POSIX.1-2008 username and group strings, respectively.
func validOwnerSpec(owner string) bool {
	var pattern = regexp.MustCompile(`^[a-zA-Z]+[a-zA-Z0-9.-]*:[a-zA-Z]+[a-zA-Z0-9.-]*$`)
	return pattern.MatchString(owner)
}

// getMemorySize attempts to get the size of the installed RAM.
func getMemorySize() (size uint64, err error) {
	file, err := os.Open(meminfoFile)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	for err == nil {
		if strings.Contains(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return 0, err
			}
			size, err := strconv.Atoi(parts[1])
			if err != nil {
				return 0, err
			}
			return uint64(size) * 1024, nil
		}
		line, err = reader.ReadString('\n')
	}
	return 0, err
}

// RouteEntry represents a entry from the route command
type RouteEntry struct {
	Destination string
	Gateway     string
	Genmask     string
	Flags       string
	Metric      int
	Ref         int
	Use         int
	Iface       string
}

// wrapper around the route command
func routeCmd() (routes []RouteEntry, err error) {
	output, err := exec.Command("/sbin/route", "-A", "inet").Output()
	if err != nil {
		return routes, err
	}

	columnMap := make(map[string]int)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return routes, fmt.Errorf("no routes found")
	}
	routes = make([]RouteEntry, len(lines)-2)
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		// skip first line
		case lineNum == 0:
			continue
		case lineNum == 1:
			for number, name := range strings.Fields(line) {
				columnMap[name] = number
			}
			continue
		default:
			fields := strings.Fields(line)
			metric, err := strconv.Atoi(fields[columnMap["Metric"]])
			if err != nil {
				return routes, err
			}
			ref, err := strconv.Atoi(fields[columnMap["Ref"]])
			if err != nil {
				return routes, err
			}
			use, err := strconv.Atoi(fields[columnMap["Use"]])
			if err != nil {
				return routes, err
			}
			routes[lineNum-2] = RouteEntry{
				Destination: fields[columnMap["Destination"]],
				Gateway:     fields[columnMap["Gateway"]],
				Genmask:     fields[columnMap["Genmask"]],
				Flags:       fields[columnMap["Flags"]],
				Metric:      metric,
				Ref:         ref,
				Use:         use,
				Iface:       fields[columnMap["Iface"]],
			}
		}
	}
	return routes, err
}

// GetIPAddress attempts to find the IP address to the default outbout interface.
func GetIPAddress() (ip string, err error) {
	ip, err = getIPAddrFromHostname()
	if err != nil || strings.HasPrefix(ip, "127") {
		ip, err = getIPAddrFromOutGoingConnection()
		if err == nil && strings.HasPrefix(ip, "127") {
			return "", fmt.Errorf("unable to identify local ip address")
		}
	}

	return ip, err
}

// getIPAddrFromHostname returns the ip address associated with hostname -i.
func getIPAddrFromHostname() (ip string, err error) {
	output, err := exec.Command("hostname", "-i").Output()
	if err != nil {
		return ip, err
	}
	return strings.TrimSpace(string(output)), err
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

// CurrentContextAsHost creates a dao.Host object of the host running
// this method. The passed in poolID is used as the resource pool in the result.
func CurrentContextAsHost(poolID string) (host *dao.Host, err error) {
	cpus := runtime.NumCPU()
	memory, err := getMemorySize()
	if err != nil {
		return nil, err
	}
	host = dao.NewHost()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	host.Name = hostname
	hostidStr, err := HostID()
	if err != nil {
		return nil, err
	}

	host.IpAddr, err = GetIPAddress()
	if err != nil {
		return host, err
	}

	host.Id = hostidStr
	host.Cores = cpus
	host.Memory = memory

	routes, err := routeCmd()
	if err != nil {
		return nil, err
	}
	for _, route := range routes {
		if route.Iface == "docker0" {
			host.PrivateNetwork = route.Destination + "/" + route.Genmask
			break
		}
	}
	host.PoolId = poolID
	return host, err
}

// ExecPath returns the path to the currently running executable.
func ExecPath() (string, string, error) {
	path, err := os.Readlink("/proc/self/exe")
	if err != nil {
		return "", "", err
	}
	return filepath.Dir(path), filepath.Base(path), nil
}

// DockerVersion contains the tuples that describe the version of docker.
type DockerVersion struct {
	Client []int
	Server []int
}

// equals compares two DockerVersion structs and returns true if they are equal.
func (a *DockerVersion) equals(b *DockerVersion) bool {
	if len(a.Client) != len(b.Client) {
		return false
	}
	for i, aI := range a.Client {
		if aI != b.Client[i] {
			return false
		}
	}
	if len(a.Server) != len(b.Server) {
		return false
	}
	for i, aI := range a.Server {
		if aI != b.Server[i] {
			return false
		}
	}
	return true
}

// GetDockerVersion returns docker version number.
func GetDockerVersion() (DockerVersion, error) {
	cmd := exec.Command("docker", "version")
	output, err := cmd.Output()
	if err != nil {
		return DockerVersion{}, err
	}
	return parseDockerVersion(string(output))
}

// parseDockerVersion parses the output of the 'docker version' commmand and
// returns a DockerVersion object.
func parseDockerVersion(output string) (version DockerVersion, err error) {

	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		if strings.HasPrefix(parts[0], "Client version") {
			a := strings.SplitN(strings.TrimSpace(parts[1]), "-", 2)
			b := strings.Split(a[0], ".")
			version.Client = make([]int, len(b))
			for i, v := range b {
				x, err := strconv.Atoi(v)
				if err != nil {
					return version, err
				}
				version.Client[i] = x
			}
		}
		if strings.HasPrefix(parts[0], "Server version") {
			a := strings.SplitN(strings.TrimSpace(parts[1]), "-", 2)
			b := strings.Split(a[0], ".")
			version.Server = make([]int, len(b))
			for i, v := range b {
				x, err := strconv.Atoi(v)
				if err != nil {
					return version, err
				}
				version.Server[i] = x
			}
		}
	}
	if len(version.Client) == 0 {
		return version, fmt.Errorf("no client version found")
	}
	if len(version.Server) == 0 {
		return version, fmt.Errorf("no server version found")
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

var dockerRun = func(imageSpec string, args ...string) (output string, err error) {
	targs := []string{"run", "run", "-rm", imageSpec}
	for _, s := range args {
		targs = append(targs, s)
	}
	docker := exec.Command("docker", targs...)
	var outputBytes []byte
	outputBytes, err = docker.CombinedOutput()
	if err != nil {
		return
	}
	output = string(outputBytes)
	return
}

// TODO: refactor createVolumeDir to use this function
func getInternalImageIds(userSpec, imageSpec string) (uid, gid int, err error) {
	var output string
	output, err = dockerRun(imageSpec, "/bin/sh", "-c",
		fmt.Sprintf(`touch test.txt && chown %s test.txt && ls -ln | awk '{ print $3 " " $4 }'`,
			userSpec))
	if err != nil {
		return
	}
	s := strings.TrimSpace(string(output))
	pattern := regexp.MustCompile(`^\d+ \d+$`)

	if !pattern.MatchString(s) {
		err = fmt.Errorf("unexpected output from getInternalImageIds: %s", s)
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
	return
}

var createVolumeDirMutex sync.Mutex

// createVolumeDir() creates a directory on the running host using the user ids
// found within the specified image. For example, it can create a directory owned
// by the mysql user (as seen by the container) despite there being no mysql user
// on the host system
func createVolumeDir(hostPath, containerSpec, imageSpec, userSpec, permissionSpec string) error {

	createVolumeDirMutex.Lock()
	defer createVolumeDirMutex.Unlock()
	// FIXME: this relies on the underlying container to have /bin/sh that supports
	// some advanced shell options. This should be rewriten so that serviced injects itself in the
	// container and performs the operations using only go!

	var err error
	var output []byte
	for i := 0; i < 1; i++ {
		docker := exec.Command("docker", "run", "-rm",
			"-v", hostPath+":/tmp",
			imageSpec,
			"/bin/sh", "-c",
			fmt.Sprintf(`
chown %s /tmp && \
chmod %s /tmp && \
if [ ! -d "%s" ]; then
	echo "ERROR: srcdir %s does not exist in container"
	exit 2
else
	rsync -a %s/ /tmp/
fi
`, userSpec, permissionSpec, containerSpec, containerSpec, containerSpec))
		output, err = docker.CombinedOutput()
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}

	glog.Errorf("could not create host volume: %s", string(output))
	return err
}
