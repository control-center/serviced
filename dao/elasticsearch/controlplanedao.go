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
	docker "github.com/zenoss/go-dockerclient"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dfs"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/volume"
	"github.com/zenoss/serviced/zzk"

	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DOCKER_ENDPOINT string = "unix:///var/run/docker.sock"
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
	userExists         func(string) (bool, error) = exists(&Pretty, "controlplane", "user")

	//model index functions
	newHost                   func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "host")
	newService                func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "service")
	newResourcePool           func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "resourcepool")
	newServiceDeployment      func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "servicedeployment")
	newServiceTemplateWrapper func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "servicetemplatewrapper")
	newAddressAssignment      func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "addressassignment")
	newUser                   func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "user")

	//model index functions
	indexHost                   func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "host")
	indexService                func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "service")
	indexServiceState           func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "servicestate")
	indexServiceTemplateWrapper func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "servicetemplatewrapper")
	indexResourcePool           func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "resourcepool")
	indexUser                   func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "user")

	//model delete functions
	deleteHost                   func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "host")
	deleteService                func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "service")
	deleteServiceState           func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "servicestate")
	deleteResourcePool           func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "resourcepool")
	deleteServiceTemplateWrapper func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "servicetemplatewrapper")
	deleteAddressAssignment      func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "addressassignment")
	deleteUser                   func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "user")

	//model get functions
	getHost                   func(string, interface{}) error = getSource("controlplane", "host")
	getService                func(string, interface{}) error = getSource("controlplane", "service")
	getServiceState           func(string, interface{}) error = getSource("controlplane", "servicestate")
	getResourcePool           func(string, interface{}) error = getSource("controlplane", "resourcepool")
	getServiceTemplateWrapper func(string, interface{}) error = getSource("controlplane", "servicetemplatewrapper")
	getUser                   func(string, interface{}) error = getSource("controlplane", "user")

	//model search functions, using uri based query
	searchHostUri           func(string) (core.SearchResult, error) = searchUri("controlplane", "host")
	searchServiceUri        func(string) (core.SearchResult, error) = searchUri("controlplane", "service")
	searchServiceStateUri   func(string) (core.SearchResult, error) = searchUri("controlplane", "servicestate")
	searchResourcePoolUri   func(string) (core.SearchResult, error) = searchUri("controlplane", "resourcepool")
	searchAddressAssignment func(string) (core.SearchResult, error) = searchUri("controlplane", "addressassignment")
	searchUserUri           func(string) (core.SearchResult, error) = searchUri("controlplane", "user")
)

// each time Serviced starts up a new password will be generated. This will be passed into
// the containers so that they can authenticate against the API
var SYSTEM_USER_NAME = "system_user"
var INSTANCE_PASSWORD string

