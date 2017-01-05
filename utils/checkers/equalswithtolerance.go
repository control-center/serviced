package checkers

import (
	"fmt"
	"math"

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
