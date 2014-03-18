package client


type Driver interface {
	Create(path string, data []byte) error
	CreateDir(path string) error
	Exists(path string) (bool, error)
}


