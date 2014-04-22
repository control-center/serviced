package api

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/zenoss/serviced/dao"
)

// Handles URL data
type URL struct {
	Host string
	Port string
}

func (u *URL) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("bad format: %s; must be formatted as HOST:PORT", value)
	}

	(*u).Host = parts[0]
	(*u).Port = parts[1]
	return nil
}

func (u *URL) String() string {
	return fmt.Sprintf("%s:%s", u.Host, u.Port)
}

// Mapping of docker image data
type ImageMap map[string]string

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

// Mapping of port data
type PortMap map[string]dao.ServiceEndpoint

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
	(*m)[port] = dao.ServiceEndpoint{Protocol: protocol, PortNumber: uint16(portnum), Application: portname}
	return nil
}

func (m *PortMap) String() string {
	var mapping []string
	for _, v := range *m {
		mapping = append(mapping, fmt.Sprintf("%s:%s:%s", v.Protocol, v.PortNumber, v.Application))
	}
	return strings.Join(mapping, " ")
}

// Tree Mapping of service data
type ServiceMap map[string][]*dao.Service

// Creates a new service map from a slice of services
func NewServiceMap(services *[]dao.Service) ServiceMap {
	var m = make(ServiceMap)
	for _, s := range *services {
		m.Add(s)
	}
	return m
}

// Appends a service to the service map
func (m *ServiceMap) Add(service dao.Service) {
	list := (*m)[service.ParentServiceId]
	(*m)[service.ParentServiceId] = append(list, &service)
}

// Gets the services by parent id
func (m *ServiceMap) Get(parentId string) []*dao.Service {
	ss := NewServiceSlice(m.getDepths())
	ss.services = (*m)[parentId]

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

// A list of service pointers
type ServiceSlice struct {
	depths   map[string]int
	services []*dao.Service
}

func NewServiceSlice(depths map[string]int) *ServiceSlice {
	return &ServiceSlice{depths: depths}
}

func (s *ServiceSlice) Append(service *dao.Service) {
	s.services = append(s.services, service)
}

func (s *ServiceSlice) Get(i int) *dao.Service {
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