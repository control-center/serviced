package retry

import (
	"time"
)

// Policy is the interface for a retry policy type.
type Policy interface {
	Name() string
	AllowRetry(retryCount int, elapsed time.Duration) bool
	Close()
}
