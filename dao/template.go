package dao

import (
	"github.com/zenoss/glog"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/zenoss/serviced/utils"
)

func (s ServiceDefinition) String() string {
	return s.Name
}

// ByName implements sort.Interface for []ServiceDefinition based
// on Name field
type ServiceDefinitionByName []ServiceDefinition

func (a ServiceDefinitionByName) Len() int           { return len(a) }
func (a ServiceDefinitionByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ServiceDefinitionByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func ServiceDefinitionFromPath(path string) (*ServiceDefinition, error) {
	return getServiceDefinition(path)
}

// Is this the best way to do a set?
var fileswecareabout = map[string]struct{}{
	"service.json": struct{}{},
	"-CONFIGS-":    struct{}{},
	"FILTERS":      struct{}{},
}

func serviceJSONFilter(filename string) bool {
	_, ok := fileswecareabout[filepath.Base(filename)]
	return ok
}

// pathContains tells you whether s is a segment of path.
func pathContains(s, path string) bool {
	for path != "" {
		dir, file := filepath.Split(path)
		if file == s {
			return true
		}
		path = strings.TrimSuffix(dir, "/")
	}
	return false
}

// pathSplit splits a path on a given segment, returning the parent and child
// paths with separators trimmed appropriately.
//
// Example:
//     fmt.Println(pathSplit("/a/b/c/d", "c"))
// Result:
//    /a/b d
//
func pathSplit(path, segment string) (string, string) {
	split := strings.Split(path, segment)
	parent := split[0]
	child := ""
	if len(split) > 1 {
		child = split[1]
	}
	parent = strings.TrimSuffix(parent, "/")
	if parent == "" {
		parent = "."
	}
	return parent, strings.TrimSuffix(child, "/")
}

func getServiceDefinition(path string) (serviceDef *ServiceDefinition, err error) {

	files, err := utils.NewArchiveIterator(path)
	if err != nil {
		return nil, err
	}

	defs := map[string]*ServiceDefinition{}

	getOrCreateSvc := func(name string) *ServiceDefinition {
		if _, ok := defs[name]; !ok {
			svc := ServiceDefinition{}
			svc.Name = filepath.Base(name)
			if svc.ConfigFiles == nil {
				svc.ConfigFiles = map[string]ConfigFile{}
			}
			defs[name] = &svc
		}
		return defs[name]
	}

	// Create necessary objects from the reader
	for files.Iterate(nil) {
		name := files.Name()
		base := filepath.Base(name)
		switch {
		case base == "service.json":
			// load blob
			svcname := filepath.Dir(name)
			svc := getOrCreateSvc(svcname)
			err = json.NewDecoder(files).Decode(svc)
			if err != nil {
				glog.Errorf("Could not unmarshal service at %s", path)
				return nil, err
			}
			if svcname != "." {
				// This is a top-down walk so we always have the parent
				p := getOrCreateSvc(filepath.Dir(svcname))
				if p.Services == nil {
					p.Services = []ServiceDefinition{}
				}
				p.Services = append(p.Services, *svc)
			}
		case pathContains("-CONFIGS-", name):
			buffer, err := ioutil.ReadAll(files)
			if err != nil {
				return nil, err
			}
			p, child := pathSplit(name, "-CONFIGS-")
			confpath := "/" + child
			svc := getOrCreateSvc(p)
			if _, ok := svc.ConfigFiles[confpath]; !ok {
				svc.ConfigFiles[confpath] = ConfigFile{
					Filename: confpath,
					Content:  string(buffer),
				}
			} else {
				configFile := svc.ConfigFiles[confpath]
				configFile.Content = string(buffer)
			}
		case pathContains("FILTERS", name):
			// make sure it is a valid filter
			if !strings.HasSuffix(base, ".conf") {
				glog.Warning("Skipping %s because it doesn't have a .conf extension", base)
				continue
			}
			contents, err := ioutil.ReadAll(files)
			if err != nil {
				glog.Errorf("Unable to read the file %s, skipping", name)
				continue
			}
			p, _ := pathSplit(name, "FILTERS")
			svc := getOrCreateSvc(p)
			if svc.LogFilters == nil {
				svc.LogFilters = map[string]string{}
			}
			base = strings.TrimSuffix(base, ".conf")
			svc.LogFilters[base] = string(contents)
		default:
			glog.V(4).Infof("Unrecognized file %s at %s", base, filepath.Dir(name))
		}
	}
	if _, ok := defs["."]; !ok {
		msg := fmt.Sprintf("No service.json at the root of %s", path)
		glog.Errorf(msg)
		return nil, errors.New(msg)
	}
	svcdef := defs["."]
	return svcdef, nil
}
