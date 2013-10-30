package elasticsearch

import "github.com/zenoss/serviced/dao"
import "github.com/zenoss/serviced/isvcs"
import "github.com/samuel/go-zookeeper/zk"
import "github.com/mattbaird/elastigo/api"
import "github.com/mattbaird/elastigo/core"
import "github.com/mattbaird/elastigo/search"
import "github.com/zenoss/glog"
import "encoding/json"
import "os/exec"
import "strconv"
import "strings"
import "errors"
import "time"
import "math/rand"
import "fmt"

//assert interface
var _ dao.ControlPlane = &ControlPlaneDao{}

// closure for geting a model
func getSource(index string, _type string) func(string, interface{}) error {
	return func(id string, source interface{}) error {
		return core.GetSource(index, _type, id, &source)
	}
}

// closure for searching a model
func searchUri(index string, _type string) func(string) (core.SearchResult, error) {
	return func(query string) (core.SearchResult, error) {
		return core.SearchUri(index, _type, query, "", 0)
	}
}

// closure for testing model existence
func exists(pretty *bool, index string, _type string) func(string) (bool, error) {
	return func(id string) (bool, error) {
		return core.Exists(*pretty, index, _type, id)
	}
}

// closure for indexing a model
func create(pretty *bool, index string, _type string) func(string, interface{}) (api.BaseResponse, error) {
	var (
		parentId  string = ""
		version   int    = 0
		op_type   string = "create"
		routing   string = ""
		timestamp string = ""
		ttl       int    = 0
		percolate string = ""
		timeout   string = ""
		refresh   bool   = true
	)
	return func(id string, data interface{}) (api.BaseResponse, error) {
		return core.IndexWithParameters(
			*pretty, index, _type, id, parentId, version, op_type, routing, timestamp, ttl, percolate, timeout, refresh, data)
	}
}

// closure for indexing a model
func index(pretty *bool, index string, _type string) func(string, interface{}) (api.BaseResponse, error) {
	var (
		parentId  string = ""
		version   int    = 0
		op_type   string = ""
		routing   string = ""
		timestamp string = ""
		ttl       int    = 0
		percolate string = ""
		timeout   string = ""
		refresh   bool   = true
	)
	return func(id string, data interface{}) (api.BaseResponse, error) {
		return core.IndexWithParameters(
			*pretty, index, _type, id, parentId, version, op_type, routing, timestamp, ttl, percolate, timeout, refresh, data)
	}
}

// closure for deleting a model
func _delete(pretty *bool, index string, _type string) func(string) (api.BaseResponse, error) {
	return func(id string) (api.BaseResponse, error) {
		//version=-1 and routing="" are not supported as of 9/30/13
		return core.Delete(*pretty, index, _type, id, -1, "")
	}
}

var (
	//enable pretty printed responses
	Pretty bool = false

	//model existance functions
	hostExists         func(string) (bool, error) = exists(&Pretty, "controlplane", "host")
	serviceExists      func(string) (bool, error) = exists(&Pretty, "controlplane", "service")
	serviceStateExists func(string) (bool, error) = exists(&Pretty, "controlplane", "servicestate")
	resourcePoolExists func(string) (bool, error) = exists(&Pretty, "controlplane", "resourcepool")

	//model index functions
	newHost                   func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "host")
	newService                func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "service")
	newServiceState           func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "servicestate")
	newResourcePool           func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "resourcepool")
	newServiceDeployment      func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "servicedeployment")
	newServiceTemplateWrapper func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "servicetemplatewrapper")

	//model index functions
	indexHost         func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "host")
	indexService      func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "service")
	indexServiceState func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "servicestate")
	indexResourcePool func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "resourcepool")

	//model delete functions
	deleteHost                   func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "host")
	deleteService                func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "service")
	deleteServiceState           func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "servicestate")
	deleteResourcePool           func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "resourcepool")
	deleteServiceTemplateWrapper func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "servicetemplatewrapper")

	//model get functions
	getHost                   func(string, interface{}) error = getSource("controlplane", "host")
	getService                func(string, interface{}) error = getSource("controlplane", "service")
	getServiceState           func(string, interface{}) error = getSource("controlplane", "servicestate")
	getResourcePool           func(string, interface{}) error = getSource("controlplane", "resourcepool")
	getServiceTemplateWrapper func(string, interface{}) error = getSource("controlplane", "servicetemplatewrapper")

	//model search functions, using uri based query
	searchHostUri         func(string) (core.SearchResult, error) = searchUri("controlplane", "host")
	searchServiceUri      func(string) (core.SearchResult, error) = searchUri("controlplane", "service")
	searchServiceStateUri func(string) (core.SearchResult, error) = searchUri("controlplane", "servicestate")
	searchResourcePoolUri func(string) (core.SearchResult, error) = searchUri("controlplane", "resourcepool")
)