type ControlPlaneDao struct {
	hostName   string
	port       int
	varpath    string
	vfs        string
	zookeepers []string
	zkDao      *zzk.ZkDao
	dfs        *dfs.DistributedFileSystem
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

// convert search result of json host to dao.Host array
func toAddressAssignments(result *core.SearchResult) (*[]dao.AddressAssignment, error) {
	var err error = nil
	var total = len(result.Hits.Hits)
	var addressAssignments = make([]dao.AddressAssignment, total)
	for i := 0; i < total; i += 1 {
		var addressAssignment dao.AddressAssignment
		err = json.Unmarshal(result.Hits.Hits[i].Source, &addressAssignment)
		if err == nil {
			addressAssignments[i] = addressAssignment
		} else {
			return nil, err
		}
	}

	return &addressAssignments, err
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
	err = getService(serviceId, &service)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceEndpoints service=%+v err=%s", service, err)
		return
	}

	service_imports := service.GetServiceImports()
	if len(service_imports) > 0 {
		glog.V(2).Infof("%+v service imports=%+v", service, service_imports)

		var servicesList []*dao.Service
		err = getServices(&servicesList)
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

//hashPassword returns the sha-1 of a password
func hashPassword(password string) string {
	h := sha1.New()
	io.WriteString(h, password)
	return fmt.Sprintf("% x", h.Sum(nil))
}

//addUser places a new user record into elastic searchp
func (this *ControlPlaneDao) AddUser(user dao.User, userName *string) error {
	glog.V(2).Infof("ControlPlane.NewUser: %+v", user)
	name := strings.TrimSpace(*userName)
	user.Password = hashPassword(user.Password)

	// save the user
	response, err := newUser(name, user)
	if response.Ok {
		*userName = name
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
	return getTenantId(serviceId, tenantId)
}

var getTenantId = func(serviceId string, tenantId *string) (err error) {
	glog.V(2).Infof("ControlPlaneDao.GetTenantId: %s", serviceId)
	id := strings.TrimSpace(serviceId)
	if id == "" {
		return errors.New("empty serviceId not allowed")
	}

	var traverse func(string) (string, error)

	traverse = func(id string) (string, error) {
		var service dao.Service
		if err := getService(id, &service); err != nil {
			return "", err
		} else if service.ParentServiceId != "" {
			return traverse(service.ParentServiceId)
		} else {
			glog.Infof("parent service: %+v", service)
			return service.Id, nil
		}
	}

	*tenantId, err = traverse(id)
	return
}

//
func (this *ControlPlaneDao) AddService(service dao.Service, serviceId *string) error {
	return addService(this, service, serviceId)
}

var addService = func(this *ControlPlaneDao, service dao.Service, serviceId *string) error {
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

//UpdateUser updates the user entry in elastic search. NOTE: It is assumed the
//pasword is NOT hashed when updating the user record
func (this *ControlPlaneDao) UpdateUser(user dao.User, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateUser: %+v", user)

	id := strings.TrimSpace(user.Name)
	if id == "" {
		return errors.New("empty User.Name not allowed")
	}

	user.Name = id
	user.Password = hashPassword(user.Password)
	response, err := indexUser(id, user)
	glog.V(2).Infof("ControlPlaneDao.UpdateUser response: %+v", response)
	if response.Ok {
		return nil
	}
	return err
}

// updateService internal method to use when service has been validated
var updateService = func(this *ControlPlaneDao, service *dao.Service) error {
	id := strings.TrimSpace(service.Id)
	if id == "" {
		return errors.New("empty Service.Id not allowed")
	}
	service.Id = id
	response, err := indexService(id, service)
	glog.V(2).Infof("ControlPlaneDao.UpdateService response: %+v", response)
	if response.Ok {
		//add address assignment info to ZK Service
		for idx := range service.Endpoints {
			assignment, err := this.getEndpointAddressAssignments(service.Id, service.Endpoints[idx].Name)
			if err != nil {
				glog.Errorf("ControlPlaneDao.UpdateService Error looking up address assignments: %v", err)
				return err
			}
			if assignment != nil {
				//assignment exists
				glog.V(4).Infof("ControlPlaneDao.UpdateService setting address assignment on endpoint: %s, %v", service.Endpoints[idx].Name, assignment)
				service.Endpoints[idx].SetAssignment(assignment)
			}
		}
		return this.zkDao.UpdateService(service)
	}
	return err
}

//
func (this *ControlPlaneDao) UpdateService(service dao.Service, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateService: %+v", service)
	//cannot update service without validating it.
	if service.DesiredState == dao.SVC_RUN {
		if err := this.validateServicesForStarting(service, nil); err != nil {
			return err
		}

	}
	return updateService(this, &service)
}

var updateServiceTemplate = func(template dao.ServiceTemplate) error {
	id := strings.TrimSpace(template.Id)
	if id == "" {
		return errors.New("empty Template Id not allowed")
	}
	template.Id = id
	if e := template.Validate(); e != nil {
		return fmt.Errorf("Error validating template: %v", e)
	}
	data, e := jsonMarshal(template)
	if e != nil {
		glog.Errorf("Failed to marshal template")
		return e
	}
	var wrapper dao.ServiceTemplateWrapper
	templateExists := false
	if e := getServiceTemplateWrapper(id, &wrapper); e == nil {
		templateExists = true
	}
	wrapper.Id = id
	wrapper.Name = template.Name
	wrapper.Description = template.Description
	wrapper.ApiVersion = 1
	wrapper.TemplateVersion = 1
	wrapper.Data = string(data)

	if templateExists {
		glog.V(2).Infof("ControlPlaneDao.updateServiceTemplate updating %s", id)
		response, e := indexServiceTemplateWrapper(id, wrapper)
		glog.V(2).Infof("ControlPlaneDao.updateServiceTemplate update %s response: %+v", id, response)
		return e
	} else {
		glog.V(2).Infof("ControlPlaneDao.updateServiceTemplate creating %s", id)
		response, e := newServiceTemplateWrapper(id, wrapper)
		glog.V(2).Infof("ControlPlaneDao.updateServiceTemplate create %s response: %+v", id, response)
		return e
	}
}

//
func (this *ControlPlaneDao) RemoveResourcePool(id string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveResourcePool: %s", id)
	response, err := deleteResourcePool(id)
	glog.V(2).Infof("ControlPlaneDao.RemoveResourcePool response: %+v", response)

	//TODO: remove AddressAssignments with this host

	return err
}

//
func (this *ControlPlaneDao) RemoveHost(id string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveHost: %s", id)
	response, err := deleteHost(id)
	glog.V(2).Infof("ControlPlaneDao.RemoveHost response: %+v", response)
	//TODO: remove AddressAssignments with this host
	return err
}

// RemoveUser removes the user specified by the userName string
func (this *ControlPlaneDao) RemoveUser(userName string, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.RemoveUser: %s", userName)
	response, err := deleteUser(userName)
	glog.V(2).Infof("ControlPlaneDao.RemoveUser response: %+v", response)
	return err
}

//
func (this *ControlPlaneDao) RemoveService(id string, unused *int) error {
	this.walkServices(id, func(svc dao.Service) error {
		this.zkDao.RemoveService(svc.Id)
		return nil
	})

	this.walkServices(id, func(svc dao.Service) error {
		_, err := deleteService(svc.Id)
		if err != nil {
			glog.Errorf("Error removing service %s	 %s ", svc.Id, err)
		}
		return err
	})

	glog.V(2).Infof("ControlPlaneDao.RemoveService: %s", id)
	response, err := deleteService(id)
	glog.V(2).Infof("ControlPlaneDao.RemoveService response: %+v", response)
	if err != nil {
		glog.Errorf("Error removing service %s: %v", id, err)
		return err
	}
	//TODO: remove AddressAssignments with this Service
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

func (this *ControlPlaneDao) GetUser(userName string, user *dao.User) error {
	glog.V(2).Infof("ControlPlaneDao.GetUser: userName=%s", userName)
	request := dao.User{}
	err := getUser(userName, &request)
	glog.V(2).Infof("ControlPlaneDao.GetHost: userName=%s, user=%+v, err=%s", userName, request, err)
	*user = request
	return err
}

//ValidateCredentials takes a user name and password and validates them against a stored user
func (this *ControlPlaneDao) ValidateCredentials(user dao.User, result *bool) error {
	glog.V(2).Infof("ControlPlaneDao.ValidateCredentials: userName=%s", user.Name)
	storedUser := dao.User{}
	err := this.GetUser(user.Name, &storedUser)
	if err != nil {
		*result = false
		return err
	}

	// hash the passed in password
	hashedPassword := hashPassword(user.Password)

	// confirm the password
	if storedUser.Password != hashedPassword {
		*result = false
		return nil
	}

	// at this point we found the user and confirmed the password
	*result = true
	return nil
}

//GetSystemUser returns the system user's credentials. The "unused int" is required by the RPC interface.
func (this *ControlPlaneDao) GetSystemUser(unused int, user *dao.User) error {
	systemUser := dao.User{
		Name:     SYSTEM_USER_NAME,
		Password: INSTANCE_PASSWORD,
	}
	*user = systemUser
	return nil
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
	return getResourcePools(pools)
}

var getResourcePools = func(pools *map[string]*dao.ResourcePool) error {
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
	return getServices(services)
}

var getServices = func(services *[]*dao.Service) error {
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

func (this *ControlPlaneDao) initializedAddressConfig(endpoint dao.ServiceEndpoint) bool {
	// has nothing defined in the service definition
	if endpoint.AddressConfig.Port == 0 && endpoint.AddressConfig.Protocol == "" {
		return false
	}
	return true
}

func (this *ControlPlaneDao) needsAddressAssignment(serviceID string, endpoint dao.ServiceEndpoint) (bool, string, error) {
	// does the endpoint's AddressConfig have any config associated with it?
	if this.initializedAddressConfig(endpoint) {
		addressAssignment, err := this.getEndpointAddressAssignments(serviceID, endpoint.Name)
		if err != nil {
			glog.Errorf("getEndpointAddressAssignments failed: %v", err)
			return false, "", err
		}

		// if there exists some AddressConfig that is initialized to anything (port and protocol are not the default values)
		// and there does NOT exist an AddressAssignment corresponding to this AddressConfig
		// then this service needs an AddressAssignment
		if addressAssignment == nil {
			glog.Infof("Service: %s endpoint: %s needs an address assignment", serviceID, endpoint.Name)
			return true, "", nil
		}

		// if there exists some AddressConfig that is initialized to anything (port and protocol are not the default values)
		// and there already exists an AddressAssignment corresponding to this AddressConfig
		// then this service does NOT need an AddressAssignment (as one already exists)
		return false, addressAssignment.Id, nil
	}

	// this endpoint has no need for an AddressAssignment ever
	return false, "", nil
}

// determine whether the services are ready for deployment
func (this *ControlPlaneDao) validateServicesForStarting(service dao.Service, _ *struct{}) error {
	// ensure all endpoints with AddressConfig have assigned IPs
	for _, endpoint := range service.Endpoints {
		needsAnAddressAssignment, addressAssignmentId, err := this.needsAddressAssignment(service.Id, endpoint)
		if err != nil {
			return err
		}

		if needsAnAddressAssignment {
			msg := fmt.Sprintf("Service ID %s is in need of an AddressAssignment: %s", service.Id, addressAssignmentId)
			return errors.New(msg)
		} else if addressAssignmentId != "" {
			glog.Infof("AddressAssignment: %s already exists", addressAssignmentId)
		}
	}

	if service.RAMCommitment < 0 {
		return fmt.Errorf("service RAM commitment cannot be negative")
	}

	// add additional validation checks to the services
	return nil
}

// Show pool IP address information
func (this *ControlPlaneDao) GetPoolsIPInfo(poolId string, poolsIpInfo *[]dao.HostIPResource) error {
	// retrieve all the hosts that are in the requested pool
	var poolHosts []*dao.PoolHost
	err := this.GetHostsForResourcePool(poolId, &poolHosts)
	if err != nil {
		glog.Errorf("Could not get hosts for Pool %s: %v", poolId, err)
		return err
	}

	for _, poolHost := range poolHosts {
		// retrieve the IPs of the hosts contained in the requested pool
		host := dao.Host{}
		err = this.GetHost(poolHost.HostId, &host)
		if err != nil {
			glog.Errorf("Could not get host %s: %v", poolHost.HostId, err)
			return err
		}

		//aggregate all the IPResources from all the hosts in the requested pool
		for _, poolHostIPResource := range host.IPs {
			if poolHostIPResource.HostId != "" && poolHostIPResource.InterfaceName != "" && poolHostIPResource.IPAddress != "" {
				*poolsIpInfo = append(*poolsIpInfo, poolHostIPResource)
			}
		}
	}

	return nil
}

// used in the walkServices function
type visit func(service dao.Service) error

// assign an IP address to a service (and all its child services) containing non default AddressResourceConfig
func (this *ControlPlaneDao) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) error {
	service := dao.Service{}
	err := getService(assignmentRequest.ServiceId, &service)
	if err != nil {
		return err
	}

	// populate poolsIpInfo
	var poolsIpInfo []dao.HostIPResource
	err = this.GetPoolsIPInfo(service.PoolId, &poolsIpInfo)
	if err != nil {
		glog.Errorf("GetPoolsIPInfo failed: %v", err)
		return err
	}

	if len(poolsIpInfo) < 1 {
		msg := fmt.Sprintf("No IP addresses are available in pool %s.", service.PoolId)
		return errors.New(msg)
	}
	glog.Infof("Pool %v contains %v available IP(s)", service.PoolId, len(poolsIpInfo))

	rand.Seed(time.Now().UTC().UnixNano())
	ipIndex := 0
	userProvidedIPAssignment := false

	if assignmentRequest.AutoAssignment {
		// automatic IP requested
		glog.Infof("Automatic IP Address Assignment")
		ipIndex = rand.Intn(len(poolsIpInfo))
	} else {
		// manual IP provided
		// verify that the user provided IP address is available in the pool
		glog.Infof("Manual IP Address Assignment")
		validIp := false
		userProvidedIPAssignment = true

		for index, hostIPResource := range poolsIpInfo {
			if assignmentRequest.IpAddress == hostIPResource.IPAddress {
				// WHAT HAPPENS IF THERE EXISTS THE SAME IP ON MORE THAN ONE HOST???
				validIp = true
				ipIndex = index
				break
			}
		}

		if !validIp {
			msg := fmt.Sprintf("The requested IP address: %s is not contained in pool %s.", assignmentRequest.IpAddress, service.PoolId)
			return errors.New(msg)
		}
	}
	assignmentRequest.IpAddress = poolsIpInfo[ipIndex].IPAddress
	selectedHostId := poolsIpInfo[ipIndex].HostId
	glog.Infof("Attempting to set IP address(es) to %s", assignmentRequest.IpAddress)

	assignments := []dao.AddressAssignment{}
	this.GetServiceAddressAssignments(assignmentRequest.ServiceId, &assignments)
	if err != nil {
		glog.Errorf("controlPlaneDao.GetServiceAddressAssignments failed in anonymous function: %v", err)
		return err
	}

	visitor := func(service dao.Service) error {
		// if this service is in need of an IP address, assign it an IP address
		for _, endpoint := range service.Endpoints {
			needsAnAddressAssignment, addressAssignmentId, err := this.needsAddressAssignment(service.Id, endpoint)
			if err != nil {
				return err
			}

			// if an address assignment is needed (does not yet exist) OR
			// if a specific IP address is provided by the user AND an address assignment already exists
			if needsAnAddressAssignment || (userProvidedIPAssignment && addressAssignmentId != "") {
				if addressAssignmentId != "" {
					glog.Infof("Removing AddressAssignment: %s", addressAssignmentId)
					err = this.RemoveAddressAssignment(addressAssignmentId, nil)
					if err != nil {
						glog.Errorf("controlPlaneDao.RemoveAddressAssignment failed in AssignIPs anonymous function: %v", err)
						return err
					}
				}
				assignment := dao.AddressAssignment{}
				assignment.AssignmentType = "static"
				assignment.HostId = selectedHostId
				assignment.PoolId = service.PoolId
				assignment.IPAddr = assignmentRequest.IpAddress
				assignment.Port = endpoint.AddressConfig.Port
				assignment.ServiceId = service.Id
				assignment.EndpointName = endpoint.Name
				glog.Infof("Creating AddressAssignment for Endpoint: %s", assignment.EndpointName)

				var unusedStr string
				err = this.AssignAddress(assignment, &unusedStr)
				if err != nil {
					glog.Errorf("AssignAddress failed in AssignIPs anonymous function: %v", err)
					return err
				}
				glog.Infof("Created AddressAssignment: %s for Endpoint: %s", assignment.Id, assignment.EndpointName)
			}
		}
		return nil
	}

	// traverse all the services
	err = this.walkServices(assignmentRequest.ServiceId, visitor)
	if err != nil {
		return err
	}

	glog.Infof("All services requiring an explicit IP address (at this moment) from service: %v and down ... have been assigned: %s", assignmentRequest.ServiceId, assignmentRequest.IpAddress)
	return nil
}

// validate the provided service
func (this *ControlPlaneDao) validateService(serviceId string) error {
	//TODO: create map of IPs to ports and ensure that an IP does not have > 1 process listening on the same port
	visitor := func(service dao.Service) error {
		// validate the service is ready to start
		err := this.validateServicesForStarting(service, nil)
		if err != nil {
			glog.Errorf("Services failed validation for starting")
			return err
		}
		return nil
	}

	// traverse all the services
	return this.walkServices(serviceId, visitor)
}

// start the provided service
func (this *ControlPlaneDao) StartService(serviceId string, unused *string) error {
	// this will traverse all the services
	err := this.validateService(serviceId)
	if err != nil {
		return err
	}

	visitor := func(service dao.Service) error {
		//start this service
		service.DesiredState = dao.SVC_RUN
		err = updateService(this, &service)
		if err != nil {
			return err
		}
		return nil
	}

	// traverse all the services
	return this.walkServices(serviceId, visitor)
}

// traverse all the services (including the children of the provided service)
func (this *ControlPlaneDao) walkServices(serviceId string, visitFn visit) error {
	//get the original service
	service := dao.Service{}
	err := getService(serviceId, &service)
	if err != nil {
		return err
	}

	// do what you requested to do while visiting this node
	err = visitFn(service)
	if err != nil {
		return err
	}

	var query = fmt.Sprintf("ParentServiceId:%s", serviceId)
	subServices, err := this.queryServices(query, "100")
	if err != nil {
		return err
	}
	for _, service := range subServices {
		err = this.walkServices(service.Id, visitFn)
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
	return stopService(this, id)
}

var stopService func(*ControlPlaneDao, string) error

func init() {
	stopService = func(this *ControlPlaneDao, id string) error {
		glog.V(0).Info("ControlPlaneDao.StopService id=", id)
		var service dao.Service
		err := getService(id, &service)
		if err != nil {
			return err
		}
		service.DesiredState = dao.SVC_STOP
		err = updateService(this, &service)
		if err != nil {
			return err
		}
		query := fmt.Sprintf("ParentServiceId:%s AND NOT Launch:manual", id)
		subservices, err := this.queryServices(query, "100")
		if err != nil {
			return err
		}
		for _, service := range subservices {
			subServiceErr := stopService(this, service.Id)
			// if we encounter an error log it and keep trying to shut down the services
			if subServiceErr != nil {
				// keep track of the last err we encountered so
				// the client of this method can know that something went wrong
				err = subServiceErr
				glog.Errorf("Unable to stop service %s because of error: %s", service.Id, subServiceErr)
			}
		}
		return err
	}
}

func (this *ControlPlaneDao) StopRunningInstance(request dao.HostServiceRequest, unused *int) error {
	return this.zkDao.TerminateHostService(request.HostId, request.ServiceStateId)
}

func (this *ControlPlaneDao) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantId *string) error {
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
	return this.deployServiceDefinitions(template.Services, request.TemplateId, request.PoolId, "", volumes, request.DeploymentId, tenantId)
}

func (this *ControlPlaneDao) deployServiceDefinitions(sds []dao.ServiceDefinition, template string, pool string, parentServiceId string, volumes map[string]string, deploymentId string, tenantId *string) error {
	for _, sd := range sds {
		if err := this.deployServiceDefinition(sd, template, pool, parentServiceId, volumes, deploymentId, tenantId); err != nil {
			return err
		}
	}
	return nil
}

func (this *ControlPlaneDao) deployServiceDefinition(sd dao.ServiceDefinition, template string, pool string, parentServiceId string, volumes map[string]string, deploymentId string, tenantId *string) error {
	svcuuid, _ := dao.NewUuid()
	now := time.Now()

	ctx, err := json.Marshal(sd.Context)
	if err != nil {
		return err
	}

	// Always deploy in stopped state, starting is a separate step
	ds := dao.SVC_STOP

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
	svc.Hostname = sd.Hostname
	svc.ConfigFiles = sd.ConfigFiles
	svc.Endpoints = sd.Endpoints
	svc.Tasks = sd.Tasks
	svc.ParentServiceId = parentServiceId
	svc.CreatedAt = now
	svc.UpdatedAt = now
	svc.Volumes = sd.Volumes
	svc.DeploymentId = deploymentId
	svc.LogConfigs = sd.LogConfigs
	svc.Snapshot = sd.Snapshot
	svc.RAMCommitment = sd.RAMCommitment
	svc.Runs = sd.Runs

	//for each endpoint, evaluate it's Application
	if err = svc.EvaluateEndpointTemplates(this); err != nil {
		return err
	}

	//for each endpoint, evaluate it's Application
	if err = svc.EvaluateEndpointTemplates(this); err != nil {
		return err
	}

	if parentServiceId == "" {
		*tenantId = svc.Id
	}

	// Using the tenant id, tag the base image with the tenantID
	if svc.ImageId != "" {
		repotag := strings.SplitN(svc.ImageId, ":", 2)
		path := strings.SplitN(repotag[0], "/", 3)
		path[len(path)-1] = *tenantId + "_" + path[len(path)-1]
		repo := strings.Join(path, "/")

		dockerclient, err := docker.NewClient("unix:///var/run/docker.sock")
		if err != nil {
			glog.Errorf("unable to start docker client")
			return err
		}

		image, err := dockerclient.InspectImage(svc.ImageId)
		if err != nil {
			glog.Errorf("could not look up image: %s", sd.ImageId)
			return err
		}

		options := docker.TagImageOptions{
			Repo:  repo,
			Force: true,
		}

		if err := dockerclient.TagImage(image.ID, options); err != nil {
			glog.Errorf("could not tag image: %s options: %+v", image.ID, options)
			return err
		}

		svc.ImageId = repo
	}

	var serviceId string
	err = addService(this, svc, &serviceId)
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

	return this.deployServiceDefinitions(sd.Services, template, pool, svc.Id, exportedVolumes, deploymentId, tenantId)
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

	if err = serviceTemplate.Validate(); err != nil {
		return fmt.Errorf("Error validating template: %v", err)
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
	go reloadLogstashContainer(this)
	return err
}

var jsonMarshal = func(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (this *ControlPlaneDao) UpdateServiceTemplate(template dao.ServiceTemplate, unused *int) error {
	result := updateServiceTemplate(template)
	go reloadLogstashContainer(this) // don't block the main thread
	return result
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
	go reloadLogstashContainer(this)
	return nil
}

// RemoveAddressAssignemnt Removes an AddressAssignment by id
func (this *ControlPlaneDao) RemoveAddressAssignment(id string, _ *struct{}) error {
	aas, err := this.queryAddressAssignments(fmt.Sprintf("Id:%s", id))
	if err != nil {
		return err
	}
	if len(*aas) == 0 {
		return fmt.Errorf("No AddressAssignment with id %v", id)
	}
	_, err = deleteAddressAssignment(id)
	if err != nil {
		return err
	}
	return nil
}

// AssignAddress Creates an AddressAssignment, verifies that an assignment for the service/endpoint does not already exist
// id param contains id of newly created assignment if successful
func (this *ControlPlaneDao) AssignAddress(assignment dao.AddressAssignment, id *string) error {
	err := assignment.Validate()
	if err != nil {
		return err
	}

	switch assignment.AssignmentType {
	case "static":
		{
			//check host and IP exist
			if err = this.validStaticIp(assignment.HostId, assignment.IPAddr); err != nil {
				return err
			}
		}
	case "virtual":
		{
			// TODO: need to check if virtual IP exists
			return fmt.Errorf("Not yet supported type %v", assignment.AssignmentType)
		}
	default:
		//Validate above should handle this but left here for completenes
		return fmt.Errorf("Invalid assignment type %v", assignment.AssignmentType)
	}

	//check service and endpoint exists
	if err = this.validEndpoint(assignment.ServiceId, assignment.EndpointName); err != nil {
		return err
	}

	//check for existing assignments to this endpoint
	existing, err := this.getEndpointAddressAssignments(assignment.ServiceId, assignment.EndpointName)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("Address Assignment already exists")
	}
	assignment.Id, err = dao.NewUuid()
	if err != nil {
		return err
	}
	_, err = newAddressAssignment(assignment.Id, &assignment)
	if err != nil {
		return err
	}
	*id = assignment.Id
	return nil
}

func (this *ControlPlaneDao) validStaticIp(hostId string, ipAddr string) error {

	hosts, err := this.queryHosts(fmt.Sprintf("Id:%s", hostId))
	if err != nil {
		return err
	}
	if len(hosts) != 1 {
		return fmt.Errorf("Found %v Hosts with id %v", len(hosts), hostId)
	}
	host := hosts[0]
	found := false
	for _, ip := range host.IPs {
		if ip.IPAddress == ipAddr {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Requested static IP is not available: %v", ipAddr)
	}
	return nil
}

func (this *ControlPlaneDao) validEndpoint(serviceId string, endpointName string) error {
	services, err := this.queryServices(fmt.Sprintf("Id:%s", serviceId), "1")
	if err != nil {
		return err
	}
	if len(services) != 1 {
		return fmt.Errorf("Found %v Services with id %v", len(services), serviceId)
	}
	service := services[0]
	found := false
	for _, endpoint := range service.Endpoints {
		if endpointName == endpoint.Name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Endpoint %v not found on service %v", endpointName, serviceId)
	}
	return nil
}

// GetServiceAddressAssignments fills in all AddressAssignments for the specified serviced id.
func (this *ControlPlaneDao) GetServiceAddressAssignments(serviceId string, assignments *[]dao.AddressAssignment) error {
	query := fmt.Sprintf("ServiceId:%s", serviceId)
	results, err := this.queryAddressAssignments(query)
	if err != nil {
		return err
	}
	*assignments = *results
	return nil
}

// queryAddressAssignments query for host ips; returns empty array if no results for query
func (this *ControlPlaneDao) queryAddressAssignments(query string) (*[]dao.AddressAssignment, error) {
	result, err := searchAddressAssignment(query)
	if err != nil {
		return nil, err
	}
	return toAddressAssignments(&result)
}

// getEndpointAddressAssignments returns the AddressAssignment for the service and endpoint, if no assignments the AddressAssignment will be nil
func (this *ControlPlaneDao) getEndpointAddressAssignments(serviceId string, endpointName string) (*dao.AddressAssignment, error) {
	//TODO: this can probably be done w/ a query
	assignments := []dao.AddressAssignment{}
	err := this.GetServiceAddressAssignments(serviceId, &assignments)
	if err != nil {
		return nil, err
	}

	for _, result := range assignments {
		if result.EndpointName == endpointName {
			return &result, nil
		}
	}
	return nil, nil
}

func (this *ControlPlaneDao) GetServiceTemplates(unused int, templates *map[string]*dao.ServiceTemplate) error {
	return getServiceTemplates(templates)
}

var getServiceTemplates = func(templates *map[string]*dao.ServiceTemplate) error {
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
	return deleteSnapshot(this.dfs, snapshotId)
}

func (this *ControlPlaneDao) DeleteSnapshots(serviceId string, unused *int) error {
	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.DeleteSnapshots err=%s", err)
		return err
	}

	if serviceId != tenantId {
		glog.Infof("ControlPlaneDao.DeleteSnapshots service is not the parent, service=%s, tenant=%s", serviceId, tenantId)
		return nil
	}

	return this.dfs.DeleteSnapshots(tenantId)
}

var deleteSnapshot = func(dfs *dfs.DistributedFileSystem, snapshotId string) error {
	return dfs.DeleteSnapshot(snapshotId)
}

func (this *ControlPlaneDao) Rollback(snapshotId string, unused *int) error {
	return rollback(this.dfs, snapshotId)
}

var rollback = func(dfs *dfs.DistributedFileSystem, snapshotId string) error {
	return dfs.Rollback(snapshotId)
}

// Takes a snapshot of the DFS via the host
func (this *ControlPlaneDao) LocalSnapshot(serviceId string, label *string) error {
	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.LocalSnapshot err=%s", err)
		return err
	}

	if id, err := this.dfs.Snapshot(tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.LocalSnapshot err=%s", err)
		return err
	} else {
		*label = id
	}

	return nil
}

// Snapshot is called via RPC by the CLI to take a snapshot for a serviceId
func (this *ControlPlaneDao) Snapshot(serviceId string, label *string) error {
	return snapshot(this, serviceId, label)
}

var snapshot = func(this *ControlPlaneDao, serviceId string, label *string) error {
	glog.V(3).Infof("ControlPlaneDao.Snapshot entering snapshot with service=%s", serviceId)
	defer glog.V(3).Infof("ControlPlaneDao.Snapshot finished snapshot for service=%s", serviceId)

	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao: dao.LocalSnapshot err=%s", err)
		return err
	}

	// request a snapshot by placing request znode in zookeeper - leader will notice
	snapshotRequest, err := dao.NewSnapshotRequest(serviceId, "")
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao: dao.NewSnapshotRequest err=%s", err)
		return err
	}
	if err := this.zkDao.AddSnapshotRequest(snapshotRequest); err != nil {
		glog.V(2).Infof("ControlPlaneDao.zkDao.AddSnapshotRequest err=%s", err)
		return err
	}
	// TODO:
	//	requestId := snapshotRequest.Id
	//	defer this.zkDao.RemoveSnapshotRequest(requestId)

	glog.V(1).Infof("added snapshot request: %+v", snapshotRequest)

	// wait for completion of snapshot request
	timeout := 60
	for i := 0; i < timeout; i++ {
		glog.V(2).Infof("watching for snapshot completion for request: %+v", snapshotRequest)
		_, _, err := this.zkDao.LoadSnapshotRequestW(snapshotRequest.Id, snapshotRequest)
		switch {
		case err != nil:
			glog.V(2).Infof("ControlPlaneDao: watch snapshot request err=%s", err)
			return err
		case snapshotRequest.SnapshotError != "":
			glog.V(2).Infof("ControlPlaneDao: watch snapshot request err=%s", snapshotRequest.SnapshotError)
			return errors.New(snapshotRequest.SnapshotError)
		case snapshotRequest.SnapshotLabel != "":
			*label = snapshotRequest.SnapshotLabel
			glog.V(1).Infof("completed snapshot request: %+v", snapshotRequest)
			return nil
		}

		time.Sleep(time.Second)
	}

	err = errors.New(fmt.Sprintf("timed out waiting %v for snapshot: %+v", time.Duration(timeout), snapshotRequest))
	glog.Error(err)
	return err
}

func (this *ControlPlaneDao) GetVolume(serviceId string, theVolume *volume.Volume) error {
	return getVolume(this.vfs, serviceId, theVolume)
}

var getVolume = func(vfs, serviceId string, theVolume *volume.Volume) error {
	var tenantId string
	if err := getTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v err=%s", serviceId, err)
		return err
	}
	glog.V(3).Infof("ControlPlaneDao.GetVolume service=%+v tenantId=%s", serviceId, tenantId)
	var service dao.Service
	if err := getService(tenantId, &service); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v err=%s", serviceId, err)
		return err
	}
	glog.V(3).Infof("ControlPlaneDao.GetVolume service=%+v poolId=%s", service, service.PoolId)

	aVolume, err := getSubvolume(vfs, service.PoolId, tenantId)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v err=%s", serviceId, err)
		return err
	}
	if aVolume == nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v volume=nil", serviceId)
		return errors.New("volume is nil")
	}

	glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v volume2=%+v %v", serviceId, aVolume, aVolume)
	*theVolume = *aVolume
	return nil
}

