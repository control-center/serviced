package statistics_test

import (
	"testing"

	. "github.com/control-center/serviced/commons/statistics"
	"github.com/control-center/serviced/utils/checkers"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type StatisticsSuite struct{}

var (
	_ = Suite(&StatisticsSuite{})

	RoughlyEquals = checkers.EqualsWithTolerance(0.0000000001)
	zeroFloat     float64
)

// ycoords is a utility func to generate the series of y-coordinates from
// a series of x-coordinates based on a slope and y-intercept provided
func ycoords(xs []float64, m, b float64) (ys []float64) {
	for _, x := range xs {
		ys = append(ys, x*m+b)
	}
	return
}

func (s *StatisticsSuite) TestLeastSquares(c *C) {
	// Test empty series produces error
	m, b, err := LeastSquares([]float64{}, []float64{})
	c.Assert(m, Equals, zeroFloat)
	c.Assert(b, Equals, zeroFloat)
	c.Assert(err, Equals, ErrInsufficientData)

	// Test fit against a straight line. It should match the original line.
	m1 := 0.2
	b1 := 4.0
	xs := []float64{-1, 0, 1, 2, 3}
	ys := ycoords(xs, m1, b1)
	m, b, err = LeastSquares(xs, ys)
	c.Assert(m, RoughlyEquals, m1)
	c.Assert(b, RoughlyEquals, b1)
	c.Assert(err, IsNil)

	// Test that a noisy series matches the answer produced by numpy.linalg.lstsq
	xs1 := []float64{0, 1, 2, 3}
	ys1 := []float64{-1, 0.2, 0.9, 2.1}
	m, b, err = LeastSquares(xs1, ys1)
	c.Assert(m, RoughlyEquals, float64(1))
	c.Assert(b, RoughlyEquals, -0.95)
	c.Assert(err, IsNil)
}
