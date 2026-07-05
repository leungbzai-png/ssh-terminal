// Package sysmon provides pure parsers for lightweight, agentless Linux VPS
// monitoring (v1.2.0 VPS Monitor Sidebar). It turns the raw text output of a
// single compact remote command (see Command) into a Snapshot the frontend can
// render, and computes CPU usage as a delta between successive /proc/stat
// samples via Manager.
//
// This package is deliberately self-contained: it performs no SSH command
// execution, no file I/O, and no network access. All inputs are byte slices
// captured elsewhere, which keeps every function unit-testable with fixture
// strings. CPU usage cannot be derived from a single /proc/stat reading, so
// ParseAll leaves CPUPercent at 0 / CPUValid false; the delta is supplied
// separately through Manager.Sample.
package sysmon

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Command is the exact compact Linux monitor command intended for remote
// execution in a later commit. It emits section markers on their own lines so
// ParseAll can split the combined output deterministically. It is defined here
// for reuse and is NOT executed by this package.
const Command = "echo @@OS@@; uname -s 2>/dev/null; " +
	"echo @@STAT@@; grep '^cpu ' /proc/stat 2>/dev/null; " +
	"echo @@MEM@@; cat /proc/meminfo 2>/dev/null; " +
	"echo @@LOAD@@; cat /proc/loadavg 2>/dev/null; " +
	"echo @@UP@@; cat /proc/uptime 2>/dev/null; " +
	"echo @@DF@@; df -P / 2>/dev/null"

// Section markers emitted by Command, each on its own line.
const (
	markerOS   = "@@OS@@"
	markerStat = "@@STAT@@"
	markerMem  = "@@MEM@@"
	markerLoad = "@@LOAD@@"
	markerUp   = "@@UP@@"
	markerDf   = "@@DF@@"
)

// CPUCounters holds the aggregate /proc/stat jiffy counters needed for a CPU
// usage delta. Both fields are cumulative since boot.
type CPUCounters struct {
	Total uint64 `json:"total"`
	Idle  uint64 `json:"idle"`
}

// MemInfo holds the /proc/meminfo fields used for memory and swap usage, in kB.
type MemInfo struct {
	TotalKB     uint64 `json:"totalKB"`
	AvailableKB uint64 `json:"availableKB"`
	SwapTotalKB uint64 `json:"swapTotalKB"`
	SwapFreeKB  uint64 `json:"swapFreeKB"`
}

// DiskUsage holds parsed `df -P /` output for the root filesystem.
type DiskUsage struct {
	Filesystem string  `json:"filesystem"`
	SizeKB     uint64  `json:"sizeKB"`
	UsedKB     uint64  `json:"usedKB"`
	AvailKB    uint64  `json:"availKB"`
	UsePercent float64 `json:"usePercent"`
}

// LoadAvg holds the 1/5/15-minute load averages from /proc/loadavg.
type LoadAvg struct {
	One     float64 `json:"one"`
	Five    float64 `json:"five"`
	Fifteen float64 `json:"fifteen"`
}

// Snapshot is the parsed, frontend-facing view of one monitor sample.
// CPUPercent/CPUValid are populated by the caller via Manager.Sample, not by
// ParseAll, because CPU usage requires two samples.
type Snapshot struct {
	OS          string    `json:"os"`
	Supported   bool      `json:"supported"`
	CPUPercent  float64   `json:"cpuPercent"`
	CPUValid    bool      `json:"cpuValid"`
	MemPercent  float64   `json:"memPercent"`
	SwapPercent float64   `json:"swapPercent"`
	SwapPresent bool      `json:"swapPresent"`
	Disk        DiskUsage `json:"disk"`
	Load        LoadAvg   `json:"load"`
	UptimeSec   float64   `json:"uptimeSec"`
	SampledAt   int64     `json:"sampledAt"`
}

// ParseUname trims the `uname -s` output and reports whether the host is a
// supported (Linux) monitor target.
func ParseUname(b []byte) (osName string, supported bool) {
	osName = strings.TrimSpace(string(b))
	return osName, osName == "Linux"
}

// ParseStat parses the aggregate "cpu " line of /proc/stat into CPUCounters.
// Total is the sum of every numeric field; Idle is the idle field plus iowait
// (when present). Fields after the standard ten (steal, guest, ...) are still
// summed into Total, matching the kernel's own accounting.
func ParseStat(b []byte) (CPUCounters, error) {
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "cpu" {
			continue
		}
		nums := fields[1:]
		if len(nums) < 4 {
			return CPUCounters{}, errors.New("sysmon: malformed /proc/stat cpu line (too few fields)")
		}
		var c CPUCounters
		for i, f := range nums {
			v, err := strconv.ParseUint(f, 10, 64)
			if err != nil {
				return CPUCounters{}, errors.New("sysmon: non-numeric field in /proc/stat cpu line: " + f)
			}
			c.Total += v
			// Field index 3 is idle, index 4 is iowait (both count as idle time).
			if i == 3 || i == 4 {
				c.Idle += v
			}
		}
		return c, nil
	}
	return CPUCounters{}, errors.New("sysmon: no 'cpu' aggregate line in /proc/stat")
}

