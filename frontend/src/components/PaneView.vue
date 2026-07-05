<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useSessions, type Pane } from "../stores/sessions";
import { useSettings } from "../stores/settings";
import {
  useWorkspaceLayout,
  TERM_MIN,
  SPLITTER_W,
} from "../composables/useWorkspaceLayout";
import TabBar from "./TabBar.vue";
import Terminal from "./Terminal.vue";
import SftpPanel from "./SftpPanel.vue";
import MonitorSidebar from "./MonitorSidebar.vue";
import CommandBar from "./CommandBar.vue";

const props = defineProps<{ pane: Pane }>();
const emit = defineEmits<{ (e: "activate"): void; (e: "close-pane"): void }>();

const sessions = useSessions();
const settings = useSettings();
const layout = useWorkspaceLayout();

const activeTab = computed(() => {
  const id = props.pane.activeTabId;
  return id ? sessions.tabs[id] : null;
});
const showMonitor = computed(() => !!activeTab.value?.showMonitor);
const showSftp = computed(() => !!activeTab.value?.showSftp);

// Live width of the .split container, tracked so the rendered panel widths can
// be scaled down on narrow windows (keeping the terminal usable) without
// mutating the user's stored preference.
const splitEl = ref<HTMLElement | null>(null);
const containerWidth = ref(0);
let ro: ResizeObserver | null = null;

// eff = the *rendered* panel widths. When the desired monitor+sftp widths plus
// the terminal minimum don't fit, both panels scale down proportionally so the
// terminal keeps TERM_MIN and the grid never overflows. On a comfortable window
// eff equals the stored desired widths.
const eff = computed(() => {
  const showM = showMonitor.value;
  const showS = showSftp.value;
  let mon = showM ? layout.monitorWidth.value : 0;
  let sftp = showS ? layout.sftpWidth.value : 0;
  const splitters = (showM ? SPLITTER_W : 0) + (showS ? SPLITTER_W : 0);
  const W = containerWidth.value;
  if (W > 0) {
    const avail = W - TERM_MIN - splitters;
    const desired = mon + sftp;
    if (desired > 0 && desired > avail) {
      const scale = Math.max(0, avail) / desired;
      mon = Math.floor(mon * scale);
      sftp = Math.floor(sftp * scale);
    }
  }
  return { mon, sftp, showM, showS };
});

const gridCols = computed(() => {
  const { mon, sftp, showM, showS } = eff.value;
  const parts: string[] = [];
  if (showM) {
    parts.push(`${mon}px`, `${SPLITTER_W}px`);
  }
  parts.push(`minmax(${TERM_MIN}px, 1fr)`);
  if (showS) {
    parts.push(`${SPLITTER_W}px`, `${sftp}px`);
  }
  return parts.join(" ");
});

// --- splitter drag ---
type DragKind = "monitor" | "sftp";
const draggingKind = ref<DragKind | null>(null);
let startX = 0;
let startVal = 0;
let raf = 0;
let pendingPx = 0;

function applyPending() {
  raf = 0;
  const kind = draggingKind.value;
  if (!kind) return;
  if (kind === "monitor") layout.setMonitorWidth(startVal + pendingPx);
  else layout.setSftpWidth(startVal - pendingPx); // sftp sits on the right: drag left widens
}

function onMove(e: PointerEvent) {
  if (!draggingKind.value) return;
  pendingPx = e.clientX - startX;
  if (!raf) raf = requestAnimationFrame(applyPending);
}

function endDrag() {
  if (!draggingKind.value) return;
  draggingKind.value = null;
  if (raf) {
    cancelAnimationFrame(raf);
    raf = 0;
  }
  document.body.style.cursor = "";
  document.body.style.userSelect = "";
  window.removeEventListener("pointermove", onMove);
  window.removeEventListener("pointerup", endDrag);
}

