package snapshot

import (
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/volume"

	"testing"
)

type MockControlPlane struct {
	dao.ControlPlane
}

func (c *MockControlPlane) GetTenantId(serviceId string, tenantId *string) error {
	return nil
}

func (c *MockControlPlane) GetService(serviceId string, service *dao.Service) error {
	return nil
}

func (c *MockControlPlane) GetServices(request dao.EntityRequest, services *[]*dao.Service) error {
	return nil
}

func (c *MockControlPlane) GetVolume(serviceId string, volume *volume.Volume) error {
	return nil
}

func TestSnapshot(t *testing.T) {
	var label string
	dfs, err := NewDistributedFileSystem(&MockControlPlane{})
	if err != nil {
		t.Fatalf("failed to initialize dfs: %+v", err)
	}

	// * error while acquiring the tenant id
	dfs.Snapshot("serviceId", &label)

	// * error while acquiring the service
	dfs.Snapshot("serviceId", &label)

	// * error while acquiring the user
	dfs.Snapshot("serviceId", &label)

	// ** user is not root
	dfs.Snapshot("serviceId", &label)

	// * error while acquiring all services
	dfs.Snapshot("serviceId", &label)

	// ~*~ service pause/resume ~*~
	dfs.Snapshot("serviceId", &label)

	// pause is empty OR resume is empty
	dfs.Snapshot("serviceId", &label)

	// pause fail
	dfs.Snapshot("serviceId", &label)

	// * error while getting the volume
	dfs.Snapshot("serviceId", &label)

	// * error while taking the snapshot
	dfs.Snapshot("serviceId", &label)

	// * success
	dfs.Snapshot("serviceId", &label)

	// ** resume is run for all snapshots
	// ** label value is populated
	// ** error is not nil
}

func TestCommit(t *testing.T) {
	var label string
	dfs, err := NewDistributedFileSystem(&MockControlPlane{})
	if err != nil {
		t.Fatalf("failed to initialize dfs: %+v", err)
	}

	// * wait for lock
	dfs.Commit("containerId", &label)

	// * error while acquiring the container
	dfs.Commit("containerId", &label)

	// * container is still running
	dfs.Commit("containerId", &label)

	// * error while acquiring the images
	dfs.Commit("containerId", &label)

	// * stale image
	dfs.Commit("containerId", &label)

	// * error while snapshotting the dfs
	dfs.Commit("containerId", &label)

	// * error while committing the container
	dfs.Commit("containerId", &label)

	// * error while getting the tenant id
	dfs.Commit("containerId", &label)

	// * error while getting the volume
	dfs.Commit("containerId", &label)

	// * error while getting the snapshots
	dfs.Commit("containerId", &label)

	// * error while marshalling the images
	dfs.Commit("containerId", &label)

	// * error while writing the images to file
	dfs.Commit("containerId", &label)

	// * success
	dfs.Commit("containerId", &label)

	// ** lock is released
	// ** label value is populated
	// ** error is not nil
}

func TestRollback(t *testing.T) {
	dfs, err := NewDistributedFileSystem(&MockControlPlane{})
	if err != nil {
		t.Fatalf("failed to initialize dfs: %+v", err)
	}

	// * wait for lock
	dfs.Rollback("snapshotId")

	// * bad snapshot id
	dfs.Rollback("snapshotId")

	// * error while acquiring the tenantId
	dfs.Rollback("snapshotId")

	// * error while acquiring the service
	dfs.Rollback("snapshotId")

	// * error while acquiring the volume
	dfs.Rollback("snapshotId")

	// * error while reading the images file
	dfs.Rollback("snapshotId")

	// * error while unmarshalling json
	dfs.Rollback("snapshotId")

	// * error while finding images
	dfs.Rollback("snapshotId")

	// * error while looking up the docker binary
	dfs.Rollback("snapshotId")

	// * error while retagging images
	dfs.Rollback("snapshotId")

	// * error while running rollback
	dfs.Rollback("snapshotId")

	// * success
	dfs.Rollback("snapshotId")

}

func TestPauseResume(t *testing.T) {
	dfs, err := NewDistributedFileSystem(&MockControlPlane{})
	if err != nil {
		t.Fatalf("failed to initialize dfs: %+v", err)
	}
	service := new(dao.Service)
	state := new(dao.ServiceState)

	// * pause success
	dfs.Pause(service, state)

	// * pause fail
	dfs.Pause(service, state)

	// * resume success
	dfs.Resume(service, state)

	// * resume fail
	dfs.Resume(service, state)

}