package sysmon

import (
	"math"
	"strings"
	"testing"
)

func approx(t *testing.T, got, want, tol float64, label string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s: got %v, want ~%v (tol %v)", label, got, want, tol)
	}
}

func TestParseUnameLinuxSupported(t *testing.T) {
	os, ok := ParseUname([]byte("Linux\n"))
	if os != "Linux" || !ok {
		t.Fatalf("got (%q, %v), want (Linux, true)", os, ok)
	}
}

func TestParseUnameNonLinuxUnsupported(t *testing.T) {
	for _, in := range []string{"Darwin\n", "FreeBSD", "  Windows_NT \n"} {
		os, ok := ParseUname([]byte(in))
		if ok {
			t.Fatalf("input %q: got supported=true, want false (os=%q)", in, os)
		}
	}
}

func TestParseStatNormal(t *testing.T) {
	// user nice system idle iowait irq softirq steal
	// Total = 100+10+50+8000+40+5+5+0 = 8210; Idle = idle+iowait = 8040
	c, err := ParseStat([]byte("cpu  100 10 50 8000 40 5 5 0\ncpu0 50 5 25 4000 20 2 2 0\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Total != 8210 {
		t.Fatalf("Total: got %d, want 8210", c.Total)
	}
	if c.Idle != 8040 {
		t.Fatalf("Idle: got %d, want 8040", c.Idle)
	}
}

func TestParseStatMalformed(t *testing.T) {
	cases := map[string][]byte{
		"missing cpu line": []byte("intr 12345\nctxt 6789\n"),
		"too few fields":   []byte("cpu  100 10\n"),
		"non-numeric":      []byte("cpu  100 10 abc 8000 40\n"),
	}
	for name, in := range cases {
		if _, err := ParseStat(in); err == nil {
			t.Fatalf("%s: expected error, got nil", name)
		}
	}
}

func TestParseMeminfoWithSwap(t *testing.T) {
	in := []byte(`MemTotal:       16333764 kB
MemFree:         1000000 kB
MemAvailable:    8000000 kB
Buffers:          200000 kB
SwapTotal:       4194300 kB
SwapFree:        4000000 kB
`)
	mi, err := ParseMeminfo(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mi.TotalKB != 16333764 || mi.AvailableKB != 8000000 {
		t.Fatalf("mem: got total=%d avail=%d", mi.TotalKB, mi.AvailableKB)
	}
	if mi.SwapTotalKB != 4194300 || mi.SwapFreeKB != 4000000 {
		t.Fatalf("swap: got total=%d free=%d", mi.SwapTotalKB, mi.SwapFreeKB)
	}
}

func TestParseMeminfoWithoutSwap(t *testing.T) {
	in := []byte(`MemTotal:        2048000 kB
MemAvailable:    1024000 kB
`)
	mi, err := ParseMeminfo(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mi.SwapTotalKB != 0 || mi.SwapFreeKB != 0 {
		t.Fatalf("expected zero swap, got total=%d free=%d", mi.SwapTotalKB, mi.SwapFreeKB)
	}
}

func TestParseMeminfoMissingRequired(t *testing.T) {
	// MemAvailable absent -> error rather than misleading zero.
	if _, err := ParseMeminfo([]byte("MemTotal: 2048000 kB\nMemFree: 1000000 kB\n")); err == nil {
		t.Fatal("expected error for missing MemAvailable, got nil")
	}
	if _, err := ParseMeminfo([]byte("MemAvailable: 1024000 kB\n")); err == nil {
		t.Fatal("expected error for missing MemTotal, got nil")
	}
}

func TestParseDfNormal(t *testing.T) {
	in := []byte(`Filesystem     1024-blocks     Used Available Capacity Mounted on
/dev/sda1         41251136 12345678  26805458      32% /
`)
	du, err := ParseDf(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if du.Filesystem != "/dev/sda1" {
		t.Fatalf("filesystem: got %q", du.Filesystem)
	}
	if du.SizeKB != 41251136 || du.UsedKB != 12345678 || du.AvailKB != 26805458 {
		t.Fatalf("sizes: got size=%d used=%d avail=%d", du.SizeKB, du.UsedKB, du.AvailKB)
	}
	approx(t, du.UsePercent, 32, 0.001, "UsePercent")
}

func TestParseDfLongFilesystemName(t *testing.T) {
	// A long device-mapper name keeps df -P on one line; anchoring on the
	// Capacity column must still extract the right numeric fields.
	in := []byte(`Filesystem                            1024-blocks    Used Available Capacity Mounted on
/dev/mapper/ubuntu--vg-ubuntu--lv--root   102687672 5242880  92198520       6% /
`)
	du, err := ParseDf(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if du.Filesystem != "/dev/mapper/ubuntu--vg-ubuntu--lv--root" {
		t.Fatalf("filesystem: got %q", du.Filesystem)
	}
	if du.SizeKB != 102687672 || du.UsedKB != 5242880 || du.AvailKB != 92198520 {
		t.Fatalf("sizes: got size=%d used=%d avail=%d", du.SizeKB, du.UsedKB, du.AvailKB)
	}
	approx(t, du.UsePercent, 6, 0.001, "UsePercent")
}

func TestParseDfMalformed(t *testing.T) {
	if _, err := ParseDf([]byte("Filesystem 1024-blocks Used Available Capacity Mounted on\n")); err == nil {
		t.Fatal("expected error for header-only df, got nil")
	}
}

func TestParseLoadavg(t *testing.T) {
	la, err := ParseLoadavg([]byte("0.15 0.25 0.35 1/234 5678\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	approx(t, la.One, 0.15, 0.0001, "One")
	approx(t, la.Five, 0.25, 0.0001, "Five")
	approx(t, la.Fifteen, 0.35, 0.0001, "Fifteen")
}

func TestParseUptime(t *testing.T) {
	sec, err := ParseUptime([]byte("123456.78 987654.32\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	approx(t, sec, 123456.78, 0.001, "UptimeSec")
}

const fullLinuxFixture = `@@OS@@
Linux
@@STAT@@
cpu  100 10 50 8000 40 5 5 0
@@MEM@@
MemTotal:       16000000 kB
MemAvailable:    8000000 kB
SwapTotal:       4000000 kB
SwapFree:        3000000 kB
@@LOAD@@
0.10 0.20 0.30 1/100 2000
@@UP@@
54321.00 98765.00
@@DF@@
Filesystem     1024-blocks     Used Available Capacity Mounted on
/dev/sda1         40000000 10000000  30000000      25% /
`

func TestParseAllFullLinux(t *testing.T) {
	snap := ParseAll([]byte(fullLinuxFixture))
	if snap.OS != "Linux" || !snap.Supported {
		t.Fatalf("os: got %q supported=%v", snap.OS, snap.Supported)
	}
	// CPU must remain unset in ParseAll (needs a delta).
	if snap.CPUValid || snap.CPUPercent != 0 {
		t.Fatalf("CPU should be unset by ParseAll: valid=%v percent=%v", snap.CPUValid, snap.CPUPercent)
	}
	approx(t, snap.MemPercent, 50, 0.001, "MemPercent")      // 1 - 8000000/16000000
	if !snap.SwapPresent {
		t.Fatal("SwapPresent: got false, want true")
	}
	approx(t, snap.SwapPercent, 25, 0.001, "SwapPercent")    // 1 - 3000000/4000000
	approx(t, snap.Disk.UsePercent, 25, 0.001, "Disk.UsePercent")
	if snap.Disk.Filesystem != "/dev/sda1" {
		t.Fatalf("disk fs: got %q", snap.Disk.Filesystem)
	}
	approx(t, snap.Load.One, 0.10, 0.0001, "Load.One")
	approx(t, snap.Load.Fifteen, 0.30, 0.0001, "Load.Fifteen")
	approx(t, snap.UptimeSec, 54321.00, 0.001, "UptimeSec")
	if snap.SampledAt == 0 {
		t.Fatal("SampledAt: got 0, want current unix time")
	}
}

func TestParseAllUnsupportedOS(t *testing.T) {
	in := `@@OS@@
Darwin
@@STAT@@
@@MEM@@
@@LOAD@@
@@UP@@
@@DF@@
`
	snap := ParseAll([]byte(in))
	if snap.OS != "Darwin" || snap.Supported {
		t.Fatalf("os: got %q supported=%v, want Darwin/false", snap.OS, snap.Supported)
	}
	// Non-fatal: no /proc-derived fields should be populated.
	if snap.MemPercent != 0 || snap.SwapPresent || snap.Disk.SizeKB != 0 {
		t.Fatalf("unsupported OS should leave metrics zero: %+v", snap)
	}
	if snap.SampledAt == 0 {
		t.Fatal("SampledAt should still be set for unsupported OS")
	}
}

func TestParseAllMissingSectionsDegrades(t *testing.T) {
	// Linux OS but every metric section empty: must not panic, metrics stay zero.
	in := `@@OS@@
Linux
@@STAT@@
@@MEM@@
@@LOAD@@
@@UP@@
@@DF@@
`
	snap := ParseAll([]byte(in))
	if !snap.Supported {
		t.Fatal("expected Supported=true for Linux")
	}
	if snap.MemPercent != 0 || snap.Disk.SizeKB != 0 || snap.Load.One != 0 || snap.UptimeSec != 0 {
		t.Fatalf("missing sections should degrade to zero: %+v", snap)
	}
}

func TestCommandContainsAllMarkers(t *testing.T) {
	for _, m := range []string{markerOS, markerStat, markerMem, markerLoad, markerUp, markerDf} {
		if !strings.Contains(Command, m) {
			t.Fatalf("Command missing marker %q", m)
		}
	}
}

func TestManagerFirstSampleInvalid(t *testing.T) {
	m := NewManager()
	_, valid := m.Sample("s1", CPUCounters{Total: 1000, Idle: 800})
	if valid {
		t.Fatal("first sample should be invalid")
	}
}

func TestManagerNormalDelta(t *testing.T) {
	m := NewManager()
	m.Sample("s1", CPUCounters{Total: 1000, Idle: 800})
	// Next: total +1000, idle +600 -> busy 400/1000 -> 40%.
	pct, valid := m.Sample("s1", CPUCounters{Total: 2000, Idle: 1400})
	if !valid {
		t.Fatal("second sample should be valid")
	}
	approx(t, pct, 40, 0.001, "CPU%")
}

func TestManagerZeroDeltaInvalid(t *testing.T) {
	m := NewManager()
	c := CPUCounters{Total: 5000, Idle: 4000}
	m.Sample("s1", c)
	_, valid := m.Sample("s1", c) // identical counters -> totalDelta 0
	if valid {
		t.Fatal("zero total delta should be invalid")
	}
}

func TestManagerCounterResetInvalid(t *testing.T) {
	m := NewManager()
	m.Sample("s1", CPUCounters{Total: 9000, Idle: 8000})
	// Counters go backwards (reboot / new connection reusing the id).
	_, valid := m.Sample("s1", CPUCounters{Total: 100, Idle: 80})
	if valid {
		t.Fatal("counter reset should be invalid")
	}
	// Baseline must have been updated: the next valid delta works from the reset.
	pct, valid := m.Sample("s1", CPUCounters{Total: 1100, Idle: 580})
	if !valid {
		t.Fatal("sample after reset baseline should be valid")
	}
	approx(t, pct, 50, 0.001, "CPU% after reset") // total +1000, idle +500 -> 50%
}

func TestManagerForget(t *testing.T) {
	m := NewManager()
	m.Sample("s1", CPUCounters{Total: 1000, Idle: 800})
	m.Forget("s1")
	// After Forget, the next sample is treated as a first sample again.
	_, valid := m.Sample("s1", CPUCounters{Total: 2000, Idle: 1400})
	if valid {
		t.Fatal("sample after Forget should be invalid (baseline cleared)")
	}
}
