/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package serviced

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

var urandomFilename = "/dev/urandom"

// Generate a new UUID
func newUuid() (string, error) {
	f, err := os.Open(urandomFilename)
	if err != nil {
		return "", err
	}
	b := make([]byte, 16)
	defer f.Close()
	f.Read(b)
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, err
}

var hostIdCmdString = "/usr/bin/hostid"

// hostId retreives the system's unique id, on linux this maps
// to /usr/bin/hostid.
func HostId() (hostid string, err error) {
	cmd := exec.Command(hostIdCmdString)
	stdout, err := cmd.Output()
	if err != nil {
		return hostid, err
	}
	return strings.TrimSpace(string(stdout)), err
}

// Path to meminfo file. Placed here so getMemorySize() is testable.
var meminfoFile = "/proc/meminfo"

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

// Represent a entry from the route command
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

func getIpAddr() (ip string, err error) {
	output, err := exec.Command("hostname", "-i").Output()
	if err != nil {
		return ip, err
	}
	return strings.TrimSpace(string(output)), err
}

// Create a new Host struct from the running host's values
func CurrentContextAsHost() (host *Host, err error) {
	cpus := runtime.NumCPU()
	memory, err := getMemorySize()
	if err != nil {
		return nil, err
	}
	host = NewHost()
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	host.Name = hostname
	hostid_str, err := HostId()
	if err != nil {
		return nil, err
	}

	host.IpAddr, err = getIpAddr()
	if err != nil {
		return host, err
	}

	host.Id = hostid_str
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

	return host, err
}

type DatabaseConnectionInfo struct {
	Dialect  string
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Options  map[string]string
}

func (connInfo *DatabaseConnectionInfo) UrlString() string {
	url := connInfo.Dialect + "://"
	if len(connInfo.User) > 0 {
		url += connInfo.User
		if len(connInfo.Password) > 0 {
			url += ":" + connInfo.Password
		}
		url += "@"
	}
	url += connInfo.Host
	if connInfo.Port > 0 {
		url += fmt.Sprintf(":%d", connInfo.Port)
	}
	url += "/" + connInfo.Database
	return url
}

// Parse a URI and create a database connection info object. Eg
// mysql://user:password@127.0.0.1:3306/test
func ParseDatabaseUri(str string) (connInfo *DatabaseConnectionInfo, err error) {
	connInfo = &DatabaseConnectionInfo{}
	u, err := url.Parse(str)
	if err != nil {
		return connInfo, err
	}
	connInfo.Dialect = u.Scheme
	if strings.Contains(u.Host, ":") {
		parts := strings.SplitN(u.Host, ":", 2)
		connInfo.Host = parts[0]
		if len(parts) > 1 {
			connInfo.Port, _ = strconv.Atoi(parts[1])
		}
	}
	if u.User != nil {
		password, _ := u.User.Password()
		connInfo.User = u.User.Username()
		connInfo.Password = password
	}
	if len(u.Path) > 1 {
		connInfo.Database = u.Path[1:]
	}
	return connInfo, nil
}

func ToMymysqlConnectionString(cInfo *DatabaseConnectionInfo) string {
	return fmt.Sprintf("tcp:%s:%d*%s/%s/%s", cInfo.Host, cInfo.Port,
		cInfo.Database, cInfo.User, cInfo.Password)
}
