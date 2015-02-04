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
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/host"
	"github.com/zenoss/glog"
)

const monitorSubDir string = "/monitor"

// Monitor monitors the exporting of a file system to clients.
type Monitor struct {
	driver StorageDriver

	monitoredHosts     map[string]bool
	monitoredHostsLock sync.RWMutex

	shouldRestart   bool
	monitorInterval time.Duration
	previousRestart time.Time

	conn               client.Connection
	storageClientsPath string
}

// NewMonitor returns a Monitor object to monitor the exported file system
func NewMonitor(driver StorageDriver, monitorInterval time.Duration) (*Monitor, error) {
	m := &Monitor{
		driver:          driver,
		monitoredHosts:  make(map[string]bool),
		shouldRestart:   getShouldRestartDFSOnFailure(),
		monitorInterval: monitorInterval,
	}

	return m, nil
}

// UpdateRemoteMonitorFile is used by remote clients to write a tiny file to the DFS volume at the given cycle
func UpdateRemoteMonitorFile(localPath string, writeInterval time.Duration, ipAddr string, shutdown <-chan interface{}) {
	monitorPath := path.Join(localPath, monitorSubDir)
	remoteFile := path.Join(localPath, monitorSubDir, ipAddr)
	glog.Infof("updating DFS volume monitor file %s at write interval: %s", remoteFile, writeInterval)

	for {
		glog.V(2).Infof("checking DFS monitor path %s", monitorPath)
		_, err := os.Stat(monitorPath)
		if err != nil {
			glog.V(2).Infof("unable to stat DFS monitor path: %s %s", monitorPath, err)
			if err := os.MkdirAll(monitorPath, 0755); err != nil {
				glog.Warningf("unable to create DFS volume monitor path %s: %s", monitorPath, err)
			} else {
				glog.Infof("created DFS volume monitor path %s", monitorPath)
			}
		}

		glog.V(2).Infof("writing DFS file %s", remoteFile)
		if err := ioutil.WriteFile(remoteFile, []byte(ipAddr), 0600); err != nil {
			glog.Warningf("unable to write DFS file %s: %s", remoteFile, err)
		}

		// wait for next cycle or shutdown
		select {
		case <-time.After(writeInterval):

		case <-shutdown:
			glog.Infof("no longer writing remote monitor status for DFS volume %s to %s", localPath, remoteFile)
			return
		}
	}
}

// SetMonitorStorageClients sets the monitored remote IPs that are active
func (m *Monitor) SetMonitorStorageClients(conn client.Connection, storageClientsPath string) {
	m.conn = conn
	m.storageClientsPath = storageClientsPath
	m.SetMonitorRemoteHosts(getActiveRemoteHosts(conn, storageClientsPath, m.monitorInterval)...)
}

// SetMonitorRemoteHosts sets the remote hosts to monitor
func (m *Monitor) SetMonitorRemoteHosts(ipAddrs ...string) {
	if len(ipAddrs) == 0 {
		glog.Warningf("disabled DFS volume monitoring - no remote IPs to monitor")
	} else {
		glog.V(2).Infof("enabled DFS volume monitoring for remote IPs: %+v", ipAddrs)
	}
	m.monitoredHostsLock.Lock()
	defer m.monitoredHostsLock.Unlock()
	for _, ipAddr := range ipAddrs {
		m.monitoredHosts[ipAddr] = true
	}
}

// setMonitorRemoteHost updates the status of the monitored remote
func (m *Monitor) setMonitorRemoteHost(ipAddr string, hasUpdated bool) {
	m.monitoredHostsLock.Lock()
	defer m.monitoredHostsLock.Unlock()
	m.monitoredHosts[ipAddr] = hasUpdated
}

// ProcessDFSVolumePollUpdateFunc is called by MonitorDFSVolume when updates are detected
type DFSVolumeMonitorPollUpdateFunc func(mountpoint, remoteIP string, isUpdated bool)

