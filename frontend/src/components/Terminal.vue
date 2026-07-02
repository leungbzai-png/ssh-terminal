<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, nextTick, computed } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { SearchAddon } from "@xterm/addon-search";
import { useSessions } from "../stores/sessions";
import { useSettings } from "../stores/settings";
import { useTheme } from "../composables/useTheme";

const props = defineProps<{ tabId: string }>();
const sessions = useSessions();
const settings = useSettings();
const theme = useTheme();

const wrap = ref<HTMLElement | null>(null);
const searchInput = ref<HTMLInputElement | null>(null);
let term: Terminal | null = null;
let fit: FitAddon | null = null;
let search: SearchAddon | null = null;
let ro: ResizeObserver | null = null;
let unlistenData: (() => void) | null = null;
let unlistenClose: (() => void) | null = null;

const showSearch = ref(false);
const searchText = ref("");
const caseSensitive = ref(false);
const wholeWord = ref(false);
const useRegex = ref(false);
// Live match feedback from the SearchAddon (resultIndex is -1 when no match).
const matchIndex = ref(-1);
const matchCount = ref(0);
const matchLabel = computed(() => {
  if (!searchText.value) return "";
  if (matchCount.value === 0) return "无匹配";
  return `${matchIndex.value + 1}/${matchCount.value}`;
});

const tab = computed(() => sessions.tabs[props.tabId]);

function readPalette() {
  const css = getComputedStyle(document.documentElement);
  const g = (n: string) => css.getPropertyValue(n).trim();
  return {
    background: g("--t-bg"),
    foreground: g("--t-fg"),
    cursor: g("--t-cursor"),
    cursorAccent: g("--t-bg"),
    selectionBackground: g("--t-selection"),
    black: g("--t-black"),
    red: g("--t-red"),
    green: g("--t-green"),
    yellow: g("--t-yellow"),
    blue: g("--t-blue"),
    magenta: g("--t-magenta"),
    cyan: g("--t-cyan"),
    white: g("--t-white"),
    brightBlack: g("--t-bright-black"),
    brightRed: g("--t-bright-red"),
    brightGreen: g("--t-bright-green"),
    brightYellow: g("--t-bright-yellow"),
    brightBlue: g("--t-bright-blue"),
    brightMagenta: g("--t-bright-magenta"),
    brightCyan: g("--t-bright-cyan"),
    brightWhite: g("--t-bright-white"),
  };
}

async function startSession() {
  if (!term || !fit) return;
  fit.fit();
  const cols = term.cols;
  const rows = term.rows;
  sessions.setTabStatus(props.tabId, "connecting");
  try {
    const quick = sessions.quickParams[props.tabId];
    if (tab.value?.quick && quick) {
      await window.go.main.App.SshOpenQuick(props.tabId, quick, cols, rows);
    } else {
      await window.go.main.App.OpenSession(props.tabId, tab.value!.hostId, cols, rows);
    }
    sessions.setTabStatus(props.tabId, "open");
  } catch (e: any) {
    const msg = String(e?.message || e);
    term.writeln(`\x1b[31m${msg}\x1b[0m`);
    sessions.setTabStatus(props.tabId, "error", msg);
  }
}

async function reconnect() {
  if (!term) return;
  term.writeln("\r\n\x1b[36m— reconnecting…\x1b[0m\r\n");
  await startSession();
}

function applySettings() {
  if (!term) return;
  term.options.fontFamily = settings.settings.fontFamily;
  term.options.fontSize = settings.settings.fontSize;
  term.options.cursorStyle = settings.settings.cursorStyle;
  term.options.cursorBlink = settings.settings.cursorBlink;
  term.options.scrollback = settings.settings.scrollBack;
  term.options.theme = readPalette();
  fit?.fit();
}

function searchOpts() {
  return {
    regex: useRegex.value,
    wholeWord: wholeWord.value,
    caseSensitive: caseSensitive.value,
    decorations: {
      matchBackground: "#7dd3fc",
      matchBorder: "#7dd3fc",
      matchOverviewRuler: "#7dd3fc",
      activeMatchBackground: "#fcd34d",
      activeMatchBorder: "#fcd34d",
      activeMatchColorOverviewRuler: "#fcd34d",
    },
  };
}

