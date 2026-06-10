<!--
  FinalShell-style command input bar.
  - Enter sends to the current tab.
  - Ctrl+Enter broadcasts to all tabs in the same pane.
  - Up/Down browses per-session history.
  - Esc clears.
  - Alt+H toggles the history dropdown.
  History is kept in localStorage so it survives reloads but is per-host (by name).
-->
<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue";
import { useSessions } from "../stores/sessions";

const props = defineProps<{ tabId: string; paneId: string }>();
const sessions = useSessions();

const cmd = ref("");
const broadcast = ref(false);
const showHistory = ref(false);
const inputEl = ref<HTMLInputElement | null>(null);

const tab = computed(() => sessions.tabs[props.tabId]);
const historyKey = computed(() => `cmdhist:${tab.value?.hostName || "default"}`);

const history = ref<string[]>([]);
let historyIdx = -1;

function loadHistory() {
  try {
    const raw = localStorage.getItem(historyKey.value);
    history.value = raw ? JSON.parse(raw) : [];
  } catch {
    history.value = [];
  }
  historyIdx = -1;
}
function pushHistory(item: string) {
  const s = item.trim();
  if (!s) return;
  // Dedupe: remove existing occurrence, then push to front.
  history.value = [s, ...history.value.filter((x) => x !== s)].slice(0, 200);
  try {
    localStorage.setItem(historyKey.value, JSON.stringify(history.value));
  } catch {}
  historyIdx = -1;
}

onMounted(loadHistory);
watch(historyKey, loadHistory);

function encodeUtf8B64(s: string) {
  // btoa requires Latin-1; encode UTF-8 first.
  const bytes = new TextEncoder().encode(s);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin);
}

async function sendToTab(tabId: string, text: string) {
  try {
    await window.go.main.App.WriteSession(tabId, encodeUtf8B64(text));
  } catch (e) {
    console.warn("send failed", tabId, e);
  }
}

async function send(forceBroadcast = false) {
  const text = cmd.value;
  if (!text) return;
  const payload = text + "\n";
  const doBroadcast = forceBroadcast || broadcast.value;
  if (doBroadcast) {
    const pane = sessions.panes.find((p) => p.id === props.paneId);
    const targets = pane?.tabIds.filter((id) => sessions.tabs[id]?.status === "open") || [];
    await Promise.all(targets.map((id) => sendToTab(id, payload)));
  } else {
    await sendToTab(props.tabId, payload);
  }
  pushHistory(text);
  cmd.value = "";
  historyIdx = -1;
}

function onKey(e: KeyboardEvent) {
  if (e.key === "Enter") {
    e.preventDefault();
    send(e.ctrlKey);
  } else if (e.key === "Escape") {
    e.preventDefault();
    if (showHistory.value) {
      showHistory.value = false;
    } else {
      cmd.value = "";
    }
  } else if (e.key === "ArrowUp") {
    if (history.value.length === 0) return;
    e.preventDefault();
    historyIdx = Math.min(historyIdx + 1, history.value.length - 1);
    cmd.value = history.value[historyIdx];
  } else if (e.key === "ArrowDown") {
    if (history.value.length === 0) return;
    e.preventDefault();
    historyIdx = Math.max(historyIdx - 1, -1);
    cmd.value = historyIdx === -1 ? "" : history.value[historyIdx];
  } else if (e.altKey && (e.key === "h" || e.key === "H")) {
    e.preventDefault();
    showHistory.value = !showHistory.value;
  }
}

function pickHistory(item: string) {
  cmd.value = item;
  showHistory.value = false;
  nextTick(() => inputEl.value?.focus());
}

function clearHistory() {
  if (!confirm("清空本主机的命令历史？")) return;
  history.value = [];
  try {
    localStorage.removeItem(historyKey.value);
  } catch {}
  showHistory.value = false;
}

// Expose focus so parent can focus on demand.
defineExpose({ focus: () => inputEl.value?.focus() });
</script>

<template>
  <div class="cmdbar" :class="{ broadcast }">
    <button
      type="button"
      class="icon-btn"
      :class="{ on: showHistory }"
      title="历史 (Alt+H)"
      @click="showHistory = !showHistory"
    >
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M3 12a9 9 0 1 0 3-6.7" />
        <polyline points="3 4 3 10 9 10" />
        <polyline points="12 7 12 12 15 14" />
      </svg>
    </button>

    <input
      ref="inputEl"
      v-model="cmd"
      type="text"
      class="cmdinput"
      :placeholder="broadcast ? '广播命令到本分屏所有标签 — Enter 发送' : '命令输入 — Enter 发送  Ctrl+Enter 广播  ↑↓ 历史  Esc 清空'"
      spellcheck="false"
      autocomplete="off"
      @keydown="onKey"
    />

    <label class="toggle" title="广播：同时发送到本分屏所有打开的标签">
      <input type="checkbox" v-model="broadcast" />
      <span>广播</span>
    </label>

    <button type="button" class="send" @click="send(false)">发送</button>

    <transition name="hist">
      <div v-if="showHistory" class="history">
        <div class="hist-head">
          <span>历史命令</span>
          <button type="button" class="icon-btn" @click="clearHistory" title="清空">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M6 6l1 14a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-14"/></svg>
          </button>
        </div>
        <ul v-if="history.length" class="hist-list">
          <li v-for="(h, i) in history" :key="i" @click="pickHistory(h)">{{ h }}</li>
        </ul>
        <div v-else class="hist-empty">暂无历史</div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
.cmdbar {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  background: var(--bg-elev);
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}
.cmdbar.broadcast {
  background: color-mix(in oklab, var(--warning) 8%, var(--bg-elev));
  border-top-color: var(--warning);
}

.cmdinput {
  flex: 1;
  font-family: var(--mono, "JetBrains Mono", Consolas, monospace);
  font-size: 12.5px;
  padding: 5px 9px;
  background: var(--bg);
}
.cmdinput::placeholder {
  color: var(--fg-subtle);
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11.5px;
  color: var(--fg-muted);
  padding: 4px 6px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  user-select: none;
}
.toggle:hover {
  background: var(--bg-hover);
  color: var(--fg);
}
.toggle input {
  margin: 0;
  width: 13px;
  height: 13px;
  accent-color: var(--accent);
}

.send {
  background: var(--accent);
  color: var(--accent-fg);
  border-color: var(--accent);
  padding: 5px 14px;
  font-weight: 500;
  font-size: 12px;
}
.send:hover {
  filter: brightness(1.08);
  background: var(--accent);
}

.icon-btn.on {
  background: var(--bg-active);
  color: var(--accent);
}

.history {
  position: absolute;
  bottom: calc(100% + 4px);
  left: 8px;
  right: 8px;
  background: var(--bg-elev-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  max-height: 240px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  z-index: 50;
}
.hist-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  border-bottom: 1px solid var(--border);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--fg-subtle);
}
.hist-list {
  list-style: none;
  margin: 0;
  padding: 4px;
  overflow-y: auto;
}
.hist-list li {
  padding: 5px 8px;
  border-radius: var(--radius-sm);
  font-family: var(--mono, monospace);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.hist-list li:hover {
  background: var(--bg-hover);
}
.hist-empty {
  padding: 16px;
  text-align: center;
  color: var(--fg-muted);
  font-size: 12px;
}

.hist-enter-active,
.hist-leave-active {
  transition: opacity 0.12s ease, transform 0.12s ease;
}
.hist-enter-from,
.hist-leave-to {
  opacity: 0;
  transform: translateY(4px);
}
</style>
