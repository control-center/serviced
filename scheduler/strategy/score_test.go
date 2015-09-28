// Copyright 2015 The Serviced Authors.
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

package strategy_test

import (
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

func MockHost(cores int, memory uint64) *host.Host {
	uuid, _ := utils.NewUUID36()
	return &host.Host{
		ID:     uuid,
		Cores:  cores,
		Memory: memory,
	}
}

func (s *StrategySuite) TestScoring(c *C) {

}