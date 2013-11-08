package elasticsearch

import (
	"github.com/mattbaird/elastigo/api"
	"github.com/mattbaird/elastigo/core"
	"github.com/mattbaird/elastigo/search"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/zzk"

	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

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
	zkDao      *zzk.ZkDao
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

// query for hosts using uri
func (this *ControlPlaneDao) queryHosts(query string) ([]*dao.Host, error) {
	result, err := searchHostUri(query)
	if err == nil {
		return toHosts(&result)
	}
	return nil, err
}

// query for services using uri
func (this *ControlPlaneDao) queryServices(queryStr, quantity string) ([]*dao.Service, error) {
	query := search.Query().Search(queryStr)
	result, err := search.Search("controlplane").Type("service").Size(quantity).Query(query).Result()
	if err == nil {
		return toServices(result)
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
	glog.V(2).Infof(" getServiceTree = %s", serviceId)
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
	glog.V(2).Infof("ControlPlaneDao.GetServiceEndpoints serviceId=%s", serviceId)
	var service dao.Service
	err = this.GetService(serviceId, &service)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceEndpoints service=%+v err=%s", service, err)
		return
	}

	service_imports := service.GetServiceImports()
	if len(service_imports) > 0 {
		//glog.Infof( "  %+v service imports=%+v", service, service_imports)

		var request dao.EntityRequest
		var servicesList []*dao.Service
		err = this.GetServices(request, &servicesList)
		if err != nil {
			return
		}

		// Map all services by Id so we can construct a tree for the current service ID
		glog.V(2).Infof("ServicesList: %d", len(servicesList))
		//for i,s := range servicesList {
		//  glog.Infof(" %d = %+v", i, s)
		//}
		_, topService := this.getServiceTree(serviceId, &servicesList)
		// We should now have the top-level service for the current service ID
		remoteEndpoints := make(map[string][]*dao.ApplicationEndpoint)

		//build 'OR' query to grab all service states with in "service" tree
		relatedServiceIds := walkTree(topService)
		var states []*dao.ServiceState
		err = this.zkDao.GetServiceStates(&states, relatedServiceIds...)
		if err != nil {
			return
		}

		// for each proxied port, find list of potential remote endpoints
		for _, endpoint := range service_imports {
			//glog.Infof( "Finding exports for import: %+v", endpoint)
			key := fmt.Sprintf("%s:%d", endpoint.Protocol, endpoint.PortNumber)
			if _, exists := remoteEndpoints[key]; !exists {
				remoteEndpoints[key] = make([]*dao.ApplicationEndpoint, 0)
			}

			for _, ss := range states {
				port := ss.GetHostPort(endpoint.Protocol, endpoint.Application, endpoint.PortNumber)
				if port > 0 {
					var ep dao.ApplicationEndpoint
					ep.ServiceId = ss.ServiceId
					ep.ContainerPort = endpoint.PortNumber
					ep.HostPort = port
					ep.HostIp = ss.HostIp
					ep.ContainerIp = ss.PrivateIp
					ep.Protocol = endpoint.Protocol
					remoteEndpoints[key] = append(remoteEndpoints[key], &ep)
				}
			}
		}

		*response = remoteEndpoints
		glog.Infof("Return for %s is %+v", serviceId, remoteEndpoints)
	}
	return
}

// add resource pool to index
func (this *ControlPlaneDao) AddResourcePool(pool dao.ResourcePool, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.NewResourcePool: %+v", pool)
	id := strings.TrimSpace(pool.Id)
	if id == "" {
		return errors.New("empty ResourcePool.Id not allowed")
	}

	pool.Id = id
	response, err := newResourcePool(id, pool)
	glog.V(2).Infof("ControlPlaneDao.NewResourcePool response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) AddHost(host dao.Host, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.AddHost: %+v", host)
	id := strings.TrimSpace(host.Id)
	if id == "" {
		return errors.New("empty Host.Id not allowed")
	}

	host.Id = id
	response, err := newHost(id, host)
	glog.V(2).Infof("ControlPlaneDao.AddHost response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) AddService(service dao.Service, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.AddService: %+v", service)
	id := strings.TrimSpace(service.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}

	service.Id = id
	response, err := newService(id, service)
	glog.V(2).Infof("ControlPlaneDao.AddService response: %+v", response)
	if response.Ok {
		return this.zkDao.AddService(&service)
	}
	return err
}

//
func (this *ControlPlaneDao) UpdateResourcePool(pool dao.ResourcePool, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateResourcePool: %+v", pool)

	id := strings.TrimSpace(pool.Id)
	if id == "" {
		return errors.New("empty ResourcePool.Id not allowed")
	}

	pool.Id = id
	response, err := indexResourcePool(id, pool)
	glog.V(2).Infof("ControlPlaneDao.UpdateResourcePool response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) UpdateHost(host dao.Host, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateHost: %+v", host)

	id := strings.TrimSpace(host.Id)
	if id == "" {
		return errors.New("empty Host.Id not allowed")
	}

	host.Id = id
	response, err := indexHost(id, host)
	glog.V(2).Infof("ControlPlaneDao.UpdateHost response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) UpdateService(service dao.Service, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateService: %+v", service)
	id := strings.TrimSpace(service.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}

	service.Id = id
	response, err := indexService(id, service)
	glog.V(2).Infof("ControlPlaneDao.UpdateService response: %+v", response)
	if response.Ok {
		return this.zkDao.UpdateService(&service)
	}
	return err
}

//
func (this *ControlPlaneDao) RemoveResourcePool(id string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveResourcePool: %s", id)
	response, err := deleteResourcePool(id)
	glog.V(2).Infof("ControlPlaneDao.RemoveResourcePool response: %+v", response)
	return err
}

//
func (this *ControlPlaneDao) RemoveHost(id string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveHost: %s", id)
	response, err := deleteHost(id)
	glog.V(2).Infof("ControlPlaneDao.RemoveHost response: %+v", response)
	return err
}

//
func (this *ControlPlaneDao) RemoveService(id string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveService: %s", id)
	response, err := deleteService(id)
	glog.V(2).Infof("ControlPlaneDao.RemoveService response: %+v", response)
	return err
}

//
func (this *ControlPlaneDao) GetResourcePool(id string, pool *dao.ResourcePool) error {
	glog.V(2).Infof("ControlPlaneDao.GetResourcePool: id=%s", id)
	request := dao.ResourcePool{}
	err := getResourcePool(id, &request)
	glog.V(2).Infof("ControlPlaneDao.GetResourcePool: id=%s, resourcepool=%+v, err=%s", id, request, err)
	*pool = request
	return err
}

//
func (this *ControlPlaneDao) GetHost(id string, host *dao.Host) error {
	glog.V(2).Infof("ControlPlaneDao.GetHost: id=%s", id)
	request := dao.Host{}
	err := getHost(id, &request)
	glog.V(2).Infof("ControlPlaneDao.GetHost: id=%s, host=%+v, err=%s", id, request, err)
	*host = request
	return err
}

//
func (this *ControlPlaneDao) GetService(id string, service *dao.Service) error {
	glog.V(2).Infof("ControlPlaneDao.GetService: id=%s", id)
	request := dao.Service{}
	err := getService(id, &request)
	glog.V(2).Infof("ControlPlaneDao.GetService: id=%s, service=%+v, err=%s", id, request, err)
	*service = request
	return err
}

func (this *ControlPlaneDao) GetRunningServices(request dao.EntityRequest, services *[]*dao.RunningService) error {
	return this.zkDao.GetAllRunningServices(services)
}

func (this *ControlPlaneDao) GetRunningServicesForHost(hostId string, services *[]*dao.RunningService) error {
	return this.zkDao.GetRunningServicesForHost(hostId, services)
}

func (this *ControlPlaneDao) GetRunningServicesForService(serviceId string, services *[]*dao.RunningService) error {
	return this.zkDao.GetRunningServicesForService(serviceId, services)
}

func (this *ControlPlaneDao) GetServiceLogs(id string, logs *string) error {
	glog.Infof("ControlPlaneDao.GetServiceLogs id=%s", id)
	var serviceStates []*dao.ServiceState
	err := this.zkDao.GetServiceStates(&serviceStates, id)
	if err != nil {
		return err
	}
	if len(serviceStates) == 0 {
		glog.Infoln("Unable to find any running services for %s", id)
		return nil
	}
	cmd := exec.Command("docker", "logs", serviceStates[0].DockerId)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Unable to return logs because: %v", err)
		return err
	}
	*logs = string(output)
	return nil

}