// Commits a container to an image and saves it on the DFS
func (this *ControlPlaneDao) Commit(containerId string, label *string) error {
	if id, err := this.dfs.Commit(containerId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume containerId=%s err=%s", containerId, err)
		return err
	} else {
		*label = id
	}

	return nil
}

func getSubvolume(vfs, poolId, tenantId string) (*volume.Volume, error) {
	baseDir, err := filepath.Abs(path.Join(varPath(), "volumes", poolId))
	if err != nil {
		return nil, err
	}
	glog.Infof("Mounting vfs:%v tenantId:%v baseDir:%v\n", vfs, tenantId, baseDir)
	return volume.Mount(vfs, tenantId, baseDir)
}

func varPath() string {
	if len(os.Getenv("SERVICED_HOME")) > 0 {
		return path.Join(os.Getenv("SERVICED_HOME"), "var")
	} else if user, err := user.Current(); err == nil {
		return path.Join(os.TempDir(), "serviced-"+user.Username, "var")
	} else {
		defaultPath := "/tmp/serviced/var"
		glog.Warningf("Defaulting varPath to:%v\n", defaultPath)
		return defaultPath
	}
}

func (this *ControlPlaneDao) Snapshots(serviceId string, labels *[]string) error {

	var tenantId string
	if err := getTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshots service=%+v err=%s", serviceId, err)
		return err
	}
	var service dao.Service
	err := getService(tenantId, &service)
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

