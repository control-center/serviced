package linux

import (
	"reflect"
	"testing"
)

var testVmstat Vmstat

func init() {
	procVmstatFile = "testproc/vmstat"
	testVmstat = Vmstat{
		NrFreePages:                1,
		NrInactiveAnon:             2,
		NrActiveAnon:               3,
		NrInactiveFile:             4,
		NrActiveFile:               5,
		NrUnevictable:              6,
		NrMlock:                    7,
		NrAnonPages:                8,
		NrMapped:                   9,
		NrFilePages:                10,
		NrDirty:                    11,
		NrWriteback:                12,
		NrSlabReclaimable:          13,
		NrSlabUnreclaimable:        14,
		NrPageTablePages:           15,
		NrKernelStack:              16,
		NrUnstable:                 17,
		NrBounce:                   18,
		NrVmscanWrite:              19,
		NrVmscanImmediateReclaim:   20,
		NrWritebackTemp:            21,
		NrIsolatedAnon:             22,
		NrIsolatedFile:             23,
		NrShmem:                    24,
		NrDirtied:                  25,
		NrWritten:                  26,
		NumaHit:                    27,
		NumaMiss:                   28,
		NumaForeign:                29,
		NumaInterleave:             30,
		NumaLocal:                  31,
		NumaOther:                  32,
		NrAnonTransparentHugepages: 33,
		NrFreeCma:                  34,
		NrDirtyThreshold:           35,
		NrDirtyBackgroundThreshold: 36,
		Pgpgin:                    37,
		Pgpgout:                   38,
		Pswpin:                    39,
		Pswpout:                   40,
		PgallocDma:                41,
		PgallocDma32:              42,
		PgallocNormal:             43,
		PgallocMovable:            44,
		Pgfree:                    45,
		Pgactivate:                46,
		Pgdeactivate:              47,
		Pgfault:                   48,
		Pgmajfault:                49,
		PgrefillDma:               50,
		PgrefillDma32:             51,
		PgrefillNormal:            52,
		PgrefillMovable:           53,
		PgstealKswapdDma:          54,
		PgstealKswapdDma32:        55,
		PgstealKswapdNormal:       56,
		PgstealKswapdMovable:      57,
		PgstealDirectDma:          58,
		PgstealDirectDma32:        59,
		PgstealDirectNormal:       60,
		PgstealDirectMovable:      61,
		PgscanKswapdDma:           62,
		PgscanKswapdDma32:         63,
		PgscanKswapdNormal:        64,
		PgscanKswapdMovable:       65,
		PgscanDirectDma:           66,
		PgscanDirectDma32:         67,
		PgscanDirectNormal:        68,
		PgscanDirectMovable:       69,
		PgscanDirectThrottle:      70,
		ZoneReclaimFailed:         71,
		Pginodesteal:              72,
		SlabsScanned:              73,
		KswapdInodesteal:          74,
		KswapdLowWmarkHitQuickly:  75,
		KswapdHighWmarkHitQuickly: 76,
		KswapdSkipCongestionWait:  77,
		Pageoutrun:                78,
		Allocstall:                79,
		Pgrotated:                 80,
		NumaPteUpdates:            81,
		NumaHintFaults:            82,
		NumaHintFaultsLocal:       83,
		NumaPagesMigrated:         84,
		PgmigrateSuccess:          85,
		PgmigrateFail:             86,
		CompactMigrateScanned:     87,
		CompactFreeScanned:        88,
		CompactIsolated:           89,
		CompactStall:              90,
		CompactFail:               91,
		CompactSuccess:            92,
		HtlbBuddyAllocSuccess:     93,
		HtlbBuddyAllocFail:        94,
		UnevictablePgsCulled:      95,
		UnevictablePgsScanned:     96,
		UnevictablePgsRescued:     97,
		UnevictablePgsMlocked:     98,
		UnevictablePgsMunlocked:   99,
		UnevictablePgsCleared:     100,
		UnevictablePgsStranded:    101,
		ThpFaultAlloc:             102,
		ThpFaultFallback:          103,
		ThpCollapseAlloc:          104,
		ThpCollapseAllocFailed:    105,
		ThpSplit:                  106,
		ThpZeroPageAlloc:          107,
		ThpZeroPageAllocFailed:    108,
	}
}

func TestReadVmstat(t *testing.T) {
	vmstat, err := ReadVmstat()
	if err != nil {
		t.Log("unexpected error reading vmstat: %s", err)
		t.Fail()
	}
	if !reflect.DeepEqual(vmstat, testVmstat) {
		t.Log("testVmstat != read vmstat: %v != %v", testVmstat, vmstat)
		t.Fail()
	}
}
