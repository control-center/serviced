package linux

import (
	"log"
	"path"
	"reflect"
	"runtime"
	"testing"
)

var testProcStatVal Stat

func init() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot determine caller path")
	}
	// mock the /proc/stat path
	procStatFile = path.Join(path.Dir(file), "testproc", "stat")

	testProcStatVal = Stat{
		Cpu: []uint64{458114, 15531, 137971, 8314282, 364950, 10, 2854, 0, 0, 0},
		CpuX: []CpuStat{
			CpuStat{152101, 3712, 51148, 2084268, 23152, 9, 326, 0, 0, 0},
			CpuStat{75843, 1767, 21476, 2219249, 10596, 0, 132, 0, 0, 0},
			CpuStat{158215, 2274, 45842, 2039814, 75192, 0, 1120, 0, 0, 0},
			CpuStat{71954, 7777, 19503, 1970950, 256009, 0, 1275, 0, 0, 0},
		},
		Intr:         20735701,
		IntrX:        []uint64{18, 87477, 0, 0, 0, 0, 0, 0, 1, 16648, 0, 0, 3879862, 0, 0},
		Ctxt:         60862438,
		Btime:        1393643169,
		Processes:    12316,
		ProcsRunning: 1,
		ProcsBlocked: 0,
	}
}

func TestReadCpuStat(t *testing.T) {
	s, err := ReadStat()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if !reflect.DeepEqual(s, testProcStatVal) {
		t.Log("read /proc/stat does not match test value")
		t.Fail()
	}
}

var result Stat

func BenchmarkReadCpuStat(b *testing.B) {
	f := procStatFile
	defer func() {
		procStatFile = f
	}()
	procStatFile = "/proc/stat"
	for n := 0; n < b.N; n++ {
		result, _ = ReadStat()
	}
}
