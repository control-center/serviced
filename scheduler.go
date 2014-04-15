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

func (s *scheduler) loop() {
	glog.V(3).Infoln("entering scheduler")

	var err error
	var this_node string
	defer func() {
		glog.V(3).Infoln("leaving scheduler")
		s.shutdown <- err
	}()

	/*
		// create scheduler node
		scheduler_path := s.cluster_path + "/election"
		err = zzk.CreateNode(scheduler_path, s.conn)
		if err != nil {
			glog.Error("could not create scheduler node: ", err)
			return
		}

		services_path := s.cluster_path + "/services"
		err = zzk.CreateNode(services_path, s.conn)
		if err != nil {
			glog.Error("could not create services node: ", err)
			return
		}

		// voter node path
		voter_path := scheduler_path + "/"
		instance_data := []byte(s.instance_id)
		err = zzk.DeleteNodebyData(scheduler_path, s.conn, instance_data)
		if err != nil {
			glog.Error("could not remove old over node: ", err)
			return
		}

		// create voting node
		this_node, err = s.conn.Create(
			voter_path, instance_data,
			zk.FlagEphemeral|zk.FlagSequence,
			zk.WorldACL(zk.PermAll))
		if err != nil {
			glog.Error("Could not create voting node:", err)
			return
		}
		glog.V(1).Infof("Created voting node: %s", this_node)

		for {
			s.conn.Sync(scheduler_path)
			// get children
			children, _, err := s.conn.Children(scheduler_path)
			if err != nil {
				glog.Error("Could not get children of schduler path")
				return
			}
			sort.Strings(children)

			leader_path := voter_path + children[0]
			if this_node == leader_path {
				glog.V(0).Info("I am the leader!")
				exists, _, event, err := s.conn.ExistsW(leader_path)
				if err != nil {
					if err == zk.ErrNoNode {
						continue
					}
					return
				}
				if !exists {
					continue
				}
				s.zkleaderFunc(s.cpDao, s.conn, event)
				return
			} else {
				glog.V(1).Infof("I must wait for %s to die.", children[0])

				exists, _, event, err := s.conn.ExistsW(leader_path)
				if err != nil && err != zk.ErrNoNode {
					return
				}
				if err == zk.ErrNoNode {
					continue
				}
				if err == nil && !exists {
					continue
				}
				select {
				case <-zzk.TimeoutAfter(time.Second * 30):
					glog.V(1).Info("I've been listening. I'm going to reinitialize.")
					continue
				case errc := <-s.closing:
					errc <- err
					return
				case evt := <-event:
					if evt.Type == zk.EventNodeDeleted {
						continue
					}
					return
				}
			}
		}
	*/
}
