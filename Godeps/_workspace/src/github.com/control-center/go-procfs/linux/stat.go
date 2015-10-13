package linux

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	cpuStatLineUser = iota
	cpuStatLineNice
	cpuStatLineSystem
	cpuStatLineIdle
	cpuStatLineIowait
	cpuStatLineIrq
	cpuStatLineSoftIrq
	cpuStatLineSteal
	cpuStatLineGuest
	cpuStatLineGuestNice
)

var systemHz uint8

func init() {
	systemHz = 100
}

type CpuStat []uint64

// User returns time spent in user mode.
func (c CpuStat) User() uint64 {
	return c[cpuStatLineUser]
}

// Nice returns spent in user mode with low priority (nice).
func (c CpuStat) Nice() uint64 {
	return c[cpuStatLineNice]
}

// System returns time spent in system mode.
func (c CpuStat) System() uint64 {
	return c[cpuStatLineSystem]
}

// Idle returns time spent in idle task.
func (c CpuStat) Idle() uint64 {
	return c[cpuStatLineIdle]
}

// Iowait returns time spent waiting for I/O to complete.
func (c CpuStat) Iowait() uint64 {
	return c[cpuStatLineIowait]
}

// Irq returns time spent servicing all interrupts since boot.
func (c CpuStat) Irq() uint64 {
	return c[cpuStatLineIrq]
}

// Softirq returns time spent serviced all soft interrupts since boot.
func (c CpuStat) Softirq() uint64 {
	return c[cpuStatLineSoftIrq]
}

// StealSupported returns true if the Steal() statistic is present.
func (c CpuStat) StealSupported() bool {
	return cpuStatLineSteal <= len(c)
}

// Steal returns stolen time, which is the time spent in other operating systems when running in a virtualized environment
func (c CpuStat) Steal() uint64 {
	return c[cpuStatLineSteal]
}

// GuestSupported returns true if the Guest() statistic is present.
func (c CpuStat) GuestSupported() bool {
	return cpuStatLineGuest <= len(c)
}

// Guest returns time spent running a virtual CPU for guest operating systems under the control of the Linux kernel.
func (c CpuStat) Guest() uint64 {
	return c[cpuStatLineGuest]
}

// GuestNiceSupported returns true if the Guest() statistic is present.
func (c CpuStat) GuestNiceSupported() bool {
	return cpuStatLineGuestNice <= len(c)
}

// GuestNice returns time spent running a niced guest (virtual CPU for guest operating systems under the control of the Linux kernel).
func (c CpuStat) GuestNice() uint64 {
	return c[cpuStatLineGuestNice]
}

var procStatFile = "/proc/stat"

func ReadStat() (stat Stat, err error) {
	file, err := os.Open(procStatFile)
	if err != nil {
		return stat, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var name string
	var val []uint64
	for i := 0; scanner.Scan(); i++ {
		if name, val, err = parseStatLine(scanner.Text()); err != nil {
			return stat, err
		}
		switch {
		case name == "cpu":
			stat.Cpu = CpuStat(val)
		case strings.HasPrefix(name, "cpu"):
			stat.CpuX = append(stat.CpuX, CpuStat(val))
		case name == "intr":
			stat.Intr = val[0]
			stat.IntrX = val[1:]
		case name == "ctxt":
			stat.Ctxt = val[0]
		case name == "btime":
			stat.Btime = val[0]
		case name == "processes":
			stat.Processes = val[0]
		case name == "procs_running":
			stat.ProcsRunning = val[0]
		case name == "procs_blocked":
			stat.ProcsBlocked = val[0]
		case name == "softirq":
		// ignoring
		default:
			return stat, fmt.Errorf("unexecpected stat line: %s", name)
		}
	}
	return stat, nil
}

// parseStatLine parses a stat line or the form: name value0 value1.... valueN.
func parseStatLine(line string) (name string, vals []uint64, err error) {
	fields := strings.Fields(line)
	totalVals := len(fields) - 1
	if totalVals <= 0 {
		err = fmt.Errorf("not enough fields in stat line (%d): %s", len(fields), line)
		return
	}
	name = fields[0]
	vals = make([]uint64, totalVals)
	var val uint64
	for i := 0; i < totalVals; i++ {
		if val, err = strconv.ParseUint(fields[i+1], 10, 64); err != nil {
			return name, nil, errors.New("could not parse stat line: " + err.Error())
		}
		vals[i] = val
	}
	return
}

// Stat represents data parsed from a Linux host's /proc/stat file.
type Stat struct {
	Cpu          CpuStat   // Cpu stat averaged across all cpus.
	CpuX         []CpuStat // Cpu stats for cpus. The CpuX[0] contains cpu0 stats.
	Intr         uint64    // Total number of interrupts since boot.
	IntrX        []uint64  // Total number of interrupts for each interrupt. IntrX[0] contains interrupts for intr0.
	Ctxt         uint64    // Number of context switches since boot.
	Btime        uint64    // Boot time, in seconds since Epoch.
	Processes    uint64    // Number of forks since boot.
	ProcsRunning uint64    // Number of processes in the running state (Linnux >= 2.5.45).
	ProcsBlocked uint64    // Number of processes blocked waiting on I/O to complete (Linux >= 2.5.45)
}
