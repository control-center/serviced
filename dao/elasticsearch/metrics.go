// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elasticsearch

import (
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/metrics"
	"github.com/zenoss/glog"
)

func (dao *ControlPlaneDao) GetHostMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	s, err := dao.metricClient.GetHostMemoryStats(req.StartTime, req.HostID)
	if err != nil {
		glog.Errorf("Could not get host memory stats for %s: %s", req.HostID, err)
		return err
	}
	*stats = *s
	return nil
}

func (dao *ControlPlaneDao) GetServiceMemoryStats(req dao.MetricRequest, stats *metrics.MemoryUsageStats) error {
	s, err := dao.metricClient.GetServiceMemoryStats(req.StartTime, req.ServiceID)
	if err != nil {
		glog.V(2).Infof("Could not get service memory stats for %s: %s", req.ServiceID, err)
		return err
	}
	*stats = *s
	return nil
}

func (dao *ControlPlaneDao) GetInstanceMemoryStats(req dao.MetricRequest, stats *[]metrics.MemoryUsageStats) error {
	s, err := dao.facade.GetInstanceMemoryStats(req.StartTime, req.Instances...)
	if err != nil {
		glog.V(2).Infof("Could not get service instance stats for %+v: %s", req.Instances, err)
		return err
	}
	*stats = s
	return nil
}
