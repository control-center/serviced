// Copyright 2014 The Serviced Authors.
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

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/proxy"
)

// Initializer for serviced host subcommands
func (c *ServicedCli) initMux() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "mux",
		Usage:       "Shows mux connection info",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "connections",
				Usage:        "Shows mux connections for hosts",
				Description:  "serviced mux connections [HOSTID]",
				BashComplete: c.printHostsFirst,
				Action:       c.cmdMuxConnectionInfo,
				Flags: []cli.Flag{
					cli.BoolFlag{"verbose, v", "Show JSON format"},
				},
			},
		},
	})
}

// serviced mux connections [--verbose, -v] [HOSTID]
func (c *ServicedCli) cmdMuxConnectionInfo(ctx *cli.Context) {
	var err error
	var info map[string]proxy.TCPMuxConnectionInfo
	if len(ctx.Args()) > 0 {
		hostID := ctx.Args()[0]
		info, err = c.driver.GetMuxConnectionInfoForHost(hostID)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	} else {
		info, err = c.driver.GetMuxConnectionInfo()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	}
	if info == nil || len(info) == 0 {
		fmt.Fprintln(os.Stderr, "unable to find any mux connection info")
		return
	}

	if ctx.Bool("verbose") {
		if jsonInfo, err := json.MarshalIndent(info, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal mux connection info: %s", err)
		} else {
			fmt.Println(string(jsonInfo))
		}
	} else {
		sortedkeys := make([]string, len(info))
		i := 0
		for k, _ := range info {
			sortedkeys[i] = k
			i++
		}
		sort.Strings(sortedkeys)

		tableStatus := newtable(0, 8, 2)
		tableStatus.printrow("DEST_HOST_IP", "DEST_ADDRESS", "DEST_SVC_NAME", "DEST_SVC_INST",
			"SRC_HOST_IP", "SRC_ADDRESS", "SRC_SVC_NAME", "SRC_SVC_INST", "DURATION")
		for _, k := range sortedkeys {
			ms := info[k]
			tableStatus.printrow(ms.AgentHostIP, ms.DstRemoteAddr,
				fmt.Sprintf("%s/%d", ms.ApplicationEndpoint.Application, ms.ApplicationEndpoint.InstanceID),
				fmt.Sprintf("%s/%d", ms.ApplicationEndpoint.ServiceID, ms.ApplicationEndpoint.InstanceID),
				ms.Src.AgentHostIP, ms.SrcRemoteAddr,
				fmt.Sprintf("%s/%s", ms.Src.ServiceName, ms.Src.InstanceID),
				fmt.Sprintf("%s/%s", ms.Src.ServiceID, ms.Src.InstanceID),
				time.Since(ms.CreatedAt))
		}
		tableStatus.flush()
	}
	return
}
