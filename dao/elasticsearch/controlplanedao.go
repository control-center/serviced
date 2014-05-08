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
	"github.com/zenoss/glog"
	docker "github.com/zenoss/go-dockerclient"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/dfs"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicestate"
	"github.com/zenoss/serviced/domain/servicetemplate"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/volume"
	"github.com/zenoss/serviced/zzk"

	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
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
	serviceExists      func(string) (bool, error) = exists(&Pretty, "controlplane", "service")
	serviceStateExists func(string) (bool, error) = exists(&Pretty, "controlplane", "servicestate")
	userExists         func(string) (bool, error) = exists(&Pretty, "controlplane", "user")

	//model index functions
	newService           func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "service")
	newAddressAssignment func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "addressassignment")
	newUser              func(string, interface{}) (api.BaseResponse, error) = create(&Pretty, "controlplane", "user")

	//model index functions
	indexService      func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "service")
	indexServiceState func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "servicestate")
	indexUser         func(string, interface{}) (api.BaseResponse, error) = index(&Pretty, "controlplane", "user")

	//model delete functions
	deleteService           func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "service")
	deleteServiceState      func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "servicestate")
	deleteAddressAssignment func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "addressassignment")
	deleteUser              func(string) (api.BaseResponse, error) = _delete(&Pretty, "controlplane", "user")

	//model get functions
	getService      func(string, interface{}) error = getSource("controlplane", "service")
	getServiceState func(string, interface{}) error = getSource("controlplane", "servicestate")
	getUser         func(string, interface{}) error = getSource("controlplane", "user")

	//model search functions, using uri based query
	searchServiceUri        func(string) (core.SearchResult, error) = searchUri("controlplane", "service")
	searchServiceStateUri   func(string) (core.SearchResult, error) = searchUri("controlplane", "servicestate")
	searchAddressAssignment func(string) (core.SearchResult, error) = searchUri("controlplane", "addressassignment")
	searchUserUri           func(string) (core.SearchResult, error) = searchUri("controlplane", "user")
)

// each time Serviced starts up a new password will be generated. This will be passed into
// the containers so that they can authenticate against the API
var SYSTEM_USER_NAME = "system_user"
var INSTANCE_PASSWORD string

type ControlPlaneDao struct {
	hostName string
	port     int
	varpath  string
	vfs      string
	zclient  *coordclient.Client
	zkDao    *zzk.ZkDao
	dfs      *dfs.DistributedFileSystem
	//needed while we move things over
	facade *facade.Facade
}

