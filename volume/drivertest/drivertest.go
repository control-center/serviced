package drivertest

import (
	"io/ioutil"
	"os"
	"path"
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

func newDriver(t *testing.T, name, root string) *Driver {
	var err error
	if root == "" {
		root, err = ioutil.TempDir("", "serviced-drivertest-")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(root, 0755); err != nil {
			t.Fatal(err)
		}
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

// DriverTestCreateEmpty verifies that a driver can create a volume, and that
// is is empty (and owned by the current user) after creation.
func DriverTestCreateEmpty(t *testing.T, drivername, root string) {
	driver := newDriver(t, drivername, root)
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
	verifyFile(t, vol.Path(), 0755|os.ModeDir, uint32(os.Getuid()), uint32(os.Getgid()))
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

func createBase(t *testing.T, driver *Driver, name string) volume.Volume {
	// We need to be able to set any perms
	oldmask := syscall.Umask(0)
	defer syscall.Umask(oldmask)

	if _, err := driver.Create(name); err != nil {
		t.Fatal(err)
	}

	volume, err := driver.Get(name)
	if err != nil {
		t.Fatal(err)
	}

	subdir := path.Join(volume.Path(), "a subdir")
	if err := os.Mkdir(subdir, 0705|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(subdir, 1, 2); err != nil {
		t.Fatal(err)
	}

	file := path.Join(volume.Path(), "a file")
	if err := ioutil.WriteFile(file, []byte("Some data"), 0222|os.ModeSetuid); err != nil {
		t.Fatal(err)
	}
	return volume
}

func verifyBase(t *testing.T, driver *Driver, vol volume.Volume) {
	subdir := path.Join(vol.Path(), "a subdir")
	verifyFile(t, subdir, 0705|os.ModeDir|os.ModeSticky, 1, 2)

	file := path.Join(vol.Path(), "a file")
	verifyFile(t, file, 0222|os.ModeSetuid, 0, 0)

	fis, err := ioutil.ReadDir(vol.Path())
	if err != nil {
		t.Fatal(err)
	}

	if len(fis) != 2 {
		t.Fatal("Unexpected files in base image")
	}

}

func DriverTestCreateBase(t *testing.T, drivername, root string) {
	driver := newDriver(t, drivername, root)
	defer cleanup(t, driver)

	vol := createBase(t, driver, "Base")
	verifyBase(t, driver, vol)

	if err := driver.Remove("Base"); err != nil {
		t.Fatal(err)
	}
}

func DriverTestSnapshots(t *testing.T, drivername, root string) {
	driver := newDriver(t, drivername, root)
	defer cleanup(t, driver)

	vol := createBase(t, driver, "Base")
	err := vol.Snapshot("Snap")
	if err != nil {
		t.Fatal(err)
	}

	verifyBase(t, driver, vol)

	file := path.Join(vol.Path(), "differentfile")
	if err := ioutil.WriteFile(file, []byte("Some other data"), 0222|os.ModeSetuid); err != nil {
		t.Fatal(err)
	}

	if err := vol.Snapshot("Snap2"); err != nil {
		t.Fatal(err)
	}
	if err := vol.Rollback("Snap"); err != nil {
		t.Fatal(err)
	}

	verifyBase(t, driver, vol)

	if err := driver.Remove("Base"); err != nil {
		t.Fatal(err)
	}
}