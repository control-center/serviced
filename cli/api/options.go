package api

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
)

// URL parses and handles URL typed options
type URL struct {
	Host string
	Port int
}

// Set converts a URL string to a URL object
func (u *URL) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("bad format: %s; must be formatted as HOST:PORT", value)
	}

	u.Host = parts[0]
	if port, err := strconv.Atoi(parts[1]); err != nil {
		return fmt.Errorf("port does not parse as an integer")
	} else {
		u.Port = port
	}
	return nil
}

func (u *URL) String() string {
	return fmt.Sprintf("%s:%d", u.Host, u.Port)
}

// ImageMap parses docker image data
type ImageMap map[string]string

// Set converts a docker image mapping into an ImageMap
func (m *ImageMap) Set(value string) error {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return fmt.Errorf("bad format")
	}

	(*m)[parts[0]] = parts[1]
	return nil
}

func (m *ImageMap) String() string {
	var mapping []string
	for k, v := range *m {
		mapping = append(mapping, k+","+v)
	}

	return strings.Join(mapping, " ")
}

// PortMap parses remote and local port data from the command line
type PortMap map[string]servicedefinition.EndpointDefinition

// Set converts a port mapping string from the command line to a PortMap object
func (m *PortMap) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return fmt.Errorf("bad format: %s; must be PROTOCOL:PORTNUM:PORTNAME", value)
	}
	protocol := parts[0]
	switch protocol {
	case "tcp", "udp":
	default:
		return fmt.Errorf("unsupported protocol: %s (udp|tcp)", protocol)
	}
	portnum, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", parts[1])
	}
	portname := parts[2]
	if portname == "" {
		return fmt.Errorf("port name cannot be empty")
	}
	port := fmt.Sprintf("%s:%d", protocol, portnum)
	(*m)[port] = servicedefinition.EndpointDefinition{Protocol: protocol, PortNumber: uint16(portnum), Application: portname}
	return nil
}

func (m *PortMap) String() string {
	var mapping []string
	for _, v := range *m {
		mapping = append(mapping, fmt.Sprintf("%s:%s:%s", v.Protocol, v.PortNumber, v.Application))
	}
	return strings.Join(mapping, " ")
}

// ServiceMap maps services to their parent
type ServiceMap map[string][]*service.Service

// NewServiceMap creates a new service map from a slice of services
func NewServiceMap(services []*service.Service) ServiceMap {
	var m = make(ServiceMap)
	for i := range services {
		m.Add(services[i])
	}
	return m
}

// Add appends a service to the service map
func (m *ServiceMap) Add(service *service.Service) {
	list := (*m)[service.ParentServiceId]
	(*m)[service.ParentServiceId] = append(list, service)
}

// Get procures services by parent id
func (m *ServiceMap) Get(parentID string) []*service.Service {
	ss := NewServiceSlice(m.getDepths())
	ss.services = (*m)[parentID]

	if !sort.IsSorted(ss) {
		sort.Sort(ss)
	}

	return ss.services
}

func (m *ServiceMap) getDepths() map[string]int {
	var depths = make(map[string]int)
	var setdepth func(string) int
	setdepth = func(id string) int {
		if d, ok := depths[id]; ok {
			return d
		}

		depths[id] = -1
		for _, s := range (*m)[id] {
			d := setdepth(s.Id)
			if d > depths[id] {
				depths[id] = d
			}
		}
		depths[id]++
		return depths[id]
	}

	for id := range *m {
		setdepth(id)
	}

	return depths
}

// ServiceSlice organizes a list of service pointers by depth and alphabetically
type ServiceSlice struct {
	depths   map[string]int
	services []*service.Service
}

// NewServiceSlice initializes a new service slice object
func NewServiceSlice(depths map[string]int) *ServiceSlice {
	return &ServiceSlice{depths: depths}
}

// Add appends a new service to the object
func (s *ServiceSlice) Add(service *service.Service) {
	s.services = append(s.services, service)
}

// Get procures a service by index
func (s *ServiceSlice) Get(i int) *service.Service {
	return s.services[i]
}

func (s *ServiceSlice) Len() int {
	return len(s.services)
}

func (s *ServiceSlice) Less(i, j int) bool {
	serviceI := s.Get(i)
	serviceJ := s.Get(j)

	if s.depths[serviceI.Id] == s.depths[serviceJ.Id] {
		return serviceI.Name < serviceJ.Name
	}

	return s.depths[serviceI.Id] < s.depths[serviceJ.Id]
}

func (s *ServiceSlice) Swap(i, j int) {
	temp := s.Get(i)
	s.services[i] = s.Get(j)
	s.services[j] = temp
}
