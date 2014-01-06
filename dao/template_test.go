package dao

import (
    "fmt"
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
                ImageId: "ubuntu",
                ConfigFiles: map[string]ConfigFile{
                    "/etc/my.cnf": ConfigFile{Filename: "/etc/my.cnf", Content: "\n# SAMPLE config file for mysql\n\n[mysqld]\n\ninnodb_buffer_pool_size = 16G\n\n"},
                },
                Endpoints: []ServiceEndpoint{
                    ServiceEndpoint{
                        Protocol:    "tcp",
                        PortNumber:  8080,
                        Application: "www",
                        Purpose:     "export",
                    },
                    ServiceEndpoint{
                        Protocol:    "tcp",
                        PortNumber:  8081,
                        Application: "websvc",
                        Purpose:     "import",
                    },
                },
                LogConfigs: []LogConfig{
                    LogConfig{
                        Path: "/tmp/foo",
                        Type: "foo",
                    },
                },
            },
            ServiceDefinition{
                Name:    "s2",
                Command: "/usr/bin/python -m SimpleHTTPServer",
                ImageId: "ubuntu",
                Endpoints: []ServiceEndpoint{
                    ServiceEndpoint{
                        Protocol:    "tcp",
                        PortNumber:  8080,
                        Application: "websvc",
                        Purpose:     "export",
                    },
                },
                LogConfigs: []LogConfig{
                    LogConfig{
                        Path: "/tmp/foo",
                        Type: "foo",
                    },
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
    if a.ImageId != b.ImageId {
        return false, fmt.Sprintf("ImageIds are not equal %s != %s", a.ImageId, b.ImageId)
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

    for i, _ := range a.Services {
        identical, msg := a.Services[i].equals(&b.Services[i])
        if identical != true {
            return identical, msg
        }
    }

    // check config files
    if len(a.ConfigFiles) != len(b.ConfigFiles) {
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

    return true, ""
}

func TestServiceDefinitionFromPath(t *testing.T) {

    sd, err := ServiceDefinitionFromPath("./testsvc")

    if err != nil {
        t.Fatalf("Problem parsing template: %s", err)
    }
    identical, msg := testSvc.equals(sd)
    if !identical {
        t.Fatalf(msg)
    }

}
