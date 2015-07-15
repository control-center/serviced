package main

import (
  "fmt"

  "github.com/control-center/serviced/domain/host"
  "github.com/control-center/serviced/rpc/agent"
  "github.com/control-center/serviced/utils"
  "github.com/zenoss/glog"
)

type MockAgent struct {
  mockHost *host.Host
}

func (m *MockAgent) BuildHost(request agent.BuildHostRequest, hostResponse *host.Host) error {
  *hostResponse = host.Host{}

  glog.Infof("Build Host Request: %s:%d, %s, %s", request.IP, request.Port, request.PoolID, request.Memory)

  if mem, err := utils.ParseEngineeringNotation(request.Memory); err == nil {
    m.mockHost.RAMCommitment = mem
  } else if mem, err := utils.ParsePercentage(request.Memory, m.mockHost.Memory); err == nil {
    m.mockHost.RAMCommitment = mem
  } else {
    return fmt.Errorf("Could not parse RAM Commitment: %v", err)
  }
  if request.PoolID != m.mockHost.PoolID {
    m.mockHost.PoolID = request.PoolID
  }
  *hostResponse = *m.mockHost
  return nil
  
}