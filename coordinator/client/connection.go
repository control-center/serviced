// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.
package client

type Driver interface {
	GetConnection(dsn, basePath string) (Connection, error)
}

type Lock interface {
	Lock() error
	Unlock() error
}
type Leader interface {
	TakeLead() error
	ReleaseLead() error
	Current() (data []byte, err error)
}

type Connection interface {
	Close()
	SetOnClose(func())
	Create(path string, data []byte) error
	CreateDir(path string) error
	Exists(path string) (bool, error)
	Delete(path string) error

	NewLock(path string) Lock
	NewLeader(path string, data []byte) Leader
}
