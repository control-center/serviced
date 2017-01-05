package statistics_test

import (
	"fmt"
	"math"
	"testing"

	. "github.com/control-center/serviced/commons/statistics"
	. "gopkg.in/check.v1"
)

type withToleranceChecker struct {
	*CheckerInfo
	tolerance float64
}

func (checker *withToleranceChecker) Check(params []interface{}, names []string) (result bool, error string) {
	defer func() {
		if v := recover(); v != nil {
			result = false
			error = fmt.Sprint(v)
		}
	}()

	obtained, ok := params[0].(float64)
	expected, ok2 := params[1].(float64)

	if !ok || !ok2 {
		return false, "Obtained and expected values must be of type float64"
	}

	diff := math.Abs(obtained - expected)
	maxdiff := checker.tolerance * math.Max(1.0, math.Max(math.Abs(obtained), math.Abs(expected)))

	return diff <= maxdiff, ""
}

func EqualsWithTolerance(tolerance float64) Checker {
	return &withToleranceChecker{
		&CheckerInfo{Name: "EqualsWithTolerance", Params: []string{"obtained", "expected"}},
		tolerance,
	}
}

var RoughlyEquals = EqualsWithTolerance(0.0000000001)

func Test(t *testing.T) { TestingT(t) }

type StatisticsSuite struct{}

var _ = Suite(&StatisticsSuite{})

var zeroFloat float64

// ycoords is a utility func to generate the series of y-coordinates from
// a series of x-coordinates based on a slope and y-intercept provided
func ycoords(xs []float64, m, b float64) (ys []float64) {
	for _, x := range xs {
		ys = append(ys, x*m+b)
	}
	return
}

func (s *StatisticsSuite) TestLeastSquares(c *C) {
	// Test empty series produces zeroes
	m, b := LeastSquares([]float64{}, []float64{})
	c.Assert(m, Equals, zeroFloat)
	c.Assert(b, Equals, zeroFloat)

	// Test fit against a straight line. It should match the original line.
	m1 := 0.2
	b1 := 4.0
	xs := []float64{-1, 0, 1, 2, 3}
	ys := ycoords(xs, m1, b1)
	m, b = LeastSquares(xs, ys)
	c.Assert(m, RoughlyEquals, m1)
	c.Assert(b, RoughlyEquals, b1)

	// Test that a noisy series matches the answer produced by numpy.linalg.lstsq
	xs1 := []float64{0, 1, 2, 3}
	ys1 := []float64{-1, 0.2, 0.9, 2.1}
	m, b = LeastSquares(xs1, ys1)
	c.Assert(m, RoughlyEquals, float64(1))
	c.Assert(b, RoughlyEquals, -0.95)
}
