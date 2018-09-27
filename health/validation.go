// Copyright 2018 The Serviced Authors.
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

package health

import (
	"fmt"
	"github.com/control-center/serviced/validation"
)

func (hc HealthCheck) ValidEntity() error {
	violations := validation.NewValidationError()
	if len(hc.KillExitCodes) > 0 && hc.KillCountLimit == 0 {
		violations.Add(fmt.Errorf("the KillCountLimit must be set if KillExitCodes are specified"))
	}

	if violations.HasError() {
		return violations
	}
	return nil
}
