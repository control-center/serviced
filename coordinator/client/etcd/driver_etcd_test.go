package client

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var etcdBinaryPath string

func setEtcdBinaryPath() {
	var err error
	etcdBinaryPath, err = exec.LookPath("etcd")
	if err == nil {
		return
	}

	gopath := os.Getenv("GOPATH")
	if len(gopath) <= 0 {
		log.Fatal("GOPATH is not set")
	}

	err = exec.Command("go", "get", "github.com/coreos/etcd").Run()
	if err != nil {
		log.Fatalf("Could not run go get github.com/coreos/etcd")
	}

	for _, pth := range filepath.SplitList(gopath) {
		if len(pth) <= 0 {
			break
		}
		etcdBinaryPath, err = exec.LookPath(pth + "/bin/etcd")
		if err == nil {
			return
		}
		break
	}
	log.Fatal("Could not find etcd")
}

func init() {
	// call here so compiler doesn't optimise this away
	setEtcdBinaryPath()
}

func TestPath(t *testing.T) {
	if len(etcdBinaryPath) == 0 {
		t.Fatal()
	}
}

func TestEtcdDriver(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "etcDriverTest-")
	if err != nil {
		t.Logf("Could not create tempdir: %s", err)
		t.FailNow()
	}
	defer os.RemoveAll(tmpdir)

	procAttr := new(os.ProcAttr)
	procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}
	args := []string{etcdBinaryPath, "-name=etcDriverTest", "-data-dir=" + tmpdir}

	process, err := os.StartProcess(etcdBinaryPath, append(args, "-f"), procAttr)
	if err != nil {
		t.Fatal("start process failed:" + err.Error())
		return
	}

	time.Sleep(time.Second)
	defer process.Kill()

	drv, err := NewEtcdDriver([]string{"http://localhost:4001"}, time.Second)
	if err != nil {
		t.Fatalf("Could not create a client: %s", err)
	}

	exists, err := drv.Exists("/foo")
	if err != nil {
		t.Fatalf("err calling exists: %s", err)
	}
	if exists {
		t.Fatal("foo should not exist")
	}

	err = drv.Delete("/foo")
	if err == nil {
		t.Fatalf("delete on non-existent object should fail")
	}

	err = drv.CreateDir("/foo")
	if err != nil {
		t.Fatalf("creating /foo should work: %s", err)
	}

	err = drv.Create("/foo/bar", []byte("test"))
	if err != nil {
		t.Fatalf("creating /foo/bar should work: %s", err)
	}

	exists, err = drv.Exists("/foo/bar")
	if err != nil {
		t.Fatalf("could not call exists: %s", err)
	}

	if !exists {
		t.Fatal("/foo/bar should not exist")
	}

	err = drv.Delete("/foo")
	if err != nil {
		t.Fatalf("delete of /foo should work: %s", err)
	}
}
