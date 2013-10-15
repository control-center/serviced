package elasticsearch

import "github.com/zenoss/serviced/dao"
import "github.com/mattbaird/elastigo/api"
import "github.com/mattbaird/elastigo/core"
import "github.com/zenoss/glog"
import "strconv"
import "strings"
import "errors"
import "encoding/json"
import "fmt"

//assert interface
var _ dao.ControlPlane = &ControlPlaneDao{}

// closure for geting a model
func getSource(index string, _type string) func( string, interface{}) error  {
  return func( id string, source interface{}) error {
    return core.GetSource(index, _type, id, &source)
  }
}

// closure for searching a model
func search(index string, _type string) func( string) (core.SearchResult, error) {
  return func(query string) (core.SearchResult, error) {
    return core.SearchUri(index, _type, query, "", 0)
  }
}

// closure for testing model existence
func exists(pretty *bool, index string, _type string) func( string) (bool, error)  {
  return func( id string) (bool, error) {
    return core.Exists( *pretty, index, _type, id)
  }
}

// closure for indexing a model
func create(pretty *bool, index string, _type string) func( string, interface{}) (api.BaseResponse, error)  {
  var (
    parentId string = ""
    version int = 0
    op_type string = "create"
    routing string = ""
    timestamp string = ""
    ttl int = 0
    percolate string = ""
    timeout string = ""
    refresh bool = true
  )
  return func( id string, data interface{}) (api.BaseResponse, error) {
    return core.IndexWithParameters(
      *pretty, index, _type, id, parentId, version, op_type, routing, timestamp, ttl, percolate, timeout, refresh, data);
  }
}

// closure for indexing a model
func index(pretty *bool, index string, _type string) func( string, interface{}) (api.BaseResponse, error)  {
  var (
    parentId string = ""
    version int = 0
    op_type string = ""
    routing string = ""
    timestamp string = ""
    ttl int = 0
    percolate string = ""
    timeout string = ""
    refresh bool = true
  )
  return func( id string, data interface{}) (api.BaseResponse, error) {
    return core.IndexWithParameters(
      *pretty, index, _type, id, parentId, version, op_type, routing, timestamp, ttl, percolate, timeout, refresh, data);
  }
}

// closure for deleting a model
func _delete(pretty *bool, index string, _type string) func( string) (api.BaseResponse, error)  {
  return func( id string) (api.BaseResponse, error) {
    //version=-1 and routing="" are not supported as of 9/30/13
    return core.Delete( *pretty, index, _type, id, -1, "");
  }
}

var (
  //enable pretty printed responses
  Pretty bool = false

  //model existance functions
  hostExists         func (string) (bool, error) = exists( &Pretty, "controlplane", "host")
  serviceExists      func (string) (bool, error) = exists( &Pretty, "controlplane", "service")
  serviceStateExists func (string) (bool, error) = exists( &Pretty, "controlplane", "servicestate")
  resourcePoolExists func (string) (bool, error) = exists( &Pretty, "controlplane", "resourcepool")

  //model index functions
  newHost         func(string, interface{}) (api.BaseResponse, error) = create( &Pretty, "controlplane", "host")
  newService      func(string, interface{}) (api.BaseResponse, error) = create( &Pretty, "controlplane", "service")
  newServiceState func(string, interface{}) (api.BaseResponse, error) = create( &Pretty, "controlplane", "servicestate")
  newResourcePool func(string, interface{}) (api.BaseResponse, error) = create( &Pretty, "controlplane", "resourcepool")

  //model index functions
  indexHost         func(string, interface{}) (api.BaseResponse, error) = index( &Pretty, "controlplane", "host")
  indexService      func(string, interface{}) (api.BaseResponse, error) = index( &Pretty, "controlplane", "service")
  indexServiceState func(string, interface{}) (api.BaseResponse, error) = index( &Pretty, "controlplane", "servicestate")
  indexResourcePool func(string, interface{}) (api.BaseResponse, error) = index( &Pretty, "controlplane", "resourcepool")

  //model delete functions
  deleteHost         func(string) (api.BaseResponse, error) = _delete( &Pretty, "controlplane", "host")
  deleteService      func(string) (api.BaseResponse, error) = _delete( &Pretty, "controlplane", "service")
  deleteServiceState func(string) (api.BaseResponse, error) = _delete( &Pretty, "controlplane", "servicestate")
  deleteResourcePool func(string) (api.BaseResponse, error) = _delete( &Pretty, "controlplane", "resourcepool")

  //model get functions
  getHost         func(string, interface{}) error = getSource( "controlplane", "host")
  getService      func(string, interface{}) error = getSource( "controlplane", "service")
  getServiceState func(string, interface{}) error = getSource( "controlplane", "servicestate")
  getResourcePool func(string, interface{}) error = getSource( "controlplane", "resourcepool")

  //model search functions
  searchHost         func(string) (core.SearchResult, error) = search( "controlplane", "host")
  searchService      func(string) (core.SearchResult, error) = search( "controlplane", "service")
  searchServiceState func(string) (core.SearchResult, error) = search( "controlplane", "servicestate")
  searchResourcePool func(string) (core.SearchResult, error) = search( "controlplane", "resourcepool")
)