var commandAsRoot = func(name string, arg ...string) (*exec.Cmd, error) {
	user, e := user.Current()
	if e != nil {
		return nil, e
	}
	if user.Uid == "0" {
		return exec.Command(name, arg...), nil
	}
	_, e = exec.Command("sudo", "-n", "echo").CombinedOutput()
	if e != nil {
		return nil, e
	}
	return exec.Command("sudo", append([]string{"-n", name}, arg...)...), nil //Go, you make me sad.
}

var writeDirectoryToTgz = func(src, filename string) error {
	cmd, e := commandAsRoot("tar", "-czf", filename, "-C", src, ".")
	if e != nil {
		return e
	}
	return cmd.Run()
}

var writeDirectoryFromTgz = func(dest, filename string) (err error) {
	if _, e := osStat(dest); e != nil {
		if !os.IsNotExist(e) {
			glog.Errorf("Could not stat %s: %v", dest, e)
			return e
		}
		if e := osMkdirAll(dest, os.ModeDir|0755); e != nil {
			glog.Errorf("Could not find nor create %s: %v", dest, e)
			return e
		}
		defer func() {
			if err != nil {
				if e := osRemoveAll(dest); e != nil {
					glog.Errorf("Could not remove %s: %v", dest, e)
				}
			}
		}()
	}
	cmd, e := commandAsRoot("tar", "-xpUf", filename, "-C", dest, "--numeric-owner")
	if e != nil {
		return e
	}
	return cmd.Run()
}

