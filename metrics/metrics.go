package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/logging"
	gometrics "github.com/rcrowley/go-metrics"
	"github.com/zenoss/logri"
)

var (
	log = logging.PackageLogger()
)

// Metrics stores metric data.
// To record metrics, enable metrics for a function with ctx.Metrics().Enabled = true
// and: defer ctx.Metrics().Stop(ctx.Metrics().Start("FunctionName")) where
// you want to time some code through the end of the method. If you want to time
// specific parts of a function, you can break it out the Start/Stop into two calls.
// Call the Log() method to capture the timing information and clear the data.  For
// running logs use the go-metrics Log() method, passing in the Metrics.Registy object.
type Metrics struct {
	sync.Mutex
	Enabled   bool
	Registry  gometrics.Registry
	Timers    map[string]gometrics.Timer
	GroupName string
}

// NewMetrics returns a new Metrics object.
func NewMetrics() *Metrics {
	return &Metrics{
		Registry: gometrics.NewRegistry(), // Keep these metrics separate from others in the app
		Timers:   make(map[string]gometrics.Timer),
	}
}

// MetricTimer represents a named timer.
type MetricTimer struct {
	Name  string
	Timer gometrics.Timer
	Time  time.Time
}

// Start returns a new timing object.
// This will be used as an argument to Stop() to record the duration/count.
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

// Stop calculates the duration.
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

// Log the current timers.  Turns off metric logging and clears
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
				"count":     t.Count(),
				"sum":       fmt.Sprintf("%.4f", float64(t.Sum())/du),
				"min":       fmt.Sprintf("%.4f", float64(t.Min())/du),
				"max":       fmt.Sprintf("%.4f", float64(t.Max())/du),
				"mean":      fmt.Sprintf("%.4f", t.Mean()/du),
				"stddev":    fmt.Sprintf("%.4f", t.StdDev()/du),
				"median":    fmt.Sprintf("%.4f", ps[0]/du),
				"units":     units,
				"groupname": m.GroupName,
			}).Debug(name)
		}
	})

	// Disable and clear all metrics.
	m.Enabled = false
	r.UnregisterAll()
	m.Timers = make(map[string]gometrics.Timer)
}

// LogAndCleanUp is used in a defer call on methods for which metric logging is desired.
// To write metrics for a method invocation to the log, add the following at the top of the method:
//
//   ctx.Metrics().Enabled = true
//   defer ctx.Metrics().LogAndCleanUp(ctx.Metrics().Start("methodname"))
//
// if Enabled is true, the metrics will be gathered and written at the end of the method.
// if Enabled is false, this will gather metrics for the method, but only report them if the
// method is called by another method with metrics enabled. I.E. it should behave similarly to
// 'defer <metrics>.Stop(<metrics>.Start("methodname"))'
// It is not necessary to reset Metrics().Enabled to false, as the Log() method does so before
// exiting.
func (m *Metrics) LogAndCleanUp(ssTimer *MetricTimer) {
	m.Stop(ssTimer)
	if m.Enabled {
		metricsLogger := logri.GetLogger("metrics")
		// FIXME: this is temporary - remove once log level configuration is available via logger
		saveLevel := metricsLogger.GetEffectiveLevel()
		// FIXME: this is temporary - remove once log level configuration is available via logger
		metricsLogger.SetLevel(logrus.DebugLevel, true)
		m.Log()
		// FIXME: this is temporary - remove once log level configuration is available via logger
		metricsLogger.SetLevel(saveLevel, false)
	}
}