type ControlPlaneDao struct {
	hostName   string
	port       int
	zookeepers []string
	scheduler  *scheduler
}

// convert search result of json host to dao.Host array
func toHosts(result *core.SearchResult) ([]*dao.Host, error) {
	var err error = nil
	var total = len(result.Hits.Hits)
	var hosts []*dao.Host = make([]*dao.Host, total)
	for i := 0; i < total; i += 1 {
		var host dao.Host
		err = json.Unmarshal(result.Hits.Hits[i].Source, &host)
		if err == nil {
			hosts[i] = &host
		} else {
			return nil, err
		}
	}

	return hosts, err
}

// convert search result of json host to dao.Host array
func toServiceTemplateWrappers(result *core.SearchResult) ([]*dao.ServiceTemplateWrapper, error) {
	var err error = nil
	var total = len(result.Hits.Hits)
	var wrappers []*dao.ServiceTemplateWrapper = make([]*dao.ServiceTemplateWrapper, total)
	for i := 0; i < total; i += 1 {
		var wrapper dao.ServiceTemplateWrapper
		err = json.Unmarshal(result.Hits.Hits[i].Source, &wrapper)
		if err == nil {
			wrappers[i] = &wrapper
		} else {
			return nil, err
		}
	}

	return wrappers, err
}

// convert search result of json services to dao.Service array
func toServices(result *core.SearchResult) ([]*dao.Service, error) {
	var err error = nil
	var total = len(result.Hits.Hits)
	var services []*dao.Service = make([]*dao.Service, total)
	for i := 0; i < total; i += 1 {
		var service dao.Service
		err = json.Unmarshal(result.Hits.Hits[i].Source, &service)
		if err == nil {
			services[i] = &service
		} else {
			return nil, err
		}
	}

	return services, err
}

// convert search result of json services to dao.Service array
func toServiceStates(result *core.SearchResult) ([]*dao.ServiceState, error) {
	var err error = nil
	var total = len(result.Hits.Hits)
	var states []*dao.ServiceState = make([]*dao.ServiceState, total)
	for i := 0; i < total; i += 1 {
		var state dao.ServiceState
		err = json.Unmarshal(result.Hits.Hits[i].Source, &state)
		if err == nil {
			states[i] = &state
		} else {
			return nil, err
		}
	}

	return states, err
}

// query for hosts using uri
func (this *ControlPlaneDao) queryHosts(query string) ([]*dao.Host, error) {
	result, err := searchHostUri(query)
	if err == nil {
		return toHosts(&result)
	}
	return nil, err
}

// query for services using uri
func (this *ControlPlaneDao) queryServices(query string) ([]*dao.Service, error) {
	result, err := searchServiceUri(query)
	if err == nil {
		return toServices(&result)
	}
	return nil, err
}

// query for service states using uri
func (this *ControlPlaneDao) queryServiceStates(query string) ([]*dao.ServiceState, error) {
	result, err := searchServiceStateUri(query)
	if err == nil {
		return toServiceStates(&result)
	}
	return nil, err
}

func walkTree(node *treenode) []string {
	if len(node.children) == 0 {
		return []string{node.id}
	}
	relatedServiceIds := make([]string, 0)
	for _, childNode := range node.children {
		for _, childId := range walkTree(childNode) {
			relatedServiceIds = append(relatedServiceIds, childId)
		}
	}
	return append(relatedServiceIds, node.id)
}

type treenode struct {
	id       string
	parent   string
	children []*treenode
}

func (this *ControlPlaneDao) getServiceTree(serviceId string, servicesList *[]*dao.Service) (servicesMap map[string]*treenode, topService *treenode) {
  servicesMap = make(map[string]*treenode)
  for _, service := range *servicesList {
    servicesMap[service.Id] = &treenode{
      service.Id,
      service.ParentServiceId,
      []*treenode{},
    }
  }

  // second time through builds our tree
  root := treenode{"root", "", []*treenode{}}
  for _, service := range *servicesList {
    node := servicesMap[service.Id]
    parent, found := servicesMap[service.ParentServiceId]
    // no parent means this node belongs to root
    if !found {
      parent = &root
    }
    parent.children = append(parent.children, node)
  }

  // now walk up the tree, then back down capturing all siblings for this service ID
  topService = servicesMap[serviceId]
  for len(topService.parent) != 0 {
    topService = servicesMap[topService.parent]
  }
  return
}

