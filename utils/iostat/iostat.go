package iostat

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/logging"
)

var plog = logging.PackageLogger()

// Getter is an interface to get a channel that reports iostats.
type Getter interface {
	GetIOStatsCh() (<-chan map[string]DeviceUtilizationReport, error)
	GetStatInterval() time.Duration
}

// Reporter implements IOStatGetter and uses interval and quit to control reporting.
type Reporter struct {
	interval time.Duration
	quit     <-chan interface{}
}

// NewReporter creates a new IOStatReporter with interval and quit.
func NewReporter(interval time.Duration, quit <-chan interface{}) *Reporter {
	return &Reporter{
		interval: interval,
		quit:     quit,
	}
}

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
	emptyFloat64 float64
	// ErrIOStatNoDevice occurs when the device name is not in iostat output
	ErrIOStatNoDevice = errors.New("No device metric in iostat output")
)

// ToSimpleIOStat is a simple version of a DeviceUtilizationReport
func (d DeviceUtilizationReport) ToSimpleIOStat() (SimpleIOStat, error) {
	if d.Device == "" {
		return SimpleIOStat{}, ErrIOStatNoDevice
	}
	return SimpleIOStat{
		Device: d.Device,
		RPS:    d.RPS,
		WPS:    d.WPS,
		Await:  d.Await,
	}, nil
}

// ParseIOStat creates a map of DeviceUtilizationReports (device name as keys) from a reader with iostat output.
func ParseIOStat(r io.Reader) (map[string]DeviceUtilizationReport, error) {
	scanner := bufio.NewScanner(r)
	var err error
	var fields []string
	reports := make(map[string]DeviceUtilizationReport, 0)
	passedSysInfo := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// iostat might print out some system info and some spaces
		if !passedSysInfo {
			if strings.HasPrefix(line, "Device") {
				passedSysInfo = true
				fields = strings.Fields(line)
			}
			continue
		}
		metrics := strings.Fields(line)
		report := DeviceUtilizationReport{}

		// In some cases, we may have an extra newline at the end of a report,
		// in which case the metrics len will be 0 and we need to return
		if len(metrics) == 0 {
			return reports, nil
		}

		for index, field := range fields {
			switch field {
			case "Device", "Device:":
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

// GetIOStatsCh calls iostat with -dNxy and an interval defined in reporter.
// It parses the output and creates a DeviceUtilizationReport for each device and sends it to the returned channel.
func (reporter *Reporter) GetIOStatsCh() (<-chan map[string]DeviceUtilizationReport, error) {
	cmd := exec.Command("iostat", "-dNxy", fmt.Sprintf("%.0f", reporter.interval.Seconds()))
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	c := make(chan map[string]DeviceUtilizationReport)

	shutdown := func() {
		cmd.Process.Kill()
		err := cmd.Wait()
		if err != nil {
			plog.WithError(err).Error("iostat error while waiting to exit")
		}
	}

	go func() {
		defer shutdown()
		defer out.Close()
		defer close(c)
		parseIOStatWatcher(out, c, reporter.quit)
	}()

	return c, nil
}

// GetStatInterval implements Getter, returns the stat reporting interval
func (reporter *Reporter) GetStatInterval() time.Duration {
	return reporter.interval
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
	for scanned := true; scanned; {
		scanc := make(chan struct{})
		go func() {
			defer close(scanc)
			scanned = scanner.Scan()
		}()

		select {
		case <-qCh:
			return
		case <-scanc:
			if scanned {
				out := scanner.Text()
				parseReader := strings.NewReader(out)
				report, err := ParseIOStat(parseReader)
				if err != nil {
					plog.WithError(err).Error("Failed to parse iostat output, exiting")
					return
				} else if len(report) == 0 {
					plog.Warn("Got an empty report from iostat")
					continue
				}
				select {
				case <-qCh:
					return
				case c <- report:
				}
			}
		}
	}
	plog.Warn("parseIOStatWatcher exiting")
	if err := scanner.Err(); err != nil {
		plog.WithError(err).Error("Error reading iostat")
	}
}
