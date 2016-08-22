package logri

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	ConfigurationError = errors.New("Unable to parse configuration")
)

// LogriConfig is the configuration for a logri manager
type LogriConfig []LoggerConfig

// LoggerConfig is the configuration for a single logger
type LoggerConfig struct {
	Logger string
	Level  string
	Local  bool
	Out    []OutConfig
}

type OutConfig struct {
	Type    OutputType
	Options map[string]string
	Local   bool
}

func ConfigFromBytes(b []byte) (LogriConfig, error) {
	var (
		cfg LogriConfig
	)
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	sort.Sort(&cfg)
	return cfg, nil
}

func ConfigFromYAML(r io.Reader) (LogriConfig, error) {
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return ConfigFromBytes(buf.Bytes())
}

func (c LogriConfig) Len() int      { return len(c) }
func (c LogriConfig) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

// Sort loggers by depth in the hierarchy
func (c LogriConfig) Less(i, j int) bool {
	a, b := c[i].Logger, c[j].Logger
	if a == "*" || a == "" {
		return true
	}
	if b == "*" || b == "" {
		return false
	}
	return strings.Count(a, ".") < strings.Count(b, ".")
}