func (this *ControlPlaneDao) GetServiceStateLogs(request dao.ServiceStateRequest, logs *string) error {
	/* TODO: This command does not support logs on other hosts */
	glog.Infof("ControlPlaneDao.GetServiceStateLogs id=%s", request)
	var serviceState dao.ServiceState
	err := this.zkDao.GetServiceState(&serviceState, request.ServiceId, request.ServiceStateId)
	if err != nil {
		glog.Errorf("ControlPlaneDao.GetServiceStateLogs servicestate=%+v err=%s", serviceState, err)
		return err
	}

	cmd := exec.Command("docker", "logs", serviceState.DockerId)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Unable to return logs because: %v", err)
		return err
	}
	*logs = string(output)
	return nil
}

//
func (this *ControlPlaneDao) GetResourcePools(request dao.EntityRequest, pools *map[string]*dao.ResourcePool) error {
	glog.V(2).Infof("ControlPlaneDao.GetResourcePools")
	result, err := searchResourcePoolUri("_exists_:Id")
	glog.V(2).Infof("ControlPlaneDao.GetResourcePools: err=%s", err)

	var resourcePools map[string]*dao.ResourcePool
	if err != nil {
		return err
	}
	var total = len(result.Hits.Hits)
	var pool dao.ResourcePool
	resourcePools = make(map[string]*dao.ResourcePool)
	for i := 0; i < total; i += 1 {
		err := json.Unmarshal(result.Hits.Hits[i].Source, &pool)
		if err != nil {
			return err
		}
		resourcePools[pool.Id] = &pool
	}

	*pools = resourcePools
	return nil
}

