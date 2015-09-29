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

type RequireSeparateStrategy struct{}

func (s *RequireSeparateStrategy) Name() string {
	return "require_separate"
}

func (s *RequireSeparateStrategy) SelectHost(service ServiceConfig, hosts []Host) (Host, error) {
	under, over := ScoreHosts(service, hosts)

	if under != nil && len(under) > 0 {
		for _, h := range under {
			if h.NumInstances == 0 {
				return h.Host, nil
			}
		}
	}
	if over != nil && len(over) > 0 {
		for _, h := range over {
			if h.NumInstances == 0 {
				return h.Host, nil
			}
		}
	}
	return nil, nil
}
