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

package strategy

type PackStrategy struct{}

func (s *PackStrategy) Name() string {
	return "pack"
}

func (s *PackStrategy) SelectHost(service ServiceConfig, hosts []Host) (Host, error) {
	under, over := ScoreHosts(service, hosts)

	// Return the host with the least amount of free resources that can handle
	// the service. In case of a tie, choose the one running more instances.
	if under != nil && len(under) > 0 {
		last := len(under) - 1
		choice := under[last]
		for i := last; i >= 0; i-- {
			scored := under[i]
			if scored.Score != choice.Score {
				break
			}
			if len(scored.Host.RunningServices()) > len(choice.Host.RunningServices()) {
				choice = scored
			}
		}
		return choice.Host, nil
	}

	// Return the host for which this service will least oversubscribe memory
	if over != nil && len(over) > 0 {
		return over[0].Host, nil
	}

	return nil, nil
}
