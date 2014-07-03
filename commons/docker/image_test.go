package docker

import (
	"net"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

const (
	bogusimage = "iam/bogus:zenoss"
	rawbase    = "busybox:latest"
	basetag    = "localhost:5000/testimageapi/busybox:latest"
	snaptag    = "localhost:5000/testimageapi/busybox:snapshot"
	victim     = "localhost:5000/testimageapi/busybox:victim"
)

func TestImageAPI(t *testing.T) {
	TestingT(t)
}

type ImageTestSuite struct {
	regid string
}

var _ = Suite(&ImageTestSuite{})

func (s *ImageTestSuite) SetUpSuite(c *C) {
	cmd := []string{"docker", "run", "-d", "-p", "5000:5000", "registry"}
	regid, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		panic("can't start registry")
	}

	s.regid = strings.Trim(string(regid), "\n")

	regup := make(chan struct{})
	timeout := make(chan struct{})

	go func(timeout, regup chan struct{}) {
	WaitForRegistryConnection:
		for {
			select {
			case <-timeout:
				panic("start registry timed out: can't connect")
			default:
				if _, err := net.Dial("tcp", ":5000"); err != nil {
					time.Sleep(1 * time.Second)
					continue WaitForRegistryConnection
				}
				break WaitForRegistryConnection
			}
		}

	WaitForRegistryPing:
		for {
			select {
			case <-timeout:
				panic("start registry timed out: can't ping it")
			default:
				_, err := http.Get("http://localhost:5000/v1/_ping")
				if err != nil {
					time.Sleep(1 * time.Second)
					continue WaitForRegistryPing
				}
				regup <- struct{}{}
			}
		}
	}(timeout, regup)

	select {
	case <-time.After(60 * time.Second):
		timeout <- struct{}{}
	case <-regup:
		break
	}

	cmd = []string{"docker", "export", s.regid, "/tmp/regexp.tar"}
	if err = exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
		panic("can't export registry container for import testing: ", err)
	}
}

func (s *ImageTestSuite) TearDownSuite(c *C) {
	cmd := []string{"docker", "kill", s.regid}
	if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
		panic("can't kill the registry")
	}

	cmd = []string{"docker", "rmi", basetag}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"docker", "rmi", snaptag}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"docker", "rmi", rawbase}
	exec.Command(cmd[0], cmd[1:]...).Run()

	cmd = []string{"rm", "/tmp/regexp.tar"}
	exec.Command(cmd[0], cmd[1:]...).Run()
}

func (s *ImageTestSuite) TestFindImage(c *C) {
	_, err := FindImage(rawbase, true)
	if err != nil {
		c.Errorf("can't find %s: %v", rawbase, err)
	}
}

func (s *ImageTestSuite) TestFindNonexistentImage(c *C) {
	_, err := FindImage(bogusimage, false)
	if err == nil {
		c.Errorf("should not be able to find %s", bogusimage)
	}
}

func (s *ImageTestSuite) TestTagImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Errorf("can't find %s: %v ", rawbase, err)
	}

	ti, err := img.Tag(basetag)
	if err != nil {
		c.Error("can't tag %s as %s: %v", rawbase, basetag, err)
	}

	_, err = FindImage(ti.ID.String(), false)
	if err != nil {
		c.Error("can't find %s: %v", ti.ID, err)
	}
}

func (s *ImageTestSuite) TestDoubleTagImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Errorf("can't find %s: %v", rawbase, err)
	}

	bt, err := img.Tag(basetag)
	if err != nil {
		c.Errorf("can't tag %s as %s: %v", rawbase, basetag, err)
	}

	st, err := img.Tag(snaptag)
	if err != nil {
		c.Errorf("can't tag %s as %s: %v", bt.ID.String(), snaptag, err)
	}

	_, err = FindImage(st.ID.String(), false)
	if err != nil {
		c.Errorf("can't find %s: %v", snaptag, err)
	}
}

func (s *ImageTestSuite) TestDeleteImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Errorf("can't find %s: %v", rawbase, err)
	}

	ti, err := img.Tag(victim)
	if err != nil {
		c.Errorf("can't tag %s as %s: %v", rawbase, victim, err)
	}

	if err = ti.Delete(); err != nil {
		c.Errorf("can't delete %s: %v", ti.ID.String(), err)
	}

	img, err = FindImage(ti.ID.String(), false)
	if img != nil {
		c.Error("should not have found: ", ti.ID.String())
	}
}

func (s *ImageTestSuite) TestFindThruLocalRepository(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Error("can't find %s: %v", rawbase, err)
	}

	ti, err := img.Tag(basetag)
	if err != nil {
		c.Errorf("can't tag %s as %s: %v", rawbase, basetag, err)
	}

	if err = ti.Delete(); err != nil {
		c.Errorf("can't delete %s: %v", ti.ID.String(), err)
	}

	_, err = FindImage(basetag, false)
	if err == nil {
		c.Errorf("%s should not be in the repo", basetag)
	}

	_, err = FindImage(basetag, true)
	if err != nil {
		c.Errorf("can't find %s in local registry: %v", basetag, err)
	}
}