var writeJsonToFile = func(v interface{}, filename string) (err error) {
	file, e := osCreate(filename)
	if e != nil {
		glog.Errorf("Could not create file %s: %v", filename, e)
		return e
	}
	defer func() {
		if e := file.Close(); e != nil {
			glog.Errorf("Error while closing file %s: %v", filename, e)
			if err == nil {
				err = e
			}
		}
	}()
	encoder := json.NewEncoder(file)
	if e := encoder.Encode(v); e != nil {
		glog.Errorf("Could not write JSON data to %s: %v", filename, e)
		return e
	}
	return nil
}

var readJsonFromFile = func(v interface{}, filename string) error {
	file, e := osOpen(filename)
	if e != nil {
		glog.Errorf("Could not open file %s: %v", filename, e)
		return e
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if e := decoder.Decode(v); e != nil {
		glog.Errorf("Could not read JSON data from %s: %v", filename, e)
		return e
	}
	return nil
}

var newDockerImporter = func() (dockerImporter, error) {
	return docker.NewClient(DOCKER_ENDPOINT)
}

var newDockerExporter = func() (dockerExporter, error) {
	return docker.NewClient(DOCKER_ENDPOINT)
}

type dockerExporter interface {
	CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
	RemoveContainer(docker.RemoveContainerOptions) error
	ExportContainer(docker.ExportContainerOptions) error
	ListImages(bool) ([]docker.APIImages, error)
}

type dockerImporter interface {
	ImportImage(docker.ImportImageOptions) error
	InspectImage(string) (*docker.Image, error)
	TagImage(string, docker.TagImageOptions) error
}

var getDockerImageNameIds = func(client dockerExporter) (map[string]string, error) {
	images, e := client.ListImages(true)
	if e != nil {
		return nil, e
	}
	result := make(map[string]string)
	for _, image := range images {
		result[image.ID] = image.ID
		for _, repotag := range image.RepoTags {
			repo, tag := repoAndTag(repotag)
			if tag == "" || tag == "latest" {
				result[repo] = image.ID
			} else {
				result[repotag] = image.ID
			}
		}
	}
	return result, nil
}

var exportDockerImageToFile = func(client dockerExporter, imageId, filename string) (err error) {
	file, e := osCreate(filename)
	if e != nil {
		glog.Errorf("Could not create file %s: %v", filename, e)
		return e
	}

	// Close (and perhaps delete) file on the way out
	defer func() {
		if e := file.Close(); e != nil {
			glog.Errorf("Error while closing file %s: %v", filename, e)
			if err == nil {
				err = e
			}
		}
		if err != nil && file != nil {
			if e := osRemoveAll(filename); e != nil {
				glog.Errorf("Error while removing file %s: %v", filename, e)
			}
		}
	}()

	createOpts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Cmd:   []string{"echo ''"},
			Image: imageId,
		},
	}

	container, e := client.CreateContainer(createOpts)
	if e != nil {
		glog.Errorf("Could not create container from image %s: %v", imageId, e)
		return e
	}

	glog.Infof("Created container %s based on image %s", container.ID, imageId)

	// Remove container on the way out
	defer func() {
		removeOpts := docker.RemoveContainerOptions{ID: container.ID}
		if e := client.RemoveContainer(removeOpts); e != nil {
			glog.Errorf("Could not remove container %s: %v", container.ID, e)
			if err == nil {
				err = e
			}
		} else {
			glog.Infof("Removed container %s", container.ID)
		}
	}()

	exportOpts := docker.ExportContainerOptions{
		ID:           container.ID,
		OutputStream: file,
	}

	if e = client.ExportContainer(exportOpts); e != nil {
		glog.Errorf("Could not export container %s: %v", container.ID, e)
		return e
	}

	glog.Infof("Exported container %s (based on image %s) to %s", container.ID, imageId, filename)
	return nil
}