type ControlPlaneDao struct {
  hostName string
  port int
}

//
func (this *ControlPlaneDao) queryHosts(query string) ([]*dao.Host, error) {
  result, err := searchHost( query)

  var hosts []*dao.Host
  if err == nil {
    var total = len(result.Hits.Hits)
    hosts = make( []*dao.Host, total)
    for i:=0; i<total; i+=1 {
      var host dao.Host
      err = json.Unmarshal( result.Hits.Hits[i].Source, &host)
      if err == nil {
        hosts[i] = &host
      } else {
        return nil, err
      }
    }
  }

  return hosts, err
}

//
func (this *ControlPlaneDao) queryServices(query string) ([]*dao.Service, error) {
  result, err := searchService( query)

  var services []*dao.Service
  if err == nil {
    var total = len(result.Hits.Hits)
    services = make( []*dao.Service, total)
    for i:=0; i<total; i+=1 {
      service := dao.Service{}
      err = json.Unmarshal( result.Hits.Hits[i].Source, &service)
      if err == nil {
        services[i] = &service
      } else {
        return nil, err
      }
    }
  }

  return services, err
}

//
func (this *ControlPlaneDao) AddResourcePool(pool dao.ResourcePool, unused *int) error {
  glog.Infof( "ControlPlaneDao.NewResourcePool: %+v", pool)

  id := strings.TrimSpace( pool.Id)
  if id == "" {
    return errors.New( "empty ResourcePool.Id not allowed")
  }

  pool.Id = id
  response, err := newResourcePool( id, pool)
  glog.Infof( "ControlPlaneDao.NewResourcePool response: %+v", response)
  if response.Ok {
    return nil
  }
  return err
}

//
func (this *ControlPlaneDao) AddHost(host dao.Host, unused *int) error {
  glog.Infof( "ControlPlaneDao.AddHost: %+v", host)

  id := strings.TrimSpace( host.Id)
  if id == "" {
    return errors.New( "empty Host.Id not allowed")
  }

  host.Id = id
  response, err := newHost( id, host)
  glog.Infof( "ControlPlaneDao.AddHost response: %+v", response)
  if response.Ok {
    return nil
  }
  return err
}

//
func (this *ControlPlaneDao) AddService(service dao.Service, unused *int) error {
  glog.Infof( "ControlPlaneDao.AddService: %+v", service)
  id := strings.TrimSpace( service.Id)
  if id == "" {
    return errors.New( "empty Service.Id not allowed")
  }

  service.Id = id
  response, err := newService( id, service)
  glog.Infof( "ControlPlaneDao.AddService response: %+v", response)
  if response.Ok {
    return nil
  }
  return err
}

