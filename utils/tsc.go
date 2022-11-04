package utils

// #include "tsc.h"
import "C"

// IntelCPUFreq /proc/cpuinfo model name
const intelCPUFreq = 2800_000000

func GetTimestampFromTSC() uint64 {
	cycleNumber := uint64(C.get_tsc())
	return cycleNumber / intelCPUFreq
}

