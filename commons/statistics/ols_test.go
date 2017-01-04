package statistics_test

import (
	"testing"

	. "github.com/control-center/serviced/commons/statistics"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type StatisticsSuite struct{}

var _ = Suite(&StatisticsSuite{})

var zeroFloat float64

type testPoint struct {
	x, y float64
}

func (t testPoint) X() float64 {
	return t.x
}

func (t testPoint) Y() float64 {
	return t.y
}

var _ Point = testPoint{}

func (s *StatisticsSuite) TestLeastSquares(c *C) {

	// Test empty series produces zeroes
	m, b := LeastSquares([]Point{})
	c.Assert(m, Equals, zeroFloat)
	c.Assert(b, Equals, zeroFloat)

	// Test fit against a straight line
	m1 := 0.2
	b1 := 4.0
	line := []Point{
		testPoint{0, 0*m1 + b1},
		testPoint{1, 1*m1 + b1},
		testPoint{2, 2*m1 + b1},
		testPoint{3, 3*m1 + b1},
	}
	m, b = LeastSquares(line)
	c.Assert(m, Equals, m1)
	c.Assert(b, Equals, b1)

	// Test that a noisy series matches the answer produced by numpy.linalg.lstsq
	points := []Point{
		testPoint{0, -1},
		testPoint{1, 0.2},
		testPoint{2, 0.9},
		testPoint{3, 2.1},
	}
	m, b = LeastSquares(points)
	c.Assert(m, Equals, float64(1))
	c.Assert(b, Equals, -0.95)
}
