package api

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/zenoss/serviced/dao"
)

var DefaultHostDaoTest = HostDaoTest{hosts: DefaultTestHosts}

var DefaultTestHosts = []dao.Host{
	{
		Id:             "test-host-id-1",
		PoolId:         "default",
		Name:           "alpha",
		IpAddr:         "127.0.0.1",
		Cores:          4,
		Memory:         4 * 1024 * 1024 * 1024,
		PrivateNetwork: "172.16.42.0/24",
	}, {
		Id:             "test-host-id-2",
		PoolId:         "default",
		Name:           "beta",
		IpAddr:         "192.168.0.1",
		Cores:          2,
		Memory:         512 * 1024 * 1024,
		PrivateNetwork: "10.0.0.1/66",
	}, {
		Id:             "test-host-id-3",
		PoolId:         "testpool",
		Name:           "gamma",
		IpAddr:         "0.0.0.0",
		Cores:          1,
		Memory:         1 * 1024 * 1024 * 1024,
		PrivateNetwork: "158.16.4.27/9090",
	},
}

type HostDaoTest struct {
	dao.ControlPlane
	hosts []dao.Host
}

func (t HostDaoTest) GetHosts(unused dao.EntityRequest, hosts *map[string]*dao.Host) error {
	if t.hosts == nil {
		return fmt.Errorf("not found")
	}

	var hostmap = make(map[string]*dao.Host)
	for _, h := range t.hosts {
		hostmap[h.Id] = &h
	}
	*hosts = hostmap

	return nil
}

func TestListHosts(t *testing.T) {
	a := &api{}
	a.client = DefaultHostDaoTest

	hosts, err := a.ListHosts()
	if err != nil {
		t.Fatalf("error calling api.ListHosts: %s", err)
	}

	var hostmap = make(map[string]*dao.Host)
	if err := a.client.GetHosts(&empty, &hostmap); err != nil {
		t.Fatalf("error calling api.client.GetHosts: %s", err)
	}

	for _, h := range hosts {
		if !reflect.DeepEqual(h, hostmap[h.Id]) {
			t.Errorf("cannot find host: %s", h.Id)
		}
	}
}

func BenchmarkListHosts(b *testing.B) {
	// Start Server
	// Call list hosts
}

func TestGetHost(t *testing.T) {
}

func BenchmarkGetHost(b *testing.B) {
}

func TestAddHost(t *testing.T) {
}

func BenchmarkAddHost(b *testing.B) {
}

func TestRemoveHost(t *testing.T) {
}

func BenchmarkRemoveHost(b *testing.B) {
}