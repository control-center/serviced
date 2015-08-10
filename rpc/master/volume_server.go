package master
import (
	"github.com/zenoss/glog"
	"github.com/control-center/serviced/volume"
	"errors"
)

// GetVolumeStatus gets the volume status
func (s *Server) GetVolumeStatus(volumeNames []string, reply *volume.Statuses) error {
	glog.V(2).Infof("[hosts_server.go]master.GetVolumeStatus(%v, %v)\n", volumeNames, reply)
	response := volume.GetStatus(volumeNames)
	if response == nil {
		glog.V(2).Infof("\tCall to volume.getStatus failed: (%v). Returning error.", response)
		return errors.New("hosts_server.go host not found")
	}
	*reply = *response
	return nil
}