// Get a service endpoint.
func (this *ControlPlaneDao) GetServiceEndpoints(serviceId string, response *map[string][]*dao.ApplicationEndpoint) (err error) {
  var service dao.Service
  err = this.GetService( serviceId, &service)
  if err == nil {
    service_imports := service.GetServiceImports()
    if len(service_imports) > 0 {
      var request dao.EntityRequest
      var servicesList []*dao.Service
      err = this.GetServices( request, &servicesList)

      if err == nil {
	      // Map all services by Id so we can construct a tree for the current service ID
	      _, topService := this.getServiceTree(serviceId, &servicesList)
        // We should now have the top-level service for the current service ID
        remoteEndpoints := make(map[string][]*dao.ApplicationEndpoint)

        //build 'OR' query to grab all service states with in "service" tree
        var goQuery string
	      relatedServiceIds := walkTree(topService)
	      lastIndex := len(relatedServiceIds) - 1
        for idx, rsid := range relatedServiceIds {
          if idx == lastIndex {
            goQuery += "ServiceId:" + rsid + " OR "
          } else {
            goQuery += "ServiceId:" + rsid
          }
        }
        goQuery += " AND Terminated:0001"
	      glog.Infof("Query: %s", goQuery)
        query := search.Query().Search( goQuery)
	      result, err := search.Search("controlplane").Type("servicestate").Size("1000").Query(query).Result()
        if err == nil {
          states, err := toServiceStates(result)

          if err == nil {
            // for each proxied port, find list of potential remote endpoints
            for _, endpoint := range service_imports {
              key := fmt.Sprintf("%s:%d", endpoint.Protocol, endpoint.PortNumber)
              if _, exists := remoteEndpoints[key]; !exists {
                remoteEndpoints[key] = make([]*dao.ApplicationEndpoint, 0)
              }

              for _, ss := range states {
                port := ss.GetHostPort( endpoint.Protocol, endpoint.Application, endpoint.PortNumber)
                if port > 0 {
                  var ep dao.ApplicationEndpoint
                  ep.ServiceId = ss.ServiceId
                  ep.ContainerPort = endpoint.PortNumber
                  ep.HostPort = port
                  ep.HostIp = ss.HostIp
                  ep.ContainerIp = ss.PrivateIp
                  ep.Protocol = endpoint.Protocol
                  remoteEndpoints[key] = append( remoteEndpoints[key], &ep)
                }
              }
            }

            *response = remoteEndpoints
            glog.Infof("Return for %s is %v", serviceId, remoteEndpoints)
          }
        }
      }
    }
  }
  return
}

// add resource pool to index
func (this *ControlPlaneDao) AddResourcePool(pool dao.ResourcePool, unused *int) error {
	//glog.Infof("ControlPlaneDao.NewResourcePool: %+v", pool)
	id := strings.TrimSpace(pool.Id)
	if id == "" {
		return errors.New("empty ResourcePool.Id not allowed")
	}

	pool.Id = id
	response, err := newResourcePool(id, pool)
	//glog.Infof("ControlPlaneDao.NewResourcePool response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) AddHost(host dao.Host, unused *int) error {
	//glog.Infof("ControlPlaneDao.AddHost: %+v", host)
	id := strings.TrimSpace(host.Id)
	if id == "" {
		return errors.New("empty Host.Id not allowed")
	}

	host.Id = id
	response, err := newHost(id, host)
	//glog.Infof("ControlPlaneDao.AddHost response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) AddService(service dao.Service, unused *int) error {
	//glog.Infof("ControlPlaneDao.AddService: %+v", service)
	id := strings.TrimSpace(service.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}

	service.Id = id
	response, err := newService(id, service)
	//glog.Infof("ControlPlaneDao.AddService response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) UpdateResourcePool(pool dao.ResourcePool, unused *int) error {
	//glog.Infof("ControlPlaneDao.UpdateResourcePool: %+v", pool)

	id := strings.TrimSpace(pool.Id)
	if id == "" {
		return errors.New("empty ResourcePool.Id not allowed")
	}

	pool.Id = id
	response, err := indexResourcePool(id, pool)
	//glog.Infof("ControlPlaneDao.UpdateResourcePool response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) UpdateHost(host dao.Host, unused *int) error {
	//glog.Infof("ControlPlaneDao.UpdateHost: %+v", host)

	id := strings.TrimSpace(host.Id)
	if id == "" {
		return errors.New("empty Host.Id not allowed")
	}

	host.Id = id
	response, err := indexHost(id, host)
	//glog.Infof("ControlPlaneDao.UpdateHost response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) UpdateService(service dao.Service, unused *int) error {
	//glog.Infof("ControlPlaneDao.UpdateService: %+v", service)
	id := strings.TrimSpace(service.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}

	service.Id = id
	response, err := indexService(id, service)
	//glog.Infof("ControlPlaneDao.UpdateService response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) RemoveResourcePool(id string, unused *int) error {
	//glog.Infof("ControlPlaneDao.RemoveResourcePool: %s", id)
	_, err := deleteResourcePool(id)
	//glog.Infof("ControlPlaneDao.RemoveResourcePool response: %+v", response)
	return err
}

