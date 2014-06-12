package domain

//MonitorProfile describes metrics, thresholds and graphs to monitor an entity's performance
type MonitorProfile struct {
	MetricConfigs []MetricConfig
	//TODO Thresholds
	//TODO Graphs
}

//Equals equality test for Monitor
func (profile *MonitorProfile) Equals(that *MonitorProfile) bool {
	if profile.MetricConfigs == nil && that.MetricConfigs == nil {
		return true
	}

	if profile.MetricConfigs == nil && that.MetricConfigs != nil {
		return false
	}

	if profile.MetricConfigs != nil && that.MetricConfigs == nil {
		return false
	}

	if len(profile.MetricConfigs) != len(that.MetricConfigs) {
		return false
	}

	for i := range profile.MetricConfigs {
		metric := &profile.MetricConfigs[i]
		if !metric.Equals(&that.MetricConfigs[i]) {
			return false
		}
	}

	return true
}
