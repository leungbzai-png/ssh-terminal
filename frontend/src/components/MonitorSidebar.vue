<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from "vue";
import { useSessions } from "../stores/sessions";
import Sparkline from "./Sparkline.vue";

// MonitorSidebar is the left-side, per-tab VPS monitor panel (v1.2.0). It owns a
// single poll timer that calls App.MonitorSample on the interval selected for
// the tab; all sampled state lives in the sessions store (keyed by tab id) so it
// survives tab switches — this component instance is reused across tabs. Polling
// runs only while the tab is connected AND the panel is shown (hiding the panel
// unmounts this component, stopping the timer), so there is no background
// monitoring after disconnect or close.

const props = defineProps<{ tabId: string }>();
const sessions = useSessions();

const tab = computed(() => sessions.tabs[props.tabId]);
const connected = computed(() => tab.value?.status === "open");

// Polling interval options (seconds); per-tab, default 5.
const INTERVALS = [2, 5, 10] as const;
const intervalSec = computed(() => sessions.monitorInterval[props.tabId] ?? 5);

// Live sample + trend buffers, read from the per-tab store state.
const snapshot = computed(() => sessions.monitorSnapshot[props.tabId] ?? null);
const error = computed(() => sessions.monitorError[props.tabId] ?? "");
const cpuHistory = computed(() => sessions.monitorHistory[props.tabId]?.cpu ?? []);
const memHistory = computed(() => sessions.monitorHistory[props.tabId]?.mem ?? []);

// --- polling ---
let timer: ReturnType<typeof setInterval> | null = null;
let inFlight = false;

function stopTimer() {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
}

// poll takes one sample for the current tab. It skips if a previous sample is
// still in flight (so a slow link at 2s never stacks calls) and re-checks the
// tab id / connection after the await to avoid writing a late result to the
// wrong tab or after a disconnect.
async function poll() {
  if (inFlight) return;
  const id = props.tabId;
  if (sessions.tabs[id]?.status !== "open") return;
  inFlight = true;
  try {
    const snap = await window.go.main.App.MonitorSample(id);
    if (props.tabId !== id || sessions.tabs[id]?.status !== "open") return;
    sessions.setMonitorSnapshot(id, snap);
    sessions.setMonitorError(id, "");
    if (snap.supported) {
      sessions.pushMonitorSample(id, snap.cpuValid ? snap.cpuPercent : null, snap.memPercent);
    }
  } catch (e: any) {
    // A drop mid-poll surfaces here; show it only while the tab is still open.
    if (props.tabId === id && sessions.tabs[id]?.status === "open") {
      sessions.setMonitorError(id, String(e?.message || e));
    }
  } finally {
    inFlight = false;
  }
}

// rebuild (re)starts the timer for the current tab/interval. Called whenever the
// tab, connection state, or interval changes. When disconnected it just stops.
function rebuild() {
  stopTimer();
  if (!connected.value) return;
  poll(); // immediate first sample
  timer = setInterval(poll, Math.max(1, intervalSec.value) * 1000);
}

watch([() => props.tabId, connected, intervalSec], rebuild, { immediate: true });
// Clear a tab's stale reading/trend when it disconnects so a reconnect is fresh.
watch(connected, (isConn) => {
  if (!isConn) sessions.resetMonitor(props.tabId);
});
onBeforeUnmount(stopTimer);

function selectInterval(s: number) {
  sessions.setMonitorInterval(props.tabId, s);
}

function close() {
  sessions.toggleMonitor(props.tabId);
}