// convert search result of json services to service.Service array
func toServices(result *core.SearchResult) ([]*service.Service, error) {
	var err error = nil
	var total = len(result.Hits.Hits)
	var services []*service.Service = make([]*service.Service, total)
	for i := 0; i < total; i += 1 {
		var service service.Service
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
func toAddressAssignments(result *core.SearchResult) (*[]service.AddressAssignment, error) {
	var err error = nil
	var total = len(result.Hits.Hits)
	var addressAssignments = make([]service.AddressAssignment, total)
	for i := 0; i < total; i += 1 {
		var addressAssignment service.AddressAssignment
		err = json.Unmarshal(result.Hits.Hits[i].Source, &addressAssignment)
		if err == nil {
			addressAssignments[i] = addressAssignment
		} else {
			return nil, err
		}
	}

	return &addressAssignments, err
}

// query for services using uri
func (this *ControlPlaneDao) queryServices(queryStr, quantity string) ([]*service.Service, error) {
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

func (this *ControlPlaneDao) getServiceTree(serviceId string, servicesList *[]*service.Service) (servicesMap map[string]*treenode, topService *treenode) {
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
	var myService service.Service
	err = this.GetService(serviceId, &myService)
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceEndpoints service=%+v err=%s", myService, err)
		return
	}

	service_imports := myService.GetServiceImports()
	if len(service_imports) > 0 {
		glog.V(2).Infof("%+v service imports=%+v", myService, service_imports)

		var request dao.EntityRequest
		var servicesList []*service.Service
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
		var states []*servicestate.ServiceState
		err = this.zkDao.GetServiceStates(&states, relatedServiceIds...)
		if err != nil {
			return
		}

		// for each proxied port, find list of potential remote endpoints
		for _, endpoint := range service_imports {
			glog.V(2).Infof("Finding exports for import: %s %+v", endpoint.Application, endpoint)
			matchedEndpoint := false
			applicationRegex, err := regexp.Compile(fmt.Sprintf("^%s$", endpoint.Application))
			if err != nil {
				continue //Don't spam error message; it was reported at validation time
			}
			for _, ss := range states {
				hostPort, containerPort, protocol, match := ss.GetHostEndpointInfo(applicationRegex)
				if match {
					glog.V(1).Infof("Matched endpoint: %s.%s -> %s:%d (%s/%d)",
						myService.Name, endpoint.Application, ss.HostIp, hostPort, protocol, containerPort)
					// if port/protocol undefined in the import, use the export's values
					if endpoint.PortNumber != 0 {
						containerPort = endpoint.PortNumber
					}
					if endpoint.Protocol != "" {
						protocol = endpoint.Protocol
					}
					var ep dao.ApplicationEndpoint
					ep.ServiceId = ss.ServiceId
					ep.ContainerPort = containerPort
					ep.HostPort = hostPort
					ep.HostIp = ss.HostIp
					ep.ContainerIp = ss.PrivateIp
					ep.Protocol = protocol
					ep.VirtualAddress = endpoint.VirtualAddress

					key := fmt.Sprintf("%s:%d", protocol, containerPort)
					if _, exists := remoteEndpoints[key]; !exists {
						remoteEndpoints[key] = make([]*dao.ApplicationEndpoint, 0)
					}
					remoteEndpoints[key] = append(remoteEndpoints[key], &ep)
					matchedEndpoint = true
				}
			}
			if !matchedEndpoint {
				glog.V(1).Infof("Unmatched endpoint %s.%s", myService.Name, endpoint.Application)
			}
		}

		*response = remoteEndpoints
		glog.V(2).Infof("Return for %s is %+v", serviceId, remoteEndpoints)
	}
	return
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

// The tenant id is the root service uuid. Walk the service tree to root to find the tenant id.
func (this *ControlPlaneDao) GetTenantId(serviceId string, tenantId *string) (err error) {
	glog.V(2).Infof("ControlPlaneDao.GetTenantId: %s", serviceId)
	id := strings.TrimSpace(serviceId)
	if id == "" {
		return errors.New("empty serviceId not allowed")
	}

	var traverse func(string) (string, error)

	traverse = func(id string) (string, error) {
		var service service.Service
		if err := this.GetService(id, &service); err != nil {
			return "", err
		} else if service.ParentServiceId != "" {
			return traverse(service.ParentServiceId)
		} else {
			glog.V(1).Infof("parent service: %+v", service)
			return service.Id, nil
		}
	}

	*tenantId, err = traverse(id)
	return
}

//
func (this *ControlPlaneDao) AddService(service service.Service, serviceId *string) error {
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
func (this *ControlPlaneDao) updateService(service *service.Service) error {
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
func (this *ControlPlaneDao) UpdateService(service service.Service, unused *int) error {
	glog.V(2).Infof("ControlPlaneDao.UpdateService: %+v", service)
	//cannot update service without validating it.
	if service.DesiredState == dao.SVC_RUN {
		if err := this.validateServicesForStarting(service, nil); err != nil {
			return err
		}

	}
	return this.updateService(&service)
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
	this.walkServices(id, func(svc service.Service) error {
		this.zkDao.RemoveService(svc.Id)
		return nil
	})

	this.walkServices(id, func(svc service.Service) error {
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

func (this *ControlPlaneDao) GetUser(userName string, user *dao.User) error {
	glog.V(2).Infof("ControlPlaneDao.GetUser: userName=%s", userName)
	request := dao.User{}
	err := getUser(userName, &request)
	glog.V(2).Infof("ControlPlaneDao.GetUser: userName=%s, user=%+v, err=%s", userName, request, err)
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
func (this *ControlPlaneDao) GetService(id string, myService *service.Service) error {
	glog.V(3).Infof("ControlPlaneDao.GetService: id=%s", id)
	request := service.Service{}
	err := getService(id, &request)
	glog.V(3).Infof("ControlPlaneDao.GetService: id=%s, service=%+v, err=%s", id, request, err)
	*myService = request
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
	var serviceStates []*servicestate.ServiceState
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
	var serviceState servicestate.ServiceState
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
func (this *ControlPlaneDao) GetServices(request dao.EntityRequest, services *[]*service.Service) error {
	glog.V(3).Infof("ControlPlaneDao.GetServices")
	query := search.Query().Search("_exists_:Id")
	results, err := search.Search("controlplane").Type("service").Size("50000").Query(query).Result()
	if err != nil {
		glog.Error("ControlPlaneDao.GetServices: err=", err)
		return err
	}
	var service_results []*service.Service
	service_results, err = toServices(results)
	if err != nil {
		return err
	}

	*services = service_results
	return nil
}

//
func (this *ControlPlaneDao) GetTaggedServices(request dao.EntityRequest, services *[]*service.Service) error {
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

		var service_results []*service.Service
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

func (this *ControlPlaneDao) initializedAddressConfig(endpoint service.ServiceEndpoint) bool {
	// has nothing defined in the service definition
	if endpoint.AddressConfig.Port == 0 && endpoint.AddressConfig.Protocol == "" {
		return false
	}
	return true
}

func (this *ControlPlaneDao) needsAddressAssignment(serviceID string, endpoint service.ServiceEndpoint) (bool, string, error) {
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
		return false, addressAssignment.ID, nil
	}

	// this endpoint has no need for an AddressAssignment ever
	return false, "", nil
}

// determine whether the services are ready for deployment
func (this *ControlPlaneDao) validateServicesForStarting(service service.Service, _ *struct{}) error {
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

// used in the walkServices function
type visit func(service service.Service) error

// assign an IP address to a service (and all its child services) containing non default AddressResourceConfig
func (this *ControlPlaneDao) AssignIPs(assignmentRequest dao.AssignmentRequest, _ *struct{}) error {
	myService := service.Service{}
	err := this.GetService(assignmentRequest.ServiceId, &myService)
	if err != nil {
		return err
	}

	// populate poolsIpInfo
	poolIPs, err := this.facade.GetPoolIPs(datastore.Get(), myService.PoolId)
	if err != nil {
		glog.Errorf("GetPoolsIPInfo failed: %v", err)
		return err
	}
	poolsIpInfo := poolIPs.HostIPs
	if len(poolsIpInfo) < 1 {
		msg := fmt.Sprintf("No IP addresses are available in pool %s.", myService.PoolId)
		return errors.New(msg)
	}
	glog.Infof("Pool %v contains %v available IP(s)", myService.PoolId, len(poolsIpInfo))

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
			msg := fmt.Sprintf("The requested IP address: %s is not contained in pool %s.", assignmentRequest.IpAddress, myService.PoolId)
			return errors.New(msg)
		}
	}
	assignmentRequest.IpAddress = poolsIpInfo[ipIndex].IPAddress
	selectedHostId := poolsIpInfo[ipIndex].HostID
	glog.Infof("Attempting to set IP address(es) to %s", assignmentRequest.IpAddress)

	assignments := []service.AddressAssignment{}
	this.GetServiceAddressAssignments(assignmentRequest.ServiceId, &assignments)
	if err != nil {
		glog.Errorf("controlPlaneDao.GetServiceAddressAssignments failed in anonymous function: %v", err)
		return err
	}

	visitor := func(myService service.Service) error {
		// if this service is in need of an IP address, assign it an IP address
		for _, endpoint := range myService.Endpoints {
			needsAnAddressAssignment, addressAssignmentId, err := this.needsAddressAssignment(myService.Id, endpoint)
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
				assignment := service.AddressAssignment{}
				assignment.AssignmentType = "static"
				assignment.HostID = selectedHostId
				assignment.PoolID = myService.PoolId
				assignment.IPAddr = assignmentRequest.IpAddress
				assignment.Port = endpoint.AddressConfig.Port
				assignment.ServiceID = myService.Id
				assignment.EndpointName = endpoint.Name
				glog.Infof("Creating AddressAssignment for Endpoint: %s", assignment.EndpointName)

				var unusedStr string
				err = this.AssignAddress(assignment, &unusedStr)
				if err != nil {
					glog.Errorf("AssignAddress failed in AssignIPs anonymous function: %v", err)
					return err
				}
				glog.Infof("Created AddressAssignment: %s for Endpoint: %s", assignment.ID, assignment.EndpointName)
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
	visitor := func(service service.Service) error {
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

	visitor := func(service service.Service) error {
		//start this service
		service.DesiredState = dao.SVC_RUN
		err = this.updateService(&service)
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
	service := service.Service{}
	err := this.GetService(serviceId, &service)
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

func (this *ControlPlaneDao) GetServiceState(request dao.ServiceStateRequest, serviceState *servicestate.ServiceState) error {
	glog.V(3).Infof("ControlPlaneDao.GetServiceState: request=%v", request)
	return this.zkDao.GetServiceState(serviceState, request.ServiceId, request.ServiceStateId)
}

func (this *ControlPlaneDao) GetRunningService(request dao.ServiceStateRequest, running *dao.RunningService) error {
	glog.V(3).Infof("ControlPlaneDao.GetRunningService: request=%v", request)
	return this.zkDao.GetRunningService(request.ServiceId, request.ServiceStateId, running)
}

func (this *ControlPlaneDao) GetServiceStates(serviceId string, serviceStates *[]*servicestate.ServiceState) error {
	glog.V(2).Infof("ControlPlaneDao.GetServiceStates: serviceId=%s", serviceId)
	return this.zkDao.GetServiceStates(serviceStates, serviceId)
}

/* This method assumes that if a service instance exists, it has not yet been terminated */
func (this *ControlPlaneDao) getNonTerminatedServiceStates(serviceId string, serviceStates *[]*servicestate.ServiceState) error {
	glog.V(2).Infof("ControlPlaneDao.getNonTerminatedServiceStates: serviceId=%s", serviceId)
	return this.zkDao.GetServiceStates(serviceStates, serviceId)
}

// Update the current state of a service instance.
func (this *ControlPlaneDao) UpdateServiceState(state servicestate.ServiceState, unused *int) error {
	glog.V(2).Infoln("ControlPlaneDao.UpdateServiceState state=%+v", state)
	return this.zkDao.UpdateServiceState(&state)
}

func (this *ControlPlaneDao) RestartService(serviceId string, unused *int) error {
	return dao.ControlPlaneError{"Unimplemented"}
}

func (this *ControlPlaneDao) StopService(id string, unused *int) error {
	glog.V(0).Info("ControlPlaneDao.StopService id=", id)
	var service service.Service
	err := this.GetService(id, &service)
	if err != nil {
		return err
	}
	service.DesiredState = dao.SVC_STOP
	err = this.updateService(&service)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("ParentServiceId:%s AND NOT Launch:manual", id)
	subservices, err := this.queryServices(query, "100")
	if err != nil {
		return err
	}
	for _, service := range subservices {
		subServiceErr := this.StopService(service.Id, unused)
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

func (this *ControlPlaneDao) StopRunningInstance(request dao.HostServiceRequest, unused *int) error {
	return this.zkDao.TerminateHostService(request.HostId, request.ServiceStateId)
}

func (this *ControlPlaneDao) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantId *string) error {
	store := servicetemplate.NewStore()
	template, err := store.Get(datastore.Get(), request.TemplateId)
	if err != nil {
		glog.Errorf("unable to load template: %s", request.TemplateId)
		return err
	}

	pool, err := this.facade.GetResourcePool(datastore.Get(), request.PoolId)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %s", request.PoolId)
		return err
	}
	if pool == nil {
		return fmt.Errorf("poolid %s not found", request.PoolId)
	}

	volumes := make(map[string]string)
	return this.deployServiceDefinitions(template.Services, request.TemplateId, request.PoolId, "", volumes, request.DeploymentId, tenantId)
}

func getSubServiceImageIds(ids map[string]struct{}, svc servicedefinition.ServiceDefinition) {
	found := struct{}{}

	if len(svc.ImageID) != 0 {
		ids[svc.ImageID] = found
	}
	for _, s := range svc.Services {
		getSubServiceImageIds(ids, s)
	}
}

func (this *ControlPlaneDao) deployServiceDefinitions(sds []servicedefinition.ServiceDefinition, template string, pool string, parentServiceId string, volumes map[string]string, deploymentId string, tenantId *string) error {
	// ensure that all images in the templates exist
	imageIds := make(map[string]struct{})
	for _, svc := range sds {
		getSubServiceImageIds(imageIds, svc)
	}

	dockerclient, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Errorf("unable to start docker client")
		return err
	}

	for imageId, _ := range imageIds {
		_, err := dockerclient.InspectImage(imageId)
		if err != nil {
			glog.Errorf("could not look up image: %s", imageId)
			return err
		}
	}

	for _, sd := range sds {
		if err := this.deployServiceDefinition(sd, template, pool, parentServiceId, volumes, deploymentId, tenantId); err != nil {
			return err
		}
	}
	return nil
}

func (this *ControlPlaneDao) deployServiceDefinition(sd servicedefinition.ServiceDefinition, template string, pool string, parentServiceId string, volumes map[string]string, deploymentId string, tenantId *string) error {
	// Always deploy in stopped state, starting is a separate step
	ds := dao.SVC_STOP

	exportedVolumes := make(map[string]string)
	for k, v := range volumes {
		exportedVolumes[k] = v
	}
	svc, err := service.BuildService(sd, parentServiceId, pool, ds, deploymentId)
	if err != nil {
		return err
	}

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
			glog.Errorf("could not look up image: %s", sd.ImageID)
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
	err = this.AddService(*svc, &serviceId)
	if err != nil {
		return err
	}

	return this.deployServiceDefinitions(sd.Services, template, pool, svc.Id, exportedVolumes, deploymentId, tenantId)
}

func (this *ControlPlaneDao) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateId *string) error {
	uuid, err := dao.NewUuid()
	if err != nil {
		return err
	}
	serviceTemplate.ID = uuid

	store := servicetemplate.NewStore()
	if err = store.Put(datastore.Get(), serviceTemplate); err != nil {
		return err
	}

	*templateId = uuid
	// this takes a while so don't block the main thread
	go this.reloadLogstashContainer()
	return err
}

func (this *ControlPlaneDao) UpdateServiceTemplate(template servicetemplate.ServiceTemplate, unused *int) error {
	store := servicetemplate.NewStore()
	if err := store.Put(datastore.Get(), template); err != nil {
		return err
	}
	go this.reloadLogstashContainer() // don't block the main thread
	return nil
}

func (this *ControlPlaneDao) RemoveServiceTemplate(id string, unused *int) error {
	// make sure it is a valid template first
	store := servicetemplate.NewStore()

	_, err := store.Get(datastore.Get(), id)
	if err != nil {
		return fmt.Errorf("Unable to find template: %s", id)
	}

	glog.V(2).Infof("ControlPlaneDao.RemoveServiceTemplate: %s", id)
	if err != store.Delete(datastore.Get(), id) {
		return err
	}
	go this.reloadLogstashContainer()
	return nil
}

// RemoveAddressAssignemnt Removes an AddressAssignment by id
func (this *ControlPlaneDao) RemoveAddressAssignment(id string, _ *struct{}) error {
	aas, err := this.queryAddressAssignments(fmt.Sprintf("ID:%s", id))
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
func (this *ControlPlaneDao) AssignAddress(assignment service.AddressAssignment, id *string) error {
	err := assignment.Validate()
	if err != nil {
		return err
	}

	switch assignment.AssignmentType {
	case "static":
		{
			//check host and IP exist
			if err = this.validStaticIp(assignment.HostID, assignment.IPAddr); err != nil {
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
	if err = this.validEndpoint(assignment.ServiceID, assignment.EndpointName); err != nil {
		return err
	}

	//check for existing assignments to this endpoint
	existing, err := this.getEndpointAddressAssignments(assignment.ServiceID, assignment.EndpointName)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("Address Assignment already exists")
	}
	assignment.ID, err = dao.NewUuid()
	if err != nil {
		return err
	}
	_, err = newAddressAssignment(assignment.ID, &assignment)
	if err != nil {
		return err
	}
	*id = assignment.ID
	return nil
}

func (this *ControlPlaneDao) validStaticIp(hostId string, ipAddr string) error {

	host, err := this.facade.GetHost(datastore.Get(), hostId)
	if err != nil {
		return err
	}
	if host == nil {
		return fmt.Errorf("host not found: %v", hostId)
	}
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
func (this *ControlPlaneDao) GetServiceAddressAssignments(serviceId string, assignments *[]service.AddressAssignment) error {
	query := fmt.Sprintf("ServiceID:%s", serviceId)
	results, err := this.queryAddressAssignments(query)
	if err != nil {
		return err
	}
	*assignments = *results
	return nil
}

// queryAddressAssignments query for host ips; returns empty array if no results for query
func (this *ControlPlaneDao) queryAddressAssignments(query string) (*[]service.AddressAssignment, error) {
	result, err := searchAddressAssignment(query)
	if err != nil {
		return nil, err
	}
	return toAddressAssignments(&result)
}

// getEndpointAddressAssignments returns the AddressAssignment for the service and endpoint, if no assignments the AddressAssignment will be nil
func (this *ControlPlaneDao) getEndpointAddressAssignments(serviceId string, endpointName string) (*service.AddressAssignment, error) {
	//TODO: this can probably be done w/ a query
	assignments := []service.AddressAssignment{}
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

func (this *ControlPlaneDao) GetServiceTemplates(unused int, templates *map[string]*servicetemplate.ServiceTemplate) error {
	glog.V(2).Infof("ControlPlaneDao.GetServiceTemplates")
	store := servicetemplate.NewStore()
	results, err := store.GetServiceTemplates(datastore.Get())
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceTemplates: err=%s", err)
		return err
	}
	templatemap := make(map[string]*servicetemplate.ServiceTemplate)
	for _, st := range results {
		templatemap[st.ID] = st
	}
	*templates = templatemap
	return nil
}

func (this *ControlPlaneDao) StartShell(service service.Service, unused *int) error {
	// TODO: implement stub
	return nil
}

func (this *ControlPlaneDao) ExecuteShell(service service.Service, command *string) error {
	// TODO: implement stub
	return nil
}

func (this *ControlPlaneDao) ShowCommands(service service.Service, unused *int) error {
	// TODO: implement stub
	return nil
}

func (this *ControlPlaneDao) DeleteSnapshot(snapshotId string, unused *int) error {
	return this.dfs.DeleteSnapshot(snapshotId)
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

func (this *ControlPlaneDao) Rollback(snapshotId string, unused *int) error {
	return this.dfs.Rollback(snapshotId)
}

// Takes a snapshot of the DFS via the host
func (this *ControlPlaneDao) LocalSnapshot(serviceId string, label *string) error {
	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.Errorf("ControlPlaneDao.LocalSnapshot err=%s", err)
		return err
	}

	if id, err := this.dfs.Snapshot(tenantId); err != nil {
		glog.Errorf("ControlPlaneDao.LocalSnapshot err=%s", err)
		return err
	} else {
		*label = id
	}

	return nil
}

// Snapshot is called via RPC by the CLI to take a snapshot for a serviceId
func (this *ControlPlaneDao) Snapshot(serviceId string, label *string) error {
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

	glog.Infof("added snapshot request: %+v", snapshotRequest)

	// wait for completion of snapshot request - check only once a second
	// BEWARE: this.zkDao.LoadSnapshotRequestW does not block like it should
	//         thus cannot use idiomatic select on eventChan and time.After() channels
	timeOutValue := time.Second * 60
	endTime := time.Now().Add(timeOutValue)
	for time.Now().Before(endTime) {
		glog.V(2).Infof("watching for snapshot completion for request: %+v", snapshotRequest)
		_, err := this.zkDao.LoadSnapshotRequestW(snapshotRequest.Id, snapshotRequest)
		switch {
		case err != nil:
			glog.Infof("failed snapshot request: %+v  error: %s", snapshotRequest, err)
			return err
		case snapshotRequest.SnapshotError != "":
			glog.Infof("failed snapshot request: %+v  error: %s", snapshotRequest, snapshotRequest.SnapshotError)
			return errors.New(snapshotRequest.SnapshotError)
		case snapshotRequest.SnapshotLabel != "":
			*label = snapshotRequest.SnapshotLabel
			glog.Infof("completed snapshot request: %+v  label: %s", snapshotRequest, *label)
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	err = fmt.Errorf("timed out waiting %v for snapshot: %+v", timeOutValue, snapshotRequest)
	glog.Error(err)
	return err
}

func (this *ControlPlaneDao) GetVolume(serviceId string, theVolume *volume.Volume) error {
	var tenantId string
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v err=%s", serviceId, err)
		return err
	}
	glog.V(3).Infof("ControlPlaneDao.GetVolume service=%+v tenantId=%s", serviceId, tenantId)
	var service service.Service
	if err := this.GetService(tenantId, &service); err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetVolume service=%+v err=%s", serviceId, err)
		return err
	}
	glog.V(3).Infof("ControlPlaneDao.GetVolume service=%+v poolId=%s", service, service.PoolId)

	aVolume, err := getSubvolume(this.vfs, service.PoolId, tenantId)
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
	if err := this.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("ControlPlaneDao.Snapshots service=%+v err=%s", serviceId, err)
		return err
	}
	var service service.Service
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
			*labels = snaplabels
		}
	}
	return nil
}

func (this *ControlPlaneDao) Get(service service.Service, file *string) error {
	// TODO: implement stub
	return nil
}

func (this *ControlPlaneDao) Send(service service.Service, files *[]string) error {
	// TODO: implment stub
	return nil
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int, facade *facade.Facade) (*ControlPlaneDao, error) {
	glog.V(0).Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
	api.Domain = hostName
	api.Port = strconv.Itoa(port)

	dao := &ControlPlaneDao{
		hostName: hostName,
		port:     port,
	}
	if dfs, err := dfs.NewDistributedFileSystem(dao, facade); err != nil {
		return nil, err
	} else {
		dao.dfs = dfs
	}

	return dao, nil
}

//createSystemUser updates the running instance password as well as the user record in elastic
func createSystemUser(s *ControlPlaneDao) error {
	user := dao.User{}
	err := s.GetUser(SYSTEM_USER_NAME, &user)
	if err != nil {
		glog.Warningf("%s", err)
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

func NewControlSvc(hostName string, port int, facade *facade.Facade, zclient *coordclient.Client, varpath, vfs string) (*ControlPlaneDao, error) {
	glog.V(2).Info("calling NewControlSvc()")
	defer glog.V(2).Info("leaving NewControlSvc()")

	s, err := NewControlPlaneDao(hostName, port, facade)
	if err != nil {
		return nil, err
	}

	//Used to bridge old to new
	s.facade = facade

	s.varpath = varpath
	s.vfs = vfs

	s.zclient = zclient
	s.zkDao = zzk.NewZkDao(zclient)

	// create the account credentials
	if err = createSystemUser(s); err != nil {
		return nil, err
	}

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
	var templatesMap map[string]*servicetemplate.ServiceTemplate
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
func (s *ControlPlaneDao) reloadLogstashContainer() error {
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
