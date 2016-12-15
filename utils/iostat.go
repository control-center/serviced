package utils

import (
	"bufio"
	"io"
	"strconv"
	"string"
	"fmt"
)

type SimpleIOStat struct {
	Device   string
	KBReadPS float64
	KBWrtnPS float64
	RPS      float64
	WPS      float64
	PctUtil  float64
}

type DeviceUtilizationReport struct {
	Device    string  // Device name
	TPS       float64 // Transfers Per Second
	BlkReadPS float64 // Reads in blocks per Second
	BlkWrtnPS float64 // Writes in blocks per Second
	BlkRead   float64 // Blocks Read total
	BlkWrtn   float64 // Blocks Written total
	KBReadPS  float64 // Kilobytes Read Per Second
	KBWrtnPS  float64 // Kilobytes Written Per Second
	KBRead    float64 // Kilobytes Read total
	KBWrtn    float64 // Kilobytes Written total
	MBReadPS  float64 // Megabytes Read Per Second
	MBWrtnPS  float64 // Megabytes Written Per Second
	MBRead    float64 // Megabytes Read total
	MBWrtn    float64 // Megabytes Written total
	RRQMPS    float64 // Read Requests Merged Per Second
	WRQMPS    float64 // Write Requests Merged Per Second
	RPS       float64 // Reads Per Second
	WPS       float64 // Writes Per Second
	RSecPS    float64 // Read Sectors Per Second
	WSecPS    float64 // Written Sectors Per Second
	RKBPS     float64 // Read Kilobytes Per Second
	WKBPS     float64 // Written Kilobytes Per Second
	RMBPS     float64 // Read Megabytes Per Second
	WMBPS     float64 // Written Megabytes Per Second
	AvgRqSz   float64 // Average Request Size (in sectors)
	AvgQuSz   float64 // Average Queue Size
	PctUtil   float64 // CPU bandwidth utilization by device requests
}

func (d DeviceUtilizationReport) ToSimpleIOStat() (SimpleIOStat, error) {
	iostat := SimpleIOStat{}
	// We should have one of:
	// -KBReadPS
	// -MBReadPS
	// -RKBPS
	// -RMBPS
	// And one of:
	// -KBWrtnPS
	// -MBWrtnPS
	// -WKBPS
	// -WMBPS
	// Find out which we have, and convert if necessary...
	var emptyFloat float64
	if d.KBReadPS != emptyFloat {
		iostat.KBReadPS = d.KBReadPS
	} else if d.MBReadPS != emptyFloat {
		iostat.KBReadPS = d.MBReadPS * 1000
	} else if d.RKBPS != emptyFloat {
		iostat.KBReadPS = d.RKBPS
	} else if d.RMBPS != emptyFloat {
		iostat.KBReadPS = d.RMBPS * 1000
	} else {
		return SimpleIOStat{}, fmt.Errorf("Could not find kBRead/s of MBRead/s")
	}

	if d.KBWrtnPS != emptyFloat {
		iostat.KBWrtnPS = d.KBWrtnPS
	} else if d.MBWrtnPS != emptyFloat {
		iostat.KBWrtnPS = d.MBWrtnPS * 1000
	} else if d.WKBPS != emptyFloat {
		iostat.KBWrtnPS = d.WKBPS
	} else if d.WMBPS != emptyFloat {
		iostat.KBWrtnPS = d.WMBPS * 1000
	} else {
		return SimpleIOStat{}, fmt.Errorf("Could not find kBWrtn/s of MBWrtn/s")
	}

	iostat.Device = d.Device
	iostat.RPS = d.RPS
	iostat.WPS = d.WPS
	iostat.PctUtil = d.PctUtil
	return iostat, nil
}

func parseIOStat(r io.Reader) ([]DeviceUtilizationReport, error) {
	scanner := bufio.NewScanner(r)
	var err error
	fields := make([]string, 0)
	reports := make([]DeviceUtilizationReport, 0)
	passedSysInfo := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// iostat might print out some system info and some spaces
		if !passedSysInfo {
			if strings.HasPrefix(strings.TrimSpace(line), "Device") {
				passedSysInfo = true
				fields = strings.Fields(line)
			}
			continue
		}
		metrics := strings.Fields(line)
		report := DeviceUtilizationReport{}

		for index, field := range fields {
			switch field {
			case "Device:":
				report.Device = metrics[index]
			case "tps":
				report.TPS, err = strconv.ParseFloat(metrics[index], 64)
			case "Blk_read/s":
				report.BlkReadPS, err = strconv.ParseFloat(metrics[index], 64)
			case "Blk_wrtn/s":
				report.BlkWrtnPS, err = strconv.ParseFloat(metrics[index], 64)
			case "Blk_read":
				report.BlkRead, err = strconv.ParseFloat(metrics[index], 64)
			case "Blk_wrtn":
				report.BlkWrtn, err = strconv.ParseFloat(metrics[index], 64)
			case "kB_read/s":
				report.KBReadPS, err = strconv.ParseFloat(metrics[index], 64)
			case "kB_wrtn/s":
				report.KBWrtnPS, err = strconv.ParseFloat(metrics[index], 64)
			case "kB_read":
				report.KBRead, err = strconv.ParseFloat(metrics[index], 64)
			case "kB_wrtn":
				report.KBWrtn, err = strconv.ParseFloat(metrics[index], 64)
			case "MB_read/s":
				report.MBReadPS, err = strconv.ParseFloat(metrics[index], 64)
			case "MB_wrtn/s":
				report.MBWrtnPS, err = strconv.ParseFloat(metrics[index], 64)
			case "MB_read":
				report.MBRead, err = strconv.ParseFloat(metrics[index], 64)
			case "MB_wrtn":
				report.MBWrtn, err = strconv.ParseFloat(metrics[index], 64)
			case "rrqm/s":
				report.RRQMPS, err = strconv.ParseFloat(metrics[index], 64)
			case "wrqm/s":
				report.WRQMPS, err = strconv.ParseFloat(metrics[index], 64)
			case "r/s":
				report.RPS, err = strconv.ParseFloat(metrics[index], 64)
			case "w/s":
				report.WPS, err = strconv.ParseFloat(metrics[index], 64)
			case "rsec/s":
				report.RSecPS, err = strconv.ParseFloat(metrics[index], 64)
			case "wsec/s":
				report.WSecPS, err = strconv.ParseFloat(metrics[index], 64)
			case "rkB/s":
				report.RKBPS, err = strconv.ParseFloat(metrics[index], 64)
			case "wkB/s":
				report.WKBPS, err = strconv.ParseFloat(metrics[index], 64)
			case "rMB/s":
				report.RMBPS, err = strconv.ParseFloat(metrics[index], 64)
			case "wMB/s":
				report.WMBPS, err = strconv.ParseFloat(metrics[index], 64)
			case "avgrq-sz":
				report.AvgRqSz, err = strconv.ParseFloat(metrics[index], 64)
			case "avgqu-sz":
				report.AvgQuSz, err = strconv.ParseFloat(metrics[index], 64)
			case "%util":
				report.PctUtil, err = strconv.ParseFloat(metrics[index], 64)
			}
			if err != nil {
				return nil, err
			}
		}
		reports = append(reports, report)
	}
	return reports, nil
}