// DFSVolumeMonitorPollUpdateFunc restarts nfs based on status of monitored remotes
func (m *Monitor) DFSVolumeMonitorPollUpdateFunc(mountpoint, remoteIP string, hasUpdatedFile bool) {
	// monitor dfs; log warnings each cycle; restart dfs if needed

	m.monitoredHostsLock.RLock()
	defer m.monitoredHostsLock.RUnlock()

	if hasUpdatedFile {
		return
	} else if len(m.monitoredHosts) == 0 {
		return
	}

	glog.Warningf("DFS NFS volume %s is not seen by remoteIP:%s - further action may be needed i.e: restart nfs", mountpoint, remoteIP)
	now := time.Now()
	since := now.Sub(m.previousRestart)
	if !m.shouldRestart {
		glog.Warningf("Not restarting DFS NFS service due to configuration setting: SERVICED_MONITOR_DFS_MASTER_RESTART=0")
		return
	} else if since < m.monitorInterval {
		glog.Warningf("Not restarting DFS NFS service - have not surpassed interval: %s since last restart", m.monitorInterval)
		return
	}

	m.previousRestart = now
	if err := m.driver.Restart(); err != nil {
		glog.Errorf("Error restarting DFS NFS service: %s", err)
	}
}

// MonitorVolume monitors the DFS volume - logs on failure and calls pollUpdateFunc
func (m *Monitor) MonitorDFSVolume(mountpoint string, shutdown <-chan interface{}, pollUpdateFunc DFSVolumeMonitorPollUpdateFunc) {
	glog.Infof("monitoring DFS export info for DFS volume %s at polling interval: %s", mountpoint, m.monitorInterval)

	m.previousRestart = time.Now()

	monitorPath := path.Join(mountpoint, monitorSubDir)
	os.RemoveAll(monitorPath)
	if err := os.MkdirAll(monitorPath, 0755); err != nil {
		glog.Errorf("no longer monitoring status for DFS volume %s - unable to mkdir %+v: %s", mountpoint, monitorPath, err)
		return
	}

	for {
		// check all active remotes
		m.monitoredHostsLock.RLock()
		remoteIPs := make([]string, 0, len(m.monitoredHosts))
		for k := range m.monitoredHosts {
			remoteIPs = append(remoteIPs, k)
		}
		m.monitoredHostsLock.RUnlock()

		activeRemotes := remoteIPs
		if m.conn != nil && len(m.storageClientsPath) != 0 {
			activeRemotes = getActiveRemoteHosts(m.conn, m.storageClientsPath, m.monitorInterval)
			sort.Sort(sort.StringSlice(activeRemotes))
			sort.Sort(sort.StringSlice(remoteIPs))
			if !reflect.DeepEqual(activeRemotes, remoteIPs) {
				glog.Infof("DFS active remotes to be monitored has changed to: %+v", activeRemotes)
			}
			m.SetMonitorRemoteHosts(activeRemotes...)
		}

		for _, remoteIP := range activeRemotes {
			remoteFile := path.Join(monitorPath, remoteIP)
			sinceModified, err := fileSinceModified(remoteFile)
			hasUpdatedFile := false
			if err != nil {
				glog.Warningf("remote agent %s is not synced on DFS volume %s for file %s: %s", remoteIP, mountpoint, remoteFile, err)
			} else if sinceModified > m.monitorInterval {
				glog.Warningf("remote agent %s is not synced DFS volume %s for file %s (since modified: %s)", remoteIP, mountpoint, remoteFile, sinceModified)
			} else {
				glog.V(4).Infof("remote agent %s is synced on DFS volume %s for file %s (since modified: %s)", remoteIP, mountpoint, remoteFile, sinceModified)
				hasUpdatedFile = true
			}

			m.setMonitorRemoteHost(remoteIP, hasUpdatedFile)
			pollUpdateFunc(mountpoint, remoteIP, hasUpdatedFile)
		}

		// wait for next cycle or shutdown
		select {
		case <-time.After(m.monitorInterval):

		case <-shutdown:
			glog.Infof("no longer monitoring status for DFS volume %s", mountpoint)
			return
		}
	}
}

// fileSinceModified returns the duration of the file since last modified
func fileSinceModified(filename string) (time.Duration, error) {
	fileinfo, err := os.Stat(filename)
	if err != nil {
		glog.V(2).Infof("unable to stat file: %s %s", filename, err)
		return 0, err
	}

	timeSince := time.Now().Sub(fileinfo.ModTime())
	glog.V(4).Infof("file %s was modified %s ago", filename, timeSince)
	return timeSince, nil
}

// Get the variable determining if we will restart NFS when we detect a failure
func getShouldRestartDFSOnFailure() bool {
	shouldRestart := os.Getenv("SERVICED_MONITOR_DFS_MASTER_RESTART")
	if len(shouldRestart) == 0 {
		glog.Infof("defaulting SERVICED_MONITOR_DFS_MASTER_RESTART to true")
		return true
	} else if shouldRestart == "1" {
		return true
	} else {
		return false
	}
}