//
func (this *ControlPlaneDao) RemoveHost(id string, unused *int) error {
	//glog.Infof("ControlPlaneDao.RemoveHost: %s", id)
	_, err := deleteHost(id)
	//glog.Infof("ControlPlaneDao.RemoveHost response: %+v", response)
	return err
}

//
func (this *ControlPlaneDao) RemoveService(id string, unused *int) error {
	//glog.Infof("ControlPlaneDao.RemoveService: %s", id)
	_, err := deleteService(id)
	//glog.Infof("ControlPlaneDao.RemoveService response: %+v", response)
	return err
}

//
func (this *ControlPlaneDao) GetResourcePool(id string, pool *dao.ResourcePool) error {
	//glog.Infof("ControlPlaneDao.GetResourcePool: id=%s", id)
	request := dao.ResourcePool{}
	err := getResourcePool(id, &request)
	//glog.Infof("ControlPlaneDao.GetResourcePool: id=%s, resourcepool=%+v, err=%s", id, request, err)
	*pool = request
	return err
}

//
func (this *ControlPlaneDao) GetHost(id string, host *dao.Host) error {
	//glog.Infof("ControlPlaneDao.GetHost: id=%s", id)
	request := dao.Host{}
	err := getHost(id, &request)
	//glog.Infof("ControlPlaneDao.GetHost: id=%s, host=%+v, err=%s", id, request, err)
	*host = request
	return err
}

//
func (this *ControlPlaneDao) GetService(id string, service *dao.Service) error {
	//glog.Infof("ControlPlaneDao.GetService: id=%s", id)
	request := dao.Service{}
	err := getService(id, &request)
	//glog.Infof("ControlPlaneDao.GetService: id=%s, service=%+v, err=%s", id, request, err)
	*service = request
	return err
}

func (this *ControlPlaneDao) GetServicesForHost(hostId string, services *[]*dao.Service) (err error) {
  ssQuery := fmt.Sprintf( "HostId:%s", hostId)
  states, err := this.queryServiceStates( ssQuery)
  if err == nil {
    _services  := make( []*dao.Service, len( states))
    for i, ss := range states {
      var service dao.Service
      err = this.GetService( ss.ServiceId, &service)
      if err == nil {
        _services[i] = &service
      } else {
        return err
      }
    }

    *services = _services
  }

  return
}

func (this *ControlPlaneDao) GetRunningServices(request dao.EntityRequest, services *[]*dao.RunningService) error {
	now := time.Now().String()
	query := search.Query().Range(search.Range().Field("Terminated").From("2000-01-01T00:00:00").To(now))
	result, err := search.Search("controlplane").Type("servicestate").Size("1000").Query(query).Result()

	if err == nil {
		states, err := toServiceStates(result)
		if err == nil {
			var _services []*dao.RunningService = make([]*dao.RunningService, len(states))
			for i, ss := range states {
				var s dao.Service
				err = this.GetService(ss.ServiceId, &s)
				if err == nil {
					_services[i] = &dao.RunningService{}
					_services[i].Id = ss.Id
					_services[i].ServiceId = ss.ServiceId
					_services[i].StartedAt = ss.Started
					_services[i].Startup = s.Startup
					_services[i].Name = s.Name
					_services[i].Description = s.Description
					_services[i].Instances = s.Instances
					_services[i].PoolId = s.PoolId
					_services[i].ImageId = s.ImageId
					_services[i].DesiredState = s.DesiredState
					_services[i].ParentServiceId = s.ParentServiceId
				} else {
					return err
				}
			}
			*services = _services
		}
	}

	return err
}

