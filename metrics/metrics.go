package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/logging"
	gometrics "github.com/rcrowley/go-metrics"
)

var (
	log = logging.PackageLogger()
)

/*
 * To record metrics, enable metrics for a function with ctx.Metrics().Enabled = true
 * and: defer ctx.Metrics().Stop(ctx.Metrics().Start("FunctionName")) where
 * you want to time some code through the end of the method. If you want to time
 * specific parts of a function, you can break it out the Start/Stop into two calls.
 * Call the Log() method to capture the timing information and clear the data.  For
 * running logs use the go-metrics Log() method, passing in the Metrics.Registy object.
 */
type Metrics struct {
	sync.Mutex
	Enabled  bool
	Registry gometrics.Registry
	Timers   map[string]gometrics.Timer
}

func NewMetrics() *Metrics {
	return &Metrics{
		Registry: gometrics.NewRegistry(), // Keep these metrics separate from others in the app
		Timers:   make(map[string]gometrics.Timer),
	}
}

type MetricTimer struct {
	Name  string
	Timer gometrics.Timer
	Time  time.Time
}

// Returns a new timing object.  This will be used as an
// argument to Stop() to record the duration/count.
func (m *Metrics) Start(name string) *MetricTimer {
	if !m.Enabled {
		return nil
	}
	m.Lock()
	defer m.Unlock()

	timer, found := m.Timers[name]
	if !found {
		timer = gometrics.NewTimer()
		m.Timers[name] = timer
		m.Registry.Register(name, timer)
	}
	return &MetricTimer{Name: name, Timer: timer, Time: time.Now()}
}

// When stop is called, calculate the duration.
func (m *Metrics) Stop(timer *MetricTimer) {
	if timer != nil {
		timer.Timer.UpdateSince(timer.Time)
	}
}

// Pads the value with units to a given width.
// padUnits(14, 0.22, 2, "µs") = "0.22µs        "
func padUnits(width int, value float64, precision int, units string) string {
	format1 := fmt.Sprintf("%%-%ds", width)
	format2 := fmt.Sprintf("%%.%df%%s", precision)
	return fmt.Sprintf(format1, fmt.Sprintf(format2, value, units))
}

// Log the current timers.  Turns off metric loggina and clears
// the metric data. Note that if we want a running tally we can
// use the go-metric log method directly, providing our registry.
func (m *Metrics) Log() {
	m.Lock()
	defer m.Unlock()

	log.Debug("Logging all timers")

	scale := time.Second
	du := float64(scale)
	units := scale.String()[1:]

	r := m.Registry

	r.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		// Other types/metrics shown in https://github.com/rcrowley/go-metrics/blob/master/log.go#L21
		case gometrics.Timer:
			t := metric.Snapshot()
			ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
			log.WithFields(logrus.Fields{
				"count":  t.Count(),
				"sum":    padUnits(14, float64(t.Sum())/du, 2, units),
				"min":    padUnits(14, float64(t.Min())/du, 2, units),
				"max":    padUnits(14, float64(t.Max())/du, 2, units),
				"mean":   padUnits(14, t.Mean()/du, 4, units),
				"stddev": padUnits(14, t.StdDev()/du, 2, units),
				"median": padUnits(14, ps[0]/du, 4, units),
			}).Debug(name)
		}
	})

	// Disable and clear all metrics.
	m.Enabled = false
	r.UnregisterAll()
	m.Timers = make(map[string]gometrics.Timer)
}
