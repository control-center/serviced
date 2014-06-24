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
			"cpu.system",
			"CPU System",
			"System CPU Usage",
		},
		metric{
			"cpu.user",
			"CPU User",
			"User CPU Usage",
		},
		metric{
			"cpu.idle",
			"CPU Idle",
			"Idle CPU Usage",
		},
		metric{
			"cpu.iowait",
			"CPU IO Wait",
			"IO Wait CPU Usage",
		},
		metric{
			"cpu.nice",
			"CPU Nice",
			"Nice CPU Usage",
		},
		metric{
			"memory.buffers",
			"Memory Buffer",
			"Memory Buffers Usage",
		},
		metric{
			"memory.cached",
			"Memory Cache",
			"Memory Cache Usage",
		},
		metric{
			"memory.free",
			"Memory Free",
			"Free Memory",
		},
		metric{
			"memory.total",
			"Total Memory",
			"Total Memory",
		},
		metric{
			"memory.used",
			"Used Memory",
			"Used Memory",
		},
	}
)
