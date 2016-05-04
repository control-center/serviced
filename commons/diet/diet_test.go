package diet_test

import (
	"fmt"
	"reflect"
	"testing"

	. "github.com/control-center/serviced/commons/diet"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type DietSuite struct{}

var _ = Suite(&DietSuite{})

type equivalentChecker struct {
	*CheckerInfo
}

func (checker *equivalentChecker) Check(params []interface{}, names []string) (result bool, error string) {
	defer func() {
		if v := recover(); v != nil {
			result = false
			error = fmt.Sprint(v)
		}
	}()
	obtained, expected := params[0], params[1]
	if obtained == nil && expected == nil {
		return false, "Obtained and expected values must not be nil"
	}
	converted := obtained
	if obtained != nil && expected != nil && reflect.TypeOf(obtained).ConvertibleTo(reflect.TypeOf(expected)) {
		converted = reflect.ValueOf(obtained).Convert(reflect.TypeOf(expected)).Interface()
	}
	return reflect.DeepEqual(converted, expected), ""
}

var EquivalentTo Checker = &equivalentChecker{
	&CheckerInfo{Name: "EquivalentTo", Params: []string{"obtained", "expected"}},
}

// Test that Contains() for a range returns the correct values
func (s *DietSuite) TestContainsAfterInsert(c *C) {
	d := NewDiet()
	d.Insert(10, 20)
	c.Assert(d.Contains(10, 20), Equals, true)
	c.Assert(d.Contains(15, 15), Equals, true)
	c.Assert(d.Contains(15, 18), Equals, true)
	c.Assert(d.Contains(9, 19), Equals, false)
	c.Assert(d.Contains(0, 5), Equals, false)

	d.Insert(5, 9)
	c.Assert(d.Contains(0, 5), Equals, false)
	c.Assert(d.Contains(9, 19), Equals, true)
}

// Test that Intersection() for a range returns the correct values
func (s *DietSuite) TestSimpleIntersectionAfterInsert(c *C) {
	d := NewDiet()
	d.Insert(10, 20)
	c.Assert(d.Intersection(10, 20), EquivalentTo, 11)
	c.Assert(d.Intersection(20, 10), EquivalentTo, 11)
	c.Assert(d.Intersection(9, 9), EquivalentTo, 0)
	c.Assert(d.Intersection(9, 10), EquivalentTo, 1)

	c.Assert(d.Total(), EquivalentTo, 11)
}

// Test that inserting a range in reverse order produces the same result
func (s *DietSuite) TestInsertReversedRange(c *C) {
	d := NewDiet()
	d.Insert(20, 10)
	c.Assert(d.Intersection(10, 20), EquivalentTo, 11)
	c.Assert(d.Contains(20, 20), Equals, true)
	c.Assert(d.Contains(8, 9), Equals, false)
}

// Test that Total and Intersection work across two nonadjacent ranges
func (s *DietSuite) TestIntersectionNonAdjacentRanges(c *C) {
	d := NewDiet()
	d.Insert(1, 5)
	d.Insert(11, 15)
	c.Assert(d.Intersection(1, 15), EquivalentTo, 10)
	c.Assert(d.Intersection(5, 11), EquivalentTo, 2)
	c.Assert(d.Intersection(6, 10), EquivalentTo, 0)
	c.Assert(d.Intersection(6, 11), EquivalentTo, 1)
	c.Assert(d.Intersection(6, 20), EquivalentTo, 5)

	c.Assert(d.Total(), EquivalentTo, 10)
}

func (s *DietSuite) TestInsertAlreadyMembers(c *C) {
	d := NewDiet()
	d.Insert(1, 5)
	d.Insert(1, 5)
	d.Insert(4, 5)
	d.Insert(1, 4)
	c.Assert(d.Total(), EquivalentTo, 5)
	c.Assert(d.Intersection(1, 5), EquivalentTo, 5)
}

func (s *DietSuite) TestInsertLeftNoOverlap(c *C) {
	d := NewDiet()
	d.Insert(1, 5)
	d.Insert(11, 15)
	c.Assert(d.Contains(1, 15), Equals, false)
	c.Assert(d.Intersection(1, 15), EquivalentTo, 10)
	c.Assert(d.Total(), EquivalentTo, 10)

}

func (s *DietSuite) TestInsertLeftAdjacent(c *C) {
	d := NewDiet()
	d.Insert(5, 10)
	d.Insert(1, 4)
	c.Assert(d.Contains(1, 10), Equals, true)
	c.Assert(d.Intersection(1, 10), EquivalentTo, 10)
}

func (s *DietSuite) TestInsertLeftOverlap(c *C) {
	d := NewDiet()
	d.Insert(5, 10)
	d.Insert(1, 7)
	c.Assert(d.Contains(1, 10), Equals, true)
	c.Assert(d.Intersection(1, 10), EquivalentTo, 10)
	c.Assert(d.Total(), EquivalentTo, 10)
}

func (s *DietSuite) TestInsertRightNoOverlap(c *C) {
	d := NewDiet()
	d.Insert(11, 15)
	d.Insert(1, 5)
	c.Assert(d.Contains(1, 15), Equals, false)
	c.Assert(d.Intersection(1, 15), EquivalentTo, 10)
	c.Assert(d.Total(), EquivalentTo, 10)

}

func (s *DietSuite) TestInsertRightAdjacent(c *C) {
	d := NewDiet()
	d.Insert(6, 10)
	d.Insert(11, 15)
	c.Assert(d.Contains(6, 15), Equals, true)
	c.Assert(d.Intersection(6, 15), EquivalentTo, 10)
	c.Assert(d.Total(), EquivalentTo, 10)
}

func (s *DietSuite) TestInsertRightOverlap(c *C) {
	d := NewDiet()
	d.Insert(6, 10)
	d.Insert(8, 15)
	c.Assert(d.Contains(6, 15), Equals, true)
	c.Assert(d.Intersection(6, 15), EquivalentTo, 10)
	c.Assert(d.Total(), EquivalentTo, 10)
}

func (s *DietSuite) TestInsertOverlapBoth(c *C) {
	d := NewDiet()
	d.Insert(6, 10)
	d.Insert(1, 15)
	c.Assert(d.Contains(1, 15), Equals, true)
	c.Assert(d.Intersection(1, 15), EquivalentTo, 15)
	c.Assert(d.Total(), EquivalentTo, 15)
}

func (s *DietSuite) TestIntersectionAll(c *C) {
	d1 := NewDiet()
	d1.Insert(1, 5)
	d1.Insert(11, 15)
	d1.Insert(21, 25)
	d1.Insert(31, 35)
	d1.Insert(41, 45)

	d1.Balance()

	d2 := NewDiet()
	d2.Insert(6, 30)

	c.Assert(d1.IntersectionAll(d2), EquivalentTo, 10)
}