// ParseMeminfo parses MemTotal, MemAvailable, SwapTotal and SwapFree (kB) from
// /proc/meminfo. MemTotal and MemAvailable are required (both are present on
// every kernel since 3.14); a missing one is an error rather than a misleading
// zero. Swap fields default to zero, which ParseAll interprets as "no swap".
func ParseMeminfo(b []byte) (MemInfo, error) {
	var mi MemInfo
	var haveTotal, haveAvail bool
	for _, line := range strings.Split(string(b), "\n") {
		key, val, ok := memLine(line)
		if !ok {
			continue
		}
		switch key {
		case "MemTotal":
			mi.TotalKB = val
			haveTotal = true
		case "MemAvailable":
			mi.AvailableKB = val
			haveAvail = true
		case "SwapTotal":
			mi.SwapTotalKB = val
		case "SwapFree":
			mi.SwapFreeKB = val
		}
	}
	if !haveTotal {
		return MemInfo{}, errors.New("sysmon: /proc/meminfo missing MemTotal")
	}
	if !haveAvail {
		return MemInfo{}, errors.New("sysmon: /proc/meminfo missing MemAvailable")
	}
	return mi, nil
}

// memLine parses a single "Key:   value kB" meminfo line. It returns ok=false
// for blank or malformed lines.
func memLine(line string) (key string, val uint64, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return "", 0, false
	}
	key = strings.TrimSpace(line[:colon])
	fields := strings.Fields(line[colon+1:])
	if len(fields) == 0 {
		return "", 0, false
	}
	v, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return key, v, true
}

// ParseDf parses `df -P /` output and returns the root filesystem usage. It uses
// the last non-header data line and anchors on the Capacity ("NN%") column so a
// filesystem name containing spaces is still handled correctly. Sizes are in kB
// (df -P reports 1024-byte blocks).
func ParseDf(b []byte) (DiskUsage, error) {
	var last string
	for _, line := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		// Skip the header row (starts with "Filesystem").
		if strings.HasPrefix(t, "Filesystem") {
			continue
		}
		last = t
	}
	if last == "" {
		return DiskUsage{}, errors.New("sysmon: no data line in df output")
	}
	fields := strings.Fields(last)
	// Anchor on the Capacity column (the field ending in '%'). Layout is:
	//   Filesystem[...] 1024-blocks Used Available Capacity Mounted-on
	capIdx := -1
	for i, f := range fields {
		if strings.HasSuffix(f, "%") {
			capIdx = i
			break
		}
	}
	if capIdx < 4 {
		return DiskUsage{}, errors.New("sysmon: malformed df line: " + last)
	}
	sizeKB, err1 := strconv.ParseUint(fields[capIdx-3], 10, 64)
	usedKB, err2 := strconv.ParseUint(fields[capIdx-2], 10, 64)
	availKB, err3 := strconv.ParseUint(fields[capIdx-1], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return DiskUsage{}, errors.New("sysmon: non-numeric size field in df line: " + last)
	}
	pctStr := strings.TrimSuffix(fields[capIdx], "%")
	pct, err := strconv.ParseFloat(pctStr, 64)
	if err != nil {
		return DiskUsage{}, errors.New("sysmon: malformed capacity field in df line: " + fields[capIdx])
	}
	return DiskUsage{
		Filesystem: strings.Join(fields[:capIdx-3], " "),
		SizeKB:     sizeKB,
		UsedKB:     usedKB,
		AvailKB:    availKB,
		UsePercent: pct,
	}, nil
}

// ParseLoadavg parses the first three floats of /proc/loadavg into LoadAvg.
func ParseLoadavg(b []byte) (LoadAvg, error) {
	fields := strings.Fields(string(b))
	if len(fields) < 3 {
		return LoadAvg{}, errors.New("sysmon: malformed /proc/loadavg (too few fields)")
	}
	one, err1 := strconv.ParseFloat(fields[0], 64)
	five, err2 := strconv.ParseFloat(fields[1], 64)
	fifteen, err3 := strconv.ParseFloat(fields[2], 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return LoadAvg{}, errors.New("sysmon: non-numeric load average field")
	}
	return LoadAvg{One: one, Five: five, Fifteen: fifteen}, nil
}

// ParseUptime parses the first float of /proc/uptime (seconds since boot).
func ParseUptime(b []byte) (float64, error) {
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0, errors.New("sysmon: empty /proc/uptime")
	}
	sec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, errors.New("sysmon: non-numeric uptime: " + fields[0])
	}
	return sec, nil
}

