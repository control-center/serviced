/*
serviced-storage PATH COMMAND ACTION [OPTIONS]

Commands:

	driver
		init     TYPE
		shutdown
		status

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
	Verbose   bool           `short:"v" long:"verbose" description:"Display verbose logging"`
	Directory flags.Filename `short:"d" long:"directory" description:"Driver directory"`
}

type ServicedStorage struct {
	name    string
	Parser  *flags.Parser
	Options ServicedStorageOptions
}

func (s *ServicedStorage) Run() {
	// Set up some initial logging for the sake of parser errors
	var logLevel = log.InfoLevel

	if _, err := s.Parser.AddGroup("Basic Options", "Basic options", &s.Options); err != nil {
		log.WithFields(log.Fields{"exitcode": 1}).Info("Unable to add option group")
		os.Exit(1)
	}

	if s.Options.Verbose {
		logLevel = log.DebugLevel
	}

	if _, err := s.Parser.Parse(); err != nil {
		os.Exit(1)
	}

	s.initializeLogging(logLevel)
}

func (s *ServicedStorage) initializeLogging(level log.Level) {
	log.SetOutput(os.Stderr)
	log.SetLevel(level)
}

func main() {
	App.initializeLogging(log.DebugLevel)
	App.Run()
}