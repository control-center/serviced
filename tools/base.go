package tools

import (
	"errors"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"
)

var (
	ErrInvalidArgs = errors.New("invalid arguments provided")
	logLevel       = log.InfoLevel
)

type ErrExit struct {
	Msg      string
	ExitCode int
}

func (e ErrExit) Error() string {
	return e.Msg
}

func ErrorExit(code int, msg string) ErrExit {
	return ErrExit{msg, code}
}

type ServicedTool struct {
	name    string
	Parser  *flags.Parser
	Options struct{}
}

func (t *ServicedTool) initializeLogging(level log.Level) {
	log.SetOutput(os.Stderr)
	log.SetLevel(level)
}

func (t *ServicedTool) parseOptions() error {
	return ErrorExit(1, "Unable to parse")
}

func (t *ServicedTool) Run() {
	// Set up some initial logging for the sake of parser errors
	var logLevel = log.InfoLevel
	log.SetOutput(os.Stderr)
	log.SetLevel(log.ErrorLevel)

	if _, err := t.Parser.AddGroup("Basic Options", "Basic options", &t.Options); err != nil {
		log.WithFields(log.Fields{"exitcode": 1}).Info("Unable to add option group")
		os.Exit(1)
	}

	if t.Options.Verbose {
		logLevel = log.DebugLevel
	}

	if _, err := t.Parser.Parse(); err != nil {
		os.Exit(1)
	}

	t.initializeLogging(logLevel)
}

func NewServicedTool(name string) *ServicedTool {
	return &ServicedTool{
		name:   name,
		Parser: flags.NewNamedParser(name, flags.Default),
	}
}