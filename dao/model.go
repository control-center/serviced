package dao

// A collection of computing resources with optional quotas.
type ResourcePool struct {
	Id          string // Unique identifier for resource pool, eg "default"
	ParentId    string // The pool id of the parent pool, if this pool is embeded in another pool. An empty string means it is not embeded.
	Priority    int    // relative priority of resource pools, used for CPU priority
	CoreLimit   int    // Number of cores on the host available to serviced
	MemoryLimit uint64 // A quota on the amount (bytes) of RAM in the pool, 0 = unlimited
}

// A host that runs the control plane agent.
type Host struct {
	Id             string // Unique identifier, default to hostid
	PoolId         string // Pool that the Host belongs to
	Name           string // A label for the host, eg hostname, role
	IpAddr         string // The IP address the host can be reached at from a serviced master
	Cores          int    // Number of cores available to serviced
	Memory         uint64 // Amount of RAM (bytes) available to serviced
	PrivateNetwork string // The private network where containers run, eg 172.16.42.0/24
}

// A Service that can run in serviced.
type Service struct {
	Id              string //unique id for the service, eg ""
	Name            string //canonical name of the service
	Startup         string //?
	Description     string //description of the service
	Instances       int    //?
	ImageId         string //?
	PoolId          string //?
	ParentId        string //?
	//DesiredState    int
	//Endpoints       *[]ServiceEndpoint
}
