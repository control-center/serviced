package linux

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Meminfo struct {
	MemTotal          uint64
	MemFree           uint64
	MemAvailable      uint64
	Buffers           uint64
	Cached            uint64
	SwapCached        uint64
	Active            uint64
	Inactive          uint64
	ActiveAnon        uint64
	InactiveAnon      uint64
	ActiveFile        uint64
	InactiveFile      uint64
	Unevictable       uint64
	Mlocked           uint64
	SwapTotal         uint64
	SwapFree          uint64
	Dirty             uint64
	Writeback         uint64
	AnonPages         uint64
	Mapped            uint64
	Shmem             uint64
	Slab              uint64
	SReclaimable      uint64
	SUnreclaim        uint64
	KernelStack       uint64
	PageTables        uint64
	NFS_Unstable      uint64
	Bounce            uint64
	WritebackTmp      uint64
	CommitLimit       uint64
	Committed_AS      uint64
	VmallocTotal      uint64
	VmallocUsed       uint64
	VmallocChunk      uint64
	HardwareCorrupted uint64
	AnonHugePages     uint64
	HugePages_Total   uint64
	HugePages_Free    uint64
	HugePages_Rsvd    uint64
	HugePages_Surp    uint64
	Hugepagesize      uint64
	DirectMap4k       uint64
	DirectMap2M       uint64
	DirectMap1G       uint64
}

var procMeminfo = "/proc/meminfo"

func parseMeminfoLine(line string) (name string, val uint64, err error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		err = fmt.Errorf("meminfo line needs at least two fields: %s", line)
		return
	}
	if len(fields[0]) < 2 {
		err = fmt.Errorf("meminfo field is too short: %s", fields[0])
		return
	}
	name = fields[0]
	name = name[0 : len(name)-1] // truncate last character
	if val, err = strconv.ParseUint(fields[1], 10, 64); err != nil {
		err = errors.New("could not parse stat line: " + err.Error())
		return
	}
	if len(fields) == 3 {
		switch fields[2] {
		case "kB":
			val = val * 1024
		default:
			err = fmt.Errorf("unexpected multiplier: %s", fields[2])
			return
		}
	}
	return
}

func ReadMeminfo() (meminfo Meminfo, err error) {
	file, err := os.Open(procMeminfo)
	if err != nil {
		return meminfo, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var name string
	var val uint64
	for i := 0; scanner.Scan(); i++ {
		if name, val, err = parseMeminfoLine(scanner.Text()); err != nil {
			return meminfo, err
		}
		switch name {

		case "MemTotal":
			meminfo.MemTotal = val
		case "MemFree":
			meminfo.MemFree = val
		case "MemAvailable":
			meminfo.MemAvailable = val
		case "Buffers":
			meminfo.Buffers = val
		case "Cached":
			meminfo.Cached = val
		case "SwapCached":
			meminfo.SwapCached = val
		case "Active":
			meminfo.Active = val
		case "Inactive":
			meminfo.Inactive = val
		case "Active(anon)":
			meminfo.ActiveAnon = val
		case "Inactive(anon)":
			meminfo.InactiveAnon = val
		case "Active(file)":
			meminfo.ActiveFile = val
		case "Inactive(file)":
			meminfo.InactiveFile = val
		case "Unevictable":
			meminfo.Unevictable = val
		case "Mlocked":
			meminfo.Mlocked = val
		case "SwapTotal":
			meminfo.SwapTotal = val
		case "SwapFree":
			meminfo.SwapFree = val
		case "Dirty":
			meminfo.Dirty = val
		case "Writeback":
			meminfo.Writeback = val
		case "AnonPages":
			meminfo.AnonPages = val
		case "Mapped":
			meminfo.Mapped = val
		case "Shmem":
			meminfo.Shmem = val
		case "Slab":
			meminfo.Slab = val
		case "SReclaimable":
			meminfo.SReclaimable = val
		case "SUnreclaim":
			meminfo.SUnreclaim = val
		case "KernelStack":
			meminfo.KernelStack = val
		case "PageTables":
			meminfo.PageTables = val
		case "NFS_Unstable":
			meminfo.NFS_Unstable = val
		case "Bounce":
			meminfo.Bounce = val
		case "WritebackTmp":
			meminfo.WritebackTmp = val
		case "CommitLimit":
			meminfo.CommitLimit = val
		case "Committed_AS":
			meminfo.Committed_AS = val
		case "VmallocTotal":
			meminfo.VmallocTotal = val
		case "VmallocUsed":
			meminfo.VmallocUsed = val
		case "VmallocChunk":
			meminfo.VmallocChunk = val
		case "HardwareCorrupted":
			meminfo.HardwareCorrupted = val
		case "AnonHugePages":
			meminfo.AnonHugePages = val
		case "HugePages_Total":
			meminfo.HugePages_Total = val
		case "HugePages_Free":
			meminfo.HugePages_Free = val
		case "HugePages_Rsvd":
			meminfo.HugePages_Rsvd = val
		case "HugePages_Surp":
			meminfo.HugePages_Surp = val
		case "Hugepagesize":
			meminfo.Hugepagesize = val
		case "DirectMap4k":
			meminfo.DirectMap4k = val
		case "DirectMap2M":
			meminfo.DirectMap2M = val
		case "DirectMap1G":
			meminfo.DirectMap1G = val
		default:
		}
	}
	return
}
