package linux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var diskStatFile = "/proc/diskstats"

// DiskStatProxy is a container for a disk statistic snapshot
type DiskStatsProxy struct {
	Disk  string
	Major uint8
	Minor uint8
	Stats *Diskstat
}

/*
 * Diskstat represents a snapshot of stats for a single disk
 *
 * Disk stats representation built from: https://www.kernel.org/doc/Documentation/iostats.txt.
 * Some of these statistics are cumulative and can roll over, while others do not.  This code
 * simply collects raw values.  Refer to the link above for full docs.
 */
type Diskstat struct {
	NReads          uint64
	NReadsMerged    uint64
	NSectorsRead    uint64
	NMsReading      uint64
	NWrites         uint64
	NWritesMerged   uint64
	NSectorsWritten uint64
	NMsWriting      uint64
	NIoInProgress   uint64
	NMsIo           uint64
	NMsIoWeighted   uint64
}

// ReadDiskStat does the actual collection of disk statistics
func ReadDiskstat() ([]*DiskStatsProxy, error) {

	diskstatsproxies := []*DiskStatsProxy{}
	file, err := os.Open(diskStatFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.Fields(scanner.Text())
		if len(line) != 14 {
			err = fmt.Errorf("malformed diskstat line: %s", scanner.Text())
			return nil, err
		}

		// Each disk should only exist once in /proc/diskstat, error out if we find a dupe
		for _, proxy := range diskstatsproxies {
			if line[2] == proxy.Disk {
				return nil, fmt.Errorf("duplicate entry for disk %s found", line[2])
			}
		}

		major, err := strconv.ParseUint(line[0], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("unable to convert major device number: %s", err)
		}
		minor, err := strconv.ParseUint(line[1], 10, 8)
		if err != nil {
			return nil, fmt.Errorf("unable to convert minor device number: %s", err)
		}
		stat := Diskstat{}
		for i := 3; i < len(line); i++ {
			switch i {
			case 3:
				err = parseInt64Line(line[i], &stat.NReads)
			case 4:
				err = parseInt64Line(line[i], &stat.NReadsMerged)
			case 5:
				err = parseInt64Line(line[i], &stat.NSectorsRead)
			case 6:
				err = parseInt64Line(line[i], &stat.NMsReading)
			case 7:
				err = parseInt64Line(line[i], &stat.NWrites)
			case 8:
				err = parseInt64Line(line[i], &stat.NWritesMerged)
			case 9:
				err = parseInt64Line(line[i], &stat.NSectorsWritten)
			case 10:
				err = parseInt64Line(line[i], &stat.NMsWriting)
			case 11:
				err = parseInt64Line(line[i], &stat.NIoInProgress)
			case 12:
				err = parseInt64Line(line[i], &stat.NMsIo)
			case 13:
				err = parseInt64Line(line[i], &stat.NMsIoWeighted)
			}

			if err != nil {
				return nil, err
			}
		}
		diskstatsproxies = append(diskstatsproxies, &DiskStatsProxy{Disk: line[2], Major: uint8(major), Minor: uint8(minor), Stats: &stat})
	}

	return diskstatsproxies, nil
}