//
func (this *ControlPlaneDao) UpdateResourcePool(pool dao.ResourcePool, unused *int) error {
  glog.Infof( "ControlPlaneDao.UpdateResourcePool: %+v", pool)

  id := strings.TrimSpace( pool.Id)
  if id == "" {
    return errors.New( "empty ResourcePool.Id not allowed")
  }

  pool.Id = id
  response, err := indexResourcePool( id, pool)
  glog.Infof( "ControlPlaneDao.UpdateResourcePool response: %+v", response)
  if response.Ok {
    return nil
  }
  return err
}

//
func (this *ControlPlaneDao) UpdateHost(host dao.Host, unused *int) error {
  glog.Infof( "ControlPlaneDao.UpdateHost: %+v", host)

  id := strings.TrimSpace( host.Id)
  if id == "" {
    return errors.New( "empty Host.Id not allowed")
  }

  host.Id = id
  response, err := indexHost( id, host)
  glog.Infof( "ControlPlaneDao.UpdateHost response: %+v", response)
  if response.Ok {
    return nil
  }
  return err
}

//
func (this *ControlPlaneDao) UpdateService(service dao.Service, unused *int) error {
  glog.Infof( "ControlPlaneDao.UpdateService: %+v", service)
  id := strings.TrimSpace( service.Id)
  if id == "" {
    return errors.New( "empty Service.Id not allowed")
  }

  service.Id = id
  response, err := indexService( id, service)
  glog.Infof( "ControlPlaneDao.UpdateService response: %+v", response)
  if response.Ok {
    return nil
  }
  return err
}

//
func (this *ControlPlaneDao) RemoveResourcePool(id string, unused *int) error {
  glog.Infof( "ControlPlaneDao.RemoveResourcePool: %s", id)
  response, err := deleteResourcePool( id)
  glog.Infof( "ControlPlaneDao.RemoveResourcePool response: %+v", response)
  return err
}

//
func (this *ControlPlaneDao) RemoveHost(id string, unused *int) error {
  glog.Infof( "ControlPlaneDao.RemoveHost: %s", id)
  response, err := deleteHost( id)
  glog.Infof( "ControlPlaneDao.RemoveHost response: %+v", response)
  return err
}

//
func (this *ControlPlaneDao) RemoveService(id string, unused *int) error {
  glog.Infof( "ControlPlaneDao.RemoveService: %s", id)
  response, err := deleteService( id)
  glog.Infof( "ControlPlaneDao.RemoveService response: %+v", response)
  return err
}

//
func (this *ControlPlaneDao) GetResourcePool(id string, pool *dao.ResourcePool) error {
  glog.Infof( "ControlPlaneDao.GetResourcePool: id=%s", id)
  request := dao.ResourcePool{}
  err := getResourcePool( id, &request)
  glog.Infof( "ControlPlaneDao.GetResourcePool: id=%s, resourcepool=%+v, err=%s", id, request, err)
  *pool = request
  return err
}

//
func (this *ControlPlaneDao) GetHost(id string, host *dao.Host) error {
  glog.Infof( "ControlPlaneDao.GetHost: id=%s", id)
  request := dao.Host{}
  err := getHost( id, &request)
  glog.Infof( "ControlPlaneDao.GetHost: id=%s, host=%+v, err=%s", id, request, err)
  *host = request
  return err
}

//
func (this *ControlPlaneDao) GetService(id string, service *dao.Service) error {
  glog.Infof( "ControlPlaneDao.GetService: id=%s", id)
  request := dao.Service{}
  err := getService( id, &request)
  glog.Infof( "ControlPlaneDao.GetService: id=%s, service=%+v, err=%s", id, request, err)
  *service = request
  return err
}

