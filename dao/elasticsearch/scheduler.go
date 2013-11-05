package elasticsearch

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/zzk"

	"sort"
	"time"
)

type scheduler struct {
	conn         *zk.Conn        // the zookeeper connection
	cluster_path string          // path to the cluster node
	instance_id  string          // unique id for this node instance
	closing      chan chan error // Sending a value on this channel notifies the schduler to shut down
	shutdown     chan error      // A error is placed on this channel when the scheduler shuts down
	started      bool            // is the loop running
	zkleaderFunc func(<-chan zk.Event)
}

func newScheduler(cluster_path string, conn *zk.Conn, instance_id string, zkleaderFunc func(<-chan zk.Event)) (s *scheduler, shutdown <-chan error) {
	s = &scheduler{
		conn:         conn,
		cluster_path: cluster_path,
		instance_id:  instance_id,
		closing:      make(chan chan error),
		shutdown:     make(chan error, 1),
		zkleaderFunc: zkleaderFunc,
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
	glog.Info("entering scheduler")
	defer glog.Info("leaving scheduler")
	var err error
	var this_node string
	defer func() {
		s.shutdown <- err
	}()

	// create scheduler node
	scheduler_path := s.cluster_path + "/election"
	err = zzk.CreateNode(scheduler_path, s.conn)
	if err != nil {
		glog.Error("could not create scheduler node: ", err)
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
	glog.Infof("Created voting node: %s", this_node)

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
			glog.Info("I am the leader!")
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
			s.zkleaderFunc(event)
			return
		} else {
			glog.Infof("I must wait for %s to die.", children[0])

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
			case <- zzk.TimeoutAfter(time.Second * 30):
				glog.Info("I've been listening. I'm going to reinit")
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
}
