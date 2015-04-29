package linux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
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

func readFloat32(str string, val *float32) error {
	lval, err := strconv.ParseFloat(str, 32)
	*val = float32(lval)
	return err
}
func readUint(str string, val *uint) error {
	lval, err := strconv.ParseUint(str, 10, 32)
	*val = uint(lval)
	return err
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
	if err = readFloat32(fields[0], &stat.Avg1m); err != nil {
		return
	}
	if err = readFloat32(fields[1], &stat.Avg5m); err != nil {
		return
	}
	if err = readFloat32(fields[2], &stat.Avg10m); err != nil {
		return
	}
	parts := strings.Split(fields[3], "/")
	if len(parts) != 2 {
		err = fmt.Errorf("expected col 4 to have two one slash")
		return
	}
	if err = readUint(parts[0], &stat.RunningProcesses); err != nil {
		return
	}
	if err = readUint(parts[1], &stat.TotalProcesses); err != nil {
		return
	}
	if err = readUint(fields[4], &stat.LastPID); err != nil {
		return
	}
	return stat, nil
}
