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
	val, err := LeastSquaresPredictor.Predict(20*time.Minute, ts, ycoords(ts, m, b))
	c.Assert(val, RoughlyEquals, 4.0)
	c.Assert(err, IsNil)

	// A line with slope 1 (x=y)
	m, b = 1, 0
	val, err = LeastSquaresPredictor.Predict(time.Minute, ts, ycoords(ts, m, b))
	c.Assert(val, RoughlyEquals, now+60)
	c.Assert(err, IsNil)

	// Insufficient data
	ts2 := []float64{}
	val, err = LeastSquaresPredictor.Predict(time.Minute, ts2, ts2)
	c.Assert(val, Equals, zeroFloat)
	c.Assert(err, Equals, ErrInsufficientData)
}
