// Copyright 2014 The Serviced Authors.
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

package registry

import "github.com/control-center/serviced/validation"

const excludeChars = " \t\r\n\v\f/?"

func (image *Image) ValidEntity() error {
	violations := validation.NewValidationError()
	violations.Add(validation.NotEmpty("Image.Library", image.Library))
	violations.Add(validation.ExcludeChars("Image.Library", image.Library, excludeChars))
	violations.Add(validation.NotEmpty("Image.Repo", image.Repo))
	violations.Add(validation.ExcludeChars("Image.Repo", image.Repo, excludeChars))
	violations.Add(validation.NotEmpty("Image.Tag", image.Tag))
	violations.Add(validation.ExcludeChars("Image.Tag", image.Tag, excludeChars))
	violations.Add(validation.NotEmpty("Image.UUID", image.UUID))
	violations.Add(validation.ExcludeChars("Image.UUID", image.UUID, excludeChars))
	if violations.HasError() {
		return violations
	}
	return nil
}