func (this *ControlPlaneDao) GetRunningServicesForHost(hostId string, services *[]*dao.RunningService) error {
	now := time.Now().String()
	qs := fmt.Sprintf("HostId:%s", hostId)
	query := search.Query().Range(search.Range().Field("Terminated").From("2001-01-01T00:00:00").To(now))
	result, err := search.Search("controlplane").Type("servicestate").Size("1000").Query(query).Search(qs).Result()

	if err == nil {
		states, err := toServiceStates(result)
		if err == nil {
			var _services []*dao.RunningService = make([]*dao.RunningService, len(states))
			for i, ss := range states {
				var s dao.Service
				err = this.GetService(ss.ServiceId, &s)
				if err == nil {
					_services[i] = &dao.RunningService{}
					_services[i].Id = ss.Id
					_services[i].ServiceId = ss.ServiceId
					_services[i].StartedAt = ss.Started
					_services[i].Startup = s.Startup
					_services[i].Name = s.Name
					_services[i].Description = s.Description
					_services[i].Instances = s.Instances
					_services[i].PoolId = s.PoolId
					_services[i].ImageId = s.ImageId
					_services[i].DesiredState = s.DesiredState
					_services[i].ParentServiceId = s.ParentServiceId
				} else {
					return err
				}
			}
			*services = _services
		}
	}

	return err
}

func (this *ControlPlaneDao) GetServiceLogs(id string, logs *string) error {
  glog.Infof( "ControlPlaneDao.GetServiceLogs id=%s", id)
	query := search.Query().Search(fmt.Sprintf("ServiceId:%s", id))
	result, err := search.Search("controlplane").Type("servicestate").Size("1").Query(query).Sort(search.Sort("Started"), search.Sort("Terminated")).Result()
	if err == nil {
		states, err := toServiceStates(result)
    glog.Infof( "ControlPlaneDao.GetServiceLogs servicestates=%+v err=%s", states, err)
		if err == nil {
			if len(states) > 0 {
				cmd := exec.Command("docker", "logs", states[0].DockerId)
				output, err := cmd.Output()
				if err == nil {
					*logs = string(output)
				}
			} else {
				err = dao.ControlPlaneError{"Not found"}
			}
		}
	}
	return err
}

func (this *ControlPlaneDao) GetServiceStateLogs(id string, logs *string) error {
  glog.Infof( "ControlPlaneDao.GetServiceStateLogs id=%s", id)
	var serviceState dao.ServiceState
	err := this.GetServiceState(id, &serviceState)
  glog.Infof( "ControlPlaneDao.GetServiceStateLogs servicestate=%+v err=%s", serviceState, err)
	if err == nil {
		cmd := exec.Command("docker", "logs", serviceState.DockerId)
		output, err := cmd.Output()
		if err == nil {
			*logs = string(output)
		}
	}
	return err
}

//
func (this *ControlPlaneDao) GetResourcePools(request dao.EntityRequest, pools *map[string]*dao.ResourcePool) error {
	//glog.Infof("ControlPlaneDao.GetResourcePools")
	result, err := searchResourcePoolUri("_exists_:Id")
	//glog.Infof("ControlPlaneDao.GetResourcePools: err=%s", err)

	var resourcePools map[string]*dao.ResourcePool
	if err == nil {
		var total = len(result.Hits.Hits)
		var pool dao.ResourcePool
		resourcePools = make(map[string]*dao.ResourcePool)
		for i := 0; i < total; i += 1 {
			err := json.Unmarshal(result.Hits.Hits[i].Source, &pool)
			if err == nil {
				resourcePools[pool.Id] = &pool
			} else {
				return err
			}
		}
	}

	*pools = resourcePools
	return err
}

//
func (this *ControlPlaneDao) GetHosts(request dao.EntityRequest, hosts *map[string]*dao.Host) error {
	//glog.Infof("ControlPlaneDao.GetHosts")
	query := search.Query().Search("_exists_:Id")
	search_result, err := search.Search("controlplane").Type("host").Size("10000").Query(query).Result()
	//glog.Infof("ControlPlaneDao.GetHosts: err=%s", err)
	if err == nil {
		result, err := toHosts(search_result)
		if err == nil {
			hostmap := make(map[string]*dao.Host)
			var total = len(result)
			for i := 0; i < total; i += 1 {
				host := result[i]
				hostmap[host.Id] = host
			}
			*hosts = hostmap
		}
	}
	return err
}

//
func (this *ControlPlaneDao) GetServices(request dao.EntityRequest, services *[]*dao.Service) error {
	//glog.Infof("ControlPlaneDao.GetServices")
	result, err := this.queryServices("_exists_:Id")
	//glog.Infof("ControlPlaneDao.GetServices: err=%s", err)
	*services = result
	return err
}

