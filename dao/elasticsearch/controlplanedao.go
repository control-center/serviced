// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package elasticsearch

import (
	"github.com/mattbaird/elastigo/api"
	"github.com/mattbaird/elastigo/core"
	"github.com/mattbaird/elastigo/search"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/volume"
	"github.com/zenoss/serviced/zzk"

	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//assert interface
var _ dao.ControlPlane = &ControlPlaneDao{}

// NotFoundError is a typed error.
type NotFoundError struct {
	s string
}

func (e *NotFoundError) Error() string {
	return e.s
}

// New returns an error that formats as the given text.
func NewNotFoundError(text string) error {
	return &NotFoundError{text}
}

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
	varpath    string
	vfs        string
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
		glog.V(2).Infof("%+v service imports=%+v", service, service_imports)

		var request dao.EntityRequest
		var servicesList []*dao.Service
		err = this.GetServices(request, &servicesList)
		if err != nil {
			return
		}

		// Map all services by Id so we can construct a tree for the current service ID
		glog.V(2).Infof("ServicesList: %d", len(servicesList))
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
			glog.V(2).Infof("Finding exports for import: %+v", endpoint)
			key := fmt.Sprintf("%s:%d", endpoint.Protocol, endpoint.PortNumber)
			if _, exists := remoteEndpoints[key]; !exists {
				remoteEndpoints[key] = make([]*dao.ApplicationEndpoint, 0)
			}

			for _, ss := range states {
				port := ss.GetHostPort(endpoint.Protocol, endpoint.Application, endpoint.PortNumber)
				glog.V(2).Info("Remote port: ", port)
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
		glog.V(1).Infof("Return for %s is %+v", serviceId, remoteEndpoints)
	}
	return
}

// add resource pool to index
func (this *ControlPlaneDao) AddResourcePool(pool dao.ResourcePool, poolId *string) error {
	glog.V(2).Infof("ControlPlaneDao.NewResourcePool: %+v", pool)
	id := strings.TrimSpace(pool.Id)
	if id == "" {
		return errors.New("empty ResourcePool.Id not allowed")
	}

	pool.Id = id
	response, err := newResourcePool(id, pool)
	glog.V(2).Infof("ControlPlaneDao.NewResourcePool response: %+v", response)
	if response.Ok {
		*poolId = id
		return nil
	}
	return err
}

//
func (this *ControlPlaneDao) AddHost(host dao.Host, hostId *string) error {
	glog.V(2).Infof("ControlPlaneDao.AddHost: %+v", host)
	id := strings.TrimSpace(host.Id)
	if id == "" {
		return errors.New("empty Host.Id not allowed")
	}

	//TODO: shouldn't all this validation be in the UpdateHost method as well?
	ipAddr, err := net.ResolveIPAddr("ip4", host.IpAddr)
	if err != nil {
		glog.Errorf("Could not resolve: %s to an ip4 address: %s", host.IpAddr, err)
		return err
	}
	if ipAddr.IP.IsLoopback() {
		glog.Errorf("Can not use %s as host address because it is a loopback address", host.IpAddr)
		return errors.New("host ip can not be a loopback address")
	}

	host.Id = id
	response, err := newHost(id, host)
	glog.V(2).Infof("ControlPlaneDao.AddHost response: %+v", response)
	if response.Ok {
		*hostId = id
		return nil
	}
	return err
}

// The tenant id is the root service uuid. Walk the service tree to root to find the tenant id.
func (this *ControlPlaneDao) GetTenantId(serviceId string, tenantId *string) error {
	glog.V(2).Infof("ControlPlaneDao.GetTenantId: %s", serviceId)
	id := strings.TrimSpace(serviceId)
	if id == "" {
		return errors.New("empty serviceId not allowed")
	}

	var err error
	var service dao.Service
	for {
		err = this.GetService(id, &service)
		if err == nil {
			id = service.ParentServiceId
			if id == "" {
				*tenantId = service.Id
				return nil
			}
		} else {
			return err
		}
	}

	return err
}