var repoAndTag = func(imageId string) (string, string) {
	i := strings.LastIndex(imageId, ":")
	if i < 0 {
		return imageId, ""
	}
	tag := imageId[i+1:]
	if strings.Contains(tag, "/") {
		return imageId, ""
	}
	return imageId[:i], tag
}

var importDockerImageFromFile = func(client dockerImporter, imageId, filename string) (err error) {
	file, e := os.Open(filename)
	if e != nil {
		return e
	}
	defer file.Close()
	repo, tag := repoAndTag(imageId)
	importOpts := docker.ImportImageOptions{
		Repository:  repo,
		Source:      "-",
		InputStream: file,
		Tag:         tag,
	}
	if e = client.ImportImage(importOpts); e != nil {
		return e
	}
	return nil
}

var utcNow = func() time.Time {
	return time.Now().UTC()
}

// Find all docker images referenced by a template or service
var dockerImageSet = func(templates map[string]*dao.ServiceTemplate, services []*dao.Service) map[string]bool {
	imageSet := make(map[string]bool)
	var visit func(*[]dao.ServiceDefinition)
	visit = func(defs *[]dao.ServiceDefinition) {
		for _, serviceDefinition := range *defs {
			if serviceDefinition.ImageId != "" {
				imageSet[serviceDefinition.ImageId] = true
			}
			visit(&serviceDefinition.Services)
		}
	}
	for _, template := range templates {
		visit(&template.Services)
	}
	for _, service := range services {
		if service.ImageId != "" {
			imageSet[service.ImageId] = true
		}
	}
	return imageSet
}