//
func (this *ControlPlaneDao) GetResourcePools(request dao.EntityRequest, pools *map[string]*dao.ResourcePool) error {
  glog.Infof( "ControlPlaneDao.GetResourcePools")
  result, err := searchResourcePool( "_exists_:Id")
  glog.Infof( "ControlPlaneDao.GetResourcePools: err=%s", err)

  var resourcePools map[string]*dao.ResourcePool
  if err == nil {
    var total = len(result.Hits.Hits)
    var pool dao.ResourcePool
    resourcePools = make( map[string]*dao.ResourcePool)
    for i:=0; i<total; i+=1 {
      err := json.Unmarshal( result.Hits.Hits[i].Source, &pool)
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
  glog.Infof( "ControlPlaneDao.GetHosts")
  result, err := this.queryHosts( "_exists_:Id")
  glog.Infof( "ControlPlaneDao.GetHosts: err=%s", err)

  hostmap := make(map[string]*dao.Host)
  if err == nil {
    var total = len(result)
    for i:=0; i<total; i+=1 {
      host := result[i]
      hostmap[ host.Id] = host
    }
  }

  *hosts = hostmap
  return err
}

//
func (this *ControlPlaneDao) GetServices(request dao.EntityRequest, services *[]*dao.Service) error {
  glog.Infof( "ControlPlaneDao.GetServices")
  result, err := this.queryServices( "_exists_:Id")
  glog.Infof( "ControlPlaneDao.GetServices: err=%s", err)
  *services = result
  return err
}


//
func (this *ControlPlaneDao) GetHostsForResourcePool(poolId string, poolHosts *[]*dao.PoolHost) error {
  id := strings.TrimSpace( poolId)
  if id == "" {
    return errors.New( "Illegal poolId: empty poolId not allowed")
  }

  query := fmt.Sprintf( "PoolId:%s", id)
  result, err := this.queryHosts( query)

  if err == nil {
    var response []*dao.PoolHost = make( []*dao.PoolHost, len(result))
    for i:=0; i<len(result); i+=1 {
      poolHost := dao.PoolHost{ result[i].Id, result[i].PoolId}
      response[i] = &poolHost
    }

    *poolHosts = response
  }

  return err
}


func (this *ControlPlaneDao) StartService(serviceId string, unused *string) error {
  //get the original service
  service := dao.Service{}
  err := this.GetService( serviceId, &service)
  if err == nil {
    //start this service
    var unusedInt int
    service.DesiredState = dao.SVC_RUN
    err = this.UpdateService( service, &unusedInt)

    if err == nil {
      //start all child services 
      var query = fmt.Sprintf( "ParentServiceId:%s", serviceId)
      subServices, err := this.queryServices( query)
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

// Update the current state of a service instance.
func (this *ControlPlaneDao) UpdateServiceState(state dao.ServiceState, unused *int) error {
	glog.Infoln("ControlPlaneDao.UpdateServiceState state=%+v", state)
  response, err := indexServiceState( state.Id, &state)
  if response.Ok {
    return nil
  }
  return err
}

func (this *ControlPlaneDao) RestartService(serviceId string, unused *int) error {
  return dao.ControlPlaneError{"Unimplemented"}
}

func (this *ControlPlaneDao) StopService(serviceId string, unused *int) error {
  return dao.ControlPlaneError{"Unimplemented"}
}

func (this *ControlPlaneDao) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, unused *int) error {
	return fmt.Errorf("unimplemented DeployTemplate")
}

func (this *ControlPlaneDao) AddServiceTemplate(serviceTemplate dao.ServiceTemplate, unused *int) error {
	return fmt.Errorf("unimplemented AddServiceTemplate")
}

func (this *ControlPlaneDao) UpdateServiceTemplate(serviceTemplate dao.ServiceTemplate, unused *int) error {
	return fmt.Errorf("unimplemented UpdateServiceTemplate")
}

func (this *ControlPlaneDao) RemoveServiceTemplate(id string, unused *int) error {
	return fmt.Errorf("unimplemented RemoveServiceTemplate")
}

func (this *ControlPlaneDao) GetServiceTemplates(unused int, serviceTemplates *[]*dao.ServiceTemplate) error {
	return fmt.Errorf("unimplemented DeployTemplate")
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int) (*ControlPlaneDao, error) {
	glog.Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
  api.Domain = hostName
  api.Port = strconv.Itoa( port)
  return &ControlPlaneDao{ hostName, port}, nil
}
