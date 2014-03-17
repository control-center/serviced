package retry

import (
	"time"
)

type Policy interface {
	Name() string
	AllowRetry(retryCount int, elapsed time.Duration) bool
	Close()
}


