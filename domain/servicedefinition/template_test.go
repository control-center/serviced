package servicedefinition

import (
	"github.com/zenoss/serviced/commons"

	"fmt"
	"log"
	"sort"
	"testing"
)

var testSvc *ServiceDefinition

func init() {

	// Test definition should match on disk (filesystem based) definition
	testSvc = &ServiceDefinition{
		Name:        "testsvc",
		Description: "Top level service. This directory is part of a unit test.",
		Services: []ServiceDefinition{
			ServiceDefinition{
				Name:    "s1",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageID: "ubuntu",
				ConfigFiles: map[string]ConfigFile{
					"/etc/my.cnf": ConfigFile{Owner: "root:root", Filename: "/etc/my.cnf", Permissions: "0660", Content: "\n# SAMPLE config file for mysql\n\n[mysqld]\n\ninnodb_buffer_pool_size = 16G\n\n"},
				},
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "www",
						Name:        "www",
						Purpose:     "export",
					},
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8081,
						Application: "websvc",
						Name:        "websvc",
						Purpose:     "import",
						AddressConfig: AddressResourceConfig{
							Port:     8081,
							Protocol: commons.TCP,
						},
					},
				},
				LogConfigs: []LogConfig{
					LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
					},
				},
				Snapshot: SnapshotCommands{
					Pause:  "echo pause",
					Resume: "echo resume",
				},
			},
			ServiceDefinition{
				Name:    "s2",
				Command: "/usr/bin/python -m SimpleHTTPServer",
				ImageID: "ubuntu",
				ConfigFiles: map[string]ConfigFile{
					"/foo/bar.txt": ConfigFile{
						Filename:    "/foo/bar.txt",
						Owner:       "zenoss:zenoss",
						Permissions: "660",
						Content:     "baz\n",
					},
				},
				Endpoints: []ServiceEndpoint{
					ServiceEndpoint{
						Protocol:    "tcp",
						PortNumber:  8080,
						Application: "websvc",
						Name:        "websvc",
						Purpose:     "export",
						VHosts:      []string{"testhost"},
					},
				},
				LogConfigs: []LogConfig{
					LogConfig{
						Path: "/tmp/foo",
						Type: "foo",
					},
				},
				Snapshot: SnapshotCommands{
					Pause:  "echo pause",
					Resume: "echo resume",
				},
			},
		},
	}

}

// This function checks if the given ServiceDefinition is equivalent. True is
// returned if true, false is returned otherwise. A non-empty message is returned
// that identifies the first inequality that was discovered.
func (a *ServiceDefinition) equals(b *ServiceDefinition) (identical bool, msg string) {

	if a.Name != b.Name {
		return false, fmt.Sprintf("Names are not equal %s != %s", a.Name, b.Name)
	}
	if a.Description != b.Description {
		return false, fmt.Sprintf("Descriptions are not equal %s != %s", a.Description, b.Description)
	}
	if a.ImageID != b.ImageID {
		return false, fmt.Sprintf("ImageIDs are not equal %s != %s", a.ImageID, b.ImageID)
	}
	if a.Command != b.Command {
		return false, fmt.Sprintf("Commands are not equal %s != %s", a.Command, b.Command)
	}
	if len(a.Endpoints) != len(b.Endpoints) {
		return false, fmt.Sprintf("Number of endpoints differ between %s [%s] and %s [%s]",
			a.Name, b.Name, len(a.Endpoints), len(b.Endpoints))
	}
	if len(a.Services) != len(b.Services) {
		return false, fmt.Sprintf("Number of sub services differ between %s [%s] and %s [%s]",
			a.Name, b.Name, len(a.Endpoints), len(b.Endpoints))
	}
	sort.Sort(ServiceDefinitionByName(a.Services))
	sort.Sort(ServiceDefinitionByName(b.Services))

	for i := range a.Services {
		identical, msg := a.Services[i].equals(&b.Services[i])
		if identical != true {
			return identical, msg
		}
	}

	// check config files
	if len(a.ConfigFiles) != len(b.ConfigFiles) {
		log.Printf("s1 :%v  \n\ns2 %s", a.ConfigFiles, b.ConfigFiles)
		return false, fmt.Sprintf("%s has %d configs, %s has %d configs",
			a, len(a.ConfigFiles), b, len(b.ConfigFiles))
	}
	for filename, confFile := range a.ConfigFiles {
		if _, ok := b.ConfigFiles[filename]; !ok {
			return false, fmt.Sprintf("%s has configFile %s, but %s does not", a, filename, b)
		}
		if confFile != b.ConfigFiles[filename] {
			return false, fmt.Sprintf("ConfigFile mismatch %s, a: %v, b: %v", filename, confFile, b.ConfigFiles[filename])
		}
	}

	// check snapshot
	if a.Snapshot.Pause != b.Snapshot.Pause {
		return false, fmt.Sprintf("Snapshot pause commands are not equal %s != %s", a.Snapshot.Pause, b.Snapshot.Pause)
	}
	if a.Snapshot.Resume != b.Snapshot.Resume {
		return false, fmt.Sprintf("Snapshot resume commands are not equal %s != %s", a.Snapshot.Resume, b.Snapshot.Resume)
	}

	return true, ""
}

func TestServiceDefinitionFromPath(t *testing.T) {

	sd, err := BuildFromPath("./testsvc")

	if err != nil {
		t.Fatalf("Problem parsing template: %s", err)
	}
	identical, msg := testSvc.equals(sd)
	if !identical {
		t.Fatalf(msg)
	}

}
