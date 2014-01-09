package dao

import (
	"github.com/zenoss/glog"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
				path := p[len(path)+len(subpath.Name())+1:]
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
		case subpath.Name() == "filters":
			if !subpath.IsDir() {
				return nil, fmt.Errorf(path + "/filters must be a directory.")
			}
			filters, err := getFiltersFromDirectory(path + "/" + subpath.Name())
			if err != nil {
				glog.Errorf("Error fetching filters at "+path, err)
			} else {
				svc.LogFilters = filters
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

// this function takes a filter directory and creates a map
// of filters by looking at the content in that directory.
// it is assumed the filter name is the name of the file minus
// the .conf part. So test.conf would be a filter named "test"
func getFiltersFromDirectory(path string) (filters map[string]string, err error) {
	filters = make(map[string]string)
	subpaths, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, subpath := range subpaths {
		filterName := subpath.Name()

		// make sure it is a valid filter
		if !strings.Contains(filterName, ".conf") {
			glog.V(2).Infof("Skipping %s because it doesn't have a .conf extension", filterName)
			continue
		}
		// read the contents and add it to our map
		contents, err := ioutil.ReadFile(path + "/" + filterName)
		if err != nil {
			glog.Errorf("Unable to read the file %s, skipping", path+"/"+filterName)
			continue
		}
		filterName = strings.Replace(filterName, ".conf", "", 1)
		filters[filterName] = string(contents)
	}
	glog.V(2).Infof("Here are the filters %v from path %s", filters, path)
	return filters, nil
}
