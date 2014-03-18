package client


type Driver interface {
	Create(path string, data []byte) error
	Exists(path string) (bool, error)
}


