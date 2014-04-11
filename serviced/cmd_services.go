package main

import (
	"code.google.com/p/go.crypto/ssh/terminal"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"

	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"syscall"
	"unicode/utf8"
)

const tree_outfmt = "%-*s %-36.36s %-40.40s %-04d %-24.24s %-12s %-06d %-6s %-8.8s\n"
const tree_outfmt_header = "%-*s %-36.36s %-40.40s %4.4s %-24.24s %-12s %6.6s %-6s %-8.8s\n"

var tree_width int
var tree_charset map[string]string
var tree_utf8 map[string]string
var tree_ascii map[string]string

func init() {
	tree_width = 40
	tree_utf8 := make(map[string]string)
	tree_utf8["bar"] = "│ "
	tree_utf8["middle"] = "├─"
	tree_utf8["last"] = "└─"

	tree_ascii = make(map[string]string)
	tree_ascii["bar"] = "| "
	tree_ascii["middle"] = "|-"
	tree_ascii["last"] = "+-"

	tree_charset = tree_utf8 // set default charset for tree

}

// Print tree body of deployed services (no header)
func (node *svcStub) treePrintBody(indent string, root, last, raw bool) {

	if !root {
		fmt.Print(indent)
		width := tree_width
		if !raw {
			if last {
				fmt.Print(tree_charset["last"])
				indent = indent + "  "
			} else {
				fmt.Print(tree_charset["middle"])
				indent = indent + tree_charset["bar"]
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
			s.Launch,
			s.DeploymentId)
	}

	if node.subSvcs != nil {
		subSvcsNoSubSvcs := make(byStubName, 0, len(node.subSvcs))
		for _, svc := range node.subSvcs {
			if len(svc.subSvcs) == 0 {
				subSvcsNoSubSvcs = append(subSvcsNoSubSvcs, svc)
			}
		}
		sort.Sort(byStubName(subSvcsNoSubSvcs))

		subSvcsHasSubSvcs := make(byStubName, 0, len(node.subSvcs))
		for _, svc := range node.subSvcs {
			if len(svc.subSvcs) != 0 {
				subSvcsHasSubSvcs = append(subSvcsHasSubSvcs, svc)
			}
		}
		sort.Sort(byStubName(subSvcsHasSubSvcs))

		treePrintSubSvcs := func(subSvcs byStubName, indent string, raw bool) {
			i := 0
			for _, s := range subSvcs {
				i = i + 1
				s.treePrintBody(indent, false, i == len(subSvcs), raw)
			}
		}

		treePrintSubSvcs(subSvcsNoSubSvcs, indent, raw)
		treePrintSubSvcs(subSvcsHasSubSvcs, indent, raw)
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
			tree_width, "NAME", "SERVICE_ID", "STARTUP", "INST", "IMAGEID", "POOL", "DSTATE", "LAUNCH", "DEPIP")
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

// byStubName provides svcStub sort interface functions
type byStubName []*svcStub

func (v byStubName) Len() int           { return len(v) }
func (v byStubName) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v byStubName) Less(i, j int) bool { return v[i].value.Name < v[j].value.Name }

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

	var ascii bool
	if os.Getenv("SERVICED_ASCII") == "1" {
		ascii = true
	}
	cmd.BoolVar(&ascii, "ascii", ascii, "use ascii characters for service tree (env SERVICED_ASCII=1 will default to ascii)")

	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if ascii {
		tree_charset = tree_ascii
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

var editors [3]string

func init() {
	editors = [...]string{"vim", "vi", "nano"}
}

func findEditor(defaultEditor string) (string, error) {
	if len(defaultEditor) > 0 {
		editorPath, err := exec.LookPath(defaultEditor)
		if err != nil {
			return defaultEditor, fmt.Errorf("Editor '%s' not found.", defaultEditor)
		}
		return editorPath, nil
	}

	for _, editor := range editors {
		path, err := exec.LookPath(editor)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no editor found")
}

func editService(service *dao.Service, editor string) error {

	serviceJson, err := json.MarshalIndent(service, " ", " ")
	if err != nil {
		glog.Fatalf("Problem marshaling service object: %s", err)
	}

	var reader io.Reader
	if terminal.IsTerminal(syscall.Stdin) {
		editorPath, err := findEditor(editor)
		if err != nil {
			fmt.Printf("%s\n", err)
			return err
		}

		f, err := ioutil.TempFile("", fmt.Sprintf("serviced_edit_%s_", service.Id))
		if err != nil {
			glog.Fatalf("Could not write tempfile: %s", err)
		}
		defer f.Close()
		defer os.Remove(f.Name())
		_, err = f.Write(serviceJson)
		if err != nil {
			glog.Fatalf("Problem writing service json to file: %s", err)
		}

		editorCmd := exec.Command(editorPath, f.Name())
		editorCmd.Stdout = os.Stdout
		editorCmd.Stdin = os.Stdin
		editorCmd.Stderr = os.Stderr
		err = editorCmd.Run()

		if err != nil {
			glog.Fatal("Editor command returned error: %s", err)
		}
		_, err = f.Seek(0, 0)
		if err != nil {
			glog.Fatal("Could not seek to begining of tempfile: %s", err)
		}
		reader = f
	} else {
		_, err = os.Stdout.Write(serviceJson)
		if err != nil {
			glog.Fatal("Could not write service to terminal's stdout: %s", err)
		}
		reader = os.Stdin
	}

	serviceJson, err = ioutil.ReadAll(reader)
	if err != nil {
		glog.Fatal("Could not read tempfile back in: %s", err)
	}
	err = json.Unmarshal(serviceJson, &service)
	if err != nil {
		glog.Fatal("Could not parse json: %s", err)
	}
	return nil
}

func (cli *ServicedCli) CmdEditService(args ...string) error {
	cmd := Subcmd("edit-service", "[SERVICE_ID]", "edit a service")

	var editor string
	cmd.StringVar(&editor, "editor", os.Getenv("EDITOR"), "editor to use to edit service definition, also controlled by $EDITOR var")

	if err := cmd.Parse(args); err != nil {
		cmd.Usage()
		return nil
	}

	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}

	client := getClient()
	var service dao.Service
	err := client.GetService(cmd.Arg(0), &service)
	if err != nil {
		glog.Fatalf("Could not get service %s: %v", cmd.Arg(0), err)
	}

	err = editService(&service, editor)

	var unused int
	err = client.UpdateService(service, &unused)

	return err
}
