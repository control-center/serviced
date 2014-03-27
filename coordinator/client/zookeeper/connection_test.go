package zk_driver

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	zklib "github.com/samuel/go-zookeeper/zk"
)

func ensureZkFatjar() {
	_, err := exec.LookPath("java")
	if err != nil {
		log.Fatal("Can't find java in path")
	}

	jars, err := filepath.Glob("zookeeper-*/contrib/fatjar/zookeeper-*-fatjar.jar")
	if err != nil {
		log.Fatal("Error search for files")
	}
	if len(jars) > 0 {
		return
	}

	err = exec.Command("curl", "-O", "http://www.java2s.com/Code/JarDownload/zookeeper/zookeeper-3.3.3-fatjar.jar.zip").Run()
	if err != nil {
		log.Fatal("Could not download fatjar: %s", err)
	}

	err = exec.Command("unzip", "zookeeper-3.3.3-fatjar.jar.zip").Run()
	if err != nil {
		log.Fatal("Could not unzip fatjar: %s", err)
	}
	err = exec.Command("mkdir", "-p", "zookeeper-3.3.3/contrib/fatjar").Run()
	if err != nil {
		log.Fatal("Could not make fatjar dir: %s", err)
	}

	err = exec.Command("mv", "zookeeper-3.3.3-fatjar.jar", "zookeeper-3.3.3/contrib/fatjar/").Run()
	if err != nil {
		log.Fatal("Could not mv fatjar: %s", err)
	}

	err = exec.Command("rm", "zookeeper-3.3.3-fatjar.jar.zip").Run()
	if err != nil {
		log.Fatal("Could not rm fatjar.zip: %s", err)
	}
}

func init() {
	ensureZkFatjar()
}

func TestEnsureZkFatjar(t *testing.T) {
	ensureZkFatjar()
}

func TestZkDriver(t *testing.T) {
	tc, err := zklib.StartTestCluster(1)
	if err != nil {
		t.Fatalf("could not start test zk cluster: %s", err)
	}
	defer os.RemoveAll(tc.Path)
	defer tc.Stop()
	time.Sleep(time.Second)

	servers := []string{fmt.Sprintf("127.0.0.1:%d", tc.Servers[0].Port)}

	drv := Driver{}
	dsnBytes, err := json.Marshal(DSN{Servers: servers, Timeout: time.Second * 15})
	if err != nil {
		t.Fatal("unexpected error creating zk DSN: %s", err)
	}
	dsn := string(dsnBytes)

	conn, err := drv.GetConnection(dsn)
	if err != nil {
		t.Fatal("unexpected error getting connection")
	}
	exists, err := conn.Exists("/foo")
	if err != nil {
		t.Fatalf("err calling exists: %s", err)
	}
	if exists {
		t.Fatal("foo should not exist")
	}

	err = conn.Delete("/foo")
	if err == nil {
		t.Fatalf("delete on non-existent object should fail")
	}

	err = conn.CreateDir("/foo")
	if err != nil {
		t.Fatalf("creating /foo should work: %s", err)
	}

	err = conn.Create("/foo/bar", []byte("test"))
	if err != nil {
		t.Fatalf("creating /foo/bar should work: %s", err)
	}

	exists, err = conn.Exists("/foo/bar")
	if err != nil {
		t.Fatalf("could not call exists: %s", err)
	}

	if !exists {
		t.Fatal("/foo/bar should not exist")
	}

	err = conn.Delete("/foo")
	if err != nil {
		t.Fatalf("delete of /foo should work: %s", err)
	}

}
