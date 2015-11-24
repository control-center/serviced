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

type method struct {
	methodV reflect.Value
	replyT  reflect.Type
}

type localClient struct {
	sync.RWMutex
	rcvrs  map[string]struct{}
	rcvrMs map[string]method
}

var (
	localRpcClient = &localClient{}
	localAddrs     = map[string]struct{}{}
)

func init() {
	localRpcClient.rcvrs = make(map[string]struct{})
	localRpcClient.rcvrMs = make(map[string]method)
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
	l.rcvrs[name] = struct{}{}
	rcvrType := reflect.TypeOf(rcvr)
	rcvrV := reflect.ValueOf(rcvr)

	for i := 0; i < rcvrV.NumMethod(); i++ {
		mName := rcvrType.Method(i).Name
		mType := rcvrType.Method(i).Type
		if mType.NumIn() != 3 {
			continue
		}
		mVal := rcvrV.MethodByName(mName)
		if !mVal.IsValid() {
			continue
		}
		key := fmt.Sprintf("%s.%s", name, mName)
		m := method{}
		m.methodV = mVal
		m.replyT = mVal.Type().In(1)
		l.rcvrMs[key] = m
	}

	return nil
}

func (l *localClient) Close() error {
	return nil
}

func (l *localClient) lookup(serviceMethod string) (reflect.Value, reflect.Type, error) {
	l.RLock()
	methodHolder, ok := l.rcvrMs[serviceMethod]
	l.RUnlock()
	if !ok {
		//check if service registered
		parts := strings.SplitN(serviceMethod, ".", 2)
		if len(parts) != 2 {
			return reflect.Value{}, nil, fmt.Errorf("Invalid service method: %s", serviceMethod)
		}
		name := parts[0]
		if _, ok := l.rcvrs[name]; !ok {
			return reflect.Value{}, nil, fmt.Errorf("can't find service %s", serviceMethod)
		}
		return reflect.Value{}, nil, fmt.Errorf("can't find method %s", serviceMethod)
	}

	return methodHolder.methodV, methodHolder.replyT, nil
}

func (l *localClient) Call(serviceMethod string, args interface{}, reply interface{}, timeout time.Duration) error {
	glog.V(3).Infof("RPC service method %s", serviceMethod)
	callChan := make(chan error, 1)

	go func() {
		method, rType, err := l.lookup(serviceMethod)
		if err != nil {
			callChan <- err
			return
		}

		inputs := make([]reflect.Value, 2)

		inputs[0] = reflect.ValueOf(args)

		//make a new one of the correct type
		rValue := reflect.New(rType.Elem())
		inputs[1] = rValue

		result := method.Call(inputs)
		errInterface := result[0].Interface()
		if errInterface != nil {
			callChan <- errInterface.(error)
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
