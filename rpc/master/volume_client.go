package master
import (
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

//GetVolumeStatus gets status information for the given volume or nil
func (c *Client) GetVolumeStatus(volumeNames []string) (*volume.Statuses, error) {
	glog.V(2).Infof("[hosts_client.go]master.GetVolumeStatus(%v)", volumeNames)
	response := &volume.Statuses{}
	if err := c.call("GetVolumeStatus", volumeNames, response); err != nil {
		glog.V(2).Infof("\tcall to GetVolumeStatus returned error: %v", err)
		return nil, err
	}
	return response, nil
}
