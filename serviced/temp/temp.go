package main

import (
	"fmt"

	"github.com/kr/pretty"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/rpc/agent"
)


func main() {
	client, err := agent.NewClient("localhost:4979")
	if err != nil {
		glog.Fatalf("%s", err)
	}
	req := agent.BuildHostRequest{
		IP: "10.87.120.128:4979",
		PoolID: "default",
	}
	response, err := client.BuildHost(req)
	if err != nil {
		glog.Fatalf("%s", err)
	}
	fmt.Printf("%#\n", pretty.Formatter(response))
}