//
func (this *ControlPlaneDao) GetHostsForResourcePool(poolId string, poolHosts *[]*dao.PoolHost) error {
	id := strings.TrimSpace(poolId)
	if id == "" {
		return errors.New("Illegal poolId: empty poolId not allowed")
	}

	query := fmt.Sprintf("PoolId:%s", id)
	result, err := this.queryHosts(query)

	if err == nil {
		var response []*dao.PoolHost = make([]*dao.PoolHost, len(result))
		for i := 0; i < len(result); i += 1 {
			poolHost := dao.PoolHost{result[i].Id, result[i].PoolId, result[i].IpAddr}
			response[i] = &poolHost
		}

		*poolHosts = response
	}

	return err
}

func (this *ControlPlaneDao) StartService(serviceId string, unused *string) error {
	//get the original service
	service := dao.Service{}
	err := this.GetService(serviceId, &service)
	if err == nil {
		//start this service
		var unusedInt int
		service.DesiredState = dao.SVC_RUN
		err = this.UpdateService(service, &unusedInt)

		if err == nil {
			//start all child services
			var query = fmt.Sprintf("ParentServiceId:%s", serviceId)
			subServices, err := this.queryServices(query)
			if err == nil {
				for _, service := range subServices {
					err = this.StartService(service.Id, unused)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return err
}

//
func (this *ControlPlaneDao) GetServiceState(id string, service *dao.ServiceState) error {
	//glog.Infof("ControlPlaneDao.GetServiceState: id=%s", id)
	request := dao.ServiceState{}
	err := getServiceState(id, &request)
	//glog.Infof("ControlPlaneDao.GetServiceState: id=%s, servicestate=%+v, err=%s", id, request, err)
	*service = request
	return err
}

//
func (this *ControlPlaneDao) GetServiceStates(serviceId string, servicestates *[]*dao.ServiceState) error {
	//glog.Infof( "ControlPlaneDao.GetServiceStates: serviceId=%s", serviceId)
  qs := fmt.Sprintf("ServiceId:%s AND Terminated:%s", serviceId, "0001")
	query := search.Query().Search(qs)
	result, err := search.Search("controlplane").Type("servicestate").Size("1000").Query(query).Result()
	//glog.Infof( "ControlPlaneDao.GetServiceStates: serviceId=%s, err=%s", serviceId, err)
	if err == nil {
		_ss, err := toServiceStates(result)
		if err == nil {
			*servicestates = _ss
		}
	}
	return err
}

// Update the current state of a service instance.
func (this *ControlPlaneDao) UpdateServiceState(state dao.ServiceState, unused *int) error {
	//glog.Infoln("ControlPlaneDao.UpdateServiceState state=%+v", state)
	response, err := indexServiceState(state.Id, &state)
	if response.Ok {
		return nil
	}
	return err
}

func (this *ControlPlaneDao) RestartService(serviceId string, unused *int) error {
	return dao.ControlPlaneError{"Unimplemented"}
}

func (this *ControlPlaneDao) StopService(id string, unused *int) error {
	var service dao.Service
	err := this.GetService(id, &service)
	if err == nil {
		service.DesiredState = dao.SVC_STOP
		err = this.UpdateService(service, unused)
		if err == nil {
			query := fmt.Sprintf("ParentServiceId:%s AND NOT Launch:manual", id)
			subservices, err := this.queryServices(query)
			if err == nil {
				for _, service := range subservices {
					var pid = service.ParentServiceId
					switch {
					case pid == "":
						glog.Warningf("Missing subservice: %s", pid)
					default:
						return this.StopService(pid, unused)
					}
				}
			}
		}
	}
	return err
}

func (this *ControlPlaneDao) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, unused *int) error {
	var wrapper dao.ServiceTemplateWrapper
	err := getServiceTemplateWrapper(request.TemplateId, &wrapper)
	if err == nil {
		var pool dao.ResourcePool
		err = this.GetResourcePool(request.PoolId, &pool)
		if err == nil {
			var template dao.ServiceTemplate
			err = json.Unmarshal([]byte(wrapper.Data), &template)
			if err == nil {
				return this.deployServiceDefinitions(template.Services, request.TemplateId, request.PoolId, "")
			}
		}
	}

	return err
}

func (this *ControlPlaneDao) deployServiceDefinitions(sds []dao.ServiceDefinition, template string, pool string, parent string) error {
	for _, sd := range sds {
		if err := this.deployServiceDefinition(sd, template, pool, parent); err != nil {
			return err
		}
	}
	return nil
}

func (this *ControlPlaneDao) deployServiceDefinition(sd dao.ServiceDefinition, template string, pool string, parent string) error {
	svcuuid, _ := dao.NewUuid()
	now := time.Now()

	ctx, err := json.Marshal(sd.Context)
	if err != nil {
		return err
	}

	// determine the desired state
	ds := dao.SVC_RUN

	if sd.Launch == "MANUAL" {
		ds = dao.SVC_STOP
	}

	svc := dao.Service{}
  svc.Id = svcuuid
  svc.Name = sd.Name
  svc.Context = string(ctx)
  svc.Startup = sd.Command
  svc.Description = sd.Description
  svc.Instances = sd.Instances.Min
  svc.ImageId = sd.ImageId
  svc.PoolId = pool
  svc.DesiredState = ds
  svc.Launch = sd.Launch
  svc.Endpoints = &sd.Endpoints
  svc.ParentServiceId = parent
  svc.CreatedAt = now
  svc.UpdatedAt = now

	var unused int
	err = this.AddService(svc, &unused)
	if err == nil {
		sduuid, _ := dao.NewUuid()
		deployment := dao.ServiceDeployment{sduuid, template, svc.Id, now}
		_, err := newServiceDeployment(sduuid, &deployment)
		if err == nil {
			return this.deployServiceDefinitions(sd.Services, template, pool, svc.Id)
		}
	}
	return err
}

func (this *ControlPlaneDao) AddServiceTemplate(serviceTemplate dao.ServiceTemplate, unused *int) error {
	var err error
	var uuid string
	var response api.BaseResponse
	var wrapper dao.ServiceTemplateWrapper

	data, err := json.Marshal(serviceTemplate)
	if err == nil {
		uuid, err = dao.NewUuid()
		if err == nil {
			wrapper.Id = uuid
			wrapper.Name = serviceTemplate.Name
			wrapper.Description = serviceTemplate.Description
			wrapper.Data = string(data)
			wrapper.ApiVersion = 1
			wrapper.TemplateVersion = 1
			response, err = newServiceTemplateWrapper(uuid, wrapper)
			if response.Ok {
				err = nil
			}
		}
	}
	return err
}

func (this *ControlPlaneDao) UpdateServiceTemplate(template dao.ServiceTemplate, unused *int) error {
	return fmt.Errorf("unimplemented UpdateServiceTemplate")
}

func (this *ControlPlaneDao) RemoveServiceTemplate(id string, unused *int) error {
	//glog.Infof("ControlPlaneDao.RemoveServiceTemplate: %s", id)
	_, err := deleteServiceTemplateWrapper(id)
	//glog.Infof("ControlPlaneDao.RemoveServiceTemplate response: %+v", response)
	return err
}

func (this *ControlPlaneDao) GetServiceTemplates(unused int, templates *map[string]*dao.ServiceTemplate) error {
	//glog.Infof("ControlPlaneDao.GetServiceTemplates")
	query := search.Query().Search("_exists_:Id")
	search_result, err := search.Search("controlplane").Type("servicetemplatewrapper").Size("1000").Query(query).Result()
	//glog.Infof("ControlPlaneDao.GetServiceTemplates: err=%s", err)

	if err == nil {
		result, err := toServiceTemplateWrappers(search_result)
		templatemap := make(map[string]*dao.ServiceTemplate)
		if err == nil {
			var total = len(result)
			for i := 0; i < total; i += 1 {
				var template dao.ServiceTemplate
				wrapper := result[i]
				err = json.Unmarshal([]byte(wrapper.Data), &template)
				templatemap[wrapper.Id] = &template
			}
		}
		*templates = templatemap
	}
	return err
}

func (this *ControlPlaneDao) lead(zkEvent <-chan zk.Event) {
	shutdown_mode := false
	for {
		if shutdown_mode {
			break
		}
		time.Sleep(time.Second)
		func() error {
			select {
			case evt := <-zkEvent:
				// shut this thing down
				shutdown_mode = true
				glog.Errorf("Got a zkevent, leaving lead: %v", evt)
				return nil
			default:
				// passthru
			}

			// get all service that are supposed to be running
			services, err := this.queryServices("DesiredState:1")
			if err == nil {
				for _, service := range services {
					// check current state
					var serviceStates []*dao.ServiceState
					err = this.GetServiceStates(service.Id, &serviceStates)
					if err == nil {
						// pick services instances to start
						if len(serviceStates) < service.Instances {
							instancesToStart := service.Instances - len(serviceStates)
							for i := 0; i < instancesToStart; i++ {
								// get hosts
								var pool_hosts []*dao.PoolHost
								err = this.GetHostsForResourcePool(service.PoolId, &pool_hosts)
								if err == nil {
									if len(pool_hosts) == 0 {
										glog.Infof("Pool %s has no hosts", service.PoolId)
										break
									}

									// randomly select host
									service_host := pool_hosts[rand.Intn(len(pool_hosts))]
									serviceState, err := service.NewServiceState(service_host.HostId)
                  serviceState.HostIp = service_host.HostIp
									if err == nil {
										glog.Infof("cp: serviceState %s", serviceState.Started)
										_, err = newServiceState(serviceState.Id, &serviceState)
									} else {
										glog.Errorf("Error creating ServiceState instance: %v", err)
										break
									}
								} else {
									return err
								}
							}
							// pick service instances to kill!
						} else if len(serviceStates) > service.Instances {
							instancesToKill := len(serviceStates) - service.Instances
							for i := 0; i < instancesToKill; i++ {
								glog.Infof("CP: Choosing to kill %s:%s\n", serviceStates[i].HostId, serviceStates[i].DockerId)
								serviceStates[i].Terminated = time.Date(2, time.January, 1, 0, 0, 0, 0, time.UTC)
								var unused int
								err = this.UpdateServiceState(*serviceStates[i], &unused)
							}
						}
					} else {
						return err
					}
				}

				// find the services that should not be running
				var xservices []*dao.Service
				xservices, err := this.queryServices("NOT DesiredState:1")
				if err == nil {
					for _, service := range xservices {
						var serviceStates []*dao.ServiceState
						err = this.GetServiceStates(service.Id, &serviceStates)
						if err == nil {
							for _, ss := range serviceStates {
								glog.Infof("CP: killing %s:%s\n", ss.HostId, ss.DockerId)
								ss.Terminated = time.Date(2, time.January, 1, 0, 0, 0, 0, time.UTC)
								var unused int
								err = this.UpdateServiceState(*ss, &unused)
								if err != nil {
									glog.Warningf("CP: %s:%s wouldn't die", ss.HostId, ss.DockerId)
								}
							}
						} else {
							glog.Errorf("Got error checking service state of %s, %s", service.Id, err.Error())
							return err
						}
					}
				}
			}
			return err
		}()
	}
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int) (*ControlPlaneDao, error) {
	glog.Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
	api.Domain = hostName
	api.Port = strconv.Itoa(port)
	return &ControlPlaneDao{hostName, port, nil, nil}, nil
}

var hostIdCmdString = "/usr/bin/hostid"

// hostId retreives the system's unique id, on linux this maps
// to /usr/bin/hostid.
func hostId() (hostid string, err error) {
	cmd := exec.Command(hostIdCmdString)
	stdout, err := cmd.Output()
	if err != nil {
		return hostid, err
	}
	return strings.TrimSpace(string(stdout)), err
}

func NewControlSvc(hostName string, port int, zookeepers []string) (s *ControlPlaneDao, err error) {
	glog.Info("calling NewControlSvc()")
	defer glog.Info("leaving NewControlSvc()")
	s, err = NewControlPlaneDao(hostName, port)

	if err != nil {
		return
	}

	if len(zookeepers) == 0 {
		isvcs.ZookeeperContainer.Run()
		s.zookeepers = []string{"127.0.0.1:2181"}
	} else {
		s.zookeepers = zookeepers
	}

	// ensure that a default pool exists
	var pool dao.ResourcePool
	err = s.GetResourcePool("default", &pool)
	if err != nil {
		glog.Infof("'default' resource pool not found; creating...")
		default_pool := dao.ResourcePool{}
		default_pool.Id = "default"

		var unused int
		err = s.AddResourcePool(default_pool, &unused)
		if err != nil {
			return
		}
	}

	hid, err := hostId()
	if err != nil {
		return nil, err
	}

	go s.handleScheduler(hid)
	return s, err
}

func (s *ControlPlaneDao) handleScheduler(hostId string) {

	for {
		conn, _, err := zk.Connect(s.zookeepers, time.Second*10)
		if err != nil {
			time.Sleep(time.Second * 3)
			continue
		}
		scheduler, shutdown := newScheduler("", conn, hostId, s.lead)
		scheduler.Start()
		select {
		case <-shutdown:
		}
	}
}
