package linux

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var procVmstatFile = "/proc/vmstat"

type Vmstat struct {
	NrFreePages                uint
	NrInactiveAnon             uint
	NrActiveAnon               uint
	NrInactiveFile             uint
	NrActiveFile               uint
	NrUnevictable              uint
	NrMlock                    uint
	NrAnonPages                uint
	NrMapped                   uint
	NrFilePages                uint
	NrDirty                    uint
	NrWriteback                uint
	NrSlabReclaimable          uint
	NrSlabUnreclaimable        uint
	NrPageTablePages           uint
	NrKernelStack              uint
	NrUnstable                 uint
	NrBounce                   uint
	NrVmscanWrite              uint
	NrVmscanImmediateReclaim   uint
	NrWritebackTemp            uint
	NrIsolatedAnon             uint
	NrIsolatedFile             uint
	NrShmem                    uint
	NrDirtied                  uint
	NrWritten                  uint
	NumaHit                    uint
	NumaMiss                   uint
	NumaForeign                uint
	NumaInterleave             uint
	NumaLocal                  uint
	NumaOther                  uint
	NrAnonTransparentHugepages uint
	NrFreeCma                  uint
	NrDirtyThreshold           uint
	NrDirtyBackgroundThreshold uint
	Pgpgin                     uint
	Pgpgout                    uint
	Pswpin                     uint
	Pswpout                    uint
	PgallocDma                 uint
	PgallocDma32               uint
	PgallocNormal              uint
	PgallocMovable             uint
	Pgfree                     uint
	Pgactivate                 uint
	Pgdeactivate               uint
	Pgfault                    uint
	Pgmajfault                 uint
	PgrefillDma                uint
	PgrefillDma32              uint
	PgrefillNormal             uint
	PgrefillMovable            uint
	PgstealKswapdDma           uint
	PgstealKswapdDma32         uint
	PgstealKswapdNormal        uint
	PgstealKswapdMovable       uint
	PgstealDirectDma           uint
	PgstealDirectDma32         uint
	PgstealDirectNormal        uint
	PgstealDirectMovable       uint
	PgscanKswapdDma            uint
	PgscanKswapdDma32          uint
	PgscanKswapdNormal         uint
	PgscanKswapdMovable        uint
	PgscanDirectDma            uint
	PgscanDirectDma32          uint
	PgscanDirectNormal         uint
	PgscanDirectMovable        uint
	PgscanDirectThrottle       uint
	ZoneReclaimFailed          uint
	Pginodesteal               uint
	SlabsScanned               uint
	KswapdInodesteal           uint
	KswapdLowWmarkHitQuickly   uint
	KswapdHighWmarkHitQuickly  uint
	KswapdSkipCongestionWait   uint
	Pageoutrun                 uint
	Allocstall                 uint
	Pgrotated                  uint
	NumaPteUpdates             uint
	NumaHintFaults             uint
	NumaHintFaultsLocal        uint
	NumaPagesMigrated          uint
	PgmigrateSuccess           uint
	PgmigrateFail              uint
	CompactMigrateScanned      uint
	CompactFreeScanned         uint
	CompactIsolated            uint
	CompactStall               uint
	CompactFail                uint
	CompactSuccess             uint
	HtlbBuddyAllocSuccess      uint
	HtlbBuddyAllocFail         uint
	UnevictablePgsCulled       uint
	UnevictablePgsScanned      uint
	UnevictablePgsRescued      uint
	UnevictablePgsMlocked      uint
	UnevictablePgsMunlocked    uint
	UnevictablePgsCleared      uint
	UnevictablePgsStranded     uint
	ThpFaultAlloc              uint
	ThpFaultFallback           uint
	ThpCollapseAlloc           uint
	ThpCollapseAllocFailed     uint
	ThpSplit                   uint
	ThpZeroPageAlloc           uint
	ThpZeroPageAllocFailed     uint
}

