package linux

import (
	"reflect"
	"testing"
)

var testLoadavg Loadavg

func init() {
	procLoadavgFile = "testproc/loadavg"
	testLoadavg = Loadavg{
		Avg1m:            0.08,
		Avg5m:            0.10,
		Avg10m:           0.13,
		RunningProcesses: 2,
		TotalProcesses:   664,
		LastPID:          20863,
	}

}

func TestReadLoadavg(t *testing.T) {
	la, err := ReadLoadavg()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if !reflect.DeepEqual(la, testLoadavg) {
		t.Log("test loadavg data did not match")
		t.Fail()
	}
}
