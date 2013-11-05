package isvcs

import (
	"log"
	"testing"
)

func TestZk(t *testing.T) {

	var err error

	err = ZookeeperContainer.Kill()
	log.Printf("Killing container: %v", err)
	ZookeeperContainer.Run()

	err = ZookeeperContainer.Stop()
	log.Printf("Stopping container: %v", err)

	ZookeeperContainer.Run()
	log.Printf("Running container: %v", err)

}
