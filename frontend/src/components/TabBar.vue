<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useSessions, type Pane } from "../stores/sessions";
import { useHosts } from "../stores/hosts";
import ConfirmDialog from "./ConfirmDialog.vue";

const props = defineProps<{ pane: Pane }>();
const emit = defineEmits<{ (e: "close-pane"): void }>();

const sessions = useSessions();
const hosts = useHosts();

const tabs = computed(() =>
  props.pane.tabIds.map((id) => sessions.tabs[id]).filter(Boolean)
);
const activeTab = computed(() =>
  props.pane.activeTabId ? sessions.tabs[props.pane.activeTabId] : null
);

const ctx = ref<{ x: number; y: number; tabId: string } | null>(null);
const pendingCloseTabId = ref<string | null>(null);

function dotClass(status: string) {
  return status === "open"
    ? "ok"
    : status === "connecting"
    ? "pending"
    : status === "idle"
    ? "idle"
    : "fail";
}

function openCtx(e: MouseEvent, tabId: string) {
  e.preventDefault();
  ctx.value = { x: e.clientX, y: e.clientY, tabId };
}
function closeCtx() { ctx.value = null; }

function reconnect(tabId: string) {
  sessions.bumpReconnect(tabId);
  closeCtx();
}
function duplicate(tabId: string) {
  const t = sessions.tabs[tabId];
  if (!t) return;
  const h = hosts.hosts.find((x) => x.id === t.hostId);
  if (h) sessions.openInActivePane(h);
  closeCtx();
}
function requestCloseTab(tabId: string) {
  closeCtx();
  const t = sessions.tabs[tabId];
  if (t && (t.status === "open" || t.status === "connecting")) {
    pendingCloseTabId.value = tabId;
  } else {
    sessions.closeTab(tabId);
  }
}

function confirmClose() {
  if (pendingCloseTabId.value) {
    sessions.closeTab(pendingCloseTabId.value);
    pendingCloseTabId.value = null;
  }
}

onMounted(() => window.addEventListener("click", closeCtx));
onUnmounted(() => window.removeEventListener("click", closeCtx));
</script>

<template>
  <div class="tabbar">
    <div class="tabs">
      <div
        v-for="t in tabs"
        :key="t.id"
        class="tab"
        :class="{ active: activeTab?.id === t.id }"
        @click="sessions.setActiveTab(pane.id, t.id)"
        @contextmenu="openCtx($event, t.id)"
      >
        <span class="tab-dot" :class="dotClass(t.status)" />
        <span class="tab-label">{{ t.hostName }}</span>
        <button class="tab-close icon-btn" @click.stop="requestCloseTab(t.id)">
          <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round">
            <path d="M6 6l12 12M18 6L6 18" />
          </svg>
        </button>
      </div>
    </div>
    <div class="actions">
      <button
        v-if="activeTab"
        class="icon-btn"
        :class="{ on: activeTab.showMonitor }"
        title="切换 VPS 监控"
        @click="sessions.toggleMonitor(activeTab.id)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 12h4l3 8 4-16 3 8h4" />
        </svg>
      </button>
      <button
        v-if="activeTab"
        class="icon-btn"
        :class="{ on: activeTab.showSftp }"
        title="切换 SFTP 文件浏览器"
        @click="sessions.toggleSftp(activeTab.id)"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
        </svg>
      </button>
      <button class="icon-btn" title="向右分屏" @click="sessions.splitRight()">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <rect x="3" y="4" width="18" height="16" rx="1.5" />
          <line x1="12" y1="4" x2="12" y2="20" />
        </svg>
      </button>
      <button
        v-if="sessions.panes.length > 1"
        class="icon-btn"
        title="关闭分屏"
        @click="emit('close-pane')"
      >
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M6 6l12 12M18 6L6 18" />
        </svg>
      </button>
    </div>

    <ul v-if="ctx" class="ctxmenu" :style="{ left: ctx.x + 'px', top: ctx.y + 'px' }" @click.stop>
      <li @click="reconnect(ctx.tabId)">重新连接</li>
      <li @click="duplicate(ctx.tabId)">克隆此会话</li>
      <li class="sep" />
      <li class="danger" @click="requestCloseTab(ctx.tabId)">关闭标签</li>
    </ul>

    <ConfirmDialog
      v-if="pendingCloseTabId"
      title="关闭会话？"
      message="此 SSH 会话仍在连接中，关闭后会断开连接。"
      confirmLabel="关闭"
      :danger="true"
      @confirm="confirmClose"
      @cancel="pendingCloseTabId = null"
    />
  </div>
</template>

<style scoped>
.tabbar {
  display: flex;
  align-items: center;
  height: 36px;
  background: var(--bg-elev);
  border-bottom: 1px solid var(--border);
  padding: 0 6px;
  flex-shrink: 0;
  position: relative;
}
.tabs {
  flex: 1;
  display: flex;
  gap: 2px;
  overflow-x: auto;
  scrollbar-width: none;
}
.tabs::-webkit-scrollbar { display: none; }
.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px 6px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  max-width: 200px;
  position: relative;
  color: var(--fg-muted);
  font-size: 12px;
}
.tab:hover { background: var(--bg-hover); }
.tab.active {
  background: var(--bg);
  color: var(--fg);
  box-shadow: inset 0 -2px 0 var(--accent);
}
.tab-label {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tab-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}
.tab-dot.ok { background: var(--success); }
.tab-dot.pending {
  background: var(--warning);
  animation: pulse 1.2s ease-in-out infinite;
}
.tab-dot.fail { background: var(--danger); }
.tab-dot.idle { background: var(--fg-subtle); }
.tab-close {
  width: 18px;
  height: 18px;
  opacity: 0;
}
.tab:hover .tab-close,
.tab.active .tab-close { opacity: 0.8; }
.actions {
  display: flex;
  gap: 2px;
  padding-left: 6px;
  border-left: 1px solid var(--border);
  margin-left: 6px;
}
.icon-btn.on {
  background: var(--bg-active);
  color: var(--accent);
}
@keyframes pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 1; }
}

.ctxmenu {
  position: fixed;
  z-index: 200;
  background: var(--bg-elev-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  padding: 4px;
  min-width: 160px;
  list-style: none;
  margin: 0;
}
.ctxmenu li {
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12.5px;
}
.ctxmenu li:hover { background: var(--bg-hover); }
.ctxmenu li.sep {
  margin: 4px 0;
  padding: 0;
  border-top: 1px solid var(--border);
  pointer-events: none;
}
.ctxmenu li.danger { color: var(--danger); }
.ctxmenu li.danger:hover { background: color-mix(in oklab, var(--danger) 12%, transparent); }
</style>
