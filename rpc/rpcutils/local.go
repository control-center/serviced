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
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/zenoss/glog"
)

type localClient struct {
	sync.RWMutex
	rcvrs map[string]interface{}
}

var (
	localRpcClient = &localClient{}
	localAddrs     = map[string]struct{}{}
)

func init() {
	localRpcClient.rcvrs = make(map[string]interface{})
	localAddrs = make(map[string]struct{})
}

func RegisterLocalAddress(addrs ...string) {
	for _, addr := range addrs {
		localAddrs[addr] = struct{}{}
	}
}

func RegisterLocal(name string, rcvr interface{}) error {

	return localRpcClient.register(name, rcvr)

}

func (l *localClient) register(name string, rcvr interface{}) error {
	l.Lock()
	defer l.Unlock()
	l.rcvrs[name] = rcvr
	return nil
}

func (l *localClient) Close() error {
	return nil
}

func (l *localClient) Call(serviceMethod string, args interface{}, reply interface{}, timeout time.Duration) error {

	parts := strings.SplitN(serviceMethod, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("Invalid service method: %s", serviceMethod)
	}
	name := parts[0]
	methodName := parts[1]

	glog.V(3).Infof("RPC service method %s:%s", name, methodName)

	l.RLock()
	server, ok := l.rcvrs[name]
	l.RUnlock()
	if !ok {
		return fmt.Errorf("Server Not Found for %s", serviceMethod)

	}

	method := reflect.ValueOf(server).MethodByName(methodName)
	if !method.IsValid() {
		return fmt.Errorf("can't find method %s", serviceMethod)
	}
	callChan := make(chan error, 1)

	go func() {
		inputs := make([]reflect.Value, 2)

		inputs[0] = reflect.ValueOf(args)

		//make a new one of the correct type
		rType := method.Type().In(1)
		rValue := reflect.New(rType.Elem())
		inputs[1] = rValue

		result := method.Call(inputs)
		err := result[0].Interface()
		if err != nil {
			callChan <- err.(error)
		}

		replyValue := reflect.ValueOf(reply)
		if reply == nil {
			callChan <- nil
		} else if replyValue.Kind() != reflect.Ptr {
			callChan <- fmt.Errorf("processing response (non-pointer %v)", reflect.TypeOf(reply))
		} else if replyValue.IsNil() {
			callChan <- fmt.Errorf("processing response (nil %v)", reflect.TypeOf(reply))
		} else {
			//assign
			//get underlying value since replyValue is a ptr
			replyValue = replyValue.Elem()
			replyValue.Set(rValue.Elem())
			callChan <- nil
		}
	}()
	if timeout <= 0 {
		timeout = 365 * 24 * time.Hour
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case result := <-callChan:
		return result
	case <-t.C:
		return fmt.Errorf("call %s timedout waiting for reply", serviceMethod)
	}
}
