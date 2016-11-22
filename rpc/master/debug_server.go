// Copyright 2016 The Serviced Authors.
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

package master

import (
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/metrics"
)

var metricsTimer *metrics.MetricTimer

// instantiate the package logger
var plog = logging.PackageLogger()

func (s *Server) DebugEnableMetrics(unused struct{}, results *string) error {
	ctx := s.context()
	if ctx.Metrics().Enabled {
		*results = "metrics collection already enabled"

	} else {
		ctx.Metrics().Enabled = true
		ctx.Metrics().GroupName = "EnableDebugMetrics"
		metricsTimer = ctx.Metrics().Start("DebugMetrics")
		*results = "metrics collection enabled"
		plog.Info("Metrics collection enabled")
	}
	return nil
}

func (s *Server) DebugDisableMetrics(unused struct{}, results *string) error {
	ctx := s.context()
	if ctx.Metrics().Enabled {
		ctx.Metrics().Stop(metricsTimer)
		ctx.Metrics().LogAndCleanUp(metricsTimer)
		ctx.Metrics().Enabled = false
		metricsTimer = nil
		*results = "metrics collection disabled"
		plog.Infof("Metrics collection disabled")
	} else {
		*results = "metrics collection not enabled"
	}
	return nil
}
