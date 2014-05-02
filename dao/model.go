package dao

import (
	"time"
)

type User struct {
	Name     string // the unique identifier for a user
	Password string // no requirements on passwords yet
}

// An association between a host and a pool.
type PoolHost struct {
	HostId string
	PoolId string
	HostIp string
}

//AssignmentRequest is used to couple a serviceId to an IpAddress
type AssignmentRequest struct {
	ServiceId      string
	IpAddress      string
	AutoAssignment bool
}

// Desired states of services.
const (
	SVC_RUN     = 1
	SVC_STOP    = 0
	SVN_RESTART = -1
)

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceId     string
	ContainerPort uint16
	HostPort      uint16
	HostIp        string
	ContainerIp   string
	Protocol      string
}

////export definition
//type ServiceExport struct {
//	Protocol    string //tcp or udp
//	Application string //application type
//	Internal    string //internal port number
//	External    string //external port number
//}

// An instantiation of a Service.
//type ServiceState struct {
//	Id          string
//	ServiceId   string
//	HostId      string
//	DockerId    string
//	PrivateIp   string
//	Scheduled   time.Time
//	Terminated  time.Time
//	Started     time.Time
//	PortMapping map[string][]HostIpAndPort // protocol -> container port (internal) -> host port (external)
//	Endpoints   []servicedefinition.ServiceEndpoint
//	HostIp      string
//	InstanceId  int
//}
//
//type ServiceDeployment struct {
//	Id         string    // Primary key
//	TemplateId string    // id of template being deployed
//	ServiceId  string    // id of service created by deployment
//	DeployedAt time.Time // when the template was deployed
//}

// A request to deploy a service template
type ServiceTemplateDeploymentRequest struct {
	PoolId       string // Pool Id to deploy service into
	TemplateId   string // Id of template to be deployed
	DeploymentId string // Unique id of the instance of this template
}

// This is created by selecting from service_state and joining to service
type RunningService struct {
	Id              string
	ServiceId       string
	HostId          string
	DockerId        string
	StartedAt       time.Time
	Name            string
	Startup         string
	Description     string
	Instances       int
	ImageId         string
	PoolId          string
	DesiredState    int
	ParentServiceId string
	InstanceId      int
}

// An instantiation of a Snapshot request
type SnapshotRequest struct {
	Id            string
	ServiceId     string
	SnapshotLabel string
	SnapshotError string
}

// A new snapshot request instance (SnapshotRequest)
func NewSnapshotRequest(serviceId string, snapshotLabel string) (snapshotRequest *SnapshotRequest, err error) {
	snapshotRequest = &SnapshotRequest{}
	snapshotRequest.Id, err = NewUuid()
	if err == nil {
		snapshotRequest.ServiceId = serviceId
		snapshotRequest.SnapshotLabel = snapshotLabel
		snapshotRequest.SnapshotError = ""
	}
	return snapshotRequest, err
}
