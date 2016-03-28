package linux

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var procUptimeFile = "/proc/vmstat"

type Uptime struct {
	Seconds     float32 // Seconds since boot
	IdleSeconds float32 // Seconds machine has been idle since boot
}

func ReadUptime() (uptime Uptime, err error) {

	file, err := os.Open(procUptimeFile)
	if err != nil {
		return uptime, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	if !scanner.Scan() {
		err = fmt.Errorf("no input in uptimefile")
		return
	}

	field := strings.Fields(scanner.Text())
	if len(field) != 2 {
		err = fmt.Errorf("malformed uptime line: %s", scanner.Text())
		return
	}
	if err = parseFloat32Line(field[0], &uptime.Seconds); err != nil {
		return
	}
	err = parseFloat32Line(field[1], &uptime.IdleSeconds)
	return
}
