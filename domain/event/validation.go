// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package event

import (
	"fmt"
	"strings"
	"time"

	"github.com/control-center/serviced/validation"
)

var validTime = time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)

//ValidEntity validates Event fields
func (e *Event) ValidEntity() error {

	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("Event.ID", e.ID))
	violations.Add(validation.NotEmpty("Event.HostID", e.HostID))
	violations.Add(validation.NotEmpty("Event.Agent", e.Agent))

	if e.Severity > 5 || e.Severity == 0 {
		violations.Add(fmt.Errorf("invalid severity value"))
	}
	if e.Count < 1 {
		violations.Add(fmt.Errorf("invalid count"))
	}
	if !(strings.HasPrefix(e.Class, "/") && len(e.Class) > 2) {
		violations.Add(fmt.Errorf("invalid class"))
	}
	if e.FirstTime.Before(validTime) {
		violations.Add(fmt.Errorf("invalid first time"))
	}
	if e.LastTime.Before(validTime) {
		violations.Add(fmt.Errorf("invalid last time"))
	}
	if e.Count == 1 {
		if e.LastTime != e.FirstTime {
			violations.Add(fmt.Errorf("first time & last time mismatch"))
		}
	}

	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}
