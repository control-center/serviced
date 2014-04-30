package dfs

import (
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/volume"

	"errors"
	"fmt"
	"os/user"
	"testing"
)

var (
	MockServices       []*dao.Service
	MockPauseResume    map[string]bool
	MockVolumeInstance MockVolume
)

type MockVolume struct {
	volume.Conn
	name string
}

func (v MockVolume) Name() string {
	return v.name
}

func (v MockVolume) Snapshot(label string) (err error) {
	if v.name == "success" {
		return
	}

	return errors.New("unable to snapshot volume")
}

type MockControlPlane struct {
	dao.ControlPlane
}

func (c *MockControlPlane) GetTenantId(serviceId string, tenantId *string) (err error) {
	switch serviceId {
	case "niltenant-snapshot":
		err = errors.New("no tenant id found")
	default:
		*tenantId = serviceId
	}
	return
}

func (c *MockControlPlane) GetService(serviceId string, service *dao.Service) (err error) {
	switch serviceId {
	case "nilservice-snapshot":
		err = errors.New("no service found for serviceId")
	default:
		service = new(dao.Service)
	}
	return
}

func (c *MockControlPlane) GetServices(request dao.EntityRequest, services *[]*dao.Service) (err error) {
	*services = MockServices
	if len(MockServices) == 0 {
		err = errors.New("no services found")
	}
	return
}

func (c *MockControlPlane) GetServiceStates(serviceId string, state *[]*dao.ServiceState) (err error) {
	switch serviceId {
	case "nilstate-snapshot":
		err = errors.New("no state found for serviceId")
	case "notfound-1":
		s := make([]*dao.ServiceState, 0)
		*state = s
	case "notfound-2":
		s := make([]*dao.ServiceState, 1)
		s[0] = &dao.ServiceState{}
		*state = s
	default:
		s := make([]*dao.ServiceState, 1)
		s[0] = &dao.ServiceState{
			ServiceId: serviceId,
			DockerId:  serviceId,
		}
		*state = s
	}
	return
}

func (c *MockControlPlane) GetVolume(serviceId string, v *volume.Volume) (err error) {
	switch serviceId {
	case "nilvolume-snapshot":
		fallthrough
	case "nilstate-snapshot":
		err = errors.New("no volume found for serviceId")
	default:
		*v = volume.Volume{MockVolumeInstance}
	}
	return
}

func setUp() {
	MockServices = make([]*dao.Service, 0)
	MockPauseResume = make(map[string]bool)
	MockVolumeInstance.name = ""

	runServiceCommand = func(state *dao.ServiceState, command string) (data []byte, err error) {
		data = []byte(fmt.Sprintf("%+v", state))

		switch command {
		case "pause-fail":
			if MockPauseResume[state.DockerId] {
				err = errors.New("service already halted")
			} else {
				err = errors.New("failed to pause service")
				MockPauseResume[state.DockerId] = false
			}
		case "pause-success":
			if MockPauseResume[state.DockerId] {
				err = errors.New("service already halted")
			} else {
				MockPauseResume[state.DockerId] = true
			}
		case "resume-fail":
			if MockPauseResume[state.DockerId] {
				err = errors.New("failed to resume service")
			} else {
				err = errors.New("service already running")
				MockPauseResume[state.DockerId] = false
			}
		case "resume-success":
			if MockPauseResume[state.DockerId] {
				MockPauseResume[state.DockerId] = false
			} else {
				err = errors.New("service already running")
			}
		}
		return
	}
}

func tearDown() {
}

func TestSnapshot(t *testing.T) {
	setUp()
	defer tearDown()

	dfs, err := NewDistributedFileSystem(&MockControlPlane{}, facade.New())
	if err != nil {
		t.Fatalf("failed to initialize dfs: %+v", err)
	}

	// * error while acquiring the service
	if _, err := dfs.Snapshot("nilservice-snapshot"); err.Error() != dfs.client.GetService("nilservice-snapshot", nil).Error() {
		t.Errorf("error not caught while acquiring the service")
	}
}

