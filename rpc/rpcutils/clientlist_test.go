// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package rpcutils

import (
	"fmt"
	"testing"
)

var count = 0

type testClient struct {
	addr  string
	count int
}

func (tc *testClient) Close() error {
	return nil
}

func (tc *testClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return nil
}

func factory(addr string) (Client, error) {
	count = count + 1
	fmt.Printf("creating %d\n", count)
	return &testClient{addr, count}, nil
}

func TestCreation(t *testing.T) {
	count = 0
	size := 1
	testAddr := "foo:222"

	getFactory := func(addr string) (Client, error) {
		count = count + 1
		if count > size {
			t.Errorf("factory called %d times, max should be %d", count, size)
		}
		return &testClient{addr, count}, nil
	}

	_, err := newClientListWithFactory(getFactory, testAddr, size)
	if err != nil {
		t.Errorf("unexpected: %s", err)
	}
	if count != size {
		t.Errorf("expected factory to be called %d, was called %d", size, count)
	}

	count = 0
	size = 5
	_, err = newClientListWithFactory(getFactory, testAddr, size)
	if err != nil {
		t.Errorf("unexpected: %s", err)
	}
	if count != size {
		t.Errorf("expected factory to be called %d, was called %d", size, count)
	}

}

func TestGetOne(t *testing.T) {
	count = 0
	size := 1
	testAddr := "foo:222"

	cl, err := newClientListWithFactory(factory, testAddr, size)
	if err != nil {
		t.Errorf("unexpected: %s", err)
	}

	if count != size {
		t.Errorf("expected factory to be called %d, was called %d", size, count)
	}

	var c1, c2 Client
	c1, err = cl.getNext()
	if err != nil {
		t.Errorf("unexpected: %s", err)
	}
	c2, err = cl.getNext()
	if err != nil {
		t.Errorf("unexpected: %s", err)
	}

	if c1 == nil || c2 == nil || c1 != c2 {
		t.Errorf("unexpected: c1 = %#v, c2 = %#v", c1, c2)
	}
}

func TestGetN(t *testing.T) {
	count = 0
	size := 5
	testAddr := "foo:222"

	cl, err := newClientListWithFactory(factory, testAddr, size)
	if err != nil {
		t.Errorf("unexpected: %s", err)
	}
	if count != size {
		t.Errorf("expected factory to be called %d, was called %d", size, count)
	}

	clients := make(map[Client]interface{})

	//get more than created and verify
	for i := 0; i < size*2; i++ {
		c, err := cl.getNext()
		if err != nil {
			t.Errorf("unexpected: %s", err)
		}
		clients[c] = nil
	}

	if len(clients) != size {
		t.Errorf("unexpected: %#v", clients)
	}
}