function findNext() {
  if (!search || !searchText.value) return;
  search.findNext(searchText.value, searchOpts() as any);
}
function findPrev() {
  if (!search || !searchText.value) return;
  search.findPrevious(searchText.value, searchOpts() as any);
}
function closeSearch() {
  showSearch.value = false;
  search?.clearDecorations();
  matchIndex.value = -1;
  matchCount.value = 0;
  term?.focus();
}

// Live incremental search as the user types, so the no-match / count indicator
// updates without waiting for Enter.
watch(searchText, (v) => {
  if (!search) return;
  if (!v) {
    search.clearDecorations();
    matchIndex.value = -1;
    matchCount.value = 0;
    return;
  }
  search.findNext(v, searchOpts() as any);
});
function openSearch() {
  showSearch.value = true;
  nextTick(() => searchInput.value?.focus());
}

function onSearchKey(e: KeyboardEvent) {
  if (e.key === "Enter") {
    e.preventDefault();
    e.shiftKey ? findPrev() : findNext();
  } else if (e.key === "Escape") {
    e.preventDefault();
    closeSearch();
  }
}

function onTermKey(domEvent: KeyboardEvent): boolean {
  if (domEvent.ctrlKey && (domEvent.key === "f" || domEvent.key === "F")) {
    domEvent.preventDefault();
    openSearch();
    return false;
  }
  // Font size: Ctrl+= / Ctrl++ (increase), Ctrl+- (decrease), Ctrl+0 (reset).
  if (domEvent.ctrlKey && !domEvent.altKey && !domEvent.metaKey) {
    if (domEvent.key === "=" || domEvent.key === "+") {
      domEvent.preventDefault();
      settings.bumpFontSize(1);
      return false;
    }
    if (domEvent.key === "-" || domEvent.key === "_") {
      domEvent.preventDefault();
      settings.bumpFontSize(-1);
      return false;
    }
    if (domEvent.key === "0") {
      domEvent.preventDefault();
      settings.resetFontSize();
      return false;
    }
  }
  return true;
}

onMounted(async () => {
  if (!wrap.value) return;
  term = new Terminal({
    convertEol: false,
    allowProposedApi: true,
    fontFamily: settings.settings.fontFamily,
    fontSize: settings.settings.fontSize,
    cursorStyle: settings.settings.cursorStyle,
    cursorBlink: settings.settings.cursorBlink,
    scrollback: settings.settings.scrollBack,
    theme: readPalette(),
    macOptionIsMeta: true,
  });
  fit = new FitAddon();
  search = new SearchAddon();
  search.onDidChangeResults((e: any) => {
    matchIndex.value = e?.resultIndex ?? -1;
    matchCount.value = e?.resultCount ?? 0;
  });
  term.loadAddon(fit);
  term.loadAddon(new WebLinksAddon());
  term.loadAddon(search);
  term.open(wrap.value);
  term.attachCustomKeyEventHandler(onTermKey);
  await nextTick();
  fit.fit();

  term.onData((d) => {
    const b64 = btoa(unescape(encodeURIComponent(d)));
    window.go.main.App.WriteSession(props.tabId, b64).catch(() => {});
  });
  term.onResize(({ cols, rows }) => {
    window.go.main.App.ResizeSession(props.tabId, cols, rows).catch(() => {});
  });

  const dataEvt = `ssh:data:${props.tabId}`;
  const closeEvt = `ssh:close:${props.tabId}`;
  const onData = (b64: string) => {
    if (!term) return;
    const bin = atob(b64);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    term.write(bytes);
  };
  const onClose = (reason: string) => {
    if (term && reason) term.writeln(`\r\n\x1b[90m— session closed: ${reason}\x1b[0m`);
    sessions.setTabStatus(props.tabId, "closed");
  };
  window.runtime.EventsOn(dataEvt, onData);
  window.runtime.EventsOn(closeEvt, onClose);
  unlistenData = () => window.runtime.EventsOff(dataEvt);
  unlistenClose = () => window.runtime.EventsOff(closeEvt);

  ro = new ResizeObserver(() => fit?.fit());
  ro.observe(wrap.value);

  await startSession();
});

watch(() => settings.settings, applySettings, { deep: true });
watch(() => theme.resolved.value, () => applySettings());

// Watch the reconnect tick from sessions store so external code can trigger
// reconnect via sessions.bumpReconnect(tabId).
watch(
  () => sessions.reconnectTick[props.tabId],
  (v) => {
    if (v) reconnect();
  }
);

