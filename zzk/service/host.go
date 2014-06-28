package service

import (
	"path"
)

const (
	zkHost = "/hosts"
)

func hostpath(nodes ...string) string {
	p := append([]string{zkHost}, nodes...)
	return path.Join(p...)
}

type HostState struct {
	HostID         string
	ServiceID      string
	ServiceStateID string
	DesiredState   int
	version        interface{}
}

func (node *HostState) Version() interface{} {
	return node.version
}

func (node *HostState) SetVersion(version interface{}) {
	node.version = version
}