function onSplitterDown(kind: DragKind, e: PointerEvent) {
  // No preventDefault: it would suppress the dblclick-reset. userSelect:none on
  // the body covers text selection during the drag instead.
  draggingKind.value = kind;
  startX = e.clientX;
  // Start from the *rendered* width so grabbing the handle never jumps on a
  // narrow (scaled) window; committing rendered+delta is the intuitive result.
  startVal = kind === "monitor" ? eff.value.mon : eff.value.sftp;
  document.body.style.cursor = "col-resize";
  document.body.style.userSelect = "none";
  window.addEventListener("pointermove", onMove);
  window.addEventListener("pointerup", endDrag);
}

onMounted(() => {
  if (splitEl.value) {
    containerWidth.value = splitEl.value.clientWidth;
    ro = new ResizeObserver((entries) => {
      containerWidth.value = entries[0].contentRect.width;
    });
    ro.observe(splitEl.value);
  }
});
onBeforeUnmount(() => {
  endDrag();
  ro?.disconnect();
});
</script>

<template>
  <section class="pane" @mousedown="emit('activate')">
    <TabBar :pane="pane" @close-pane="emit('close-pane')" />
    <div class="pane-body">
      <template v-if="activeTab">
        <div class="terminal-area">
          <div ref="splitEl" class="split" :style="{ gridTemplateColumns: gridCols }">
            <MonitorSidebar v-if="showMonitor" :tab-id="activeTab.id" />
            <div
              v-if="showMonitor"
              class="splitter"
              :class="{ dragging: draggingKind === 'monitor' }"
              title="拖动调整监控面板宽度（双击重置）"
              @pointerdown="onSplitterDown('monitor', $event)"
              @dblclick="layout.resetMonitorWidth()"
            />
            <div class="term-col">
              <Terminal :tab-id="activeTab.id" />
              <CommandBar
                v-if="settings.settings.showCommandBar && activeTab.status === 'open'"
                :tab-id="activeTab.id"
                :pane-id="pane.id"
              />
            </div>
            <div
              v-if="showSftp"
              class="splitter"
              :class="{ dragging: draggingKind === 'sftp' }"
              title="拖动调整 SFTP 面板宽度（双击重置）"
              @pointerdown="onSplitterDown('sftp', $event)"
              @dblclick="layout.resetSftpWidth()"
            />
            <SftpPanel v-if="showSftp" :tab-id="activeTab.id" :pane-id="pane.id" />
          </div>
        </div>
      </template>
      <div v-else class="empty">
        <div class="empty-card">
          <h3>No session</h3>
          <p>Double-click a host in the sidebar to connect.</p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.pane {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--bg);
}
.pane-body {
  flex: 1;
  min-height: 0;
  position: relative;
}
.terminal-area {
  height: 100%;
}
/* Column order matches child order: [MonitorSidebar splitter?] term-col
   [splitter SftpPanel?]. gridTemplateColumns is set inline from the draggable
   widths. Changing a column resizes the terminal element, whose ResizeObserver
   refits xterm automatically (no manual fit call needed). */
.split {
  display: grid;
  height: 100%;
  width: 100%;
}
.term-col {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}

/* Draggable splitter: a 6px hit area with a thin line shown on hover/drag. The
   adjacent panels already carry a 1px border, so the rest state stays quiet. */
.splitter {
  position: relative;
  cursor: col-resize;
  background: transparent;
  z-index: 5;
  touch-action: none;
}
.splitter::before {
  content: "";
  position: absolute;
  top: 0;
  bottom: 0;
  left: 2px;
  right: 2px;
  border-radius: 2px;
  background: var(--accent);
  opacity: 0;
  transition: opacity 0.15s ease;
}
.splitter:hover::before,
.splitter.dragging::before {
  opacity: 0.55;
}
.splitter.dragging::before {
  opacity: 0.9;
}

.empty {
  height: 100%;
  display: grid;
  place-items: center;
  color: var(--fg-muted);
}
.empty-card {
  text-align: center;
  padding: 32px;
}
.empty-card h3 {
  margin: 0 0 6px;
  font-weight: 600;
}
.empty-card p {
  margin: 0;
  font-size: 12px;
}
</style>
