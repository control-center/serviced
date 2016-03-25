package linux

import (
	"reflect"
	"testing"
)

var testUptime Uptime

func init() {
	procUptimeFile = "testproc/uptime"
	testUptime = Uptime{
		Seconds:     258527.69,
		IdleSeconds: 311193.41,
	}

}

func TestReadUptime(t *testing.T) {
	uptime, err := ReadUptime()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if !reflect.DeepEqual(uptime, testUptime) {
		t.Logf("%v != %v", uptime, testUptime)
	}
}
