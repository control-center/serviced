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
