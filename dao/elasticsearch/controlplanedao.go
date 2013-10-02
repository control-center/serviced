package elasticsearch

import "github.com/zenoss/serviced/dao"
import "github.com/mattbaird/elastigo/api"
import "github.com/mattbaird/elastigo/core"
import "github.com/zenoss/glog"
import "strconv"
import "strings"
import "errors"
import "encoding/json"

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
func index(pretty *bool, index string, _type string) func( string, interface{}) (api.BaseResponse, error)  {
  return func( id string, data interface{}) (api.BaseResponse, error) {
    return core.Index( *pretty, index, _type, id, data);
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
  resourcePoolExists func (string) (bool, error) = exists( &Pretty, "controlplane", "resourcepool")

  //model index functions
  indexHost         func(string, interface{}) (api.BaseResponse, error) = index( &Pretty, "controlplane", "host")
  indexService      func(string, interface{}) (api.BaseResponse, error) = index( &Pretty, "controlplane", "service")
  indexResourcePool func(string, interface{}) (api.BaseResponse, error) = index( &Pretty, "controlplane", "resourcepool")

  //model delete functions
  deleteHost         func(string) (api.BaseResponse, error) = _delete( &Pretty, "controlplane", "host")
  deleteService      func(string) (api.BaseResponse, error) = _delete( &Pretty, "controlplane", "service")
  deleteResourcePool func(string) (api.BaseResponse, error) = _delete( &Pretty, "controlplane", "resourcepool")

  //model get functions
  getHost         func(string, interface{}) error = getSource( "controlplane", "host")
  getService      func(string, interface{}) error = getSource( "controlplane", "service")
  getResourcePool func(string, interface{}) error = getSource( "controlplane", "resourcepool")

  //model search functions
  searchHost         func(string) (core.SearchResult, error) = search( "controlplane", "host")
  searchService      func(string) (core.SearchResult, error) = search( "controlplane", "service")
  searchResourcePool func(string) (core.SearchResult, error) = search( "controlplane", "resourcepool")
)

type ControlPlaneDao struct {
  hostName string
  port int
}

//
func (this *ControlPlaneDao) NewResourcePool(resourcePool *dao.ResourcePool) error {
  glog.Infof( "ControlPlaneDao.NewResourcePool: %+v", *resourcePool)

  id := strings.TrimSpace( resourcePool.Id)
  if id == "" {
    return errors.New( "empty ResourcePool.Id not allowed")
  }

  //XXX elasticgo core.Exists function is broken
  //exists, _ := resourcePoolExists(id)
  //if exists {
  //  message := fmt.Sprintf( "ResourcePool with Id=%s already exists", id)
  //  return resourcePool, errors.New( message)
  //}

  resourcePool.Id = id
  response, err := indexResourcePool( id, resourcePool)
  glog.Infof( "ControlPlaneDao.NewResourcePool response: %+v", response)
  if response.Ok {
    return nil
  } else {
    return err
  }
}

//
func (this *ControlPlaneDao) NewHost(host *dao.Host) error {
  glog.Infof( "ControlPlaneDao.NewHost: %+v", *host)

  id := strings.TrimSpace( host.Id)
  if id == "" {
    return errors.New( "empty Host.Id not allowed")
  }

  //XXX elasticgo core.Exists function is broken
  //exists, _ := hostExists(id)
  //if exists {
  //  message := fmt.Sprintf( "Host with Id=%s already exists", id)
  //  return nil, errors.New( message)
  //}

  host.Id = id
  response, err := indexHost( id, host)
  glog.Infof( "ControlPlaneDao.NewHost response: %+v", response)
  if response.Ok {
    return nil
  } else {
    return err
  }
}

//
func (this *ControlPlaneDao) NewService(service *dao.Service) error {
  glog.Infof( "ControlPlaneDao.NewService: %+v", *service)
  id := strings.TrimSpace( service.Id)
  if id == "" {
    return errors.New( "empty Service.Id not allowed")
  }

  //XXX elasticgo core.Exists function is broken
  //exists, _ := serviceExists(id)
  //if exists {
  //  message := fmt.Sprintf( "Service with Id=%s already exists", id)
  //  return nil, errors.New( message)
  //}

  service.Id = id
  response, err := indexService( id, service)
  glog.Infof( "ControlPlaneDao.NewService response: %+v", response)
  if response.Ok {
    return nil
  } else {
    return err
  }
}

//
func (this *ControlPlaneDao) UpdateResourcePool(resourcePool *dao.ResourcePool) error {
  glog.Infof( "ControlPlaneDao.UpdateResourcePool: %+v", *resourcePool)

  id := strings.TrimSpace( resourcePool.Id)
  if id == "" {
    return errors.New( "empty ResourcePool.Id not allowed")
  }

  //XXX elasticgo core.Exists function is broken
  //exists, _ := resourcePoolExists(id)
  //if exists {
  //  message := fmt.Sprintf( "ResourcePool with Id=%s already exists", id)
  //  return resourcePool, errors.New( message)
  //}

  resourcePool.Id = id
  response, err := indexResourcePool( id, resourcePool)
  glog.Infof( "ControlPlaneDao.UpdateResourcePool response: %+v", response)
  if response.Ok {
    return nil
  } else {
    return err
  }
}

//
func (this *ControlPlaneDao) UpdateHost(host *dao.Host) error {
  glog.Infof( "ControlPlaneDao.UpdateHost: %+v", *host)

  id := strings.TrimSpace( host.Id)
  if id == "" {
    return errors.New( "empty Host.Id not allowed")
  }

  //XXX elasticgo core.Exists function is broken
  //exists, _ := hostExists(id)
  //if exists {
  //  message := fmt.Sprintf( "Host with Id=%s already exists", id)
  //  return nil, errors.New( message)
  //}

  host.Id = id
  response, err := indexHost( id, host)
  glog.Infof( "ControlPlaneDao.UpdateHost response: %+v", response)
  if response.Ok {
    return nil
  } else {
    return err
  }
}

//
func (this *ControlPlaneDao) UpdateService(service *dao.Service) error {
  glog.Infof( "ControlPlaneDao.UpdateService: %+v", *service)
  id := strings.TrimSpace( service.Id)
  if id == "" {
    return errors.New( "empty Service.Id not allowed")
  }

  //XXX elasticgo core.Exists function is broken
  //exists, _ := serviceExists(id)
  //if exists {
  //  message := fmt.Sprintf( "Service with Id=%s already exists", id)
  //  return nil, errors.New( message)
  //}

  service.Id = id
  response, err := indexService( id, service)
  glog.Infof( "ControlPlaneDao.UpdateService response: %+v", response)
  if response.Ok {
    return nil
  } else {
    return err
  }
}

//
func (this *ControlPlaneDao) DeleteResourcePool(id string) error {
  glog.Infof( "ControlPlaneDao.DeleteResourcePool: %s", id)
  response, err := deleteResourcePool( id)
  glog.Infof( "ControlPlaneDao.DeleteResourcePool response: %+v", response)
  return err
}

//
func (this *ControlPlaneDao) DeleteHost(id string) error {
  glog.Infof( "ControlPlaneDao.DeleteHost: %s", id)
  response, err := deleteHost( id)
  glog.Infof( "ControlPlaneDao.DeleteHost response: %+v", response)
  return err
}

//
func (this *ControlPlaneDao) DeleteService(id string) error {
  glog.Infof( "ControlPlaneDao.DeleteService: %s", id)
  response, err := deleteService( id)
  glog.Infof( "ControlPlaneDao.DeleteService response: %+v", response)
  return err
}

//
func (this *ControlPlaneDao) GetResourcePool(id string) (dao.ResourcePool, error) {
  glog.Infof( "ControlPlaneDao.GetResourcePool: id=%s", id)
  resourcePool := dao.ResourcePool{}
  err := getResourcePool( id, &resourcePool)
  glog.Infof( "ControlPlaneDao.GetResourcePool: id=%s, resourcepool=%+v, err=%s", id, resourcePool, err)
  return resourcePool, err
}

//
func (this *ControlPlaneDao) GetHost(id string) (dao.Host, error) {
  glog.Infof( "ControlPlaneDao.GetHost: id=%s", id)
  host := dao.Host{}
  err := getHost( id, &host)
  glog.Infof( "ControlPlaneDao.GetHost: id=%s, host=%+v, err=%s", id, host, err)
  return host, err
}

//
func (this *ControlPlaneDao) GetService(id string) (dao.Service, error) {
  glog.Infof( "ControlPlaneDao.GetService: id=%s", id)
  service := dao.Service{}
  err := getService( id, &service)
  glog.Infof( "ControlPlaneDao.GetService: id=%s, service=%+v, err=%s", id, service, err)
  return service, err
}

//
func (this *ControlPlaneDao) GetResourcePools() ([]dao.ResourcePool, error) {
  glog.Infof( "ControlPlaneDao.GetResourcePools")
  result, err := searchResourcePool( "_exists_:Id")
  glog.Infof( "ControlPlaneDao.GetResourcePools: err=%s", err)

  var resourcePools []dao.ResourcePool
  if err == nil {
    var total = len(result.Hits.Hits)
    resourcePools = make( []dao.ResourcePool, total)
    for i:=0; i<total; i+=1 {
      err := json.Unmarshal( result.Hits.Hits[i].Source, &resourcePools[i])
      if err != nil {
        return resourcePools, err
      }
    }
  }
  return resourcePools, err
}

//
func (this *ControlPlaneDao) GetHosts() ([]dao.Host, error) {
  glog.Infof( "ControlPlaneDao.GetHosts")
  result, err := searchHost( "_exists_:Id")
  glog.Infof( "ControlPlaneDao.GetHosts: err=%s", err)

  var hosts []dao.Host
  if err == nil {
    var total = len(result.Hits.Hits)
    hosts = make( []dao.Host, total)
    for i:=0; i<total; i+=1 {
      err := json.Unmarshal( result.Hits.Hits[i].Source, &hosts[i])
      if err != nil {
        return hosts, err
      }
    }
  }
  return hosts, err
}

//
func (this *ControlPlaneDao) GetServices() ([]dao.Service, error) {
  glog.Infof( "ControlPlaneDao.GetServices")
  result, err := searchService( "_exists_:Id")
  glog.Infof( "ControlPlaneDao.GetServices: err=%s", err)

  var services []dao.Service
  if err == nil {
    var total = len(result.Hits.Hits)
    services = make( []dao.Service, total)
    for i:=0; i<total; i+=1 {
      err := json.Unmarshal( result.Hits.Hits[i].Source, &services[i])
      if err != nil {
        return services, err
      }
    }
  }
  return services, err
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int) (*ControlPlaneDao, error) {
	glog.Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
  api.Domain = hostName
  api.Port = strconv.Itoa( port)
  return &ControlPlaneDao{ hostName, port}, nil
}
