package dao

type ControlPlaneDao interface {

  NewHost( host *Host) error
  NewService( service *Service) error
  NewResourcePool( resourcePool *ResourcePool) error

  UpdateHost( host *Host) error
  UpdateService( service *Service) error
  UpdateResourcePool( resourcePool *ResourcePool) error

  DeleteHost(id string) error
  DeleteService(id string) error
  DeleteResourcePool(id string) error

  GetHost(id string) (Host, error)
  GetService(id string) (Service, error)
  GetResourcePool(id string) (ResourcePool, error)

  GetHosts() ([]Host, error)
  GetServices() ([]Service, error)
  GetResourcePools() ([]ResourcePool, error)
}
