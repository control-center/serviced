package etcd

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/zenoss/serviced/coordinator/client"
)

type EtcdDriver struct {
	client  *etcd.Client
	servers []string
	timeout time.Duration
	onClose **func()
}

// Assert that the Ectd driver meets the Driver interface
var _ client.Driver = EtcdDriver{}

func NewEtcdDriver(servers []string, timeout time.Duration) (driver client.Driver, err error) {

	client := etcd.NewClient(servers)
	client.SetConsistency("STRONG_CONSISTENCY")

	driver = &EtcdDriver{
		client:  client,
		servers: servers,
		timeout: timeout,
		onClose: new(*func()),
	}
	return driver, nil
}

func init() {
	if err := client.RegisterDriver("etcd", NewEtcdDriver); err != nil {
		panic(err)
	}
}

func (etcd EtcdDriver) Close() {
	etcd.client.CloseCURL()
	if *etcd.onClose != nil {
		(*(*etcd.onClose))()
	}
}

func (etcd EtcdDriver) SetOnClose(f func()) {
	*etcd.onClose = &f
}

func (etcd EtcdDriver) Lock(path string) (lockId string, err error) {
	return "", errors.New("unsupported")
}

func (etcd EtcdDriver) Unlock(path string, lockId string) (err error) {
	return errors.New("unsupported")
}

func (etcd EtcdDriver) Create(path string, data []byte) error {
	_, err := etcd.client.Create(path, string(data), 1000000)
	return err
}

func (etcd EtcdDriver) CreateDir(path string) error {
	_, err := etcd.client.CreateDir(path, 1000000)
	return err
}

func (etcd EtcdDriver) Exists(path string) (bool, error) {
	_, err := etcd.client.Get(path, false, false)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (etc EtcdDriver) Delete(path string) error {
	_, err := etc.client.Delete(path, true)
	return err
}

var etcdBinaryPath string

func setEtcdBinaryPath() error {
	var err error
	etcdBinaryPath, err = exec.LookPath("etcd")
	if err == nil {
		return nil
	}

	gopath := os.Getenv("GOPATH")
	if len(gopath) <= 0 {
		log.Fatal("GOPATH is not set")
	}

	err = exec.Command("go", "get", "github.com/coreos/etcd").Run()
	if err != nil {
		return err
	}

	for _, pth := range filepath.SplitList(gopath) {
		if len(pth) <= 0 {
			break
		}
		etcdBinaryPath, err = exec.LookPath(pth + "/bin/etcd")
		if err == nil {
			return nil
		}
		break
	}
	return errors.New("Could not find etcd")
}

type TestCluster struct {
	tmpDir     string
	process    *os.Process
	clientPort int
}

func (tc TestCluster) Machines() []string {
	return []string{fmt.Sprintf("http://localhost:%d", tc.clientPort)}
}

func (tc TestCluster) Stop() {
	tc.process.Kill()
	os.RemoveAll(tc.tmpDir)
}

func NewTestCluster() (*TestCluster, error) {

	if err := setEtcdBinaryPath(); err != nil {
		return nil, err
	}

	tc := &TestCluster{}

	// get some unused ports
	lclient, _ := net.Listen("tcp", "127.0.0.1:0") // listen on localhost
	clientPort := lclient.Addr().(*net.TCPAddr).Port
	lserver, _ := net.Listen("tcp", "127.0.0.1:0") // listen on localhost
	serverPort := lserver.Addr().(*net.TCPAddr).Port

	tmpdir, err := ioutil.TempDir("", "etcDriverTest-")
	if err != nil {
		return nil, err
	}

	tc.tmpDir = tmpdir

	procAttr := new(os.ProcAttr)
	procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}
	args := []string{etcdBinaryPath, "-name=etcDriverTest",
		fmt.Sprintf("-addr=127.0.0.1:%d", clientPort),
		fmt.Sprintf("-bind-addr=127.0.0.1:%d", clientPort),
		fmt.Sprintf("-peer-addr=127.0.0.1:%d", serverPort),
		fmt.Sprintf("-peer-bind-addr=127.0.0.1:%d", serverPort),
		"-data-dir=" + tmpdir}

	lclient.Close()
	lserver.Close()
	tc.clientPort = clientPort
	process, err := os.StartProcess(etcdBinaryPath, append(args, "-f"), procAttr)
	if err != nil {
		defer os.RemoveAll(tc.tmpDir)
		return nil, err
	}
	tc.process = process
	time.Sleep(time.Second)
	return tc, nil
}
