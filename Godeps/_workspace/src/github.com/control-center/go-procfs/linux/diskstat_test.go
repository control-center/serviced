package linux

import (
	"log"
	"path"
	"reflect"
	"runtime"
	"testing"
)

var testDiskStatVals []*DiskStatsProxy

func init() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Cannot determine caller path")
	}
	// mock the /proc/diskstats path
	diskStatFile = path.Join(path.Dir(file), "testproc", "diskstats")
	stat1 := Diskstat{
		NReads:          0,
		NReadsMerged:    0,
		NSectorsRead:    0,
		NMsReading:      0,
		NWrites:         0,
		NWritesMerged:   0,
		NSectorsWritten: 0,
		NMsWriting:      0,
		NIoInProgress:   0,
		NMsIo:           0,
		NMsIoWeighted:   0,
	}
	testDiskStatVals = append(testDiskStatVals, &DiskStatsProxy{Disk: "fd0", Major: 2, Minor: 0, Stats: &stat1})
	stat2 := Diskstat{
		NReads:          1983878,
		NReadsMerged:    134013,
		NSectorsRead:    72779008,
		NMsReading:      5371375,
		NWrites:         2172269,
		NWritesMerged:   1607758,
		NSectorsWritten: 83233346,
		NMsWriting:      12941442,
		NIoInProgress:   0,
		NMsIo:           2111762,
		NMsIoWeighted:   18323695,
	}
	testDiskStatVals = append(testDiskStatVals, &DiskStatsProxy{Disk: "sda", Major: 8, Minor: 0, Stats: &stat2})
	stat3 := Diskstat{
		NReads:          814,
		NReadsMerged:    0,
		NSectorsRead:    6512,
		NMsReading:      12745,
		NWrites:         50755,
		NWritesMerged:   0,
		NSectorsWritten: 406040,
		NMsWriting:      384449,
		NIoInProgress:   0,
		NMsIo:           36859,
		NMsIoWeighted:   397195,
	}
	testDiskStatVals = append(testDiskStatVals, &DiskStatsProxy{Disk: "dm-2", Major: 253, Minor: 2, Stats: &stat3})
}

func TestReadDiskStat(t *testing.T) {
	s, err := ReadDiskstat()
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	if !reflect.DeepEqual(s, testDiskStatVals) {
		t.Log("read /proc/diskstats does not match test value")
		t.Fail()
	}
}
