package coordinator

type Driver interface {
	Close()
	Create(path string, data []byte) error
	Delete(path string, version int32) error
	Exists(path string) (bool, error)
	Sync(path string) error
}

type Retryable interface {
	ShouldContinue() bool
	MarkComplete()
	SetError(error)
}


