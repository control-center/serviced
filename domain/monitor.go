package domain

//MonitorProfile describes metrics, thresholds and graphs to monitor an entity's performance
type MonitorProfile struct {
	Metrics []MetricConfig
	//TODO Thresholds
	//TODO Graphs
}

//Equals equality test for Monitor
func (profile *MonitorProfile) Equals(that *MonitorProfile) bool {
	if profile.Metrics == nil && that.Metrics == nil {
		return true
	}

	if profile.Metrics == nil && that.Metrics != nil {
		return false
	}

	if profile.Metrics != nil && that.Metrics == nil {
		return false
	}

	if len(profile.Metrics) != len(that.Metrics) {
		return false
	}

	for i := range profile.Metrics {
		metric := &profile.Metrics[i]
		if !metric.Equals(&that.Metrics[i]) {
			return false
		}
	}

	return true
}
