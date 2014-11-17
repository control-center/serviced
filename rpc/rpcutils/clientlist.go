// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package rpcutils

import (
	"errors"
)

type clientList struct {
	clients chan Client
}

func (cl *clientList) getNext() (Client, error) {
	var client Client
	select {
	case client = <-cl.clients: //get the next client and add it back to the end
		cl.clients <- client
	default:
		return nil, errors.New("Client not available") //this shouldn't happen
	}
	return client, nil
}

func newClientList(addr string, size int) (*clientList, error) {
	return newClientListWithFactory(NewReconnectingClient, addr, size)
}

func newClientListWithFactory(factory func(add string) (Client, error), addr string, size int) (*clientList, error) {
	cList := clientList{make(chan Client, size)}

	for i := 0; i < size; i++ {
		client, err := factory(addr)
		if err != nil {
			return nil, err
		}
		select {
		case cList.clients <- client: //add to channel
		default:
			return nil, errors.New("Could not add Client to list")
		}
	}
	return &cList, nil
}
