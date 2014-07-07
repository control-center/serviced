package scheduler

import (
	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/zzk"

	"time"
)

type leaderFunc func(*facade.Facade, dao.ControlPlane, coordclient.Connection, <-chan coordclient.Event, string)

type scheduler struct {
	zkClient     *coordclient.Client // client from which connections can be created from
	cpDao        dao.ControlPlane    // ControlPlane interface
	cluster_path string              // path to the cluster node
	instance_id  string              // unique id for this node instance
	closing      chan chan error     // Sending a value on this channel notifies the schduler to shut down
	shutdown     chan error          // A error is placed on this channel when the scheduler shuts down
	started      bool                // is the loop running
	zkleaderFunc leaderFunc          // multiple implementations of leader function possible
	facade       *facade.Facade
}

func NewScheduler(cluster_path string, zkClient *coordclient.Client, instance_id string, cpDao dao.ControlPlane, facade *facade.Facade) (s *scheduler, shutdown <-chan error) {
	s = &scheduler{
		zkClient:     zkClient,
		cpDao:        cpDao,
		cluster_path: cluster_path,
		instance_id:  instance_id,
		closing:      make(chan chan error),
		shutdown:     make(chan error, 1),
		zkleaderFunc: Lead, // random scheduler implementation
		facade:       facade,
	}
	return s, s.shutdown
}

func (s *scheduler) Start() {
	if !s.started {
		s.started = true
		go s.loop()
	}
}

// Shut down node
func (s *scheduler) Stop() error {

	if !s.started {
		return nil
	}
	defer func() {
		s.started = false
	}()
	errc := make(chan error, 1)
	s.closing <- errc
	return <-errc
}

type hostNodeT struct {
	HostID  string
	version interface{}
}

func (h *hostNodeT) Version() interface{}           { return h.version }
func (h *hostNodeT) SetVersion(version interface{}) { h.version = version }

func (s *scheduler) loop() {
	glog.V(3).Infoln("entering scheduler")

	var err error
	//var this_node string
	defer func() {
		glog.V(3).Infoln("leaving scheduler")
		s.shutdown <- err
	}()

	var allPools []*pool.ResourcePool
	for {
		allPools, err = s.facade.GetResourcePools(datastore.Get())
		if err != nil {
			glog.Errorf("scheduler.go failed to get resource pools: %v", err)
			time.Sleep(time.Second * 5)
			continue
		} else if allPools == nil || len(allPools) == 0 {
			glog.Error("no resource pools found")
			time.Sleep(time.Second * 5)
			continue
		}
		break
	}

	for _, aPool := range allPools {
		poolBasedConn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(aPool.ID))
		if err != nil {
			glog.Error(err)
			return
		}

		hostNode := hostNodeT{HostID: s.instance_id}
		leader := poolBasedConn.NewLeader("/pools/"+aPool.ID+"/scheduler", &hostNode)
		events, err := leader.TakeLead()
		if err != nil {
			glog.Error("could not take lead: ", err)
			return
		}

		defer func() {
			leader.ReleaseLead()
		}()

		glog.Infof(" Creating a leader for pool: %v --- %+v", aPool.ID, poolBasedConn)
		s.zkleaderFunc(s.facade, s.cpDao, poolBasedConn, events, aPool.ID)
	}
}
