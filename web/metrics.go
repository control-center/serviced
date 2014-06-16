package web

//metric defines metrics for hosts or pools
type metric struct {
	ID          string
	Name        string
	Description string
}

var (
	metrics = []metric{
		metric{
			"CpuacctStat.system",
			"CPU System",
			"System CPU Usage",
		},
		metric{
			"CpuacctStat.user",
			"CPU User",
			"User CPU Usage",
		},
		metric{
			"MemoryStat.pgfault",
			"Memory Page Fault",
			"Page Fault Stats",
		},
		metric{
			"MemoryStat.rss",
			"Resident Memory",
			"Resident Memory Usage",
		},
	}
)
