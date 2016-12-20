package utils

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type SimpleIOStat struct {
	Device string
	RPS    float64
	WPS    float64
	Await  float64
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
	Await     float64 // The  average time (in  milliseconds) for I/O requests issued to the device to be served
}

func (d DeviceUtilizationReport) ToSimpleIOStat() (SimpleIOStat, error) {
	iostat := SimpleIOStat{}
	iostat.Device = d.Device
	iostat.RPS = d.RPS
	iostat.WPS = d.WPS
	iostat.Await = d.Await
	return iostat, nil
}

func parseIOStat(r io.Reader) (map[string]DeviceUtilizationReport, error) {
	scanner := bufio.NewScanner(r)
	var err error
	fields := make([]string, 0)
	reports := make(map[string]DeviceUtilizationReport, 0)
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
			case "await":
				report.Await, err = strconv.ParseFloat(metrics[index], 64)
			}
			if err != nil {
				return nil, err
			}
		}
		reports[report.Device] = report
	}
	return reports, nil
}

func GetSimpleIOStats(devices []string) (map[string]SimpleIOStat, error) {
	cmd := exec.Command("iostat", "-dNxy", "30", devices...)
	defer cmd.Wait()
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer out.Close()
	if err = cmd.Start(); err != nil {
		return nil, err
	}

	iostats, err := parseIOStat(out)
	if err != nil {
		return nil, err
	}

	simpleIOStats := make(map[string]SimpleIOStat, len(iostats))

	for device, stats := range iostats {
		sstat, err := stats.ToSimpleIOStat()
		if err != nil {
			plog.WithField("device", device).WithError(err).Error("Unable to get iostat for tenant device")
		} else {
			simpleIOStats[device] = sstat
		}
	}

	return simpleIOStats, nil
}
