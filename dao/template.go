package dao

import (
	"github.com/zenoss/glog"

	"encoding/json"
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
	for len(path) >= 0 {
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
	return strings.TrimSuffix(parent, "/"), strings.TrimSuffix(child, "/")
}

func getServiceDefinition(path string) (serviceDef *ServiceDefinition, err error) {

	files, err := utils.NewArchiveIterator(path)
	if err != nil {
		return nil, err
	}

	defs := map[string]*ServiceDefinition{}
	confs := map[string][]ConfigFile{}
	filters := map[string]map[string]string{}

	// Create necessary objects from the reader
	for files.Iterate(nil) {
		name := files.Name()
		parent, base := filepath.Dir(name), filepath.Base(name)
		switch {
		case base == "service.json":
			// load blob
			svc := ServiceDefinition{}
			err = json.NewDecoder(files).Decode(&svc)
			if err != nil {
				glog.Errorf("Could not unmarshal service at %s", path)
				return nil, err
			}
			svc.Name = filepath.Base(path)
			if svc.ConfigFiles == nil {
				svc.ConfigFiles = make(map[string]ConfigFile)
			}
			defs[parent] = &svc
		case pathContains("-CONFIGS-", name):
			buffer, err := ioutil.ReadAll(files)
			if err != nil {
				return nil, err
			}
			p, child := pathSplit(name, "-CONFIGS-")
			if _, ok := confs[p]; !ok {
				confs[p] = []ConfigFile{}
			}
			confs[p] = append(confs[p], ConfigFile{
				Filename: "/" + child,
				Content:  string(buffer),
			})
		case strings.Contains(name, "FILTERS"):
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
			base = strings.TrimSuffix(base, ".conf")
			if _, ok := filters[parent]; !ok {
				filters[parent] = map[string]string{}
			}
			filters[parent][base] = string(contents)
		}
	}
	fmt.Println(defs)
	// Now stitch it together
	return &ServiceDefinition{}, nil

	//// look at sub services
	//subServices := make(map[string]*ServiceDefinition)
	//subpaths, err := ioutil.ReadDir(path)
	//if err != nil {
	//	return nil, err
	//}
	//for _, subpath := range subpaths {
	//	switch {
	//	case subpath.Name() == "service.json":
	//		continue
	//	case subpath.Name() == "makefile": // ignoring makefiles present in service defs
	//		continue
	//	case subpath.Name() == "-CONFIGS-":
	//		if !subpath.IsDir() {
	//			return nil, fmt.Errorf("-CONFIGS- must be a director: %s", path)
	//		}
	//		getFiles := func(p string, f os.FileInfo, err error) error {
	//			if f.IsDir() {
	//				return nil
	//			}
	//			buffer, err := ioutil.ReadFile(p)
	//			if err != nil {
	//				return err
	//			}
	//			path := p[len(path)+len(subpath.Name())+1:]
	//			if _, ok := svc.ConfigFiles[path]; !ok {
	//				svc.ConfigFiles[path] = ConfigFile{
	//					Filename: path,
	//					Content:  string(buffer),
	//				}
	//			} else {
	//				configFile := svc.ConfigFiles[path]
	//				configFile.Content = string(buffer)
	//			}
	//			return nil
	//		}
	//		err = filepath.Walk(path+"/"+subpath.Name(), getFiles)
	//		if err != nil {
	//			return nil, err
	//		}
	//	case subpath.Name() == "FILTERS":
	//		if !subpath.IsDir() {
	//			return nil, fmt.Errorf(path + "/-FILTERS- must be a directory.")
	//		}
	//		filters, err := getFiltersFromDirectory(path + "/" + subpath.Name())
	//		if err != nil {
	//			glog.Errorf("Error fetching filters at "+path, err)
	//		} else {
	//			svc.LogFilters = filters
	//		}
	//	case subpath.IsDir():
	//		subsvc, err := getServiceDefinition(path + "/" + subpath.Name())
	//		if err != nil {
	//			return nil, err
	//		}
	//		subServices[subpath.Name()] = subsvc
	//	default:
	//		glog.Errorf("Unreconised file %s at %s", subpath, path)
	//	}
	//}
	//svc.Services = make([]ServiceDefinition, len(subServices))
	//i := 0
	//for _, subsvc := range subServices {
	//	svc.Services[i] = *subsvc
	//	i += 1
	//}
	//return &svc, err
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
