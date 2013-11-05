package isvcs

import (
	"log"
	"testing"
)

func TestOpenTsdb(t *testing.T) {

	var err error

	err = OpenTsdbContainer.Kill()
	log.Printf("Killing container: %v", err)
	OpenTsdbContainer.Run()

	err = OpenTsdbContainer.Stop()
	log.Printf("Stopping container: %v", err)

	OpenTsdbContainer.Run()
	log.Printf("Running container: %v", err)
}
