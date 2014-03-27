// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package client

type Driver interface {
	GetConnection(dsn string) (Connection, error)
}

type Connection interface {
	Close()
	SetOnClose(func())
	Create(path string, data []byte) error
	CreateDir(path string) error
	Exists(path string) (bool, error)
	Delete(path string) error
	Lock(path string) (lockId string, err error)
	Unlock(path, lockId string) error
}
