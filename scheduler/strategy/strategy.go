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

import (
	"errors"
	"strings"

	"github.com/control-center/serviced/domain/servicedefinition"
)

var (
	strategies        []Strategy
	ErrNoSuchStrategy = errors.New("no such scheduler strategy")
)

func init() {
	strategies = []Strategy{
		&BalanceStrategy{},
		&PackStrategy{},
		&PreferSeparateStrategy{},
		&RequireSeparateStrategy{},
	}
}

type Host interface {
	HostID() string
	TotalCores() int
	TotalMemory() uint64
	RunningServices() []ServiceConfig
}

type ServiceConfig interface {
	GetServiceID() string
	RequestedCorePercent() int
	RequestedMemoryBytes() uint64
	HostPolicy() servicedefinition.HostPolicy
}

type Strategy interface {
	// The name of this strategy
	Name() string
	// Chooses the best host for svc to run on
	SelectHost(svc ServiceConfig, hosts []Host) (Host, error)
}

func Get(name string) (Strategy, error) {
	for _, strategy := range strategies {
		if strings.ToLower(strategy.Name()) == strings.ToLower(name) {
			return strategy, nil
		}
	}
	return nil, ErrNoSuchStrategy
}
