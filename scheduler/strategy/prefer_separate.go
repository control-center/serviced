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

type PreferSeparateStrategy struct{}

func (s *PreferSeparateStrategy) Name() string {
	return "prefer_separate"
}

func (s *PreferSeparateStrategy) SelectHost(service ServiceConfig, hosts []Host) (Host, error) {
	under, over := ScoreHosts(service, hosts)

	if under != nil && len(under) > 0 {
		var (
			min int = under[0].NumInstances
			idx int
		)
		for i, h := range under {
			if h.NumInstances < min {
				min = h.NumInstances
				idx = i
			}
		}
		return under[idx].Host, nil
	}

	// Return the host for which this service will least oversubscribe memory
	if over != nil && len(over) > 0 {
		return over[0].Host, nil
	}
	return nil, nil
}
