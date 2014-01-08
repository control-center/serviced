package main

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"encoding/json"
	"fmt"
	"unicode/utf8"
)

const tree_outfmt = "%-*s %-8.8s %-40.40s %-04d %-24.24s %-12s %-06d %-6s\n"
const tree_outfmt_header = "%-*s %-8.8s %-40.40s %4.4s %-24.24s %-12s %6.6s %-6s\n"

var tree_width int

func init() {
	tree_width = 40
}

// Print tree body of deployed services (no header)
func (node *svcStub) treePrintBody(indent string, root, last, raw bool) {

	if !root {
		fmt.Print(indent)
		width := tree_width
		if !raw {
			if last {
				fmt.Print("└─")
				indent = indent + "  "
			} else {
				fmt.Print("├─")
				indent = indent + "│ "
			}
			width = tree_width - utf8.RuneCountInString(indent)
		}
		s := node.value
		fmt.Printf(tree_outfmt,
			width, s.Name,
			s.Id,
			s.Startup,
			s.Instances,
			s.ImageId,
			s.PoolId,
			s.DesiredState,
			s.Launch)
	}
	if node.subSvcs != nil {
		i := 0
		for _, s := range node.subSvcs {
			i = i + 1
			s.treePrintBody(indent, false, i == len(node.subSvcs), raw)
		}
	}
}

// Return maximum depth of tree
func (node *svcStub) maxDepth(depth int) int {
	max := depth
	for _, n := range node.subSvcs {
		cdepth := n.maxDepth(depth + 1)
		if cdepth > max {
			max = cdepth
		}
	}
	return max
}

// Print a tree of deployed services
func (node *svcStub) treePrint(raw bool) {

	tree_width = node.maxDepth(0)*2 + 16
	if !raw {
		fmt.Printf(tree_outfmt_header,
			tree_width, "Name", "Id", "Startup", "Inst", "ImageId", "Pool", "DState", "Launch")
	}
	node.treePrintBody("", true, false, raw)
}

// Takes an array of services and creates a nested stub struct
func generateSvcMap(parentId string, services []*dao.Service, stub *svcStub) {
	for _, s := range services {
		if s.ParentServiceId == parentId {
			stub.add(s)
			generateSvcMap(s.Id, services, stub.subSvcs[s.Id])
		}
	}
}

// A simple nested struct to create a Service tree
type svcStub struct {
	value   *dao.Service
	subSvcs map[string]*svcStub
}

// Adds a Service object to the given svcStub.
func (svcMap *svcStub) add(s *dao.Service) {
	if svcMap.subSvcs == nil {
		svcMap.subSvcs = make(map[string]*svcStub)
	}
	stub := &svcStub{
		value:   s,
		subSvcs: make(map[string]*svcStub),
	}
	svcMap.subSvcs[s.Id] = stub
}

// Print the list of available services.
func (cli *ServicedCli) CmdServices(args ...string) error {
	cmd := Subcmd("services", "[CMD]", "Show services")

	var verbose bool
	cmd.BoolVar(&verbose, "verbose", false, "Show JSON representation for each service")

	var raw bool
	cmd.BoolVar(&raw, "raw", false, "Don't show the header line")

	if err := cmd.Parse(args); err != nil {
		return nil
	}

	controlPlane := getClient()
	var services []*dao.Service
	err := controlPlane.GetServices(&empty, &services)
	if err != nil {
		glog.Fatalf("Could not get services: %v", err)
	}

	if verbose == false {
		svcMap := &svcStub{}
		svcMap.value = &dao.Service{}
		generateSvcMap("", services, svcMap)
		svcMap.treePrint(raw)
	} else {
		servicesJson, err := json.MarshalIndent(services, " ", " ")
		if err != nil {
			glog.Fatalf("Problem marshaling services object: %s", err)
		}
		fmt.Printf("%s\n", servicesJson)
	}

	return err
}
