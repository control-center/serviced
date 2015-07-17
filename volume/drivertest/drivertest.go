package drivertest

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"

	"github.com/control-center/serviced/volume"
)

var (
	drv volume.Driver
)

type Driver struct {
	volume.Driver
	// Keep a reference to the root here just in case something below doesn't work
	root string
}

func newDriver(t *testing.T, name string) *Driver {
	root, err := ioutil.TempDir("/var/tmp", "serviced-drivertest-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	d, err := volume.GetDriver(name, root)
	if err != nil {
		t.Logf("drivertest: %v\n", err)
		if err == volume.ErrDriverNotSupported {
			t.Skipf("Driver %s not supported", name)
		}
		t.Fatal(err)
	}
	return &Driver{d, root}
}

func cleanup(t *testing.T, d *Driver) {
	if err := d.Cleanup(); err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(d.root)
}

func verifyFile(t *testing.T, path string, mode os.FileMode, uid, gid uint32) {
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if fi.Mode()&os.ModeType != mode&os.ModeType {
		t.Fatalf("Expected %s type 0x%x, got 0x%x", path, mode&os.ModeType, fi.Mode()&os.ModeType)
	}

	if fi.Mode()&os.ModePerm != mode&os.ModePerm {
		t.Fatalf("Expected %s mode %o, got %o", path, mode&os.ModePerm, fi.Mode()&os.ModePerm)
	}

	if fi.Mode()&os.ModeSticky != mode&os.ModeSticky {
		t.Fatalf("Expected %s sticky 0x%x, got 0x%x", path, mode&os.ModeSticky, fi.Mode()&os.ModeSticky)
	}

	if fi.Mode()&os.ModeSetuid != mode&os.ModeSetuid {
		t.Fatalf("Expected %s setuid 0x%x, got 0x%x", path, mode&os.ModeSetuid, fi.Mode()&os.ModeSetuid)
	}

	if fi.Mode()&os.ModeSetgid != mode&os.ModeSetgid {
		t.Fatalf("Expected %s setgid 0x%x, got 0x%x", path, mode&os.ModeSetgid, fi.Mode()&os.ModeSetgid)
	}

	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		if stat.Uid != uid {
			t.Fatalf("%s no owned by uid %d", path, uid)
		}
		if stat.Gid != gid {
			t.Fatalf("%s not owned by gid %d", path, gid)
		}
	}

}

func DriverTestCreateEmpty(t *testing.T, drivername string) {
	driver := newDriver(t, drivername)
	defer cleanup(t, driver)

	volumeName := "empty"

	if _, err := driver.Create(volumeName); err != nil {
		t.Fatal(err)
	}
	if !driver.Exists(volumeName) {
		t.Fatal("Newly created image doesn't exist")
	}
	vol, err := driver.Get(volumeName)
	if err != nil {
		t.Fatal(err)
	}
	verifyFile(t, vol.Path(), 0755|os.ModeDir, 0, 0)
	fis, err := ioutil.ReadDir(vol.Path())
	if err != nil {
		t.Fatal(err)
	}
	if len(fis) != 0 {
		t.Fatal("New directory not empty")
	}

	driver.Release(volumeName)

	if err := driver.Remove(volumeName); err != nil {
		t.Fatal(err)
	}
}