/*
serviced-storage PATH COMMAND ACTION [OPTIONS]

Commands:

	volume
		list
		create   NAME
		remove   NAME
		export   NAME
*/
package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
)

var (
	name = "serviced-storage"
	App  = &ServicedStorage{
		name:   name,
		Parser: flags.NewNamedParser(name, flags.Default),
	}
)

type ServicedStorageOptions struct {
	Verbose   []bool         `short:"v" description:"Display verbose logging"`
	Directory flags.Filename `short:"d" long:"directory" description:"Driver directory"`
}

type ServicedStorage struct {
	name    string
	Parser  *flags.Parser
	Options ServicedStorageOptions
}

func (s *ServicedStorage) Run() {
	// Set up some initial logging for the sake of parser errors
	s.initializeLogging()
	if _, err := s.Parser.AddGroup("Basic Options", "Basic options", &s.Options); err != nil {
		log.WithFields(log.Fields{"exitcode": 1}).Fatal("Unable to add option group")
		os.Exit(1)
	}
	s.Parser.Parse()
}

func (s *ServicedStorage) initializeLogging() {
	log.SetOutput(os.Stderr)
	level := log.WarnLevel + log.Level(len(App.Options.Verbose))
	log.SetLevel(level)
}

func main() {
	App.Run()
}