// --- formatting helpers ---
function pct(v: number): string {
  return `${Math.round(v)}%`;
}
function humanKB(kb: number): string {
  const units = ["KB", "MB", "GB", "TB", "PB"];
  let v = kb;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  const dp = i === 0 || v >= 100 ? 0 : 1;
  return `${v.toFixed(dp)} ${units[i]}`;
}
function fmtUptime(sec: number): string {
  const s = Math.floor(sec);
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d}天 ${h}小时`;
  if (h > 0) return `${h}小时 ${m}分`;
  return `${m}分`;
}
// Severity class for a usage percentage (drives the accent color).
function levelOf(v: number): "ok" | "warn" | "crit" {
  if (v >= 90) return "crit";
  if (v >= 70) return "warn";
  return "ok";
}

const cpuText = computed(() => {
  const s = snapshot.value;
  if (!s) return "—";
  return s.cpuValid ? pct(s.cpuPercent) : "测量中…";
});
</script>

<template>
  <aside class="monitor">
    <header>
      <span class="title">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 12h4l3 8 4-16 3 8h4" />
        </svg>
        VPS 监控
      </span>
      <div class="intervals" role="group" aria-label="采样间隔">
        <button
          v-for="s in INTERVALS"
          :key="s"
          type="button"
          class="int-btn"
          :class="{ on: intervalSec === s }"
          @click="selectInterval(s)"
        >
          {{ s }}s
        </button>
      </div>
      <button class="icon-btn close" title="关闭监控" @click="close">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round">
          <path d="M6 6l12 12M18 6L6 18" />
        </svg>
      </button>
    </header>

    <!-- 1. No active session -->
    <div v-if="!connected" class="state">
      <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d="M18.36 6.64a9 9 0 1 1-12.73 0" /><line x1="12" y1="2" x2="12" y2="12" />
      </svg>
      <p class="state-title">无活动会话</p>
      <p class="state-sub">连接到主机后可查看实时资源监控。</p>
    </div>

    <!-- 2. Error -->
    <div v-else-if="error" class="state">
      <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="9" /><line x1="12" y1="8" x2="12" y2="13" /><line x1="12" y1="16.5" x2="12" y2="16.5" />
      </svg>
      <p class="state-title">监控出错</p>
      <p class="state-sub">{{ error }}</p>
    </div>

    <!-- 3. Loading (first sample not yet available) -->
    <div v-else-if="!snapshot" class="skeleton">
      <div v-for="n in 4" :key="n" class="sk-card" />
      <p class="state-sub center">等待数据…</p>
    </div>

    <!-- 4. Unsupported host (non-Linux) -->
    <div v-else-if="!snapshot.supported" class="state">
      <svg width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="4" width="18" height="14" rx="2" /><line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="18" x2="12" y2="21" />
      </svg>
      <p class="state-title">不支持的主机</p>
      <p class="state-sub">实时监控目前仅支持 Linux 主机{{ snapshot.os ? `（检测到 ${snapshot.os}）` : "" }}。</p>
    </div>

    <!-- 5. Ready: live metrics (wired in commit 4) -->
    <div v-else class="cards">
      <div class="card" :class="'lvl-' + (snapshot.cpuValid ? levelOf(snapshot.cpuPercent) : 'ok')">
        <div class="card-head"><span class="label">CPU</span><span class="value">{{ cpuText }}</span></div>
        <Sparkline :values="cpuHistory" :max="100" />
      </div>

      <div class="card" :class="'lvl-' + levelOf(snapshot.memPercent)">
        <div class="card-head"><span class="label">内存</span><span class="value">{{ pct(snapshot.memPercent) }}</span></div>
        <div class="bar"><span :style="{ width: Math.min(100, snapshot.memPercent) + '%' }" /></div>
        <Sparkline :values="memHistory" :max="100" />
      </div>

      <div class="card" :class="snapshot.swapPresent ? 'lvl-' + levelOf(snapshot.swapPercent) : 'lvl-muted'">
        <div class="card-head"><span class="label">Swap</span><span class="value">{{ snapshot.swapPresent ? pct(snapshot.swapPercent) : "无" }}</span></div>
        <div v-if="snapshot.swapPresent" class="bar"><span :style="{ width: Math.min(100, snapshot.swapPercent) + '%' }" /></div>
      </div>

      <div class="card" :class="'lvl-' + levelOf(snapshot.disk.usePercent)">
        <div class="card-head"><span class="label">磁盘 /</span><span class="value">{{ pct(snapshot.disk.usePercent) }}</span></div>
        <div class="bar"><span :style="{ width: Math.min(100, snapshot.disk.usePercent) + '%' }" /></div>
        <div class="sub-line">{{ humanKB(snapshot.disk.usedKB) }} / {{ humanKB(snapshot.disk.sizeKB) }}</div>
      </div>

      <div class="meta">
        <div class="meta-row">
          <span class="label">负载</span>
          <span class="mono">{{ snapshot.load.one.toFixed(2) }} · {{ snapshot.load.five.toFixed(2) }} · {{ snapshot.load.fifteen.toFixed(2) }}</span>
        </div>
        <div class="meta-row">
          <span class="label">运行</span>
          <span class="mono">{{ fmtUptime(snapshot.uptimeSec) }}</span>
        </div>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.monitor {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--bg-elev);
  border-right: 1px solid var(--border);
  overflow: hidden;
}
header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.title {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 600;
  color: var(--fg);
  white-space: nowrap;
}
.title svg {
  color: var(--accent);
}
.intervals {
  display: flex;
  gap: 1px;
  margin-left: auto;
}
.int-btn {
  font-size: 10.5px;
  padding: 2px 5px;
  color: var(--fg-muted);
  border-radius: var(--radius-sm);
  font-variant-numeric: tabular-nums;
}
.int-btn.on {
  background: var(--bg-active);
  color: var(--accent);
}
.close {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

/* --- empty / error / unsupported states --- */
.state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 24px 16px;
  gap: 4px;
  color: var(--fg-muted);
}
.state svg {
  color: var(--fg-subtle);
  margin-bottom: 4px;
}
.state-title {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--fg);
  margin: 0;
}
.state-sub {
  font-size: 11px;
  line-height: 1.5;
  color: var(--fg-muted);
  margin: 0;
  max-width: 220px;
}
.state-sub.center {
  text-align: center;
}

/* --- loading skeleton --- */
.skeleton {
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.sk-card {
  height: 52px;
  border-radius: var(--radius);
  background: var(--bg-hover);
  animation: sk-pulse 1.4s ease-in-out infinite;
}
.sk-card:nth-child(2) { animation-delay: 0.15s; }
.sk-card:nth-child(3) { animation-delay: 0.3s; }
.sk-card:nth-child(4) { animation-delay: 0.45s; }
@keyframes sk-pulse {
  0%, 100% { opacity: 0.45; }
  50% { opacity: 0.9; }
}

/* --- metric cards --- */
.cards {
  flex: 1;
  overflow-y: auto;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.card {
  padding: 8px 10px;
  border-radius: var(--radius);
  background: var(--bg);
  border: 1px solid var(--border);
  --lvl: var(--accent);
}
.card.lvl-warn { --lvl: var(--warning); }
.card.lvl-crit { --lvl: var(--danger); }
.card.lvl-muted { --lvl: var(--fg-subtle); }
.card-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 6px;
}
.card .label {
  font-size: 11px;
  color: var(--fg-muted);
}
.card .value {
  font-size: 15px;
  font-weight: 600;
  color: var(--lvl);
  font-variant-numeric: tabular-nums;
}
.card :deep(.spark) {
  color: var(--lvl);
}
.bar {
  height: 5px;
  border-radius: 3px;
  background: var(--bg-hover);
  overflow: hidden;
  margin-bottom: 6px;
}
.bar > span {
  display: block;
  height: 100%;
  background: var(--lvl);
  border-radius: 3px;
  transition: width 0.3s ease;
}
.sub-line {
  font-size: 10.5px;
  color: var(--fg-muted);
  font-variant-numeric: tabular-nums;
}
.meta {
  padding: 8px 10px;
  border-radius: var(--radius);
  background: var(--bg);
  border: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.meta-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}
.meta .label {
  font-size: 11px;
  color: var(--fg-muted);
}
.mono {
  font-size: 11.5px;
  color: var(--fg);
  font-variant-numeric: tabular-nums;
}
</style>
