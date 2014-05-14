package nfs

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"syscall"

	"github.com/zenoss/serviced/commons/atomicfile"
)

// Server manages exporting an NFS mount.
type Server struct {
	basePath      string
	exportedName  string
	exportOptions string
	network       string
	clients       map[string]struct{}
}

var (
	ErrInvalidExportedName = errors.New("nfs server: invalid exported name")
	ErrInvalidBasePath     = errors.New("nfs server: invalid base path")
	ErrBasePathNotDir      = errors.New("nfs server: base path not a directory")
	ErrInvalidNetwork      = errors.New("nfs server: the network value is not CIDR")
)

var (
	exportsPath = "/exports"
)

var (
	osMkdirAll = os.MkdirAll
	osChmod    = os.Chmod
)

const defaultDirectoryPerm = 0755

const hostDenyMarker = "# serviced, do not remove past this line"
const hostDenyDefaults = "\n# serviced, do not remove past this line\nrpcbind mountd nfsd statd lockd rquotad : ALL\n"

const hostAllowMarker = "# serviced, do not remove past this line"
const hostAllowDefaults = "\n# serviced, do not remove past this line\nrpcbind mountd nfsd statd lockd rquotad : 127.0.0.1"

func verifyExportsDir(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		//handle does not exist
		return osMkdirAll(path, defaultDirectoryPerm)
	}
	if !stat.IsDir() {
		return ErrBasePathNotDir
	}
	if (stat.Mode() & defaultDirectoryPerm) != defaultDirectoryPerm {
		err = osChmod(path, defaultDirectoryPerm)
	}
	return err
}

func NewServer(basePath, exportedName, network string) (*Server, error) {

	if len(exportedName) < 2 || strings.Contains(exportedName, "/") {
		return nil, ErrInvalidExportedName
	}
	if len(basePath) < 2 {
		return nil, ErrInvalidBasePath
	}
	if err := verifyExportsDir(basePath); err != nil {
		return nil, err
	}
	if err := verifyExportsDir(path.Join(exportsPath, exportedName)); err != nil {
		return nil, err
	}

	if _, _, err := net.ParseCIDR(network); err != nil {
		return nil, ErrInvalidNetwork
	}

	return &Server{
		basePath:      basePath,
		exportedName:  exportedName,
		exportOptions: "rw,nohide,insecure,no_subtree_check,async",
		clients:       make(map[string]struct{}),
		network:       network,
	}, nil
}

func (c *Server) Clients() []string {
	clients := make([]string, len(c.clients))
	i := 0
	for key, _ := range c.clients {
		clients[i] = key
	}
	return clients
}
func (c *Server) SetClients(clients ...string) {
	c.clients = make(map[string]struct{})

	for _, client := range clients {
		c.clients[client] = struct{}{}
	}
}

func (c *Server) Sync() error {
	if err := c.hostsDeny(); err != nil {
		return err
	}
	if err := c.hostsAllow(); err != nil {
		return err
	}
	if err := reload(); err != nil {
		return err
	}
	if err := c.writeExports(); err != nil {
		return err
	}
	return nil
}

func (c *Server) hostsDeny() error {

	s, err := readFileIfExists(hostDenyDefaults)
	if err != nil {
		return err
	}
	if strings.Contains(s, hostDenyDefaults) {
		return nil
	}

	if index := strings.Index(s, hostDenyMarker); index > 0 {
		s = s[:index-1]
	}
	s = s + hostDenyDefaults
	return atomicfile.WriteFile(etcHostsDeny, []byte(s), 0664)
}

func readFileIfExists(path string) (string, error) {
	s := ""
	if exists, err := doesExists(path); err != nil {
		return s, err
	} else {
		if exists {
			bytes, err := ioutil.ReadFile(path)
			if err != nil {
				return s, err
			}
			s = string(bytes)
		}
	}
	return s, nil
}

func (c *Server) hostsAllow() error {
	s, err := readFileIfExists(etcHostsAllow)
	if err != nil {
		return err
	}

	if index := strings.Index(s, hostAllowMarker); index > 0 {
		s = s[:index-1]
	}

	s = s + hostAllowDefaults
	hosts := make([]string, len(c.clients))
	i := 0
	for key, _ := range c.clients {
		hosts[i] = key
		i += 1
	}
	sort.Strings(hosts)
	for _, h := range hosts {
		s = s + " " + h
	}

	return atomicfile.WriteFile(etcHostsAllow, []byte(s), 0664)
}

func (c *Server) writeExports() error {
	s := fmt.Sprintf("/export\t%s(rw,fsid=0,insecure,no_subtree_check,async)\n"+
		"/export/%s\t%s(rw,nohide,insecure,no_subtree_check,async)",
		c.network, c.exportedName, c.network)
	if err := os.MkdirAll(exportsDir, 0775); err != nil {
		return err
	}
	edir := exportsDir + "/" + c.exportedName
	if err := os.MkdirAll(edir, 0775); err != nil {
		return err
	}
	if err := bindMount(c.basePath, edir); err != nil {
		return err
	}
	return atomicfile.WriteFile(etcExports, []byte(s), 0664)
}

type bindMountF func(string, string) error

var bindMount bindMountF = bindMountImp

// bindMountImp performs a bind mount of src to dst.
func bindMountImp(src, dst string) error {
	cmd, args := mntArgs(src, dst, "", "bind")
	mount := exec.Command(cmd, args...)
	return mount.Run()
}

func doesExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// mntArgs computes the required arguments for the mount command given
// fs (src fs), dst (mount point), fsType (ext3, xfs, etc), options (parameters
// passed to the -o flag).
func mntArgs(fs, dst, fsType, options string) (cmd string, args []string) {
	args = make([]string, 0)
	if syscall.Getuid() != 0 {
		args = append(args, "sudo")
	}
	args = append(args, "mount")
	if len(options) > 0 {
		args = append(args, "-o")
		args = append(args, options)
	}
	if len(fsType) > 0 {
		args = append(args, "-t")
		args = append(args, fsType)
	}
	if len(fs) > 0 {
		args = append(args, fs)
	}
	args = append(args, dst)
	return args[0], args[1:]
}