// ParseAll splits the combined Command output on its section markers and builds
// a Snapshot. It degrades gracefully: an unsupported (non-Linux) OS returns
// early with Supported=false, and any missing or unparseable section leaves its
// fields at zero rather than failing the whole snapshot. CPUPercent/CPUValid are
// intentionally left zero/false here — CPU usage requires a delta supplied by
// Manager.Sample. SampledAt is set to the current Unix time.
func ParseAll(raw []byte) Snapshot {
	secs := splitSections(raw)
	snap := Snapshot{SampledAt: time.Now().Unix()}

	osName, supported := ParseUname([]byte(secs[markerOS]))
	snap.OS = osName
	snap.Supported = supported
	if !supported {
		// Not a Linux host: don't attempt to parse /proc-style sections.
		return snap
	}

	if mi, err := ParseMeminfo([]byte(secs[markerMem])); err == nil {
		if mi.TotalKB > 0 {
			snap.MemPercent = clampPercent(100 * (1 - float64(mi.AvailableKB)/float64(mi.TotalKB)))
		}
		if mi.SwapTotalKB > 0 {
			snap.SwapPresent = true
			snap.SwapPercent = clampPercent(100 * (1 - float64(mi.SwapFreeKB)/float64(mi.SwapTotalKB)))
		}
	}

	if du, err := ParseDf([]byte(secs[markerDf])); err == nil {
		snap.Disk = du
	}

	if la, err := ParseLoadavg([]byte(secs[markerLoad])); err == nil {
		snap.Load = la
	}

	if up, err := ParseUptime([]byte(secs[markerUp])); err == nil {
		snap.UptimeSec = up
	}

	return snap
}

// StatCounters extracts the aggregate CPU counters from combined Command output
// so the caller can feed them to Manager.Sample for a CPU-usage delta. ok is
// false when the @@STAT@@ section is missing or malformed, in which case CPU
// usage is simply unavailable for this sample (the rest of the snapshot is
// still valid). It reuses the same section split as ParseAll.
func StatCounters(raw []byte) (CPUCounters, bool) {
	secs := splitSections(raw)
	c, err := ParseStat([]byte(secs[markerStat]))
	if err != nil {
		return CPUCounters{}, false
	}
	return c, true
}

// splitSections groups the combined command output by marker. Each marker sits
// on its own line; every line after it (until the next marker) belongs to that
// section. Content before the first marker is discarded.
func splitSections(raw []byte) map[string]string {
	out := map[string]string{}
	var cur string
	var buf []string
	flush := func() {
		if cur != "" {
			out[cur] = strings.Join(buf, "\n")
		}
	}
	for _, line := range strings.Split(string(raw), "\n") {
		switch strings.TrimSpace(line) {
		case markerOS, markerStat, markerMem, markerLoad, markerUp, markerDf:
			flush()
			cur = strings.TrimSpace(line)
			buf = buf[:0]
		default:
			if cur != "" {
				buf = append(buf, line)
			}
		}
	}
	flush()
	return out
}

// clampPercent constrains a percentage to the 0..100 range.
func clampPercent(p float64) float64 {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// Manager computes CPU usage as a delta between successive /proc/stat samples,
// keyed by session id. It holds only the previous CPUCounters per session — no
// history, no secrets, nothing persisted. It is safe for concurrent use.
type Manager struct {
	mu   sync.Mutex
	prev map[string]CPUCounters
}

// NewManager returns an empty CPU-delta Manager.
func NewManager() *Manager {
	return &Manager{prev: map[string]CPUCounters{}}
}

// Sample records the latest CPUCounters for a session and returns the CPU usage
// percentage since the previous sample. valid is false for the first sample of
// a session, when the total delta is zero, or when the counters go backwards
// (a reset / new connection) — in every case the stored baseline is updated so
// the next call can produce a valid reading. The returned percent is clamped to
// 0..100.
func (m *Manager) Sample(sessionID string, counters CPUCounters) (percent float64, valid bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prev, ok := m.prev[sessionID]
	m.prev[sessionID] = counters
	if !ok {
		return 0, false
	}
	// Counter reset or moved backwards: baseline already updated above; skip.
	if counters.Total < prev.Total || counters.Idle < prev.Idle {
		return 0, false
	}
	totalDelta := counters.Total - prev.Total
	if totalDelta == 0 {
		return 0, false
	}
	idleDelta := counters.Idle - prev.Idle
	return clampPercent(100 * (1 - float64(idleDelta)/float64(totalDelta))), true
}

// Forget drops a session's stored baseline (called when a session closes so the
// map does not grow unbounded).
func (m *Manager) Forget(sessionID string) {
	m.mu.Lock()
	delete(m.prev, sessionID)
	m.mu.Unlock()
}
