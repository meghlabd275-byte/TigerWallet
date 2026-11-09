package handlers

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Host metric readers.
//
// These read the authoritative kernel sources (/proc and statfs) rather than
// maintaining a sampled cache, so a reported value is the state of the machine
// at the moment of the request. Each reader reports whether it succeeded; a
// failure is surfaced as an absent metric instead of a fabricated one.

// hostCPUPercent returns busy CPU as a percentage over a short sampling
// interval. /proc/stat exposes cumulative jiffies, so a percentage requires two
// reads separated in time; a 100ms window keeps the request cheap while still
// producing a meaningful instantaneous figure.
func hostCPUPercent() (float64, bool) {
	first, ok := readCPUSample()
	if !ok {
		return 0, false
	}
	if !sleepMilliseconds(100) {
		return 0, false
	}
	second, ok := readCPUSample()
	if !ok || second.total <= first.total {
		return 0, false
	}
	totalDelta := second.total - first.total
	idleDelta := second.idle - first.idle
	if idleDelta > totalDelta {
		return 0, false
	}
	return float64(totalDelta-idleDelta) / float64(totalDelta) * 100, true
}

type cpuSample struct {
	idle  uint64
	total uint64
}

func readCPUSample() (cpuSample, bool) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuSample{}, false
	}
	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuSample{}, false
	}
	var sample cpuSample
	for i, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return cpuSample{}, false
		}
		sample.total += v
		if i == 3 {
			sample.idle = v
		}
	}
	if sample.total == 0 {
		return cpuSample{}, false
	}
	return sample, true
}

func sleepMilliseconds(ms int) bool {
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return true
}

// hostMemoryPercent reads MemTotal/MemAvailable from /proc/meminfo.
func hostMemoryPercent() (float64, bool) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	var total, available float64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = v
		case "MemAvailable:":
			available = v
		}
	}
	if total <= 0 {
		return 0, false
	}
	if available > total {
		available = total
	}
	return (total - available) / total * 100, true
}

// hostDiskPercent reports the used fraction of the filesystem holding the
// working directory via statfs(2).
func hostDiskPercent() (float64, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(".", &st); err != nil || st.Blocks == 0 {
		return 0, false
	}
	used := st.Blocks - st.Bavail
	return float64(used) / float64(st.Blocks) * 100, true
}
