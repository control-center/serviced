package dao

import (
	"time"

	"github.com/zenoss/serviced/utils"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain"
)

type User struct {
	Name     string // the unique identifier for a user
	Password string // no requirements on passwords yet
}

// An association between a host and a pool.
type PoolHost struct {
	HostID string
	PoolID string
	HostIP string
}

//AssignmentRequest is used to couple a serviceId to an IPAddress
type AssignmentRequest struct {
	ServiceID      string
	IPAddress      string
	AutoAssignment bool
}

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceID      string
	ContainerPort  uint16
	HostPort       uint16
	HostIP         string
	ContainerIP    string
	Protocol       string
	VirtualAddress string
}

// A request to deploy a service template
type ServiceTemplateDeploymentRequest struct {
	PoolID       string // Pool Id to deploy service into
	TemplateID   string // Id of template to be deployed
	DeploymentID string // Unique id of the instance of this template
}

// A request to deploy a service from a service definition
//  Pool and deployment ids are derived from the parent
type ServiceDeploymentRequest struct {
	ParentID     string	// ID of parent service
	Service      servicedefinition.ServiceDefinition;
}

// This is created by selecting from service_state and joining to service
type RunningService struct {
	Id              string
	ServiceID       string
	HostID          string
	DockerID        string
	StartedAt       time.Time
	Name            string
	Startup         string
	Description     string
	Instances       int
	ImageID         string
	PoolID          string
	DesiredState    int
	ParentServiceID string
	InstanceID      int
	MonitoringProfile domain.MonitorProfile
}

// An instantiation of a Snapshot request
type SnapshotRequest struct {
	Id            string
	ServiceID     string
	SnapshotLabel string
	SnapshotError string
}

// A new snapshot request instance (SnapshotRequest)
func NewSnapshotRequest(serviceId string, snapshotLabel string) (snapshotRequest *SnapshotRequest, err error) {
	snapshotRequest = &SnapshotRequest{}
	snapshotRequest.Id, err = utils.NewUUID36()
	if err == nil {
		snapshotRequest.ServiceID = serviceId
		snapshotRequest.SnapshotLabel = snapshotLabel
		snapshotRequest.SnapshotError = ""
	}
	return snapshotRequest, err
}
