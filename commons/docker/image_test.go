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
	rawbase = "base:latest"
	basetag = "localhost:5000/testimageapi/base:latest"
	victim  = "localhost:5000/testimageapi/base:victim"
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
}

func (s *ImageTestSuite) TearDownSuite(c *C) {
	cmd := []string{"docker", "kill", s.regid}
	if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
		panic("can't kill the registry")
	}

	cmd = []string{"docker", "rmi", basetag}
	if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
		panic("can't delete tagged test image")
	}
}

func (s *ImageTestSuite) TestFindImage(c *C) {
	_, err := FindImage(rawbase, false)
	if err != nil {
		c.Errorf("can't find %s: %v", rawbase, err)
	}
}

func (s *ImageTestSuite) TestTagImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Error("can't find %s: %v ", rawbase, err)
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

func (s *ImageTestSuite) TestDeleteImage(c *C) {
	img, err := FindImage(rawbase, true)
	if err != nil {
		c.Error("can't find %s: %v", rawbase, err)
	}

	ti, err := img.Tag(victim)
	if err != nil {
		c.Error("can't tag %s as %s: %v", rawbase, victim, err)
	}

	if err = ti.Delete(); err != nil {
		c.Errorf("can't delete %s: %v", ti.ID.String(), err)
	}

	img, err = FindImage(ti.ID.String(), false)
	if img != nil {
		c.Error("should not have found: ", ti.ID.String())
	}
}