// getDefaultDFSMonitorRemoteInterval returns the duration specified in seconds for the env var SERVICED_DFS_MONITOR_REMOTE_UPDATE_INTERVAL
func getDefaultDFSMonitorRemoteInterval() time.Duration {
	return getEnvMinDuration("SERVICED_MONITOR_DFS_REMOTE_UPDATE_INTERVAL", 60, 60)
}

func getDefaultNFSMonitorMasterInterval() time.Duration {
	min := getDefaultDFSMonitorRemoteInterval()
	return getEnvMinDuration("SERVICED_MONITOR_DFS_MASTER_INTERVAL", 900, int32(min.Seconds())*2)
}

// getEnvMinDuration returns the time.Duration env var meeting minimum and default duration
func getEnvMinDuration(envvar string, def, min int32) time.Duration {
	duration := def
	envval := os.Getenv(envvar)
	if len(strings.TrimSpace(envval)) == 0 {
		// ignore unset envvar
	} else if intVal, intErr := strconv.ParseInt(envval, 0, 32); intErr != nil {
		glog.Warningf("ignoring invalid %s of '%s': %s", envvar, envval, intErr)
		duration = min
	} else if int32(intVal) < min {
		glog.Warningf("ignoring invalid %s of '%s' < minimum:%v seconds", envvar, envval, min)
	} else {
		duration = int32(intVal)
	}

	return time.Duration(duration) * time.Second
}

// StorageClientHostNode is the zookeeper node for a host
type StorageClientHostNode struct {
	host.Host
	version interface{}
}

// Version is an implementation of client.Node
func (v *StorageClientHostNode) Version() interface{} { return v.version }

// SetVersion is an implementation of client.Node
func (v *StorageClientHostNode) SetVersion(version interface{}) { v.version = version }

// getActiveRemoteHosts returns a slice of activeClientIPs
func getActiveRemoteHosts(conn client.Connection, storageClientsPath string, monitorInterval time.Duration) []string {
	// clients is not full list of remotes when called from server.go - retrieve our own list from zookeeper
	var err error
	remoteIPs, err := getAllRemoteHostsFromZookeeper(conn, storageClientsPath)
	if err != nil {
		return []string{}
	}
	glog.V(2).Infof("DFS remote IPs: %+v", remoteIPs)

	// determine active hosts
	var activeClientIPs []string
	now := time.Now()
	for _, clnt := range remoteIPs {
		cp := path.Join(storageClientsPath, clnt)
		glog.V(2).Infof("retrieving info for DFS for remoteIP %s at zookeeper node %s", clnt, cp)

		hnode := StorageClientHostNode{}
		err := conn.Get(cp, &hnode)
		if err != nil && err != client.ErrEmptyNode {
			glog.Errorf("DFS could not get remote host zookeeper node %s: %s", cp, err)
			continue
		}
		if hnode.Host.UpdatedAt.IsZero() {
			glog.Infof("DFS not monitoring non-active host %+v:  HostID:%v  UpdatedAt:%+v", clnt, hnode.ID, hnode.UpdatedAt)
			continue
		}

		elapsed := now.Sub(hnode.Host.UpdatedAt)
		glog.V(2).Infof("retrieved info for DFS for remoteIP %s  UpdatedAt:%s  elapsed:%s  monitorInterval:%s", clnt, hnode.Host.UpdatedAt, elapsed, monitorInterval)
		if elapsed > monitorInterval {
			glog.Infof("DFS not monitoring non-active host %+v:  HostID:%v  UpdatedAt:%+v  lastseen:%s ago", clnt, hnode.ID, hnode.UpdatedAt, elapsed)
			continue
		}

		activeClientIPs = append(activeClientIPs, clnt)
	}

	glog.V(2).Infof("DFS remote active IPs: %+v", activeClientIPs)
	return activeClientIPs
}

// getAllRemoteHostsFromZookeeper retrieves a list of remote storage clients from zookeeper
func getAllRemoteHostsFromZookeeper(conn client.Connection, storageClientsPath string) ([]string, error) {
	clients, err := conn.Children(storageClientsPath)
	if err != nil {
		glog.Errorf("unable to retrieve list of DFS remote hosts: %s", err)
		return []string{}, err
	}

	glog.V(4).Infof("DFS remote IPs: %+v", clients)
	return clients, nil
}
