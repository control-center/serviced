package serviced

import (
	"testing"

	"github.com/zenoss/serviced/dao"
)

var (
	hostone   = dao.PoolHost{"a", "p", "0.0.0.0"}
	hosttwo   = dao.PoolHost{"b", "p", "1.1.1.1"}
	hostthree = dao.PoolHost{"c", "p", "2.2.2.2"}
)

var addressAssignmentTests = []struct {
	addressAssignments []*dao.AddressAssignment
	pool               []*dao.PoolHost
	expectedPoolHost   *dao.PoolHost
	expectingError     bool
}{
	{
		// one valid address assignment
		[]*dao.AddressAssignment{
			&dao.AddressAssignment{
				"123",
				"static",
				"a",
				"",
				"11.11.11.11",
				123,
				"test",
				"test",
			},
		},
		[]*dao.PoolHost{&hostone, &hosttwo, &hostthree},
		&hostone,
		false,
	},
	{
		// one invalid address assignment, address assigned to host not in pool
		[]*dao.AddressAssignment{
			&dao.AddressAssignment{
				"123",
				"static",
				"z",
				"",
				"22.22.22.22",
				123,
				"test",
				"test",
			},
		},
		[]*dao.PoolHost{&hostone, &hosttwo, &hostthree},
		nil,
		true,
	},
	{
		// multiple valid address assignments
		[]*dao.AddressAssignment{
			&dao.AddressAssignment{
				"123",
				"static",
				"b",
				"",
				"11.11.11.11",
				123,
				"test",
				"test",
			},
			&dao.AddressAssignment{
				"456",
				"static",
				"b",
				"",
				"22.22.22.22",
				456,
				"test",
				"test",
			},
		},
		[]*dao.PoolHost{&hostone, &hosttwo, &hostthree},
		&hosttwo,
		false,
	},
	{
		// multiple address assignments, addresses assigned to multiple hosts
		[]*dao.AddressAssignment{
			&dao.AddressAssignment{
				"123",
				"static",
				"b",
				"",
				"11.11.11.11",
				123,
				"test",
				"test",
			},
			&dao.AddressAssignment{
				"456",
				"static",
				"c",
				"",
				"22.22.22.22",
				456,
				"test",
				"test",
			},
			&dao.AddressAssignment{
				"789",
				"static",
				"b",
				"",
				"33.33.33.33",
				789,
				"test",
				"test",
			},
		},
		[]*dao.PoolHost{&hostone, &hosttwo, &hostthree},
		nil,
		true,
	},
	{
		// multiple address assignements, address assigned to host not in pool
		[]*dao.AddressAssignment{
			&dao.AddressAssignment{
				"123",
				"static",
				"b",
				"",
				"11.11.11.11",
				123,
				"test",
				"test",
			},
			&dao.AddressAssignment{
				"456",
				"static",
				"b",
				"",
				"22.22.22.22",
				456,
				"test",
				"test",
			},
			&dao.AddressAssignment{
				"789",
				"static",
				"z",
				"",
				"33.33.33.33",
				789,
				"test",
				"test",
			},
		},
		[]*dao.PoolHost{&hostone, &hosttwo, &hostthree},
		nil,
		true,
	},
	{
		// empty pool
		[]*dao.AddressAssignment{
			&dao.AddressAssignment{
				"123",
				"static",
				"b",
				"",
				"11.11.11.11",
				123,
				"test",
				"test",
			},
			&dao.AddressAssignment{
				"456",
				"static",
				"b",
				"",
				"22.22.22.22",
				456,
				"test",
				"test",
			},
		},
		[]*dao.PoolHost{},
		nil,
		true,
	},
}

func TestPoolHostFromAddressAssignments(t *testing.T) {
	for i, aat := range addressAssignmentTests {
		poolhost, err := poolHostFromAddressAssignments(aat.addressAssignments, aat.pool)
		switch {
		case poolhost != aat.expectedPoolHost:
			t.Errorf("%d, expecting host %v, got %v", i, aat.expectedPoolHost, poolhost)
		case aat.expectingError && (err == nil):
			t.Errorf("%d, expecting error, got nil", i)
		case !aat.expectingError && (err != nil):
			t.Errorf("%d, not expecting error, got %v", i, err)
		default:
			// don't worry, be happy!
		}
	}
}
