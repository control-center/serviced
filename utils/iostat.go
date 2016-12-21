package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SimpleIOStat contains basic information from iostat
type SimpleIOStat struct {
	Device string
	RPS    float64
	WPS    float64
	Await  float64
}

// DeviceUtilizationReport is a full iostat report.
// Some fields may not be used if they are not output by an iostat call.
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

var (
	emptyFloat64      float64
	ErrIOStatNoDevice error = errors.New("No device metric in iostat output")
	ErrIOStatNoRPS    error = errors.New("No Read Per Second metric in iostat output")
	ErrIOStatNoWPS    error = errors.New("No Write Per Second metric in iostat output")
	ErrIOStatNoAwait  error = errors.New("No Await metric in iostat output")
)

// ToSimpleIOStat is a simple version of a DeviceUtilizationReport
func (d DeviceUtilizationReport) ToSimpleIOStat() (SimpleIOStat, error) {
	iostat := SimpleIOStat{}
	if d.Device == "" {
		return SimpleIOStat{}, ErrIOStatNoDevice
	} else {
		iostat.Device = d.Device
	}
	if d.RPS == emptyFloat64 {
		return SimpleIOStat{}, ErrIOStatNoRPS
	} else {
		iostat.RPS = d.RPS
	}
	if d.WPS == emptyFloat64 {
		return SimpleIOStat{}, ErrIOStatNoWPS
	} else {
		iostat.WPS = d.WPS
	}
	if d.Await == emptyFloat64 {
		return SimpleIOStat{}, ErrIOStatNoDevice
	} else {
		iostat.Device = d.Device
	}
	return iostat, nil
}

// ParseIOStat creates a map of DeviceUtilizationReports (device name as keys)
// from a reader with iostat output.
func ParseIOStat(r io.Reader) (map[string]DeviceUtilizationReport, error) {
	plog.Info("-------------Parsing iostat")
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

// GetSimpleIOStatsCh calls iostat with -dNxy and an interval.
// It parses the output and creates a DeviceUtilizationReport for each device
// and sends it to the returned channel.
func GetSimpleIOStatsCh(interval time.Duration, quitCh <-chan interface{}) (<-chan map[string]DeviceUtilizationReport, error) {
	cmd := exec.Command("iostat", "-dNxy", fmt.Sprintf("%f", interval.Seconds()))
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	c := make(chan map[string]DeviceUtilizationReport)

	go func() {
		defer cmd.Process.Kill()
		defer out.Close()
		defer close(c)
		parseIOStatWatcher(out, c, quitCh)
		plog.Info("----------Closing iostat channel!!!!")
	}()

	return c, nil
}

// parseIOStatWatcher scans the reader for 2 new lines, signifying a new report
func parseIOStatWatcher(r io.Reader, c chan<- map[string]DeviceUtilizationReport, qCh <-chan interface{}) {
	// Custom bufio.Split() function to split tokens by 2 new lines
	atTwoNewLines := func(data []byte, atEOF bool) (int, []byte, error) {
		advance := 0
		var token []byte
		var prev byte

		for _, b := range data {
			// consume the newline by advancing passed it
			if string(b) == "\n" && string(prev) == "\n" {
				advance++
				plog.Info("---------------Splitting!")
				return advance, token, nil
			}
			token = append(token, b)
			advance++
			prev = b
		}
		return 0, nil, nil
	}

	scanner := bufio.NewScanner(r)
	scanner.Split(atTwoNewLines)
	for scanner.Scan() {
		out := scanner.Text()
		plog.Infof("--------Got a str from the scanner")
		parseReader := strings.NewReader(out)
		report, err := ParseIOStat(parseReader)
		if err != nil {
			plog.WithError(err).Error("----------Failed to parse iostat output.")
		}
		select {
		case <-qCh:
			plog.Info("---------------Quitting parseiostatwatcher")
			return
		case c <- report:
			plog.Infof("-----------------Sent report\n:%v", report)
		}
	}
	if err := scanner.Err(); err != nil {
		plog.Errorf("--------------error: %s", err)
	}
	plog.Info("--------BAILING!")

}