//
func (this *ControlPlaneDao) GetHosts(request dao.EntityRequest, hosts *map[string]*dao.Host) error {
	glog.V(2).Infof("ControlPlaneDao.GetHosts")
	query := search.Query().Search("_exists_:Id")
	search_result, err := search.Search("controlplane").Type("host").Size("10000").Query(query).Result()

	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetHosts: err=%s", err)
		return err
	}
	result, err := toHosts(search_result)
	if err != nil {
		return err
	}
	hostmap := make(map[string]*dao.Host)
	var total = len(result)
	for i := 0; i < total; i += 1 {
		host := result[i]
		hostmap[host.Id] = host
	}
	*hosts = hostmap
	return nil
}

//
func (this *ControlPlaneDao) GetServices(request dao.EntityRequest, services *[]*dao.Service) error {
	glog.V(2).Infof("ControlPlaneDao.GetServices")
	query := search.Query().Search("_exists_:Id")
	results, err := search.Search("controlplane").Type("service").Size("50000").Query(query).Result()
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServices: err=%s", err)
		return err
	}
	var service_results []*dao.Service
	service_results, err = toServices(results)
	if err != nil {
		return err
	}

	*services = service_results
	return nil
}

//
func (this *ControlPlaneDao) GetTaggedServices(request dao.EntityRequest, services *[]*dao.Service) error {
	glog.V(2).Infof("ControlPlaneDao.GetTaggedServices")

	switch v := request.(type) {
	case []string:
		qs := strings.Join(v, " AND ")
		query := search.Query().Search(qs)
		results, err := search.Search("controlplane").Type("service").Size("8192").Query(query).Result()
		if err != nil {
			glog.V(2).Infof("ControlPlaneDao.GetTaggedServices: err=%s", err)
			return err
		}

		var service_results []*dao.Service
		service_results, err = toServices(results)
		if err != nil {
			glog.V(2).Infof("ControlPlaneDao.GetTaggedServices: err=%s", err)
			return err
		}

		*services = service_results

		glog.V(2).Infof("ControlPlaneDao.GetTaggedServices: services=%v", services)
		return nil
	default:
		err := fmt.Errorf("Bad request type: %v", v)
		glog.V(2).Infof("ControlPlaneDao.GetTaggedServices: err=%s", err)
		return err
	}
}

func (this *ControlPlaneDao) GetHostsForResourcePool(poolId string, poolHosts *[]*dao.PoolHost) error {
	id := strings.TrimSpace(poolId)
	if id == "" {
		return errors.New("Illegal poolId: empty poolId not allowed")
	}

	query := fmt.Sprintf("PoolId:%s", id)
	result, err := this.queryHosts(query)
	if err != nil {
		return err
	}
	var response []*dao.PoolHost = make([]*dao.PoolHost, len(result))
	for i := 0; i < len(result); i += 1 {
		poolHost := dao.PoolHost{result[i].Id, result[i].PoolId, result[i].IpAddr}
		response[i] = &poolHost
	}

	*poolHosts = response
	return nil
}

