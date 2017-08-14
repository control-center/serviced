// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logfilter

import (
	"strings"

	"github.com/control-center/serviced/validation"
)

// ValidEntity validates LogFilter fields
func (lf *LogFilter) ValidEntity() error {
	trimmed := strings.TrimSpace(lf.Name)
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("LogFilter.Name", lf.Name))
	violations.Add(validation.StringsEqual(lf.Name, trimmed, "leading and trailing spaces not allowed for LogFilter name"))

	trimmed = strings.TrimSpace(lf.Version)
	violations.Add(validation.NotEmpty("LogFilter.Version", lf.Version))
	violations.Add(validation.StringsEqual(lf.Version, trimmed, "leading and trailing spaces not allowed for LogFilter version"))

	violations.Add(validation.NotEmpty("LogFilter.Filter", lf.Filter))

	if len(violations.Errors) > 0 {
		return violations
	}
	return nil
}