func (this *ControlPlaneDao) Backup(backupsDirectory string, backupFilePath *string) (err error) {
	var (
		templates      map[string]*dao.ServiceTemplate
		services       []*dao.Service
		imagesNameTags [][]string
	)
	backupName := utcNow().Format("backup-2006-01-02-150405")
	if backupsDirectory == "" {
		backupsDirectory = filepath.Join(varPath(), "backups")
	}
	*backupFilePath = path.Join(backupsDirectory, backupName+".tgz")
	defer func() {
		// Zero-value the backupFilePath if we're returning an error
		if err != nil && backupFilePath != nil && *backupFilePath != "" {
			*backupFilePath = ""
		}
	}()
	backupPath := func(relPath ...string) string {
		return filepath.Join(append([]string{backupsDirectory, backupName}, relPath...)...)
	}
	if e := osMkdirAll(backupPath("images"), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", backupPath(), e)
		return e
	}
	defer func() {
		if e := osRemoveAll(backupPath()); e != nil {
			glog.Errorf("Could not remove %s: %v", backupPath(), e)
			if err == nil {
				err = e
			}
		}
	}()
	if e := osMkdirAll(backupPath("snapshots"), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", backupPath(), e)
		return e
	}

	// Dump all template definitions
	if e := getServiceTemplates(&templates); e != nil {
		glog.Errorf("Could not get templates: %v", e)
		return e
	}
	if e := writeJsonToFile(templates, backupPath("templates.json")); e != nil {
		glog.Errorf("Could not write templates.json: %v", e)
		return e
	}

	// Dump all service definitions
	if e := getServices(&services); e != nil {
		glog.Errorf("Could not get services: %v", e)
		return e
	}
	if e := writeJsonToFile(services, backupPath("services.json")); e != nil {
		glog.Errorf("Could not write services.json: %v", err)
		return e
	}

	// Export each of the referenced docker images
	client, e := newDockerExporter()
	if e != nil {
		glog.Errorf("Could not connect to docker: %v", e)
		return e
	}
	// Note: client does not need to be .Close()'d

	imageNameIds, e := getDockerImageNameIds(client)
	if e != nil {
		glog.Errorf("Could not get image tags from docker: %v", e)
		return e
	}

	imageIdTags := make(map[string][]string)

	imageNameSet := dockerImageSet(templates, services)

	for imageName, _ := range imageNameSet {
		imageId := imageNameIds[imageName]
		imageIdTags[imageId] = []string{}
	}

	for imageName, imageId := range imageNameIds {
		if imageName == imageId {
			continue
		}
		tags := imageIdTags[imageId]
		if tags == nil {
			continue
		}
		imageIdTags[imageId] = append(tags, imageName)
	}

	i := 0
	for imageId, imageTags := range imageIdTags {
		filename := backupPath("images", fmt.Sprintf("%d.tar", i))
		if e := exportDockerImageToFile(client, imageId, filename); e != nil {
			if e == docker.ErrNoSuchImage {
				glog.Infof("Docker image %s was referenced, but does not exist. Ignoring.", imageId)
			} else {
				glog.Errorf("Error while exporting docker image %s: %v", imageId, e)
				return e
			}
		} else {
			imageNameWithTags := append([]string{imageId}, imageTags...)
			imagesNameTags = append(imagesNameTags, imageNameWithTags)
			i++
		}
	}

	if e := writeJsonToFile(imagesNameTags, backupPath("images.json")); e != nil {
		glog.Errorf("Could not write images.json: %v", e)
		return e
	}

	snapshotToTgzFile := func(service *dao.Service) (filename string, err error) {
		var snapshotId string
		if e := snapshot(this, service.Id, &snapshotId); e != nil {
			glog.Errorf("Could not snapshot service %s: %v", service.Id, e)
			return "", e
		}

		// Delete snapshot on the way out
		defer func() {
			if e := deleteSnapshot(this.dfs, snapshotId); e != nil {
				glog.Errorf("Error while deleting snapshot %s: %v", snapshotId, e)
				if err == nil {
					err = e
				}
			}
		}()
		snapDir, e := getSnapshotPath(this.vfs, service.PoolId, service.Id, snapshotId)
		if e != nil {
			glog.Errorf("Could not get subvolume %s:%s: %v", service.PoolId, service.Id, e)
			return "", e
		}
		snapFile := backupPath("snapshots", fmt.Sprintf("%s.tgz", snapshotId))
		if e := writeDirectoryToTgz(snapDir, snapFile); e != nil {
			glog.Errorf("Could not write %s to %s: %v", snapDir, snapFile, e)
			return "", e
		}
		return snapFile, nil
	}

	for _, service := range services {
		if service.ParentServiceId == "" {
			if _, e := snapshotToTgzFile(service); e != nil {
				glog.Errorf("Could not save snapshot of service %s: %v", service.Id, e)
				return e
			}
			// Note: the deferred RemoveAll (above) will cleanup the file.
		}
	}

	if e := writeDirectoryToTgz(backupPath(), *backupFilePath); e != nil {
		glog.Errorf("Could not write %s to %s: %v", backupPath(), backupFilePath, e)
		return e
	}

	return nil
}

var getSnapshotPath = func(vfs, poolId, serviceId, snapshotId string) (string, error) {
	volume, e := getSubvolume(vfs, poolId, serviceId)
	if e != nil {
		return "", e
	}
	return volume.SnapshotPath(snapshotId), nil
}

