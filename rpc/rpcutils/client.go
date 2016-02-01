// Copyright 2015 The Serviced Authors.
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

package rpcutils

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
	"time"

	"github.com/control-center/serviced/commons/pool"
	"github.com/zenoss/glog"
)

var dialTimeoutSecs = 30

// SetDialTimeout time in seconds to timeout dialing a connection
func SetDialTimeout(timeout int) {
	dialTimeoutSecs = timeout
}

type connectRPCFn func(add string) (*rpc.Client, error)

func connectRPC(addr string) (*rpc.Client, error) {
	glog.V(4).Infof("Connecting to %s", addr)
	conn, err := net.DialTimeout("tcp", addr, time.Duration(dialTimeoutSecs)*time.Second)
	if err != nil {
		return nil, err
	}
	return jsonrpc.NewClient(conn), nil
}

func connectRPCTLS(addr string) (*rpc.Client, error) {
	glog.V(4).Infof("Connecting to %s", addr)

	config := tls.Config{InsecureSkipVerify: !RPCCertVerify}
	timeoutDialer := net.Dialer{Timeout: time.Duration(dialTimeoutSecs) * time.Second}
	conn, err := tls.DialWithDialer(&timeoutDialer, "tcp4", addr, &config)
	if err != nil {
		return nil, err
	}
	return jsonrpc.NewClient(conn), nil
}

// newClient that will create at most max active rpc connections at any given time. discardClientTimeout timeout for
// discarding client from pool if a call takes too long, call will not be cancelled; assures liveliness of pool
func newClient(addr string, max int, discardClientTimeout time.Duration, fn connectRPCFn) (Client, error) {

	rpcClientFactory := func() (interface{}, error) {
		return fn(addr)
	}
	rpcPool, err := pool.NewPool(max, rpcClientFactory)
	if err != nil {
		return nil, err
	}
	rc := &reconnectingClient{addr: addr, pool: rpcPool, discardClientTimeout: discardClientTimeout}
	return rc, nil
}

// Client to limit the number of underlying rpc connections. Reuses connections and discards connections on error
type reconnectingClient struct {
	addr                 string
	pool                 pool.Pool
	activeConnections    int32
	discardClientTimeout time.Duration
}

func (rc *reconnectingClient) Call(serviceMethod string, args interface{}, reply interface{}, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 365 * 24 * time.Hour
	}

	start := time.Now()
	item, err := rc.pool.BorrowWait(timeout)
	if err != nil {
		return err
	}
	defer func() {
		// failsafe if we return without cleaning up
		if item != nil {
			rc.pool.Remove(item)
		}
	}()
	elapsed := time.Now().Sub(start)
	if timeout > 0 {
		remaining := timeout - elapsed
		if remaining < 0 {
			rc.pool.Return(item)
			item = nil
			return fmt.Errorf("RPC call to %s timed out waiting for client", serviceMethod)
		}
		timeout = remaining
	}
	rpcClient := item.Item.(*rpc.Client)
	errChan := make(chan error, 1)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		wg.Done()
		glog.V(4).Infof("rpcClient: Call %s", serviceMethod)
		rpcErr := rpcClient.Call(serviceMethod, args, reply)
		errChan <- rpcErr
	}()
	wg.Wait()
	clientRemoved := false
	start = time.Now()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case e := <-errChan:
			if e != nil {
				rpcClient.Close()
				rc.pool.Remove(item)
				item = nil
				return e
			}
			if clientRemoved {
				rpcClient.Close()
			} else {
				rc.pool.Return(item)
			}
			item = nil
			return e
		case <-timer.C:
			err = fmt.Errorf("RPC call to %s timed out after %s", serviceMethod, timeout)
			rpcClient.Close()
			rc.pool.Remove(item)
			item = nil
			return err
		case <-time.After(rc.discardClientTimeout):
			//log long calls and remove from pool to prevent blocks
			glog.V(2).Infof("Long Running Call %s %s", serviceMethod, time.Now().Sub(start))
			if !clientRemoved {
				glog.V(1).Infof("Long Running Call removing client from pool %s after %s\n", serviceMethod, time.Now().Sub(start))
				rc.pool.Remove(item)
				clientRemoved = true
			}
		}
	}
}

func (rc *reconnectingClient) Close() error {
	//ignore close as we want to reuse the underlying connections
	return nil
}
