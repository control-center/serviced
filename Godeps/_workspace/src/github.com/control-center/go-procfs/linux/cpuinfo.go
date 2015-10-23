package linux

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

type Cpuinfo struct {
	Processor      uint
	VendorId       string
	CpuFamily      uint
	Model          uint
	ModelName      string
	Stepping       uint
	Microcode      string
	CpuMHz         float32
	CacheSize      uint64
	PhysicalId     uint
	Siblings       uint
	CoreId         uint
	CpuCores       uint
	Apicid         uint
	InitialApicid  uint
	Fpu            bool
	FpuException   bool
	CpuidLevel     uint
	Wp             bool
	Flags          map[string]bool
	Bogomips       float32
	ClflushSize    uint
	CacheAlignment uint
	AddressSizes   struct {
		BitsPhysical uint8
		BitsVirtual  uint8
	}
	PowerManagement string
}

var procCpuinfo = "/proc/cpuinfo"

func ReadCpuinfo() (cpuinfos []Cpuinfo, err error) {
	file, err := os.Open(procCpuinfo)
	if err != nil {
		return cpuinfos, err
	}
	defer file.Close()
	cpuinfos = make([]Cpuinfo, 0)
	cpuinfo := Cpuinfo{}
	scanner := bufio.NewScanner(file)
	for i := 0; scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			cpuinfos = append(cpuinfos, cpuinfo)
			cpuinfo = Cpuinfo{}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			err = fmt.Errorf("expected 2 parts, got %d: %s", len(parts), line)
		}
		name := strings.TrimSpace(parts[0])
		switch name {
		case "processor":
			err = parseIntLine(parts[1], &cpuinfo.Processor)
		case "vendor_id":
			cpuinfo.VendorId = strings.TrimSpace(parts[1])
		case "cpu family":
			err = parseIntLine(parts[1], &cpuinfo.CpuFamily)
		case "model":
			err = parseIntLine(parts[1], &cpuinfo.Model)
		case "model name":
			cpuinfo.ModelName = strings.TrimSpace(parts[1])
		case "stepping":
			err = parseIntLine(parts[1], &cpuinfo.Stepping)
		case "microcode":
			cpuinfo.Microcode = strings.TrimSpace(parts[1])
		case "cpu MHz":
			err = parseFloat32Line(parts[1], &cpuinfo.CpuMHz)
		case "cache size":
			err = parseBytesLine(parts[1], &cpuinfo.CacheSize)
		case "physical id":
			err = parseIntLine(parts[1], &cpuinfo.PhysicalId)
		case "siblings":
			err = parseIntLine(parts[1], &cpuinfo.Siblings)
		case "core id":
			err = parseIntLine(parts[1], &cpuinfo.CoreId)
		case "cpu cores":
			err = parseIntLine(parts[1], &cpuinfo.CpuCores)
		case "apicid":
			err = parseIntLine(parts[1], &cpuinfo.Apicid)
		case "initial apicid":
			err = parseIntLine(parts[1], &cpuinfo.InitialApicid)
		case "fpu":
			err = parseBoolLine(parts[1], &cpuinfo.Fpu)
		case "fpu_exception":
			err = parseBoolLine(parts[1], &cpuinfo.FpuException)
		case "cpuid level":
			err = parseIntLine(parts[1], &cpuinfo.CpuidLevel)
		case "wp":
			err = parseBoolLine(parts[1], &cpuinfo.Wp)
		case "flags":
			err = parseFlagsLine(parts[1], &cpuinfo.Flags)
		case "bogomips":
			err = parseFloat32Line(parts[1], &cpuinfo.Bogomips)
		case "clflush size":
			err = parseIntLine(parts[1], &cpuinfo.ClflushSize)
		case "cache_alignment":
			err = parseIntLine(parts[1], &cpuinfo.CacheAlignment)
		case "address sizes":
			var physical, virtual uint8
			physical, virtual, err = parseAddressSizesLine(parts[1])
			if err == nil {
				cpuinfo.AddressSizes.BitsPhysical = physical
				cpuinfo.AddressSizes.BitsVirtual = virtual
			}
		case "power management":
			cpuinfo.PowerManagement = strings.TrimSpace(parts[1])
		default:
			log.Printf("cpuinfo, unknown column: %s", parts[0])
		}
		if err != nil {
			return
		}
	}
	return
}
