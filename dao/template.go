package dao

import (
	"github.com/zenoss/glog"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

func getServiceDefinition(path string) (serviceDef *ServiceDefinition, err error) {

	// is path a dir
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("Given path is not a directory")
	}

	// look for service.json
	serviceFile := fmt.Sprintf("%s/service.json", path)
	blob, err := ioutil.ReadFile(serviceFile)
	if err != nil {
		return nil, err
	}

	// load blob
	svc := ServiceDefinition{}
	err = json.Unmarshal(blob, &svc)
	if err != nil {
		glog.Errorf("Could not unmarshal service at %s", path)
		return nil, err
	}
	svc.Name = filepath.Base(path)
	if svc.ConfigFiles == nil {
		svc.ConfigFiles = make(map[string]ConfigFile)
	}

	// look at sub services
	subServices := make(map[string]*ServiceDefinition)
	subpaths, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, subpath := range subpaths {
		switch {
		case subpath.Name() == "service.json":
			continue
		case subpath.Name() == "-CONFIGS-":
			if !subpath.IsDir() {
				return nil, fmt.Errorf("-CONFIGS- must be a director: %s", path)
			}
			getFiles := func(p string, f os.FileInfo, err error) error {
				if f.IsDir() {
					return nil
				}
				buffer, err := ioutil.ReadFile(p)
				if err != nil {
					return err
				}
				path := p[len(path)+len("-CONFIGS-")-1:]
				if _, ok := svc.ConfigFiles[path]; !ok {
					svc.ConfigFiles[path] = ConfigFile{
						Filename: path,
						Content:  string(buffer),
					}
				} else {
					configFile := svc.ConfigFiles[path]
					configFile.Content = string(buffer)
				}
				return nil
			}
			err = filepath.Walk(path+"/"+subpath.Name(), getFiles)
			if err != nil {
				return nil, err
			}
		case subpath.IsDir():
			subsvc, err := getServiceDefinition(path + "/" + subpath.Name())
			if err != nil {
				return nil, err
			}
			subServices[subpath.Name()] = subsvc
		default:
			glog.Errorf("Unreconised file %s at %s", subpath, path)
		}
	}
	svc.Services = make([]ServiceDefinition, len(subServices))
	i := 0
	for _, subsvc := range subServices {
		svc.Services[i] = *subsvc
		i += 1
	}
	return &svc, err
}