onBeforeUnmount(async () => {
  unlistenData?.();
  unlistenClose?.();
  ro?.disconnect();
  try {
    await window.go.main.App.CloseSession(props.tabId);
  } catch {}
  term?.dispose();
  term = null;
});
</script>

<template>
  <div class="term-wrap">
    <div ref="wrap" class="term" />

    <Transition name="search">
      <div v-if="showSearch" class="search-bar" @keydown.stop>
        <svg class="ic" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>
        <input
          ref="searchInput"
          v-model="searchText"
          type="text"
          class="search-input"
          placeholder="搜索 (Enter 下一个 · Shift+Enter 上一个 · Esc 关闭)"
          spellcheck="false"
          @keydown="onSearchKey"
        />
        <span class="match" :class="{ none: !!searchText && matchCount === 0 }">{{ matchLabel }}</span>
        <button type="button" class="icon-btn" :class="{ on: caseSensitive }" title="区分大小写 (Aa)" @click="caseSensitive = !caseSensitive">Aa</button>
        <button type="button" class="icon-btn" :class="{ on: wholeWord }" title="全字匹配" @click="wholeWord = !wholeWord">W</button>
        <button type="button" class="icon-btn" :class="{ on: useRegex }" title="正则" @click="useRegex = !useRegex">.*</button>
        <button type="button" class="icon-btn" title="上一个" @click="findPrev">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M18 15l-6-6-6 6"/></svg>
        </button>
        <button type="button" class="icon-btn" title="下一个" @click="findNext">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M6 9l6 6 6-6"/></svg>
        </button>
        <button type="button" class="icon-btn" title="关闭" @click="closeSearch">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M6 6l12 12M18 6L6 18"/></svg>
        </button>
      </div>
    </Transition>

    <Transition name="fade">
      <div v-if="tab?.status === 'closed' || tab?.status === 'error'" class="reconnect-overlay">
        <div class="reconnect-card">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 1 1-3-6.7L21 8" />
            <path d="M21 3v5h-5" />
          </svg>
          <div class="reconnect-title">{{ tab?.status === 'error' ? '连接出错' : '已断开' }}</div>
          <div class="reconnect-sub" v-if="tab?.error">{{ tab.error }}</div>
          <button type="button" class="primary" @click="reconnect">重新连接</button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.term-wrap {
  position: relative;
  width: 100%;
  height: 100%;
  background: var(--t-bg);
  padding: 8px 4px 8px 10px;
  min-width: 0;
  min-height: 0;
}
.term {
  width: 100%;
  height: 100%;
}

.search-bar {
  position: absolute;
  top: 8px;
  right: 16px;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 6px;
  background: var(--bg-elev-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  z-index: 30;
}
.search-bar .ic {
  color: var(--fg-muted);
  margin: 0 2px 0 4px;
  flex-shrink: 0;
}
.search-input {
  width: 240px;
  padding: 4px 8px;
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--fg);
  border-radius: var(--radius-sm);
  font-size: 12px;
}
.search-input::placeholder {
  color: var(--fg-subtle);
}
.match {
  font-size: 11px;
  color: var(--fg-muted);
  min-width: 34px;
  text-align: center;
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}
.match.none {
  color: var(--danger);
}
.icon-btn.on {
  background: var(--bg-active);
  color: var(--accent);
}
.search-bar .icon-btn {
  width: 24px;
  height: 24px;
  font-size: 10.5px;
  font-weight: 600;
}

.reconnect-overlay {
  position: absolute;
  inset: 0;
  background: color-mix(in oklab, var(--t-bg) 85%, transparent);
  display: grid;
  place-items: center;
  z-index: 20;
  backdrop-filter: blur(2px);
}
.reconnect-card {
  text-align: center;
  padding: 24px 32px;
  color: var(--fg);
}
.reconnect-card svg {
  color: var(--fg-muted);
  margin-bottom: 8px;
}
.reconnect-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}
.reconnect-sub {
  font-size: 12px;
  color: var(--fg-muted);
  margin-bottom: 14px;
  max-width: 360px;
}
.reconnect-card .primary {
  background: var(--accent);
  color: var(--accent-fg);
  border-color: var(--accent);
  padding: 7px 18px;
  font-weight: 500;
}

.search-enter-active,
.search-leave-active {
  transition: opacity 0.12s ease, transform 0.12s ease;
}
.search-enter-from,
.search-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