func ReadVmstat() (vmstat Vmstat, err error) {

	file, err := os.Open(procVmstatFile)
	if err != nil {
		return vmstat, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		field := strings.Fields(scanner.Text())
		if len(field) != 2 {
			err = fmt.Errorf("malformed vmstat line: %s", scanner.Text())
			return
		}
		switch field[0] {

		case "nr_free_pages":
			err = parseIntLine(field[1], &vmstat.NrFreePages)
		case "nr_inactive_anon":
			err = parseIntLine(field[1], &vmstat.NrInactiveAnon)
		case "nr_active_anon":
			err = parseIntLine(field[1], &vmstat.NrActiveAnon)
		case "nr_inactive_file":
			err = parseIntLine(field[1], &vmstat.NrInactiveFile)
		case "nr_active_file":
			err = parseIntLine(field[1], &vmstat.NrActiveFile)
		case "nr_unevictable":
			err = parseIntLine(field[1], &vmstat.NrUnevictable)
		case "nr_mlock":
			err = parseIntLine(field[1], &vmstat.NrMlock)
		case "nr_anon_pages":
			err = parseIntLine(field[1], &vmstat.NrAnonPages)
		case "nr_mapped":
			err = parseIntLine(field[1], &vmstat.NrMapped)
		case "nr_file_pages":
			err = parseIntLine(field[1], &vmstat.NrFilePages)
		case "nr_dirty":
			err = parseIntLine(field[1], &vmstat.NrDirty)
		case "nr_writeback":
			err = parseIntLine(field[1], &vmstat.NrWriteback)
		case "nr_slab_reclaimable":
			err = parseIntLine(field[1], &vmstat.NrSlabReclaimable)
		case "nr_slab_unreclaimable":
			err = parseIntLine(field[1], &vmstat.NrSlabUnreclaimable)
		case "nr_page_table_pages":
			err = parseIntLine(field[1], &vmstat.NrPageTablePages)
		case "nr_kernel_stack":
			err = parseIntLine(field[1], &vmstat.NrKernelStack)
		case "nr_unstable":
			err = parseIntLine(field[1], &vmstat.NrUnstable)
		case "nr_bounce":
			err = parseIntLine(field[1], &vmstat.NrBounce)
		case "nr_vmscan_write":
			err = parseIntLine(field[1], &vmstat.NrVmscanWrite)
		case "nr_vmscan_immediate_reclaim":
			err = parseIntLine(field[1], &vmstat.NrVmscanImmediateReclaim)
		case "nr_writeback_temp":
			err = parseIntLine(field[1], &vmstat.NrWritebackTemp)
		case "nr_isolated_anon":
			err = parseIntLine(field[1], &vmstat.NrIsolatedAnon)
		case "nr_isolated_file":
			err = parseIntLine(field[1], &vmstat.NrIsolatedFile)
		case "nr_shmem":
			err = parseIntLine(field[1], &vmstat.NrShmem)
		case "nr_dirtied":
			err = parseIntLine(field[1], &vmstat.NrDirtied)
		case "nr_written":
			err = parseIntLine(field[1], &vmstat.NrWritten)
		case "numa_hit":
			err = parseIntLine(field[1], &vmstat.NumaHit)
		case "numa_miss":
			err = parseIntLine(field[1], &vmstat.NumaMiss)
		case "numa_foreign":
			err = parseIntLine(field[1], &vmstat.NumaForeign)
		case "numa_interleave":
			err = parseIntLine(field[1], &vmstat.NumaInterleave)
		case "numa_local":
			err = parseIntLine(field[1], &vmstat.NumaLocal)
		case "numa_other":
			err = parseIntLine(field[1], &vmstat.NumaOther)
		case "nr_anon_transparent_hugepages":
			err = parseIntLine(field[1], &vmstat.NrAnonTransparentHugepages)
		case "nr_free_cma":
			err = parseIntLine(field[1], &vmstat.NrFreeCma)
		case "nr_dirty_threshold":
			err = parseIntLine(field[1], &vmstat.NrDirtyThreshold)
		case "nr_dirty_background_threshold":
			err = parseIntLine(field[1], &vmstat.NrDirtyBackgroundThreshold)
		case "pgpgin":
			err = parseIntLine(field[1], &vmstat.Pgpgin)
		case "pgpgout":
			err = parseIntLine(field[1], &vmstat.Pgpgout)
		case "pswpin":
			err = parseIntLine(field[1], &vmstat.Pswpin)
		case "pswpout":
			err = parseIntLine(field[1], &vmstat.Pswpout)
		case "pgalloc_dma":
			err = parseIntLine(field[1], &vmstat.PgallocDma)
		case "pgalloc_dma32":
			err = parseIntLine(field[1], &vmstat.PgallocDma32)
		case "pgalloc_normal":
			err = parseIntLine(field[1], &vmstat.PgallocNormal)
		case "pgalloc_movable":
			err = parseIntLine(field[1], &vmstat.PgallocMovable)
		case "pgfree":
			err = parseIntLine(field[1], &vmstat.Pgfree)
		case "pgactivate":
			err = parseIntLine(field[1], &vmstat.Pgactivate)
		case "pgdeactivate":
			err = parseIntLine(field[1], &vmstat.Pgdeactivate)
		case "pgfault":
			err = parseIntLine(field[1], &vmstat.Pgfault)
		case "pgmajfault":
			err = parseIntLine(field[1], &vmstat.Pgmajfault)
		case "pgrefill_dma":
			err = parseIntLine(field[1], &vmstat.PgrefillDma)
		case "pgrefill_dma32":
			err = parseIntLine(field[1], &vmstat.PgrefillDma32)
		case "pgrefill_normal":
			err = parseIntLine(field[1], &vmstat.PgrefillNormal)
		case "pgrefill_movable":
			err = parseIntLine(field[1], &vmstat.PgrefillMovable)
		case "pgsteal_kswapd_dma":
			err = parseIntLine(field[1], &vmstat.PgstealKswapdDma)
		case "pgsteal_kswapd_dma32":
			err = parseIntLine(field[1], &vmstat.PgstealKswapdDma32)
		case "pgsteal_kswapd_normal":
			err = parseIntLine(field[1], &vmstat.PgstealKswapdNormal)
		case "pgsteal_kswapd_movable":
			err = parseIntLine(field[1], &vmstat.PgstealKswapdMovable)
		case "pgsteal_direct_dma":
			err = parseIntLine(field[1], &vmstat.PgstealDirectDma)
		case "pgsteal_direct_dma32":
			err = parseIntLine(field[1], &vmstat.PgstealDirectDma32)
		case "pgsteal_direct_normal":
			err = parseIntLine(field[1], &vmstat.PgstealDirectNormal)
		case "pgsteal_direct_movable":
			err = parseIntLine(field[1], &vmstat.PgstealDirectMovable)
		case "pgscan_kswapd_dma":
			err = parseIntLine(field[1], &vmstat.PgscanKswapdDma)
		case "pgscan_kswapd_dma32":
			err = parseIntLine(field[1], &vmstat.PgscanKswapdDma32)
		case "pgscan_kswapd_normal":
			err = parseIntLine(field[1], &vmstat.PgscanKswapdNormal)
		case "pgscan_kswapd_movable":
			err = parseIntLine(field[1], &vmstat.PgscanKswapdMovable)
		case "pgscan_direct_dma":
			err = parseIntLine(field[1], &vmstat.PgscanDirectDma)
		case "pgscan_direct_dma32":
			err = parseIntLine(field[1], &vmstat.PgscanDirectDma32)
		case "pgscan_direct_normal":
			err = parseIntLine(field[1], &vmstat.PgscanDirectNormal)
		case "pgscan_direct_movable":
			err = parseIntLine(field[1], &vmstat.PgscanDirectMovable)
		case "pgscan_direct_throttle":
			err = parseIntLine(field[1], &vmstat.PgscanDirectThrottle)
		case "zone_reclaim_failed":
			err = parseIntLine(field[1], &vmstat.ZoneReclaimFailed)
		case "pginodesteal":
			err = parseIntLine(field[1], &vmstat.Pginodesteal)
		case "slabs_scanned":
			err = parseIntLine(field[1], &vmstat.SlabsScanned)
		case "kswapd_inodesteal":
			err = parseIntLine(field[1], &vmstat.KswapdInodesteal)
		case "kswapd_low_wmark_hit_quickly":
			err = parseIntLine(field[1], &vmstat.KswapdLowWmarkHitQuickly)
		case "kswapd_high_wmark_hit_quickly":
			err = parseIntLine(field[1], &vmstat.KswapdHighWmarkHitQuickly)
		case "kswapd_skip_congestion_wait":
			err = parseIntLine(field[1], &vmstat.KswapdSkipCongestionWait)
		case "pageoutrun":
			err = parseIntLine(field[1], &vmstat.Pageoutrun)
		case "allocstall":
			err = parseIntLine(field[1], &vmstat.Allocstall)
		case "pgrotated":
			err = parseIntLine(field[1], &vmstat.Pgrotated)
		case "numa_pte_updates":
			err = parseIntLine(field[1], &vmstat.NumaPteUpdates)
		case "numa_hint_faults":
			err = parseIntLine(field[1], &vmstat.NumaHintFaults)
		case "numa_hint_faults_local":
			err = parseIntLine(field[1], &vmstat.NumaHintFaultsLocal)
		case "numa_pages_migrated":
			err = parseIntLine(field[1], &vmstat.NumaPagesMigrated)
		case "pgmigrate_success":
			err = parseIntLine(field[1], &vmstat.PgmigrateSuccess)
		case "pgmigrate_fail":
			err = parseIntLine(field[1], &vmstat.PgmigrateFail)
		case "compact_migrate_scanned":
			err = parseIntLine(field[1], &vmstat.CompactMigrateScanned)
		case "compact_free_scanned":
			err = parseIntLine(field[1], &vmstat.CompactFreeScanned)
		case "compact_isolated":
			err = parseIntLine(field[1], &vmstat.CompactIsolated)
		case "compact_stall":
			err = parseIntLine(field[1], &vmstat.CompactStall)
		case "compact_fail":
			err = parseIntLine(field[1], &vmstat.CompactFail)
		case "compact_success":
			err = parseIntLine(field[1], &vmstat.CompactSuccess)
		case "htlb_buddy_alloc_success":
			err = parseIntLine(field[1], &vmstat.HtlbBuddyAllocSuccess)
		case "htlb_buddy_alloc_fail":
			err = parseIntLine(field[1], &vmstat.HtlbBuddyAllocFail)
		case "unevictable_pgs_culled":
			err = parseIntLine(field[1], &vmstat.UnevictablePgsCulled)
		case "unevictable_pgs_scanned":
			err = parseIntLine(field[1], &vmstat.UnevictablePgsScanned)
		case "unevictable_pgs_rescued":
			err = parseIntLine(field[1], &vmstat.UnevictablePgsRescued)
		case "unevictable_pgs_mlocked":
			err = parseIntLine(field[1], &vmstat.UnevictablePgsMlocked)
		case "unevictable_pgs_munlocked":
			err = parseIntLine(field[1], &vmstat.UnevictablePgsMunlocked)
		case "unevictable_pgs_cleared":
			err = parseIntLine(field[1], &vmstat.UnevictablePgsCleared)
		case "unevictable_pgs_stranded":
			err = parseIntLine(field[1], &vmstat.UnevictablePgsStranded)
		case "thp_fault_alloc":
			err = parseIntLine(field[1], &vmstat.ThpFaultAlloc)
		case "thp_fault_fallback":
			err = parseIntLine(field[1], &vmstat.ThpFaultFallback)
		case "thp_collapse_alloc":
			err = parseIntLine(field[1], &vmstat.ThpCollapseAlloc)
		case "thp_collapse_alloc_failed":
			err = parseIntLine(field[1], &vmstat.ThpCollapseAllocFailed)
		case "thp_split":
			err = parseIntLine(field[1], &vmstat.ThpSplit)
		case "thp_zero_page_alloc":
			err = parseIntLine(field[1], &vmstat.ThpZeroPageAlloc)
		case "thp_zero_page_alloc_failed":
			err = parseIntLine(field[1], &vmstat.ThpZeroPageAllocFailed)
		}
		if err != nil {
			return
		}
	}
	return
}