func TestSnapshotPauseResume(t *testing.T) {
	setUp()
	defer tearDown()

	var services []*dao.Service

	dfs, err := NewDistributedFileSystem(&MockControlPlane{}, facade.New())
	if err != nil {
		t.Fatalf("failed to initialize dfs: %+v", err)
	}

	// * error while acquiring the user
	niluser_err := errors.New("user not found")
	getCurrentUser = func() (*user.User, error) {
		return nil, niluser_err
	}
	if _, err := dfs.Snapshot("niluser-snapshot"); err.Error() != niluser_err.Error() {
		t.Errorf("error not caught while acquiring the user")
	}

	// ** user is not root / error while acquiring the volume
	getCurrentUser = func() (u *user.User, err error) {
		u = &user.User{
			Username: "testuser",
		}
		return
	}

	if _, err := dfs.Snapshot("nilvolume-snapshot"); err.Error() != dfs.client.GetVolume("nilvolume-snapshot", nil).Error() {
		t.Errorf("error not caught while acquiring the volume")
	}

	// * error while acquiring all services
	getCurrentUser = func() (u *user.User, err error) {
		u = &user.User{
			Username: USER_ROOT,
		}
		return
	}

	if _, err := dfs.Snapshot("nilvolume-snapshot"); err.Error() != dfs.client.GetServices(unused, new([]*dao.Service)).Error() {
		t.Errorf("error not caught while acquiring the services")
	}

	// ~*~ service pause/resume ~*~
	// pause is empty OR resume is empty
	services = make([]*dao.Service, 3)
	services[0] = &dao.Service{
		Id: "service0",
		Snapshot: dao.SnapshotCommands{
			Pause:  "",
			Resume: "command",
		},
	}
	services[1] = &dao.Service{
		Id: "service1",
		Snapshot: dao.SnapshotCommands{
			Pause:  "command",
			Resume: "",
		},
	}
	services[2] = &dao.Service{
		Id: "service2",
		Snapshot: dao.SnapshotCommands{
			Pause:  "",
			Resume: "",
		},
	}
	MockServices = services
	if _, err := dfs.Snapshot("nilstate-snapshot"); err.Error() != dfs.client.GetVolume("nilstate-snapshot", nil).Error() {
		t.Errorf("error not caught while acquiring the volume")
	}

	// error acquiring service states
	services = make([]*dao.Service, 1)
	services[0] = &dao.Service{
		Id: "nilstate-snapshot",
		Snapshot: dao.SnapshotCommands{
			Pause:  "command",
			Resume: "command",
		},
	}
	MockServices = services
	if _, err := dfs.Snapshot("nilstate-snapshot"); err.Error() != dfs.client.GetServiceStates("nilstate-snapshot", nil).Error() {
		t.Errorf("error not caught while acquiring the service state")
	}

	// pause fail
	services = make([]*dao.Service, 3)
	services[0] = &dao.Service{
		Id: "service0",
		Snapshot: dao.SnapshotCommands{
			Pause:  "pause-success",
			Resume: "resume-success",
		},
	}
	services[1] = &dao.Service{
		Id: "service1",
		Snapshot: dao.SnapshotCommands{
			Pause:  "pause-fail",
			Resume: "resume-success",
		},
	}
	services[2] = &dao.Service{
		Id: "service2",
		Snapshot: dao.SnapshotCommands{
			Pause:  "pause-sucess",
			Resume: "resume-fail",
		},
	}
	MockServices = services
	if _, err := dfs.Snapshot("nilvolume-snapshot"); err.Error() != dfs.client.GetVolume("nilvolume-snapshot", nil).Error() {
		if paused, ok := MockPauseResume[services[0].Id]; paused || !ok {
			t.Errorf("unexpected state for %s", services[0].Id)
		} else if paused, ok := MockPauseResume[services[1].Id]; paused || !ok {
			t.Errorf("unexpected state for %s", services[1].Id)
		} else if paused, ok := MockPauseResume[services[2].Id]; paused || ok {
			t.Errorf("unexpected state for %s", services[2].Id)
		}
	} else {
		t.Errorf("error not caught while pausing and resuming services")
	}

	// error while taking the snapshot
	services = make([]*dao.Service, 3)
	services[0] = &dao.Service{
		Id: "service0",
		Snapshot: dao.SnapshotCommands{
			Pause:  "pause-success",
			Resume: "resume-success",
		},
	}
	services[1] = &dao.Service{
		Id: "service1",
		Snapshot: dao.SnapshotCommands{
			Pause:  "pause-success",
			Resume: "resume-fail",
		},
	}
	services[2] = &dao.Service{
		Id: "service2",
		Snapshot: dao.SnapshotCommands{
			Pause:  "pause-sucess",
			Resume: "resume-success",
		},
	}
	MockServices = services
	if label, err := dfs.Snapshot("errsnapshot"); err.Error() != MockVolumeInstance.Snapshot(label).Error() {
		t.Errorf("error not caught while taking the snapshot")
	} else {
		if paused, ok := MockPauseResume[services[0].Id]; paused || !ok {
			t.Errorf("unexpected state for %s", services[0].Id)
		} else if paused, ok := MockPauseResume[services[1].Id]; !paused || !ok {
			t.Errorf("unexpected state for %s", services[1].Id)
		} else if paused, ok := MockPauseResume[services[2].Id]; paused || ok {
			t.Errorf("unexpected state for %s", services[2].Id)
		}
	}

	// * success
	services = make([]*dao.Service, 1)
	services[0] = &dao.Service{
		Id: "service0",
		Snapshot: dao.SnapshotCommands{
			Pause:  "pause-success",
			Resume: "resume-success",
		},
	}
	MockServices = services
	MockVolumeInstance.name = "success"
	if _, err := dfs.Snapshot("success-snapshot"); err != nil {
		t.Errorf("unexpected error while capturing the snapshot: %+v", err)
	}
}

func TestCommit(t *testing.T) {
	// TODO: write tests!
	// * wait for lock
	// * error while acquiring the container
	// * container is still running
	// * error while acquiring the images
	// * stale image
	// * error while snapshotting the dfs
	// * error while committing the container
	// * error while getting the tenant id
	// * error while getting the volume
	// * error while marshalling the images
	// * error while writing the images to file
	// * success
	// ** lock is released
	// ** label value is populated
	// ** error is not nil
}

func TestRollback(t *testing.T) {
	// TODO: write tests!
	// * wait for lock
	// * bad snapshot id
	// * error while acquiring the tenantId
	// * error while acquiring the service
	// * error while acquiring the volume
	// * error while getting the latest images
	// * error while getting the snapshot images
	// * error while retagging images
	// * error while running rollback
	// * success
}