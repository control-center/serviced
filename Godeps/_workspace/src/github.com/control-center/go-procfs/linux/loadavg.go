package linux

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var procLoadavgFile = "/proc/loadavg"

type Loadavg struct {
	Avg1m            float32
	Avg5m            float32
	Avg10m           float32
	RunningProcesses uint
	TotalProcesses   uint
	LastPID          uint
}

func ReadLoadavg() (stat Loadavg, err error) {
	file, err := os.Open(procLoadavgFile)
	if err != nil {
		return stat, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		err = fmt.Errorf("unexpected EOF")
		return
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) != 5 {
		err = fmt.Errorf("expected 5 columns got %d", len(fields))
		return
	}
	if err = parseFloat32Line(fields[0], &stat.Avg1m); err != nil {
		return
	}
	if err = parseFloat32Line(fields[1], &stat.Avg5m); err != nil {
		return
	}
	if err = parseFloat32Line(fields[2], &stat.Avg10m); err != nil {
		return
	}
	parts := strings.Split(fields[3], "/")
	if len(parts) != 2 {
		err = fmt.Errorf("expected col 4 to have two one slash")
		return
	}
	if err = parseIntLine(parts[0], &stat.RunningProcesses); err != nil {
		return
	}
	if err = parseIntLine(parts[1], &stat.TotalProcesses); err != nil {
		return
	}
	if err = parseIntLine(fields[4], &stat.LastPID); err != nil {
		return
	}
	return stat, nil
}