func (this *ControlPlaneDao) StartService(serviceId string, unused *string) error {
	//get the original service
	service := dao.Service{}
	err := this.GetService(serviceId, &service)
	if err != nil {
		return err
	}
	//start this service
	var unusedInt int
	service.DesiredState = dao.SVC_RUN
	err = this.UpdateService(service, &unusedInt)
	if err != nil {
		return err
	}
	//start all child services
	var query = fmt.Sprintf("ParentServiceId:%s", serviceId)
	subServices, err := this.queryServices(query, "100")
	if err != nil {
		return err
	}
	for _, service := range subServices {
		err = this.StartService(service.Id, unused)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *ControlPlaneDao) GetServiceStates(serviceId string, serviceStates *[]*dao.ServiceState) error {
	glog.V(2).Infof("ControlPlaneDao.GetServiceStates: serviceId=%s", serviceId)
	return this.zkDao.GetServiceStates(serviceStates, serviceId)
}

/* This method assumes that if a service instance exists, it has not yet been terminated */
func (this *ControlPlaneDao) getNonTerminatedServiceStates(serviceId string, serviceStates *[]*dao.ServiceState) error {
	glog.V(2).Infof("ControlPlaneDao.getNonTerminatedServiceStates: serviceId=%s", serviceId)
	return this.zkDao.GetServiceStates(serviceStates, serviceId)
}

// Update the current state of a service instance.
func (this *ControlPlaneDao) UpdateServiceState(state dao.ServiceState, unused *int) error {
	glog.V(2).Infoln("ControlPlaneDao.UpdateServiceState state=%+v", state)
	return this.zkDao.UpdateServiceState(&state)
}

func (this *ControlPlaneDao) AddServiceState(state *dao.ServiceState) error {
	glog.V(2).Infoln("ControlPlaneDao.AddServiceState state=%+v", state)
	return this.zkDao.AddServiceState(state)
}

func (this *ControlPlaneDao) RestartService(serviceId string, unused *int) error {
	return dao.ControlPlaneError{"Unimplemented"}
}

func (this *ControlPlaneDao) StopService(id string, unused *int) error {
	glog.Infof("ControlPlaneDao.StopService id=%s", id)
	var service dao.Service
	err := this.GetService(id, &service)
	if err != nil {
		return err
	}
	service.DesiredState = dao.SVC_STOP
	err = this.UpdateService(service, unused)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("ParentServiceId:%s AND NOT Launch:manual", id)
	subservices, err := this.queryServices(query, "100")
	if err != nil {
		return err
	}
	for _, service := range subservices {
		return this.StopService(service.Id, unused)
	}
	return nil
}

func (this *ControlPlaneDao) StopRunningInstance(request dao.HostServiceRequest, unused *int) error {
	return this.zkDao.TerminateHostService(request.HostId, request.ServiceStateId)
}

func (this *ControlPlaneDao) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, unused *int) error {
	var wrapper dao.ServiceTemplateWrapper
	err := getServiceTemplateWrapper(request.TemplateId, &wrapper)
	if err != nil {
		glog.Errorf("Unable to load template wrapper: %s", request.TemplateId)
		return err
	}

	var pool dao.ResourcePool
	err = this.GetResourcePool(request.PoolId, &pool)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %s", request.PoolId)
		return err
	}

	var template dao.ServiceTemplate
	err = json.Unmarshal([]byte(wrapper.Data), &template)
	if err != nil {
		glog.Errorf("Unable to unmarshal template: %s", request.TemplateId)
		return err
	}
	return this.deployServiceDefinitions(template.Services, request.TemplateId, request.PoolId, "")
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
	svc.Tags = sd.Tags
	svc.Instances = sd.Instances.Min
	svc.ImageId = sd.ImageId
	svc.PoolId = pool
	svc.DesiredState = ds
	svc.Launch = sd.Launch
	svc.ConfigFiles = sd.ConfigFiles
	svc.Endpoints = &sd.Endpoints
	svc.ParentServiceId = parent
	svc.CreatedAt = now
	svc.UpdatedAt = now

	var unused int
	err = this.AddService(svc, &unused)
	if err != nil {
		return err
	}
	sduuid, _ := dao.NewUuid()
	deployment := dao.ServiceDeployment{sduuid, template, svc.Id, now}
	_, err = newServiceDeployment(sduuid, &deployment)
	if err != nil {
		return err
	}
	return this.deployServiceDefinitions(sd.Services, template, pool, svc.Id)
}

