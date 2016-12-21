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

package volume

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"text/tabwriter"

	"github.com/docker/go-units"
	"github.com/zenoss/glog"
)

type Status interface {
	String() string
	GetUsageData() []Usage
}

type SimpleStatus struct { // see Docker - look at their status struct and borrow heavily.
	Driver     DriverType
	DriverData map[string]string
	UsageData  []Usage
}

type Usage interface {
	GetMetricName() string
	GetLabel() string
	GetType() string
	GetValueUInt64() (uint64, error)
	GetValueFloat64() (float64, error)
}

var ErrWrongDataType = errors.New("Wrong data type for Usage Value")

type UsageInt struct {
	MetricName string
	Label      string
	Type       string
	Value      uint64
}

func (u UsageInt) GetMetricName() string {
	return u.MetricName
}

func (u UsageInt) GetLabel() string {
	return u.Label
}

func (u UsageInt) GetType() string {
	return u.Type
}

func (u UsageInt) GetValueUInt64() (uint64, error) {
	return u.Value, nil
}

func (u UsageInt) GetValueFloat64() (float64, error) {
	return 0.0, ErrWrongDataType
}

type UsageFloat struct {
	MetricName string
	Label      string
	Type       string
	Value      float64
}

func (u UsageFloat) GetMetricName() string {
	return u.MetricName
}

func (u UsageFloat) GetLabel() string {
	return u.Label
}

func (u UsageFloat) GetType() string {
	return u.Type
}

func (u UsageFloat) GetValueUInt64() (uint64, error) {
	return 0, ErrWrongDataType
}

func (u UsageFloat) GetValueFloat64() (float64, error) {
	return u.Value, nil
}

// This struct is stupid, for the sake of using interfaces AND RPC ser/deser.
type Statuses struct {
	DeviceMapperStatusMap map[string]*DeviceMapperStatus
	SimpleStatusMap       map[string]*SimpleStatus
}

func (s *Statuses) GetAllStatuses() map[string]Status {
	result := make(map[string]Status)
	for k, s := range s.DeviceMapperStatusMap {
		result[k] = s
	}
	for k, s := range s.SimpleStatusMap {
		result[k] = s
	}
	return result
}

// GetStatus retrieves the status for the volumeNames passed in. If volumeNames is empty, it gets all statuses.
func GetStatus() *Statuses {
	result := &Statuses{}
	result.DeviceMapperStatusMap = make(map[string]*DeviceMapperStatus)
	result.SimpleStatusMap = make(map[string]*SimpleStatus)
	driverMap := getDrivers()
	for path, driver := range *driverMap {
		status, err := driver.Status()
		if err != nil {
			glog.Warningf("Error getting driver status for path %s: %v", path, err)
		}
		if status != nil {
			if driver.DriverType() == DriverTypeDeviceMapper {
				result.DeviceMapperStatusMap[path] = status.(*DeviceMapperStatus)
			} else {
				result.SimpleStatusMap[path] = status.(*SimpleStatus)
			}
		} else {
			glog.Warningf("nil status returned for path %s", path)
		}
	}
	return result
}

func (s SimpleStatus) GetUsageData() []Usage {
	return s.UsageData
}

func (s SimpleStatus) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Driver:                 %s\n", s.Driver))
	for key, value := range s.DriverData {
		buffer.WriteString(fmt.Sprintf("%-24s%s\n", fmt.Sprintf("%s:", key), value))
	}
	buffer.WriteString(fmt.Sprintf("Usage Data:\n"))
	for _, usage := range s.UsageData {
		val, err := usage.GetValueUInt64()
		if err == ErrWrongDataType {
			val, err := usage.GetValueFloat64()
			if err != nil {
				glog.Error("Usage data has no value")
			}
			buffer.WriteString(fmt.Sprintf("\t%s %s: %s\n", usage.GetLabel(), usage.GetType(), units.BytesSize(val)))
		} else if err != nil {
			glog.Errorf("Could not get usage data: %s", err)
		} else {
			buffer.WriteString(fmt.Sprintf("\t%s %s: %s\n", usage.GetLabel(), usage.GetType(), units.BytesSize(float64(val))))
		}

	}
	return buffer.String()
}

// TenantStorageStats represents tenant-specific storage usage details.
type TenantStorageStats struct {
	TenantID            string
	VolumePath          string
	PoolAvailableBlocks uint64

	DeviceName string

	DeviceTotalBlocks       uint64
	DeviceAllocatedBlocks   uint64
	DeviceUnallocatedBlocks uint64

	FilesystemTotal     uint64
	FilesystemUsed      uint64
	FilesystemAvailable uint64

	Errors []string

	NumberSnapshots         int
	SnapshotAllocatedBlocks uint64
}

type DeviceMapperStatus struct {
	Driver     DriverType
	DriverType string
	DriverPath string
	PoolName   string

	PoolDataTotal     uint64
	PoolDataAvailable uint64
	PoolDataUsed      uint64

	PoolMetadataTotal     uint64
	PoolMetadataAvailable uint64
	PoolMetadataUsed      uint64

	DriverData map[string]string
	UsageData  []Usage
	Tenants    []TenantStorageStats

	Errors []string
}

func (s DeviceMapperStatus) GetUsageData() []Usage {
	return s.UsageData
}

var DeviceMapperStatusTemplate = `
Driver:	{{.Driver}}
Driver Type:	{{.DriverType}}
Volume Path:	{{.DriverPath}}

Thin Pool
---------
Logical Volume:	{{.PoolName}}
Metadata (total/used/avail): 	{{bytes .PoolMetadataTotal}}	/ {{bytes .PoolMetadataUsed}}	({{percent .PoolMetadataUsed .PoolMetadataTotal}})	/ {{bytes .PoolMetadataAvailable}}	({{percent .PoolMetadataAvailable .PoolMetadataTotal}})
Data (total/used/avail):	{{bytes .PoolDataTotal}}	/ {{bytes .PoolDataUsed}}	({{percent .PoolDataUsed .PoolDataTotal}})	/ {{bytes .PoolDataAvailable}}	({{percent .PoolDataAvailable .PoolDataTotal}})
{{with $parent := .}}{{range .Tenants}}
{{.TenantID}} Application Data
-----------------------------------------
Volume Mount Point:	{{.VolumePath}}
Filesystem (total/used/avail):	{{bytes .FilesystemTotal}} / {{bytes .FilesystemUsed}}	({{percent .FilesystemUsed .FilesystemTotal}}) / {{bytes .FilesystemAvailable}}	({{percent .FilesystemAvailable .FilesystemTotal}})
Virtual device size:	{{blocksToBytes .DeviceTotalBlocks}}
{{range .Errors}}
{{.}}
{{end -}}
{{end -}}
{{end -}}
{{range .Errors}}
{{.}}
{{end -}}
`

var funcMap = template.FuncMap{
	"bytes":         ToBytes,
	"blocksToBytes": BlocksToBytes,
	"bytesToBlocks": BytesToBlocks,
	"percent":       Percent,
}

func (s DeviceMapperStatus) String() string {
	var buffer bytes.Buffer
	w := tabwriter.NewWriter(&buffer, 4, 0, 1, ' ', 0)
	tpl, err := template.New("status").Funcs(funcMap).Parse(DeviceMapperStatusTemplate)
	if err != nil {
		return err.Error()
	}

	if err := tpl.Execute(w, s); err != nil {
		return err.Error()
	}
	w.Flush()
	return buffer.String()
}
