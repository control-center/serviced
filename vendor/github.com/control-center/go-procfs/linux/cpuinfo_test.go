package linux

import (
	"reflect"
	"testing"
)

var testCpuinfos []Cpuinfo

func init() {

	procCpuinfo = "testproc/cpuinfo"
	testCpuinfos = []Cpuinfo{
		Cpuinfo{
			Processor:     0,
			VendorId:      "GenuineIntel",
			CpuFamily:     6,
			Model:         58,
			ModelName:     "Intel(R) Core(TM) i5-3337U CPU @ 1.80GHz",
			Stepping:      9,
			Microcode:     "0x15",
			CpuMHz:        774.000,
			CacheSize:     3072 * 1024,
			PhysicalId:    0,
			Siblings:      4,
			CoreId:        0,
			CpuCores:      2,
			Apicid:        0,
			InitialApicid: 0,
			Fpu:           true,
			FpuException:  true,
			CpuidLevel:    13,
			Wp:            true,
			Flags: map[string]bool{
				"fpu": true, "vme": true, "de": true, "pse": true, "tsc": true, "msr": true,
				"pae": true, "mce": true, "cx8": true, "apic": true, "sep": true,
				"mtrr": true, "pge": true, "mca": true, "cmov": true, "pat": true,
				"pse36": true, "clflush": true, "dts": true, "acpi": true, "mmx": true,
				"fxsr": true, "sse": true, "sse2": true, "ss": true, "ht": true, "tm": true,
				"pbe": true, "syscall": true, "nx": true, "rdtscp": true, "lm": true,
				"constant_tsc": true, "arch_perfmon": true, "pebs": true, "bts": true,
				"rep_good": true, "nopl": true, "xtopology": true, "nonstop_tsc": true,
				"aperfmperf": true, "eagerfpu": true, "pni": true, "pclmulqdq": true,
				"dtes64": true, "monitor": true, "ds_cpl": true, "vmx": true, "est": true,
				"tm2": true, "ssse3": true, "cx16": true, "xtpr": true, "pdcm": true,
				"pcid": true, "sse4_1": true, "sse4_2": true, "x2apic": true, "popcnt": true,
				"tsc_deadline_timer": true, "aes": true, "xsave": true, "avx": true,
				"f16c": true, "rdrand": true, "lahf_lm": true, "ida": true, "arat": true,
				"epb": true, "xsaveopt": true, "pln": true, "pts": true, "dtherm": true,
				"tpr_shadow": true, "vnmi": true, "flexpriority": true, "ept": true,
				"vpid": true, "fsgsbase": true, "smep": true, "erms": true},
			Bogomips:       3591.80,
			ClflushSize:    64,
			CacheAlignment: 64,
			AddressSizes: struct {
				BitsPhysical uint8
				BitsVirtual  uint8
			}{BitsPhysical: 36, BitsVirtual: 48},
			PowerManagement: "",
		},
		Cpuinfo{
			Processor:     1,
			VendorId:      "GenuineIntel",
			CpuFamily:     6,
			Model:         58,
			ModelName:     "Intel(R) Core(TM) i5-3337U CPU @ 1.80GHz",
			Stepping:      9,
			Microcode:     "0x15",
			CpuMHz:        774.000,
			CacheSize:     3072 * 1024,
			PhysicalId:    0,
			Siblings:      4,
			CoreId:        0,
			CpuCores:      2,
			Apicid:        1,
			InitialApicid: 1,
			Fpu:           true,
			FpuException:  true,
			CpuidLevel:    13,
			Wp:            true,
			Flags: map[string]bool{
				"fpu": true, "vme": true, "de": true, "pse": true, "tsc": true, "msr": true,
				"pae": true, "mce": true, "cx8": true, "apic": true, "sep": true,
				"mtrr": true, "pge": true, "mca": true, "cmov": true, "pat": true,
				"pse36": true, "clflush": true, "dts": true, "acpi": true, "mmx": true,
				"fxsr": true, "sse": true, "sse2": true, "ss": true, "ht": true, "tm": true,
				"pbe": true, "syscall": true, "nx": true, "rdtscp": true, "lm": true,
				"constant_tsc": true, "arch_perfmon": true, "pebs": true, "bts": true,
				"rep_good": true, "nopl": true, "xtopology": true, "nonstop_tsc": true,
				"aperfmperf": true, "eagerfpu": true, "pni": true, "pclmulqdq": true,
				"dtes64": true, "monitor": true, "ds_cpl": true, "vmx": true, "est": true,
				"tm2": true, "ssse3": true, "cx16": true, "xtpr": true, "pdcm": true,
				"pcid": true, "sse4_1": true, "sse4_2": true, "x2apic": true, "popcnt": true,
				"tsc_deadline_timer": true, "aes": true, "xsave": true, "avx": true,
				"f16c": true, "rdrand": true, "lahf_lm": true, "ida": true, "arat": true,
				"epb": true, "xsaveopt": true, "pln": true, "pts": true, "dtherm": true,
				"tpr_shadow": true, "vnmi": true, "flexpriority": true, "ept": true,
				"vpid": true, "fsgsbase": true, "smep": true, "erms": true},
			Bogomips:       3591.80,
			ClflushSize:    64,
			CacheAlignment: 64,
			AddressSizes: struct {
				BitsPhysical uint8
				BitsVirtual  uint8
			}{BitsPhysical: 36, BitsVirtual: 48},
			PowerManagement: "",
		},
		Cpuinfo{
			Processor:     2,
			VendorId:      "GenuineIntel",
			CpuFamily:     6,
			Model:         58,
			ModelName:     "Intel(R) Core(TM) i5-3337U CPU @ 1.80GHz",
			Stepping:      9,
			Microcode:     "0x15",
			CpuMHz:        774.000,
			CacheSize:     3072 * 1024,
			PhysicalId:    0,
			Siblings:      4,
			CoreId:        1,
			CpuCores:      2,
			Apicid:        2,
			InitialApicid: 2,
			Fpu:           true,
			FpuException:  true,
			CpuidLevel:    13,
			Wp:            true,
			Flags: map[string]bool{
				"fpu": true, "vme": true, "de": true, "pse": true, "tsc": true, "msr": true,
				"pae": true, "mce": true, "cx8": true, "apic": true, "sep": true,
				"mtrr": true, "pge": true, "mca": true, "cmov": true, "pat": true,
				"pse36": true, "clflush": true, "dts": true, "acpi": true, "mmx": true,
				"fxsr": true, "sse": true, "sse2": true, "ss": true, "ht": true, "tm": true,
				"pbe": true, "syscall": true, "nx": true, "rdtscp": true, "lm": true,
				"constant_tsc": true, "arch_perfmon": true, "pebs": true, "bts": true,
				"rep_good": true, "nopl": true, "xtopology": true, "nonstop_tsc": true,
				"aperfmperf": true, "eagerfpu": true, "pni": true, "pclmulqdq": true,
				"dtes64": true, "monitor": true, "ds_cpl": true, "vmx": true, "est": true,
				"tm2": true, "ssse3": true, "cx16": true, "xtpr": true, "pdcm": true,
				"pcid": true, "sse4_1": true, "sse4_2": true, "x2apic": true, "popcnt": true,
				"tsc_deadline_timer": true, "aes": true, "xsave": true, "avx": true,
				"f16c": true, "rdrand": true, "lahf_lm": true, "ida": true, "arat": true,
				"epb": true, "xsaveopt": true, "pln": true, "pts": true, "dtherm": true,
				"tpr_shadow": true, "vnmi": true, "flexpriority": true, "ept": true,
				"vpid": true, "fsgsbase": true, "smep": true, "erms": true},
			Bogomips:       3591.80,
			ClflushSize:    64,
			CacheAlignment: 64,
			AddressSizes: struct {
				BitsPhysical uint8
				BitsVirtual  uint8
			}{BitsPhysical: 36, BitsVirtual: 48},
			PowerManagement: "",
		},
		Cpuinfo{
			Processor:     3,
			VendorId:      "GenuineIntel",
			CpuFamily:     6,
			Model:         58,
			ModelName:     "Intel(R) Core(TM) i5-3337U CPU @ 1.80GHz",
			Stepping:      9,
			Microcode:     "0x15",
			CpuMHz:        774.000,
			CacheSize:     3072 * 1024,
			PhysicalId:    0,
			Siblings:      4,
			CoreId:        1,
			CpuCores:      2,
			Apicid:        3,
			InitialApicid: 3,
			Fpu:           true,
			FpuException:  true,
			CpuidLevel:    13,
			Wp:            true,
			Flags: map[string]bool{
				"fpu": true, "vme": true, "de": true, "pse": true, "tsc": true, "msr": true,
				"pae": true, "mce": true, "cx8": true, "apic": true, "sep": true,
				"mtrr": true, "pge": true, "mca": true, "cmov": true, "pat": true,
				"pse36": true, "clflush": true, "dts": true, "acpi": true, "mmx": true,
				"fxsr": true, "sse": true, "sse2": true, "ss": true, "ht": true, "tm": true,
				"pbe": true, "syscall": true, "nx": true, "rdtscp": true, "lm": true,
				"constant_tsc": true, "arch_perfmon": true, "pebs": true, "bts": true,
				"rep_good": true, "nopl": true, "xtopology": true, "nonstop_tsc": true,
				"aperfmperf": true, "eagerfpu": true, "pni": true, "pclmulqdq": true,
				"dtes64": true, "monitor": true, "ds_cpl": true, "vmx": true, "est": true,
				"tm2": true, "ssse3": true, "cx16": true, "xtpr": true, "pdcm": true,
				"pcid": true, "sse4_1": true, "sse4_2": true, "x2apic": true, "popcnt": true,
				"tsc_deadline_timer": true, "aes": true, "xsave": true, "avx": true,
				"f16c": true, "rdrand": true, "lahf_lm": true, "ida": true, "arat": true,
				"epb": true, "xsaveopt": true, "pln": true, "pts": true, "dtherm": true,
				"tpr_shadow": true, "vnmi": true, "flexpriority": true, "ept": true,
				"vpid": true, "fsgsbase": true, "smep": true, "erms": true},
			Bogomips:       3591.80,
			ClflushSize:    64,
			CacheAlignment: 64,
			AddressSizes: struct {
				BitsPhysical uint8
				BitsVirtual  uint8
			}{BitsPhysical: 36, BitsVirtual: 48},
			PowerManagement: "",
		},
	}
}

func TestReadCpuinfo(t *testing.T) {

	cpuinfos, err := ReadCpuinfo()
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if !reflect.DeepEqual(cpuinfos, testCpuinfos) {
		t.Logf("orig\n%v", cpuinfos)
		t.Logf("test\n%v", testCpuinfos)
		t.Fail()
	}
}
