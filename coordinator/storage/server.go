package storage

import (
	"fmt"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/domain/host"
)

type Server struct {
	host      *host.Host
	zclient *client.Client
	closing   chan struct{}
	driver StorageDriver
	debug     chan string
}

type StorageDriver interface {
	ExportPath() string
	SetClients(clients ...string)
	Sync() error
}

func NewServer(driver StorageDriver, host *host.Host, zclient *client.Client) (*Server, error) {

	if len(driver.ExportPath()) < 9 {
		return nil, fmt.Errorf("export path can not be empty")
	}

	s := &Server{
		host:      host,
		zclient:      zclient,
		closing:   make(chan struct{}),
		driver: driver,
		debug:     make(chan string),
	}

	go s.loop()
	return s, nil
}

func (s *Server) Close() {
	close(s.closing)
}

func (s *Server) loop() {

	var err error
	var leadEventC <-chan client.Event
	var e <-chan client.Event

	conn, _ := s.zclient.GetConnection()
	children := make([]string, 0)
	node := &Node{
		Host:       *s.host,
		ExportPath: s.driver.ExportPath(),
		version:    nil,
	}

	glog.Info("creating leader")
	storageLead := conn.NewLeader("/storage/leader", node)
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

		if err = conn.CreateDir("/storage/clients"); err != nil && err != client.ErrNodeExists {
			glog.Errorf("err creating /storage/clients: %s", err)
			continue
		}

		leadEventC, err = storageLead.TakeLead()
		if err != nil && err != zookeeper.ErrDeadlock {
			glog.Errorf("err taking lead: %s", err)
			continue
		}

		children, e, err = conn.ChildrenW("/storage/clients")
		if err != nil {
			continue
		}

		s.driver.SetClients(children...)
		if err = s.driver.Sync(); err != nil {
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