func (this *ControlPlaneDao) Restore(backupFilePath string, unused *int) (err error) {
	//TODO: acquire restore mutex, defer release
	var (
		doReloadLogstashContainer bool
		existingServices          []*dao.Service
		existingPools             map[string]*dao.ResourcePool
		templates                 map[string]*dao.ServiceTemplate
		services                  []*dao.Service
		imagesNameTags            [][]string
	)
	defer func() {
		if doReloadLogstashContainer {
			go reloadLogstashContainer(this) // don't block the main thread
		}
	}()
	restorePath := func(relPath ...string) string {
		return filepath.Join(append([]string{varPath(), "restore"}, relPath...)...)
	}

	if e := osRemoveAll(restorePath()); e != nil {
		glog.Errorf("Could not remove %s: %v", restorePath(), e)
		return e
	}

	if e := osMkdirAll(restorePath(), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", restorePath(), e)
		return e
	}

	defer func() {
		if e := osRemoveAll(restorePath()); e != nil {
			glog.Errorf("Could not remove %s: %v", restorePath(), e)
			if err == nil {
				err = e
			}
		}
	}()

	if e := writeDirectoryFromTgz(restorePath(), backupFilePath); e != nil {
		glog.Errorf("Could not expand %s to %s: %v", backupFilePath, restorePath(), e)
		return e
	}

	if e := readJsonFromFile(&templates, restorePath("templates.json")); e != nil {
		glog.Errorf("Could not read templates from %s: %v", restorePath("templates.json"), e)
		return e
	}

	if e := readJsonFromFile(&services, restorePath("services.json")); e != nil {
		glog.Errorf("Could not read services from %s: %v", restorePath("services.json"), e)
		return e
	}

	if e := readJsonFromFile(&imagesNameTags, restorePath("images.json")); e != nil {
		glog.Errorf("Could not read images from %s: %v", restorePath("images.json"), e)
		return e
	}

	// Restore the service templates ...
	for templateId, template := range templates {
		template.Id = templateId
		if e := updateServiceTemplate(*template); e != nil {
			glog.Errorf("Could not update template %s: %v", templateId, e)
			return e
		}
		doReloadLogstashContainer = true
	}

	// Restore the services ...
	if e := getServices(&existingServices); e != nil {
		glog.Errorf("Could not get existing services: %v", e)
		return e
	}
	if e := getResourcePools(&existingPools); e != nil {
		glog.Errorf("Could not get existing pools: %v", e)
		return e
	}
	existingServiceMap := make(map[string]*dao.Service)
	for _, service := range existingServices {
		existingServiceMap[service.Id] = service
	}
	for _, service := range services {
		if existingService := existingServiceMap[service.Id]; existingService != nil {
			if e := stopService(this, service.Id); e != nil {
				glog.Errorf("Could not stop service %s: %v", service.Id, e)
				return e
			}
			service.PoolId = existingService.PoolId
			if existingPools[service.PoolId] == nil {
				glog.Infof("Changing PoolId of service %s from %s to default", service.Id, service.PoolId)
				service.PoolId = "default"
			}
			if e := updateService(this, service); e != nil {
				glog.Errorf("Could not update service %s: %v", service.Id, e)
				return e
			}
		} else {
			if existingPools[service.PoolId] == nil {
				glog.Infof("Changing PoolId of service %s from %s to default", service.Id, service.PoolId)
				service.PoolId = "default"
			}
			var serviceId string
			if e := addService(this, *service, &serviceId); e != nil {
				glog.Errorf("Could not add service %s: %v", service.Id, e)
				return e
			}
			if service.Id != serviceId {
				msg := fmt.Sprintf("BUG!!! ADDED SERVICE %s, BUT WITH THE WRONG ID: %s", service.Id, serviceId)
				glog.Errorf(msg)
				return errors.New(msg)
			}
			existingServiceMap[service.Id] = service
		}
	}

	// Restore the docker images ...
	client, e := newDockerImporter()
	// Note: client does not need to be .Close()'d
	if e != nil {
		glog.Errorf("Could not connect to docker: %v", e)
		return e
	}
	for i, imageNameWithTags := range imagesNameTags {
		imageId := imageNameWithTags[0]
		imageTags := imageNameWithTags[1:]
		filename := restorePath("images", fmt.Sprintf("%d.tar", i))
		imageName := "imported:" + imageId
		if e := importDockerImageFromFile(client, imageName, filename); e != nil {
			glog.Errorf("Could not import docker image %s (%+v) from file %s: %v", imageId, imageTags, filename, e)
			return e
		}
		image, e := client.InspectImage(imageName)
		if e != nil {
			glog.Errorf("Could not find imported docker image %s (%+v): %v", imageName, imageTags, e)
			return e
		}
		for _, imageTag := range imageTags {
			repo, tag := repoAndTag(imageTag)
			options := docker.TagImageOptions{
				Repo:  repo,
				Tag:   tag,
				Force: true,
			}
			if e := client.TagImage(image.ID, options); e != nil {
				glog.Errorf("Could not tag image %s (%s) options: %+v: %v", image.ID, imageName, options, e)
				return e
			}
		}
	}

	// Restore the snapshots ...
	snapFiles, e := readDirFileNames(restorePath("snapshots"))
	if e != nil {
		glog.Errorf("Could not list contents of %s: %v", restorePath("snapshots"), e)
		return e
	}
	for _, snapFile := range snapFiles {
		snapshotId := strings.TrimSuffix(snapFile, ".tgz")
		if snapshotId == snapFile {
			continue //the filename does not end with .tgz
		}
		parts := strings.Split(snapshotId, "_")
		if len(parts) != 2 {
			glog.Warningf("Skipping restoration of snapshot %s, due to malformed ID!", snapshotId)
			continue
		}
		serviceId := parts[0]
		service := existingServiceMap[serviceId]
		if service == nil {
			glog.Warningf("Could not find service %s for snapshot %s. Skipping!", serviceId, snapshotId)
			continue
		}
		snapDir, e := getSnapshotPath(this.vfs, service.PoolId, service.Id, snapshotId)
		if e != nil {
			glog.Errorf("Could not get subvolume %s:%s: %v", service.PoolId, service.Id, e)
			return e
		}
		filename := restorePath("snapshots", snapFile)
		if e := writeDirectoryFromTgz(snapDir, filename); e != nil {
			glog.Errorf("Could not write %s from %s: %v", snapDir, filename, e)
			return e
		}

		defer func() {
			if e := deleteSnapshot(this.dfs, snapshotId); e != nil {
				glog.Errorf("Couldn't delete snapshot %s: %v", snapshotId, e)
				if err == nil {
					err = e
				}
			}
		}()

		if e := this.Rollback(snapshotId, unused); e != nil {
			glog.Errorf("Could not rollback to snapshot %s: %v", snapshotId, e)
			return e
		}
	}

	//TODO: garbage collect (http://jimhoskins.com/2013/07/27/remove-untagged-docker-images.html)
	return nil
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int) (*ControlPlaneDao, error) {
	glog.V(0).Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
	api.Domain = hostName
	api.Port = strconv.Itoa(port)

	dao := &ControlPlaneDao{
		hostName: hostName,
		port:     port,
	}
	if dfs, err := dfs.NewDistributedFileSystem(dao); err != nil {
		return nil, err
	} else {
		dao.dfs = dfs
	}

	return dao, nil
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

//createSystemUser updates the running instance password as well as the user record in elastic
func createSystemUser(s *ControlPlaneDao) error {
	user := dao.User{}
	err := s.GetUser(SYSTEM_USER_NAME, &user)
	if err != nil {
		glog.Errorf("%s", err)
		glog.V(0).Info("'default' user not found; creating...")

		// create the system user
		user := dao.User{}
		user.Name = SYSTEM_USER_NAME
		userName := SYSTEM_USER_NAME

		if err := s.AddUser(user, &userName); err != nil {
			return err
		}
	}

	// update the instance password
	password, err := dao.NewUuid()
	if err != nil {
		return err
	}
	user.Name = SYSTEM_USER_NAME
	user.Password = password
	INSTANCE_PASSWORD = password
	unused := 0
	return s.UpdateUser(user, &unused)
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

	// create the account credentials
	if err = createSystemUser(s); err != nil {
		return nil, err
	}

	go s.handleScheduler(hid)

	return s, nil
}

func (s *ControlPlaneDao) ReadyDFS(unused bool, unusedint *int) (err error) {
	s.dfs.Lock()
	s.dfs.Unlock()
	return
}

// writeLogstashConfiguration takes all the available
// services and writes out the filters section for logstash.
// This is required before logstash startsup
func (s *ControlPlaneDao) writeLogstashConfiguration() error {
	var templatesMap map[string]*dao.ServiceTemplate
	if err := s.GetServiceTemplates(0, &templatesMap); err != nil {
		return err
	}

	// FIXME: eventually this file should live in the DFS or the config should
	// live in zookeeper to allow the agents to get to this
	if err := dao.WriteConfigurationFile(templatesMap); err != nil {
		return err
	}
	return nil
}

// Anytime the available service definitions are modified
// we need to restart the logstash container so it can write out
// its new filter set.
// This method depends on the elasticsearch container being up and running.
var reloadLogstashContainer = func(s *ControlPlaneDao) error {
	err := s.writeLogstashConfiguration()
	if err != nil {
		glog.Fatalf("Could not write logstash configuration: %s", err)
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

var readDirFileNames = func(dirname string) ([]string, error) {
	files, e := ioutil.ReadDir(dirname)
	result := make([]string, len(files))
	if e != nil {
		return result, e
	}
	for i, file := range files {
		result[i] = file.Name()
	}
	return result, nil
}

var ioutilWriteFile = func(filename string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(filename, data, perm)
}

var osOpen = func(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

var osCreate = func(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

var osStat = func(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

var osMkdirAll = func(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

var osRemoveAll = func(path string) error {
	return os.RemoveAll(path)
}

const HOST_ID_CMDString = "/usr/bin/hostid"
