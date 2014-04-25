package serviced

import (
	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	//"github.com/zenoss/serviced/zzk"
)

type leaderFunc func(dao.ControlPlane, coordclient.Connection, <-chan coordclient.Event)

type scheduler struct {
	conn         coordclient.Connection // the coordination service client connection
	cpDao        dao.ControlPlane       // ControlPlane interface
	cluster_path string                 // path to the cluster node
	instance_id  string                 // unique id for this node instance
	closing      chan chan error        // Sending a value on this channel notifies the schduler to shut down
	shutdown     chan error             // A error is placed on this channel when the scheduler shuts down
	started      bool                   // is the loop running
	zkleaderFunc leaderFunc             // multiple implementations of leader function possible
}

func NewScheduler(cluster_path string, conn coordclient.Connection, instance_id string, cpDao dao.ControlPlane) (s *scheduler, shutdown <-chan error) {
	s = &scheduler{
		conn:         conn,
		cpDao:        cpDao,
		cluster_path: cluster_path,
		instance_id:  instance_id,
		closing:      make(chan chan error),
		shutdown:     make(chan error, 1),
		zkleaderFunc: Lead, // random scheduler implementation
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
	HostId  string
	version int32
}

func (h *hostNodeT) Version() int32           { return h.version }
func (h *hostNodeT) SetVersion(version int32) { h.version = version }

func (s *scheduler) loop() {
	glog.V(3).Infoln("entering scheduler")

	var err error
	//var this_node string
	defer func() {
		glog.V(3).Infoln("leaving scheduler")
		s.shutdown <- err
	}()

	hostNode := hostNodeT{HostId: s.instance_id}
	leader := s.conn.NewLeader("/scheduler", &hostNode)
	events, err := leader.TakeLead()
	if err != nil {
		glog.Error("could not take lead: ", err)
		return
	}
	defer leader.ReleaseLead()
	s.zkleaderFunc(s.cpDao, s.conn, events)
}

/*
func (s *scheduler) loop() {
	glog.V(3).Infoln("entering scheduler")

	var err error
	//var this_node string
	defer func() {
		glog.V(3).Infoln("leaving scheduler")
		s.shutdown <- err
	}()

	hostNode := hostNodeT{HostId: s.instance_id}
	leader := s.conn.NewLeader("/scheduler", &hostNode)

	leaderChan := make(chan interface{})
	go func() {
		events, err := leader.TakeLead()
		if err != nil {
			leaderChan <- err
		} else {
			leaderChan <- events
		}
	}()

	for {

		select {
		case t := <-leaderChan:
			err, ok := t.(error)
			if ok {
				glog.Error("could not take lead: ", err)
				return
			}
			events, ok := t.(chan client.Event)
			if ok {
				defer leader.ReleaseLead()
				s.zkleaderFunc(s.cpDao, s.conn, events)
				return
			} else {
				panic("should not get here")
			}

		case <-time.After(time.Second):
			// i am a follower
		}
	}
}
*/