//
func (this *ControlPlaneDao) AddService(service dao.Service, serviceId *string) error {
	glog.V(2).Infof("ControlPlaneDao.AddService: %+v", service)
	id := strings.TrimSpace(service.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}

	service.Id = id
	response, err := newService(id, service)
	glog.V(2).Infof("ControlPlaneDao.AddService response: %+v", response)
	if response.Ok {
		*serviceId = id
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
	if err != nil {
		glog.Errorf("Error removing service %s: %v", id, err)
		return err
	}
	this.zkDao.RemoveService(id)
	return nil
}

//
func (this *ControlPlaneDao) GetResourcePool(id string, pool *dao.ResourcePool) error {
	glog.V(2).Infof("ControlPlaneDao.GetResourcePool: id=%s", id)
	if len(id) == 0 {
		return errors.New("Must specify a pool ID")
	}
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
	glog.V(3).Infof("ControlPlaneDao.GetService: id=%s", id)
	request := dao.Service{}
	err := getService(id, &request)
	glog.V(3).Infof("ControlPlaneDao.GetService: id=%s, service=%+v, err=%s", id, request, err)
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
	glog.V(3).Info("ControlPlaneDao.GetServiceLogs id=", id)
	var serviceStates []*dao.ServiceState
	err := this.zkDao.GetServiceStates(&serviceStates, id)
	if err != nil {
		return err
	}
	if len(serviceStates) == 0 {
		glog.V(1).Info("Unable to find any running services for ", id)
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
	glog.V(3).Info("ControlPlaneDao.GetServiceStateLogs id=", request)
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
	glog.V(3).Infof("ControlPlaneDao.GetResourcePools")
	result, err := searchResourcePoolUri("_exists_:Id")
	glog.V(3).Info("ControlPlaneDao.GetResourcePools: err=", err)

	var resourcePools map[string]*dao.ResourcePool
	if err != nil {
		return err
	}
	var total = len(result.Hits.Hits)
	resourcePools = make(map[string]*dao.ResourcePool)
	for i := 0; i < total; i += 1 {
		var pool dao.ResourcePool
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
	glog.V(3).Infof("ControlPlaneDao.GetHosts")
	query := search.Query().Search("_exists_:Id")
	search_result, err := search.Search("controlplane").Type("host").Size("10000").Query(query).Result()

	if err != nil {
		glog.Error("ControlPlaneDao.GetHosts: err=", err)
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
	glog.V(3).Infof("ControlPlaneDao.GetServices")
	query := search.Query().Search("_exists_:Id")
	results, err := search.Search("controlplane").Type("service").Size("50000").Query(query).Result()
	if err != nil {
		glog.Error("ControlPlaneDao.GetServices: err=", err)
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
	glog.V(3).Infof("ControlPlaneDao.GetTaggedServices")

	switch v := request.(type) {
	case []string:
		qs := strings.Join(v, " AND ")
		query := search.Query().Search(qs)
		results, err := search.Search("controlplane").Type("service").Size("8192").Query(query).Result()
		if err != nil {
			glog.Error("ControlPlaneDao.GetTaggedServices: err=", err)
			return err
		}

		var service_results []*dao.Service
		service_results, err = toServices(results)
		if err != nil {
			glog.Error("ControlPlaneDao.GetTaggedServices: err=", err)
			return err
		}

		*services = service_results

		glog.V(2).Infof("ControlPlaneDao.GetTaggedServices: services=%v", services)
		return nil
	default:
		err := fmt.Errorf("Bad request type: %v", v)
		glog.V(2).Info("ControlPlaneDao.GetTaggedServices: err=", err)
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
	if len(result) == 0 {
		errorMessage := fmt.Sprintf("Illegal poolId:%s was not found", id)
		return errors.New(errorMessage)
	}

	var response []*dao.PoolHost = make([]*dao.PoolHost, len(result))
	for i := 0; i < len(result); i += 1 {
		poolHost := dao.PoolHost{result[i].Id, result[i].PoolId, result[i].IpAddr}
		response[i] = &poolHost
	}

	*poolHosts = response
	return nil
}

// if AddressResourceConfig exists in a service, an IP must be assigned to the endpoint that the AddressResourceConfig belongs to
func validAddressResourceConfig(arc dao.AddressResourceConfig) error {
	if arc.Port != 0 || arc.Protocol != "" {
		msg := fmt.Sprintf("AddressConfig with no assignment. Port: %d Protocol: %s", arc.Port, arc.Protocol)
		return errors.New(msg)
	}
	return nil
}

// determine whether the services are ready for deployment
func (this *ControlPlaneDao) ValidateServicesForDeployment(service dao.Service) error {
	// ensure all endpoints with AddressConfig have assigned IPs
	for _, endPoint := range service.Endpoints {
		err := validAddressResourceConfig(endPoint.AddressConfig)
		if err != nil {
			return err
		}
	}

	// add additional validation checks to the services
	return nil
}

func (this *ControlPlaneDao) StartService(serviceId string, unused *string) error {
	//get the original service
	service := dao.Service{}
	err := this.GetService(serviceId, &service)
	if err != nil {
		return err
	}

	// validate the service is ready to start
	err = this.ValidateServicesForDeployment(service)
	if err != nil {
		glog.Errorf("Services failed validation for deployment")
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

func (this *ControlPlaneDao) GetServiceState(request dao.ServiceStateRequest, serviceState *dao.ServiceState) error {
	glog.V(3).Infof("ControlPlaneDao.GetServiceState: request=%v", request)
	return this.zkDao.GetServiceState(serviceState, request.ServiceId, request.ServiceStateId)
}

func (this *ControlPlaneDao) GetRunningService(request dao.ServiceStateRequest, running *dao.RunningService) error {
	glog.V(3).Infof("ControlPlaneDao.GetRunningService: request=%v", request)
	return this.zkDao.GetRunningService(request.ServiceId, request.ServiceStateId, running)
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

func (this *ControlPlaneDao) RestartService(serviceId string, unused *int) error {
	return dao.ControlPlaneError{"Unimplemented"}
}

func (this *ControlPlaneDao) StopService(id string, unused *int) error {
	glog.V(2).Info("ControlPlaneDao.StopService id=", id)
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

	volumes := make(map[string]string)
	return this.deployServiceDefinitions(template.Services, request.TemplateId, request.PoolId, "", volumes, request.DeploymentId)
}

func (this *ControlPlaneDao) deployServiceDefinitions(sds []dao.ServiceDefinition, template string, pool string, parent string, volumes map[string]string, deploymentId string) error {
	for _, sd := range sds {
		if err := this.deployServiceDefinition(sd, template, pool, parent, volumes, deploymentId); err != nil {
			return err
		}
	}
	return nil
}

func (this *ControlPlaneDao) deployServiceDefinition(sd dao.ServiceDefinition, template string, pool string, parent string, volumes map[string]string, deploymentId string) error {
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

	exportedVolumes := make(map[string]string)
	for k, v := range volumes {
		exportedVolumes[k] = v
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
	svc.Endpoints = sd.Endpoints
	svc.Tasks = sd.Tasks
	svc.ParentServiceId = parent
	svc.CreatedAt = now
	svc.UpdatedAt = now
	svc.Volumes = sd.Volumes
	svc.DeploymentId = deploymentId
	svc.LogConfigs = sd.LogConfigs
	svc.Snapshot = sd.Snapshot

	//for each endpoint, evaluate it's Application
	if err = svc.EvaluateEndpointTemplates(this); err != nil {
		return err
	}

	//for each endpoint, evaluate it's Application
	if err = svc.EvaluateEndpointTemplates(this); err != nil {
		return err
	}

	var serviceId string
	err = this.AddService(svc, &serviceId)
	if err != nil {
		return err
	}

	var unused int
	sduuid, _ := dao.NewUuid()
	deployment := dao.ServiceDeployment{sduuid, template, svc.Id, now}
	err = this.AddServiceDeployment(deployment, &unused)
	if err != nil {
		return err
	}

	return this.deployServiceDefinitions(sd.Services, template, pool, svc.Id, exportedVolumes, deploymentId)
}

func (this *ControlPlaneDao) AddServiceDeployment(deployment dao.ServiceDeployment, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.AddServiceDeployment: %+v", deployment)
	id := strings.TrimSpace(deployment.Id)
	if id == "" {
		return errors.New("empty ServiceDeployment.Id not allowed")
	}

	deployment.Id = id
	response, err := newServiceDeployment(id, deployment)
	glog.V(2).Infof("ControlPlaneDao.AddServiceDeployment response: %+v", response)
	return err
}

func (this *ControlPlaneDao) AddServiceTemplate(serviceTemplate dao.ServiceTemplate, templateId *string) error {
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
		*templateId = uuid
		err = nil
	}
	// this takes a while so don't block the main thread
	go this.reloadLogstashContainer()
	return err
}

func (this *ControlPlaneDao) UpdateServiceTemplate(template dao.ServiceTemplate, unused *int) error {
	return fmt.Errorf("unimplemented UpdateServiceTemplate")
}

func (this *ControlPlaneDao) RemoveServiceTemplate(id string, unused *int) error {
	// make sure it is a valid template first
	var wrapper dao.ServiceTemplateWrapper
	err := getServiceTemplateWrapper(id, &wrapper)

	if err != nil {
		return fmt.Errorf("Unable to find template: %s", id)
	}

	glog.V(2).Infof("ControlPlaneDao.RemoveServiceTemplate: %s", id)
	response, err := deleteServiceTemplateWrapper(id)
	glog.V(2).Infof("ControlPlaneDao.RemoveServiceTemplate response: %+v", response)
	if err != nil {
		return err
	}
	go this.reloadLogstashContainer()
	return nil
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

func (this *ControlPlaneDao) StartShell(service dao.Service, unused *int) error {
	// TODO: implement stub
	return nil
}

func (this *ControlPlaneDao) ExecuteShell(service dao.Service, command *string) error {
	// TODO: implement stub
	return nil
}

func (this *ControlPlaneDao) ShowCommands(service dao.Service, unused *int) error {
	// TODO: implement stub
	return nil
}

func (this *ControlPlaneDao) DeleteSnapshot(snapshotId string, unused *int) error {
	var tenantId string
	parts := strings.Split(snapshotId, "_")
	if len(parts) != 2 {
		glog.V(2).Infof("ControlPlaneDao.DeleteSnapshot malformed snapshot Id: %s", snapshotId)
		return errors.New("malformed snapshotId")
	}
	serviceId := parts[0]
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.DeleteSnapshot service=%+v err=%s", serviceId, err)
		return err
	}

	var service dao.Service
	err := this.GetService(tenantId, &service)
	glog.V(2).Infof("Getting service instance: %s", tenantId)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.DeleteSnapshot service=%+v err=%s", serviceId, err)
		return err
	}

	// delete snapshot
	if volume, err := getSubvolume(this.vfs, service.PoolId, tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.DeleteSnapshot service=%+v err=%s", serviceId, err)
		return err
	} else {
		glog.V(2).Infof("deleting snapshot %s", snapshotId)
		if err := volume.RemoveSnapshot(snapshotId); err != nil {
			return err
		}
	}
	return nil
}

func (this *ControlPlaneDao) Rollback(snapshotId string, unused *int) error {

	var tenantId string
	parts := strings.Split(snapshotId, "_")
	if len(parts) != 2 {
		glog.V(2).Infof("ControlPlaneDao.Snapshot malformed snapshot Id: %s", snapshotId)
		return errors.New("malformed snapshotId")
	}
	serviceId := parts[0]
	label := parts[1]
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshot service=%+v err=%s", serviceId, err)
		return err
	}

	this.StopService(tenantId, unused)
	// TODO: Wait for real event that confirms shutdown
	time.Sleep(time.Second * 5) // wait for shutdown

	var service dao.Service
	err := this.GetService(tenantId, &service)
	glog.V(2).Infof("Getting service instance: %s", tenantId)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.Rollback service=%+v err=%s", serviceId, err)
		return err
	}
	// rollback
	if volume, err := getSubvolume(this.vfs, service.PoolId, tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Rollback service=%+v err=%s", serviceId, err)
		return err
	} else {
		glog.V(2).Infof("performing rollback on %s to %s", tenantId, label)
		if err := volume.Rollback(snapshotId); err != nil {
			return err
		}
	}
	unusedStr := ""
	return this.StartService(tenantId, &unusedStr)
}

func (this *ControlPlaneDao) callQuiescePause() error {
	if err := this.zkDao.UpdateSnapshotState("PAUSE"); err != nil {
		glog.V(3).Infof("ControlPlaneDao.callQuiescePause err=%s", err)
		return err
	}

	// assuming lxc-attach is setuid for docker group
	//   sudo chgrp docker /usr/bin/lxc-attach
	//   sudo chmod u+s /usr/bin/lxc-attach

	var request dao.EntityRequest
	var servicesList []*dao.Service
	if err := this.GetServices(request, &servicesList); err != nil {
		return err
	}
	for _, service := range servicesList {
		if service.Snapshot.Pause != "" && service.Snapshot.Resume != "" {
			glog.V(2).Infof("quiesce pause  service: %+v", service)
			cmd := exec.Command("echo", "TODO:", "lxc-attach", "-n", string(service.Id), "--", service.Snapshot.Pause)
			output, err := cmd.CombinedOutput()
			if err != nil {
				glog.Errorf("Unable to quiesce pause service %+v with cmd %+v because: %v", service, cmd, err)
				return err
			}
			glog.V(2).Infof("quiesce paused service - output:%s", string(output))
		}
	}

	// TODO: deficiency of this algorithm is that if one service fails to pause,
	//       all paused services will stay paused
	//       Perhaps one way to fix it is to call resume for all paused services
	//       if any of them fail to pause

	return nil
}

func (this *ControlPlaneDao) callQuiesceResume() error {
	if err := this.zkDao.UpdateSnapshotState("RESUME"); err != nil {
		glog.V(2).Infof("ControlPlaneDao.callQuiesceResume err=%s", err)
		return err
	}

	var request dao.EntityRequest
	var servicesList []*dao.Service
	if err := this.GetServices(request, &servicesList); err != nil {
		return err
	}
	for _, service := range servicesList {
		if service.Snapshot.Pause != "" && service.Snapshot.Resume != "" {
			glog.V(2).Infof("quiesce resume service: %+v", service)
			cmd := exec.Command("echo", "TODO:", "lxc-attach", "-n", string(service.Id), "--", service.Snapshot.Resume)
			output, err := cmd.CombinedOutput()
			if err != nil {
				glog.Errorf("Unable to resume service %+v with cmd %+v because: %v", service, cmd, err)
				return err
			}
			glog.V(2).Infof("quiesce resume service - output:%+v", output)
		}
	}

	// TODO: deficiency of this algorithm is that if one service fails to resume,
	//       all remaining paused services will stay paused
	//       Perhaps one way to fix it is to call resume for all paused services
	//       if any of them fail to resume

	return nil
}

func (this *ControlPlaneDao) Snapshot(serviceId string, label *string) error {
	glog.V(3).Infof("ControlPlaneDao.Snapshot entering snapshot with service=%s", serviceId)
	defer glog.V(3).Infof("ControlPlaneDao.Snapshot finished snapshot with label=%s", *label)

	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshot service=%+v err=%s", serviceId, err)
		return err
	}
	var service dao.Service
	err := this.GetService(tenantId, &service)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshot service=%+v err=%s", serviceId, err)
		return err
	}

	// simplest case - do everything here

	// call quiesce pause for services with 'Snapshot' definition
	if err := this.callQuiescePause(); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshot service=%+v err=%s", serviceId, err)
		return err
	}

	// TODO: move "create a snapshot" functionality to a method called by the CP agent when the container volume is quiesced
	// create a snapshot
	if volume, err := getSubvolume(this.vfs, service.PoolId, tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshot service=%+v err=%s", serviceId, err)
		return err
	} else if volume == nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshot service=%+v volume=%+v", serviceId, volume)
		return err
	} else {
		snapLabel := snapShotName(volume.Name())
		if err := volume.Snapshot(snapLabel); err != nil {
			return err
		} else {
			*label = snapLabel
		}
	}

	// call quiesce resume for services with 'Snapshot' definition
	if err := this.callQuiesceResume(); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshot service=%+v err=%s", serviceId, err)
		return err
	}

	return nil
}

func snapShotName(volumeName string) string {
	format := "20060102-150405"
	loc := time.Now()
	utc := loc.UTC()
	return volumeName + "_" + utc.Format(format)
}

func getSubvolume(vfs, poolId, tenantId string) (*volume.Volume, error) {
	baseDir, err := filepath.Abs(path.Join(varPath(), "volumes", poolId))
	if err != nil {
		return nil, err
	}
	return volume.Mount(vfs, tenantId, baseDir)
}

func varPath() string {
	if len(os.Getenv("SERVICED_HOME")) > 0 {
		return path.Join(os.Getenv("SERVICED_HOME"), "var")
	}
	return "/tmp/serviced/var"
}

func (this *ControlPlaneDao) Snapshots(serviceId string, labels *[]string) error {

	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshots service=%+v err=%s", serviceId, err)
		return err
	}
	var service dao.Service
	err := this.GetService(tenantId, &service)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshots service=%+v err=%s", serviceId, err)
		return err
	}

	if volume, err := getSubvolume(this.vfs, service.PoolId, tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshots service=%+v err=%s", serviceId, err)
		return err
	} else {
		if snaplabels, err := volume.Snapshots(); err != nil {
			return err
		} else {
			glog.Infof("Got snap labels %v", snaplabels)
			*labels = make([]string, len(snaplabels))
			*labels = snaplabels
		}
	}
	return nil
}

