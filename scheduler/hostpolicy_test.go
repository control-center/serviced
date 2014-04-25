package scheduler

import (
	"testing"

	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
)

var (
	least, middlest, most *host.Host
	unprioritized         []*host.Host
	prioritized           []*host.Host
	hoststates            map[*host.Host][]string
	testinfo              *TestHostInfo
)

func BeforeEach() {
	least = &host.Host{ID: "least"}
	middlest = &host.Host{ID: "middlest"}
	most = &host.Host{ID: "most"}
	prioritized = []*host.Host{most, middlest, least}
	unprioritized = []*host.Host{least, middlest, most}
	hoststates = map[*host.Host][]string{least: []string{}, most: []string{}, middlest: []string{}}
	testinfo = &TestHostInfo{prioritized, hoststates}
}

// First we stub out the HostInfo to return static data
type TestHostInfo struct {
	prioritized []*host.Host
	services    map[*host.Host][]string
}

// Just satisfy the interface; we're prioritizing explicitly in the test
func (t *TestHostInfo) AvailableRAM(h *host.Host, c chan *hostitem, d <-chan bool) {}

// Return the list of hosts prioritized with no modification (ignore what's passed in)
func (t *TestHostInfo) PrioritizeByMemory(hosts []*host.Host) ([]*host.Host, error) {
	return t.prioritized, nil
}

// Don't go to ZooKeeper, just look at our local manually constructed service state.
func (t *TestHostInfo) ServicesOnHost(h *host.Host) []*dao.RunningService {
	result := []*dao.RunningService{}
	for _, s := range t.services[h] {
		result = append(result, &dao.RunningService{ServiceId: s})
	}
	return result
}

func (t *TestHostInfo) addServiceToHost(svc *dao.Service, h *host.Host) {
	t.services[h] = append(t.services[h], svc.Id)
}

func TestLeastCommitted(t *testing.T) {
	BeforeEach()
	svc := dao.Service{HostPolicy: dao.LEAST_COMMITTED}
	policy := ServiceHostPolicy{&svc, testinfo}
	if h, _ := policy.SelectHost(unprioritized); h != most {
		t.Fatalf("Expected most host but got %s", h.ID)
	}

}

func TestPreferSeparate(t *testing.T) {
	BeforeEach()
	svc := dao.Service{HostPolicy: dao.PREFER_SEPARATE}
	policy := ServiceHostPolicy{&svc, testinfo}

	testinfo.addServiceToHost(&svc, most)
	if h, _ := policy.SelectHost(unprioritized); h != middlest {
		t.Fatalf("Expected middlest host but got %s", h.ID)
	}

	testinfo.addServiceToHost(&svc, middlest)
	if h, _ := policy.SelectHost(unprioritized); h != least {
		t.Fatalf("Expected least host but got %s", h.ID)
	}

	// Start on all hosts and make sure it rolls around to most available
	testinfo.addServiceToHost(&svc, least)
	if h, _ := policy.SelectHost(unprioritized); h != most {
		t.Fatalf("Expected most host but got %s", h.ID)
	}
}

func TestRequireSeparate(t *testing.T) {
	BeforeEach()
	svc := dao.Service{HostPolicy: dao.REQUIRE_SEPARATE}
	policy := ServiceHostPolicy{&svc, testinfo}

	testinfo.addServiceToHost(&svc, most)
	if h, _ := policy.SelectHost(unprioritized); h != middlest {
		t.Fatalf("Expected middlest host but got %s", h.ID)
	}

	testinfo.addServiceToHost(&svc, middlest)
	if h, _ := policy.SelectHost(unprioritized); h != least {
		t.Fatalf("Expected least host but got %s", h.ID)
	}

	// Start on all hosts and make sure it fails to find a host
	testinfo.addServiceToHost(&svc, least)
	if _, err := policy.SelectHost(unprioritized); err == nil {
		t.Fatalf("Should have received an error but didn't")
	}
}
