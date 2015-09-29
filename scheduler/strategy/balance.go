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

type BalanceStrategy struct{}

func (s *BalanceStrategy) Name() string {
	return "balance"
}

func (s *BalanceStrategy) SelectHost(service ServiceConfig, hosts []Host) (Host, error) {
	under, over := ScoreHosts(service, hosts)

	// Return the host with the greatest amount of free resources that can handle
	// the service. In case of a tie, choose the one running fewer instances.
	if under != nil && len(under) > 0 {
		choice := under[0]
		for _, scored := range under {
			if scored.Score != choice.Score {
				break
			}
			if len(scored.Host.RunningServices()) < len(choice.Host.RunningServices()) {
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