func (this *ControlPlaneDao) AddServiceTemplate(serviceTemplate dao.ServiceTemplate, unused *int) error {
	var err error
	var uuid string
	var response api.BaseResponse
	var wrapper dao.ServiceTemplateWrapper

	data, err := json.Marshal(serviceTemplate)
	if err != nil {
		return err
	}
	uuid, err = dao.NewUuid()
	if err != nil {
		return err
	}
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
	return nil
}

func (this *ControlPlaneDao) UpdateServiceTemplate(template dao.ServiceTemplate, unused *int) error {
	return fmt.Errorf("unimplemented UpdateServiceTemplate")
}

func (this *ControlPlaneDao) RemoveServiceTemplate(id string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveServiceTemplate: %s", id)
	response, err := deleteServiceTemplateWrapper(id)
	glog.V(2).Infof("ControlPlaneDao.RemoveServiceTemplate response: %+v", response)
	return err
}

func (this *ControlPlaneDao) GetServiceTemplates(unused int, templates *map[string]*dao.ServiceTemplate) error {
	glog.V(2).Infof("ControlPlaneDao.GetServiceTemplates")
	query := search.Query().Search("_exists_:Id")
	search_result, err := search.Search("controlplane").Type("servicetemplatewrapper").Size("1000").Query(query).Result()
	glog.V(2).Infof("ControlPlaneDao.GetServiceTemplates: err=%s", err)
	if err != nil {
		return err
	}
	result, err := toServiceTemplateWrappers(search_result)
	templatemap := make(map[string]*dao.ServiceTemplate)
	if err != nil {
		return err
	}
	var total = len(result)
	for i := 0; i < total; i += 1 {
		var template dao.ServiceTemplate
		wrapper := result[i]
		err = json.Unmarshal([]byte(wrapper.Data), &template)
		templatemap[wrapper.Id] = &template
	}
	*templates = templatemap
	return nil
}

func (this *ControlPlaneDao) lead(conn *zk.Conn, zkEvent <-chan zk.Event) {
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
				glog.Info("Processing leader duties")
				// passthru
			}

			this.watchServices(conn)
			return nil
		}()
	}
}

func (this *ControlPlaneDao) watchServices(conn *zk.Conn) {
	processing := make(map[string]chan int)
	sDone := make(chan string)

	// When this function exits, ensure that any started goroutines get
	// a signal to shutdown
	defer func() {
		for _, shutdown := range processing {
			shutdown <- 1
		}
	}()

	for {
		glog.Infof("Leader watching for changes to %s", zzk.SERVICE_PATH)
		serviceIds, _, zkEvent, err := conn.ChildrenW(zzk.SERVICE_PATH)
		if err != nil {
			glog.Errorf("Leader unable to find any services: %v", err)
			return
		}
		for _, serviceId := range serviceIds {
			if processing[serviceId] == nil {
				serviceChannel := make(chan int)
				processing[serviceId] = serviceChannel
				go this.watchService(conn, serviceChannel, sDone, serviceId)
			}
		}
		select {
		case evt := <-zkEvent:
			glog.Infof("Leader event: %v", evt)
		case serviceId := <-sDone:
			glog.Infof("Leading cleaning up for service %s", serviceId)
			delete(processing, serviceId)
		}
	}
}

