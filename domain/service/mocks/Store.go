package mocks

import "github.com/control-center/serviced/domain/service"
import "github.com/stretchr/testify/mock"

import "github.com/control-center/serviced/datastore"

import "time"

type Store struct {
	mock.Mock
}

func (_m *Store) Put(ctx datastore.Context, svc *service.Service) error {
	ret := _m.Called(ctx, svc)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, *service.Service) error); ok {
		r0 = rf(ctx, svc)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) Get(ctx datastore.Context, id string) (*service.Service, error) {
	ret := _m.Called(ctx, id)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *service.Service); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) Delete(ctx datastore.Context, id string) error {
	ret := _m.Called(ctx, id)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string) error); ok {
		r0 = rf(ctx, id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) UpdateDesiredState(ctx datastore.Context, serviceID string, desiredState int) error {
	ret := _m.Called(ctx, serviceID, desiredState)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, int) error); ok {
		r0 = rf(ctx, serviceID, desiredState)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) UpdateCurrentState(ctx datastore.Context, serviceID string, currentState string) error {
	ret := _m.Called(ctx, serviceID, currentState)

	var r0 error
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) error); ok {
		r0 = rf(ctx, serviceID, currentState)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Store) GetServices(ctx datastore.Context) ([]service.Service, error) {
	ret := _m.Called(ctx)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context) []service.Service); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetUpdatedServices(ctx datastore.Context, since time.Duration) ([]service.Service, error) {
	ret := _m.Called(ctx, since)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, time.Duration) []service.Service); ok {
		r0 = rf(ctx, since)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, time.Duration) error); ok {
		r1 = rf(ctx, since)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetTaggedServices(ctx datastore.Context, tags ...string) ([]service.Service, error) {
	ret := _m.Called(ctx, tags)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, ...string) []service.Service); ok {
		r0 = rf(ctx, tags...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, ...string) error); ok {
		r1 = rf(ctx, tags...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetServicesByPool(ctx datastore.Context, poolID string) ([]service.Service, error) {
	ret := _m.Called(ctx, poolID)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.Service); ok {
		r0 = rf(ctx, poolID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, poolID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetServiceCountByImage(ctx datastore.Context, imageID string) (int, error) {
	ret := _m.Called(ctx, imageID)

	var r0 int
	if rf, ok := ret.Get(0).(func(datastore.Context, string) int); ok {
		r0 = rf(ctx, imageID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, imageID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]service.Service, error) {
	ret := _m.Called(ctx, deploymentID)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.Service); ok {
		r0 = rf(ctx, deploymentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, deploymentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetChildServices(ctx datastore.Context, parentID string) ([]service.Service, error) {
	ret := _m.Called(ctx, parentID)

	var r0 []service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string) []service.Service); ok {
		r0 = rf(ctx, parentID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, parentID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) FindChildService(ctx datastore.Context, deploymentID string, parentID string, serviceName string) (*service.Service, error) {
	ret := _m.Called(ctx, deploymentID, parentID, serviceName)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string, string) *service.Service); ok {
		r0 = rf(ctx, deploymentID, parentID, serviceName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string, string) error); ok {
		r1 = rf(ctx, deploymentID, parentID, serviceName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) FindTenantByDeploymentID(ctx datastore.Context, deploymentID string, name string) (*service.Service, error) {
	ret := _m.Called(ctx, deploymentID, name)

	var r0 *service.Service
	if rf, ok := ret.Get(0).(func(datastore.Context, string, string) *service.Service); ok {
		r0 = rf(ctx, deploymentID, name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, string) error); ok {
		r1 = rf(ctx, deploymentID, name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) Search(ctx datastore.Context, query service.Query) ([]service.ServiceDetails, error) {
	ret := _m.Called(ctx, query)

	var r0 []service.ServiceDetails
	if rf, ok := ret.Get(0).(func(datastore.Context, service.Query) []service.ServiceDetails); ok {
		r0 = rf(ctx, query)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, service.Query) error); ok {
		r1 = rf(ctx, query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *Store) GetServiceDetails(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 *service.ServiceDetails
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *service.ServiceDetails); ok {
		r0 = rf(ctx, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, serviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetServiceDetailsByParentID(ctx datastore.Context, parentID string, since time.Duration) ([]service.ServiceDetails, error) {
	ret := _m.Called(ctx, parentID, since)

	var r0 []service.ServiceDetails
	if rf, ok := ret.Get(0).(func(datastore.Context, string, time.Duration) []service.ServiceDetails); ok {
		r0 = rf(ctx, parentID, since)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, time.Duration) error); ok {
		r1 = rf(ctx, parentID, since)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetAllServiceHealth(ctx datastore.Context) ([]service.ServiceHealth, error) {
	ret := _m.Called(ctx)

	var r0 []service.ServiceHealth
	if rf, ok := ret.Get(0).(func(datastore.Context) []service.ServiceHealth); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ServiceHealth)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetServiceHealth(ctx datastore.Context, serviceID string) (*service.ServiceHealth, error) {
	ret := _m.Called(ctx, serviceID)

	var r0 *service.ServiceHealth
	if rf, ok := ret.Get(0).(func(datastore.Context, string) *service.ServiceHealth); ok {
		r0 = rf(ctx, serviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ServiceHealth)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string) error); ok {
		r1 = rf(ctx, serviceID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetAllPublicEndpoints(ctx datastore.Context) ([]service.PublicEndpoint, error) {
	ret := _m.Called(ctx)

	var r0 []service.PublicEndpoint
	if rf, ok := ret.Get(0).(func(datastore.Context) []service.PublicEndpoint); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.PublicEndpoint)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetAllExportedEndpoints(ctx datastore.Context) ([]service.ExportedEndpoint, error) {
	ret := _m.Called(ctx)

	var r0 []service.ExportedEndpoint
	if rf, ok := ret.Get(0).(func(datastore.Context) []service.ExportedEndpoint); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ExportedEndpoint)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetAllIPAssignments(ctx datastore.Context) ([]service.BaseIPAssignment, error) {
	ret := _m.Called(ctx)

	var r0 []service.BaseIPAssignment
	if rf, ok := ret.Get(0).(func(datastore.Context) []service.BaseIPAssignment); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.BaseIPAssignment)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Store) GetServiceDetailsByIDOrName(ctx datastore.Context, query string, noprefix bool) ([]service.ServiceDetails, error) {
	ret := _m.Called(ctx, query, noprefix)

	var r0 []service.ServiceDetails
	if rf, ok := ret.Get(0).(func(datastore.Context, string, bool) []service.ServiceDetails); ok {
		r0 = rf(ctx, query, noprefix)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]service.ServiceDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(datastore.Context, string, bool) error); ok {
		r1 = rf(ctx, query, noprefix)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
