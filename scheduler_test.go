package serviced

import (
	"math"
	"testing"

	"github.com/zenoss/serviced/dao"
)

var hosts []*dao.PoolHost = []*dao.PoolHost{
	&dao.PoolHost{"H1", "P", "0.0.0.0", 1024},
	&dao.PoolHost{"H2", "P", "1.1.1.1", 2048},
	&dao.PoolHost{"H3", "P", "2.2.2.2", 1024},
	&dao.PoolHost{"H4", "P", "3.3.3.3", 4096},
}

func TestMemcap(t *testing.T) {
	if mc := memcap(hosts); mc != 8192 {
		t.Fatalf("Expecting 8192 got %d", mc)
	}
}

func TestWeightedRandomChoice(t *testing.T) {
	choices := make(map[string]int)
	for i := 0; i < 1000000; i++ {
		h := weightedRandomChoice(hosts)
		if _, ok := choices[h.HostId]; ok == false {
			choices[h.HostId] = 0
		} else {
			choices[h.HostId]++
		}
	}

	if choices["H4"] < choices["H1"] || choices["H4"] < choices["H2"] || choices["H4"] < choices["H3"] {
		t.Fatal("H4 did not get the most weight")
	}

	if choices["H2"] < choices["H1"] {
		t.Fatal("H2 should have more weight than H1")
	}

	if choices["H2"] < choices["H3"] {
		t.Fatal("H2 should have more weight than H3")
	}

	delta := math.Abs(float64(choices["H1"] - choices["H3"]))
	if delta/float64(choices["H1"]) > 0.1 {
		t.Fatal("Percent difference between H1 and H3 is > 1%")
	}
}
