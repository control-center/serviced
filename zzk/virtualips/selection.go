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

package virtualips

import (
	"errors"
	"math/rand"

	h "github.com/control-center/serviced/domain/host"
)

// HostSelectionStrategy represents a selection strategy to choose from a list of hosts
type HostSelectionStrategy interface {
	Select(hosts []h.Host) (h.Host, error)
}

// ErrNoHosts is returned when a strategy is given no hosts to select.
var ErrNoHosts = errors.New("there are no hosts to select")

// RandomHostSelectionStrategy will randomly pick from one of the provided hosts
type RandomHostSelectionStrategy struct{}

// Select will randomly pick of the provided hosts. If no hosts are given,
// ErrNoHosts will be returned.
func (s *RandomHostSelectionStrategy) Select(hosts []h.Host) (h.Host, error) {
	n := len(hosts)
	if n == 0 {
		return h.Host{}, ErrNoHosts
	}
	return hosts[rand.Intn(n)], nil
}
