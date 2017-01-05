package statistics_test

import (
	"time"

	. "github.com/control-center/serviced/commons/statistics"
	. "gopkg.in/check.v1"
)

func (s *StatisticsSuite) TestLeastSquaresPredictor(c *C) {
	var m, b, now float64

	now = float64(time.Now().UTC().Unix())
	ts := []float64{now - 10, now - 8, now - 6, now - 4, now - 2}

	// A constant value
	m, b = 0, 4
	val := LeastSquaresPredictor.Predict(20*time.Minute, ts, ycoords(ts, m, b))
	c.Assert(val, RoughlyEquals, 4.0)

	// A line with slope 1 (x=y)
	m, b = 1, 0
	val = LeastSquaresPredictor.Predict(time.Minute, ts, ycoords(ts, m, b))
	c.Assert(val, RoughlyEquals, now+60)
}