func (this *ControlPlaneDao) watchService(conn *zk.Conn, shutdown <-chan int, done chan<- string, serviceId string) {
	defer func() { done <- serviceId }()
	for {
		var service dao.Service
		_, zkEvent, err := zzk.LoadServiceW(conn, serviceId, &service)
		if err != nil {
			glog.Errorf("Unable to load service %s: %v", serviceId, err)
			return
		}

		glog.Infof("Leader watching for changes to service %s", service.Name)

		// check current state
		var serviceStates []*dao.ServiceState
		err = this.getNonTerminatedServiceStates(service.Id, &serviceStates)
		if err != nil {
			return
		}

		// Is the service supposed to be running at all?
		switch {
		case service.DesiredState == dao.SVC_STOP:
			shutdownServiceInstances(conn, serviceStates, len(serviceStates))
		case service.DesiredState == dao.SVC_RUN:
			this.updateServiceInstances(conn, &service, serviceStates)
		default:
			glog.Warningf("Unexpected desired state %d for service %s", service.DesiredState, service.Name)
		}


		select {
		case evt := <-zkEvent:
			if evt.Type == zk.EventNodeDeleted {
				glog.Infof("Shutting down due to node delete %s", serviceId)
				shutdownServiceInstances(conn, serviceStates, len(serviceStates))
				return
			}
			glog.Infof("Service %s received event %v", service.Name, evt)
			continue

		case <-shutdown:
			glog.Info("Leader stopping watch on %s", service.Name)
			return

		}
	}

}

func (this *ControlPlaneDao) updateServiceInstances(conn *zk.Conn, service *dao.Service, serviceStates []*dao.ServiceState) error {
	var err error
	// pick services instances to start
	if len(serviceStates) < service.Instances {
		instancesToStart := service.Instances - len(serviceStates)
		var poolHosts []*dao.PoolHost
		err = this.GetHostsForResourcePool(service.PoolId, &poolHosts)
		if err != nil {
			glog.Errorf("Leader unable to acquire hosts for pool %s", service.PoolId)
			return err
		}
		if len(poolHosts) == 0 {
			glog.Warningf("Pool %s has no hosts", service.PoolId)
			return nil
		}

		return startServiceInstances(conn, service, poolHosts, instancesToStart)

	} else if len(serviceStates) > service.Instances {
		instancesToKill := len(serviceStates) - service.Instances
		shutdownServiceInstances(conn, serviceStates, instancesToKill)
	}
	return nil

}

func startServiceInstances(conn *zk.Conn, service *dao.Service, pool_hosts []*dao.PoolHost, numToStart int) error {
	for i := 0; i < numToStart; i++ {
		// randomly select host
		service_host := pool_hosts[rand.Intn(len(pool_hosts))]
		serviceState, err := service.NewServiceState(service_host.HostId)
		if err != nil {
			glog.Errorf("Error creating ServiceState instance: %v", err)
			return err
		}

		serviceState.HostIp = service_host.HostIp
		err = zzk.AddServiceState(conn, serviceState)
		if err != nil {
			glog.Errorf("Leader unable to add service state: %v", err)
			return err
		}
		glog.Infof("cp: serviceState %s", serviceState.Started)
	}
	return nil
}

func shutdownServiceInstances(conn *zk.Conn, serviceStates []*dao.ServiceState, numToKill int) {
	for i := 0; i < numToKill; i++ {
		glog.Infof("Killing host service state %s:%s\n", serviceStates[i].HostId, serviceStates[i].Id)
		serviceStates[i].Terminated = time.Date(2, time.January, 1, 0, 0, 0, 0, time.UTC)
		err := zzk.TerminateHostService(conn, serviceStates[i].HostId, serviceStates[i].Id)
		if err != nil {
			glog.Warningf("%s:%s wouldn't die", serviceStates[i].HostId, serviceStates[i].Id)
		}
	}
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int) (*ControlPlaneDao, error) {
	glog.Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
	api.Domain = hostName
	api.Port = strconv.Itoa(port)
	return &ControlPlaneDao{hostName, port, nil, nil}, nil
}



// hostId retreives the system's unique id, on linux this maps
// to /usr/bin/hostid.
func hostId() (hostid string, err error) {
	cmd := exec.Command(HOST_ID_CMDString)
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
	s.zkDao = &zzk.ZkDao{s.zookeepers}

	isvcs.ElasticSearchContainer.Run()

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
		func() {
			conn, _, err := zk.Connect(s.zookeepers, time.Second*10)
			if err != nil {
				time.Sleep(time.Second * 3)
				return
			}
			defer conn.Close()

			scheduler, shutdown := newScheduler("", conn, hostId, s.lead)
			scheduler.Start()
			select {
			case <-shutdown:
			}
		}()
	}
}

const HOST_ID_CMDString = "/usr/bin/hostid"
