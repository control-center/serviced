// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package retry

import (
	"time"
)

// Policy is the interface for a retry policy type.
type Policy interface {
	Name() string
	AllowRetry(retryCount int, elapsed time.Duration) (bool, time.Duration)
}