func (this *ControlPlaneDao) Get(service dao.Service, file *string) error {
	// TODO: implement stub
	return nil
}

func (this *ControlPlaneDao) Send(service dao.Service, files *[]string) error {
	// TODO: implment stub
	return nil
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int) (*ControlPlaneDao, error) {
	glog.V(0).Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
	api.Domain = hostName
	api.Port = strconv.Itoa(port)
	return &ControlPlaneDao{hostName, port, "", "", nil, nil}, nil
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

func createDefaultPool(s *ControlPlaneDao) error {
	var pool dao.ResourcePool
	// does the default pool exist
	if err := s.GetResourcePool("default", &pool); err != nil {
		glog.Errorf("%s", err)
		glog.V(0).Info("'default' resource pool not found; creating...")

		// create it
		default_pool := dao.ResourcePool{}
		default_pool.Id = "default"
		var poolId string
		if err := s.AddResourcePool(default_pool, &poolId); err != nil {
			return err
		}
	}
	return nil
}

func NewControlSvc(hostName string, port int, zookeepers []string, varpath, vfs string) (*ControlPlaneDao, error) {
	glog.V(2).Info("calling NewControlSvc()")
	defer glog.V(2).Info("leaving NewControlSvc()")

	s, err := NewControlPlaneDao(hostName, port)
	if err != nil {
		return nil, err
	}

	if err = isvcs.Mgr.Start(); err != nil {
		return nil, err
	}

	s.varpath = varpath
	s.vfs = vfs

	if len(zookeepers) == 0 {
		s.zookeepers = []string{"127.0.0.1:2181"}
	} else {
		s.zookeepers = zookeepers
	}
	s.zkDao = &zzk.ZkDao{s.zookeepers}

	if err = createDefaultPool(s); err != nil {
		return nil, err
	}

	hid, err := hostId()
	if err != nil {
		return nil, err
	}

	go s.handleScheduler(hid)

	return s, nil
}

// Anytime the available service definitions are modified
// we need to restart the logstash container so it can write out
// its new filter set.
// This method depends on the elasticsearch container being up and running.
func (s *ControlPlaneDao) reloadLogstashContainer() error {
	var templatesMap map[string]*dao.ServiceTemplate
	if err := s.GetServiceTemplates(0, &templatesMap); err != nil {
		return err
	}

	// FIXME: eventually this file should live in the DFS or the config should
	// live in zookeeper to allow the agents to get to this
	if err := dao.WriteConfigurationFile(templatesMap); err != nil {
		return err
	}
	glog.V(2).Info("Starting logstash container")
	if err := isvcs.Mgr.Notify("restart logstash"); err != nil {
		glog.Fatalf("Could not start logstash container: %s", err)
		return err
	}
	return nil
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

			sched, shutdown := serviced.NewScheduler("", conn, hostId, s)
			sched.Start()
			select {
			case <-shutdown:
			}
		}()
	}
}

const HOST_ID_CMDString = "/usr/bin/hostid"
