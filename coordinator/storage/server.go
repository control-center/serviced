package storage

import (
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/domain/host"
)

type Server struct {
	host      *host.Host
	conn      client.Connection
	closing   chan struct{}
	nfsServer StorageServer
	debug     chan string
}

type StorageServer interface {
	SetClients(clients ...string)
	Sync() error
}

func NewServer(nfsServer StorageServer, host *host.Host, conn client.Connection) (*Server, error) {
	s := &Server{
		host:      host,
		conn:      conn,
		closing:   make(chan struct{}),
		nfsServer: nfsServer,
		debug:     make(chan string),
	}

	go s.loop()
	return s, nil
}

func (s *Server) Close() {
	close(s.closing)
	s.conn.Close()
}

func (s *Server) loop() {

	var err error
	var leadEventC <-chan client.Event
	var e <-chan client.Event
	children := make([]string, 0)
	node := &Node{
		Host:    *s.host,
		version: nil,
	}

	glog.Info("creating leader")
	storageLead := s.conn.NewLeader("/storage/leader", node)
	defer storageLead.ReleaseLead()
	for {
		glog.Info("looping")
		// keep from churning if we get errors
		if err != nil {
			select {
			case <-s.closing:
				return
			case <-time.After(time.Second * 10):
			}
		}
		err = nil

		if err = s.conn.CreateDir("/storage/clients"); err != nil && err != client.ErrNodeExists {
			glog.Errorf("err creating /storage/clients: %s", err)
			continue
		}

		leadEventC, err = storageLead.TakeLead()
		if err != nil && err != zookeeper.ErrDeadlock {
			glog.Errorf("err taking lead: %s", err)
			continue
		}

		children, e, err = s.conn.ChildrenW("/storage/clients")
		if err != nil {
			continue
		}

		s.nfsServer.SetClients(children...)
		if err = s.nfsServer.Sync(); err != nil {
			continue
		}

		select {
		case <-s.closing:
			glog.Info("storage.server: received closing event")
			return
		case event := <-e:
			glog.Info("storage.server: received event: %s", event)
			continue
		case event := <-leadEventC:
			glog.Info("storage.server: received event on lock: %s", event)
			storageLead.ReleaseLead()
			continue
		}

	}
}
