// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package servicedefinition

import (
	"github.com/zenoss/glog"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func getServiceDefinition(path string) (serviceDef *ServiceDefinition, err error) {

	// is path a dir
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("given path is not a directory")
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
		case subpath.Name() == "makefile": // ignoring makefiles present in service defs
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
				path, err := filepath.Rel(filepath.Join(path, subpath.Name()), p)
				if err != nil {
					return err
				}
				path = "/" + path
				if _, ok := svc.ConfigFiles[path]; !ok {
					svc.ConfigFiles[path] = ConfigFile{
						Filename: path,
						Content:  string(buffer),
					}
				} else {
					configFile := svc.ConfigFiles[path]
					configFile.Content = string(buffer)
					svc.ConfigFiles[path] = configFile
				}
				return nil
			}
			err = filepath.Walk(path+"/"+subpath.Name(), getFiles)
			if err != nil {
				return nil, err
			}
		case subpath.Name() == "FILTERS":
			if !subpath.IsDir() {
				return nil, fmt.Errorf(path + "/-FILTERS- must be a directory.")
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
			glog.V(4).Infof("Unrecognized file %s at %s", subpath.Name(), path)
		}
	}
	svc.Services = make([]ServiceDefinition, len(subServices))
	i := 0
	for _, subsvc := range subServices {
		svc.Services[i] = *subsvc
		i++
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
		if !strings.HasSuffix(filterName, ".conf") {
			glog.Warning("Skipping %s because it doesn't have a .conf extension", filterName)
			continue
		}
		// read the contents and add it to our map
		contents, err := ioutil.ReadFile(path + "/" + filterName)
		if err != nil {
			glog.Errorf("Unable to read the file %s, skipping", path+"/"+filterName)
			continue
		}
		filterName = strings.TrimSuffix(filterName, ".conf")
		filters[filterName] = string(contents)
	}
	glog.V(2).Infof("Here are the filters %v from path %s", filters, path)
	return filters, nil
}
