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

package container

import (
	"github.com/control-center/serviced/zzk/registry"
	zkservice "github.com/control-center/serviced/zzk/service"

	"os"
	"strings"
)

var (
	master_ip = os.Getenv("SERVICED_MASTER_IP")
	hostip    = strings.Split(master_ip, ":")[0]

	cp_controlplane_bind = zkservice.ImportBinding{
		Application:    "controlplane",
		Purpose:        "import",
		PortNumber:     443,
		VirtualAddress: "",
	}
	cp_controlplane_exports = []registry.ExportDetails{
		registry.ExportDetails{
			ExportBinding: zkservice.ExportBinding{
				Application: "controlplane",
				Protocol:    "tcp",
				PortNumber:  443,
			},
			PrivateIP:  "127.0.0.1",
			HostIP:     hostip,
			InstanceID: 0,
		}}

	cp_consumer_bind = zkservice.ImportBinding{
		Application:    "controlplane_consumer",
		Purpose:        "import",
		PortNumber:     8444,
		VirtualAddress: "",
	}

	cp_consumer_exports = []registry.ExportDetails{
		registry.ExportDetails{
			ExportBinding: zkservice.ExportBinding{
				Application: "controlplane_consumer",
				Protocol:    "tcp",
				PortNumber:  8443,
			},
			PrivateIP:  "127.0.0.1",
			HostIP:     hostip,
			InstanceID: 0,
		}}

	cp_logstash_bind = zkservice.ImportBinding{
		Application:    "controlplane_logstash_tcp",
		Purpose:        "import",
		PortNumber:     5042,
		VirtualAddress: "",
	}

	cp_logstash_exports = []registry.ExportDetails{
		registry.ExportDetails{
			ExportBinding: zkservice.ExportBinding{
				Application: "controlplane_logstash_tcp",
				Protocol:    "tcp",
				PortNumber:  5042,
			},
			PrivateIP:  "127.0.0.1",
			HostIP:     hostip,
			InstanceID: 0,
		}}

	cp_filebeat_bind = zkservice.ImportBinding{
		Application:    "controlplane_logstash_filebeat",
		Purpose:        "import",
		PortNumber:     5043,
		VirtualAddress: "",
	}

	cp_filebeat_exports = []registry.ExportDetails{
		registry.ExportDetails{
			ExportBinding: zkservice.ExportBinding{
				Application: "controlplane_logstash_filebeat",
				Protocol:    "tcp",
				PortNumber:  5043,
			},
			PrivateIP:  "127.0.0.1",
			HostIP:     hostip,
			InstanceID: 0,
		}}

	cp_kibana_bind = zkservice.ImportBinding{
		Application:    "controlplane_kibana_tcp",
		Purpose:        "import",
		PortNumber:     5601,
		VirtualAddress: "",
	}

	cp_kibana_exports = []registry.ExportDetails{
		registry.ExportDetails{
			ExportBinding: zkservice.ExportBinding{
				Application: "controlplane_kibana_tcp",
				Protocol:    "tcp",
				PortNumber:  5601,
			},
			PrivateIP:  "127.0.0.1",
			HostIP:     hostip,
			InstanceID: 0,
		}}
)
