// +build windows

package daemon

import (
	"strings"

	"github.com/docker/docker/daemon/execdriver"
	derr "github.com/docker/docker/errors"
	"github.com/docker/docker/volume"
	"github.com/docker/libnetwork"
)

// DefaultPathEnv is deliberately empty on Windows as the default path will be set by
// the container. Docker has no context of what the default path should be.
const DefaultPathEnv = ""

// Container holds fields specific to the Windows implementation. See
// CommonContainer for standard fields common to all containers.
type Container struct {
	CommonContainer

	// Fields below here are platform specific.
}

func killProcessDirectly(container *Container) error {
	return nil
}

func (daemon *Daemon) setupLinkedContainers(container *Container) ([]string, error) {
	return nil, nil
}

func (container *Container) createDaemonEnvironment(linkedEnv []string) []string {
	// On Windows, nothing to link. Just return the container environment.
	return container.Config.Env
}

func (daemon *Daemon) initializeNetworking(container *Container) error {
	return nil
}

// ConnectToNetwork connects a container to the network
func (daemon *Daemon) ConnectToNetwork(container *Container, idOrName string) error {
	return nil
}

// DisconnectFromNetwork disconnects a container from, the network
func (container *Container) DisconnectFromNetwork(n libnetwork.Network) error {
	return nil
}

func (container *Container) setupWorkingDirectory() error {
	return nil
}

func (daemon *Daemon) populateCommand(c *Container, env []string) error {
	en := &execdriver.Network{
		Interface: nil,
	}

	parts := strings.SplitN(string(c.hostConfig.NetworkMode), ":", 2)
	switch parts[0] {
	case "none":
	case "default", "": // empty string to support existing containers
		if !c.Config.NetworkDisabled {
			en.Interface = &execdriver.NetworkInterface{
				MacAddress:   c.Config.MacAddress,
				Bridge:       daemon.configStore.Bridge.VirtualSwitchName,
				PortBindings: c.hostConfig.PortBindings,

				// TODO Windows. Include IPAddress. There already is a
				// property IPAddress on execDrive.CommonNetworkInterface,
				// but there is no CLI option in docker to pass through
				// an IPAddress on docker run.
			}
		}
	default:
		return derr.ErrorCodeInvalidNetworkMode.WithArgs(c.hostConfig.NetworkMode)
	}

	// TODO Windows. More resource controls to be implemented later.
	resources := &execdriver.Resources{
		CommonResources: execdriver.CommonResources{
			CPUShares: c.hostConfig.CPUShares,
		},
	}

	// TODO Windows. Further refactoring required (privileged/user)
	processConfig := execdriver.ProcessConfig{
		Privileged:  c.hostConfig.Privileged,
		Entrypoint:  c.Path,
		Arguments:   c.Args,
		Tty:         c.Config.Tty,
		User:        c.Config.User,
		ConsoleSize: c.hostConfig.ConsoleSize,
	}

	processConfig.Env = env

	var layerPaths []string
	img, err := daemon.graph.Get(c.ImageID)
	if err != nil {
		return derr.ErrorCodeGetGraph.WithArgs(c.ImageID, err)
	}
	for i := img; i != nil && err == nil; i, err = daemon.graph.GetParent(i) {
		lp, err := daemon.driver.Get(i.ID, "")
		if err != nil {
			return derr.ErrorCodeGetLayer.WithArgs(daemon.driver.String(), i.ID, err)
		}
		layerPaths = append(layerPaths, lp)
		err = daemon.driver.Put(i.ID)
		if err != nil {
			return derr.ErrorCodePutLayer.WithArgs(daemon.driver.String(), i.ID, err)
		}
	}
	m, err := daemon.driver.GetMetadata(c.ID)
	if err != nil {
		return derr.ErrorCodeGetLayerMetadata.WithArgs(err)
	}
	layerFolder := m["dir"]

	c.command = &execdriver.Command{
		CommonCommand: execdriver.CommonCommand{
			ID:            c.ID,
			Rootfs:        c.rootfsPath(),
			InitPath:      "/.dockerinit",
			WorkingDir:    c.Config.WorkingDir,
			Network:       en,
			MountLabel:    c.getMountLabel(),
			Resources:     resources,
			ProcessConfig: processConfig,
			ProcessLabel:  c.getProcessLabel(),
		},
		FirstStart:  !c.HasBeenStartedBefore,
		LayerFolder: layerFolder,
		LayerPaths:  layerPaths,
		Hostname:    c.Config.Hostname,
		Isolated:    c.hostConfig.Isolation.IsHyperV(),
	}

	return nil
}

// getSize returns real size & virtual size
func (daemon *Daemon) getSize(container *Container) (int64, int64) {
	// TODO Windows
	return 0, 0
}

// setNetworkNamespaceKey is a no-op on Windows.
func (daemon *Daemon) setNetworkNamespaceKey(containerID string, pid int) error {
	return nil
}

// allocateNetwork is a no-op on Windows.
func (daemon *Daemon) allocateNetwork(container *Container) error {
	return nil
}

func (daemon *Daemon) updateNetwork(container *Container) error {
	return nil
}

func (daemon *Daemon) releaseNetwork(container *Container) {
}

// appendNetworkMounts appends any network mounts to the array of mount points passed in.
// Windows does not support network mounts (not to be confused with SMB network mounts), so
// this is a no-op.
func appendNetworkMounts(container *Container, volumeMounts []volume.MountPoint) ([]volume.MountPoint, error) {
	return volumeMounts, nil
}

func (daemon *Daemon) setupIpcDirs(container *Container) error {
	return nil
}

func (container *Container) unmountIpcMounts(unmount func(pth string) error) {
}

func detachMounted(path string) error {
	return nil
}

func (container *Container) ipcMounts() []execdriver.Mount {
	return nil
}

func getDefaultRouteMtu() (int, error) {
	return -1, errSystemNotSupported
}
