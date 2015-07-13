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

package dfs

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/storage"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

type DistributedFilesystem struct {
	fsType        string
	varpath       string
	dockerHost    string
	dockerPort    int
	facade        *facade.Facade
	timeout       time.Duration
	networkDriver storage.StorageDriver

	// locking
	mutex sync.Mutex
	lock  client.Lock

	// logging
	logger *logger
}

func NewDistributedFilesystem(fsType, varpath, dockerRegistry string, facade *facade.Facade, timeout time.Duration, networkDriver storage.StorageDriver) (*DistributedFilesystem, error) {
	host, port, err := parseRegistry(dockerRegistry)
	if err != nil {
		return nil, err
	}

	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return nil, err
	}
	lock := zkservice.ServiceLock(conn)

	return &DistributedFilesystem{
		fsType:        fsType,
		varpath:       varpath,
		dockerHost:    host,
		dockerPort:    port,
		facade:        facade,
		timeout:       timeout,
		lock:          lock,
		networkDriver: networkDriver,
	}, nil
}

func (dfs *DistributedFilesystem) Lock() error {
	dfs.mutex.Lock()

	err := dfs.lock.Lock()
	if err != nil {
		glog.Warningf("Could not lock services! Operation may be unstable: %s", err)
	}
	dfs.logger = new(logger).init()
	return err
}

func (dfs *DistributedFilesystem) Unlock() error {
	defer dfs.mutex.Unlock()
	dfs.logger.Done()
	dfs.logger = nil
	return dfs.lock.Unlock()
}

func (dfs *DistributedFilesystem) IsLocked() (bool, error) {
	conn, err := zzk.GetLocalConnection("/")
	if err != nil {
		return false, err
	}
	return zkservice.IsServiceLocked(conn)
}

func (dfs *DistributedFilesystem) GetStatus(timeout time.Duration) string {
	if dfs.logger != nil {
		return dfs.logger.Recv(timeout)
	}
	return ""
}

func (dfs *DistributedFilesystem) log(msg string, argv ...interface{}) {
	defer glog.V(0).Infof(msg, argv...)
	if dfs.logger != nil {
		dfs.logger.Send(fmt.Sprintf(msg, argv...))
	}
}

func parseRegistry(registry string) (host string, port int, err error) {
	parts := strings.SplitN(registry, ":", 2)

	if host = parts[0]; host == "" {
		return "", 0, fmt.Errorf("malformed registry")
	} else if len(parts) > 1 {
		if port, err = strconv.Atoi(parts[1]); err != nil {
			return "", 0, fmt.Errorf("malformed registry")
		}
	}
	return host, port, nil
}

type logger struct {
	messages chan string
	q        []string
	mutex    sync.Mutex
	empty    sync.WaitGroup
}

func (l *logger) init() *logger {
	l = &logger{messages: make(chan string)}
	l.empty.Add(1)
	go l.stream()
	return l
}

func (l *logger) stream() {
	for {
		m, ok := <-l.messages
		if !ok {
			return
		}

		l.mutex.Lock()
		l.q = append(l.q, m)
		if len(l.q) == 1 {
			l.empty.Done()
		}
		l.mutex.Unlock()
	}
}

func (l *logger) Send(m string) {
	l.messages <- m
}

func (l *logger) Recv(timeout time.Duration) string {
	ready := make(chan struct{})

	go func() {
		defer close(ready)
		l.empty.Wait()
	}()

	select {
	case <-time.After(timeout):
		return "timeout"
	case <-ready:
	}

	if len(l.q) == 0 {
		return "EOF"
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()
	msg := l.q[len(l.q)-1]
	l.q = make([]string, 0)
	l.empty.Add(1)
	return msg
}

func (l *logger) Done() {
	close(l.messages)
}
