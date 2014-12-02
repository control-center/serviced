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

package api

import (
	"github.com/control-center/serviced/script"
)

// ScriptRun
func (a *api) ScriptRun(fileName string, config script.Config) error {
	r, err := script.NewRunnerFromFile(fileName, config)
	if err !=nil{
		return err
	}
	r.Run()
	return err
}

func (a *api) ScriptParse(fileName string, config script.Config) error {
	_, err := script.NewRunnerFromFile(fileName, config)
	return err
}
