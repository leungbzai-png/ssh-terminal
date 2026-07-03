<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, computed } from "vue";
import { useSettings } from "./stores/settings";
import { useHosts } from "./stores/hosts";
import { useSessions } from "./stores/sessions";
import { useTheme } from "./composables/useTheme";
import Sidebar from "./components/Sidebar.vue";
import PaneView from "./components/PaneView.vue";
import HostDialog from "./components/HostDialog.vue";
import SettingsDialog from "./components/SettingsDialog.vue";
import HostKeyDialog from "./components/HostKeyDialog.vue";
import CloseConfirmDialog from "./components/CloseConfirmDialog.vue";
import KeysDialog from "./components/KeysDialog.vue";
import QuickConnectDialog from "./components/QuickConnectDialog.vue";
import ImportConfigDialog from "./components/ImportConfigDialog.vue";
import ImportHostsDialog from "./components/ImportHostsDialog.vue";
import ShortcutHelpDialog from "./components/ShortcutHelpDialog.vue";
import type { HostRecord, QuickConnectParams, HostsImportPreview, HostsImportResult } from "./wails.d";

const settings = useSettings();
const hostsStore = useHosts();
const sessions = useSessions();
const theme = useTheme();

const editingHost = ref<HostRecord | null>(null);
const showSettings = ref(false);
const showKeys = ref(false);
const showQuick = ref(false);
const showImport = ref(false);
const showShortcuts = ref(false);
const hostsImportPreview = ref<HostsImportPreview | null>(null);

// Close confirmation
const closeConfirmCount = ref<number | null>(null);
const onConfirmClose = (count: number) => {
  closeConfirmCount.value = count;
};

// File drop: route to active tab's SFTP.
const dropOverlay = ref(false);
const dropProgress = ref<{ pct: number; current: string } | null>(null);
const onFileDrop = async (payload: { paths: string[] }) => {
  dropOverlay.value = false;
  const paths = payload?.paths || [];
  if (paths.length === 0) return;
  const tabId = sessions.activePane?.activeTabId;
  if (!tabId) {
    alert("请先打开一个 SSH 会话再拖入文件");
    return;
  }
  const tab = sessions.tabs[tabId];
  if (!tab || tab.status !== "open") {
    alert("当前标签未连接");
    return;
  }
  // Ensure SFTP panel is open so user can see / change target dir.
  if (!tab.showSftp) sessions.toggleSftp(tabId);

  let remoteDir = "";
  try {
    remoteDir = sessions.sftpCwd[tabId] || (await window.go.main.App.SftpCwd(tabId));
    sessions.sftpCwd[tabId] = remoteDir;
  } catch (e: any) {
    alert("无法获取远程目录: " + (e?.message || e));
    return;
  }
  if (!confirm(`上传 ${paths.length} 项到 ${remoteDir} ？`)) return;

  dropProgress.value = { pct: 0, current: "准备…" };
  const progressEvt = `sftp:progress:${tabId}`;
  const doneEvt = `sftp:done:${tabId}`;
  const onProg = (p: { transferred: number; total: number; current: string }) => {
    const pct = p.total > 0 ? Math.floor((p.transferred / p.total) * 100) : 0;
    dropProgress.value = { pct, current: p.current };
  };
  const onDone = (r: { ok: boolean; err: string }) => {
    window.runtime.EventsOff(progressEvt);
    window.runtime.EventsOff(doneEvt);
    if (r.ok) {
      dropProgress.value = { pct: 100, current: "完成" };
      setTimeout(() => (dropProgress.value = null), 800);
      // Bump refresh tick so SFTP panel reloads.
      sessions.bumpSftpRefresh(tabId);
    } else {
      dropProgress.value = null;
      alert("上传失败: " + r.err);
    }
  };
  window.runtime.EventsOn(progressEvt, onProg);
  window.runtime.EventsOn(doneEvt, onDone);

  try {
    await window.go.main.App.SftpUploadPaths(tabId, paths, remoteDir);
  } catch (e: any) {
    dropProgress.value = null;
    alert("上传出错: " + (e?.message || e));
  }
};

// Restore last session's saved-host tabs as idle (not auto-connected). Hosts
// that no longer exist are skipped silently — no crash.
async function restoreTabs() {
  try {
    const refs = (await window.go.main.App.GetOpenTabs()) || [];
    let skipped = 0;
    for (const r of refs) {
      const host = hostsStore.hosts.find((h) => h.id === r.hostId);
      if (host) sessions.openSavedTabIdle(host);
      else skipped++;
    }
    if (skipped > 0) console.info(`Skipped ${skipped} restored tab(s): host removed`);
  } catch (e) {
    console.info("Tab restore skipped:", e);
  }
}

function onGlobalKey(e: KeyboardEvent) {
  if (e.key === "F1") {
    e.preventDefault();
    showShortcuts.value = !showShortcuts.value;
  }
}

function dragEnter(e: DragEvent) {
  if (e.dataTransfer?.types?.includes("Files")) dropOverlay.value = true;
}
function dragLeave(e: DragEvent) {
  // Only hide when leaving window entirely.
  if ((e as any).target === document.body || (e.relatedTarget == null)) {
    dropOverlay.value = false;
  }
}

onMounted(async () => {
  await settings.load();
  theme.setMode(settings.settings.theme);
  await hostsStore.refresh();
  await restoreTabs();
  window.runtime.EventsOn("app:confirmClose", onConfirmClose);
  window.runtime.EventsOn("app:filedrop", onFileDrop);
  window.addEventListener("dragenter", dragEnter);
  window.addEventListener("dragleave", dragLeave);
  window.addEventListener("dragover", (e) => e.preventDefault());
  window.addEventListener("keydown", onGlobalKey);
});
onBeforeUnmount(() => {
  window.runtime.EventsOff("app:confirmClose");
  window.runtime.EventsOff("app:filedrop");
  window.removeEventListener("dragenter", dragEnter);
  window.removeEventListener("dragleave", dragLeave);
  window.removeEventListener("keydown", onGlobalKey);
});

watch(
  () => settings.settings.theme,
  (m) => theme.setMode(m)
);

function newHost() {
  editingHost.value = {
    id: "",
    name: "",
    address: "",
    port: 22,
    user: "",
    authType: "password",
  };
}
function editHost(h: HostRecord) {
  editingHost.value = { ...h };
}
async function saveHost(h: HostRecord) {
  try {
    await hostsStore.upsert(h);
    editingHost.value = null;
  } catch (e: any) {
    // Backend rejected the host (e.g. invalid Advanced SSH config). Keep the
    // dialog open so the user can fix it.
    window.alert("保存失败：" + (e?.message || e));
  }
}
function openHost(h: HostRecord) {
  sessions.openInActivePane(h);
}
async function quickConnect(params: QuickConnectParams, remember: boolean) {
  showQuick.value = false;
  if (remember) {
    // Save the host through the normal encrypted path, then open it as a
    // regular saved host (credentials read back from encrypted storage).
    const saved = await window.go.main.App.UpsertHost({
      id: "",
      name: params.address,
      address: params.address,
      port: params.port || 22,
      user: params.user,
      authType: params.authType,
      password: params.password,
      keyPath: params.keyPath,
      passphrase: params.passphrase,
    });
    await hostsStore.refresh();
    sessions.openInActivePane(saved);
  } else {
    // Ephemeral: credentials stay in memory only, never persisted.
    sessions.openQuickInActivePane(params);
  }
}
async function onImported(count: number) {
  showImport.value = false;
  await hostsStore.refresh();
  if (count > 0) {
    // Non-blocking confirmation; kept simple.
    console.info(`Imported ${count} host(s) from ssh config`);
  }
}
async function exportHosts() {
  try {
    const path = await window.go.main.App.ExportHosts();
    if (path) alert(`已导出主机到:\n${path}\n（不含密码/口令/私钥）`);
  } catch (e: any) {
    alert("导出失败: " + (e?.message || e));
  }
}
async function openImportHosts() {
  try {
    const preview = await window.go.main.App.PreviewHostsImport();
    if (!preview || !preview.path) return; // cancelled
    if (!preview.hosts || preview.hosts.length === 0) {
      alert("文件中没有可导入的主机。");
      return;
    }
    hostsImportPreview.value = preview;
  } catch (e: any) {
    alert("无法读取导入文件: " + (e?.message || e));
  }
}
async function onHostsImported(res: HostsImportResult) {
  hostsImportPreview.value = null;
  await hostsStore.refresh();
  console.info(
    `Imported ${res.imported}, overwrote ${res.overwritten}, skipped ${res.skipped}`
  );
}
function confirmCloseProceed() {
  closeConfirmCount.value = null;
  window.go.main.App.ConfirmQuit();
}

const hasActiveTab = computed(() => !!sessions.activePane?.activeTabId);
// Drag-upload target feedback.
const dropReady = computed(() => {
  const id = sessions.activePane?.activeTabId;
  return !!id && sessions.tabs[id]?.status === "open";
});
const dropTargetDir = computed(() => {
  const id = sessions.activePane?.activeTabId;
  return (id && sessions.sftpCwd[id]) || "会话工作目录";
});
</script>

<template>
  <div class="shell">
    <Sidebar
      :hosts="hostsStore.hosts"
      @new="newHost"
      @quick="showQuick = true"
      @import="showImport = true"
      @export-hosts="exportHosts"
      @import-hosts="openImportHosts"
      @edit="editHost"
      @open="openHost"
      @delete="(id) => hostsStore.remove(id)"
      @settings="showSettings = true"
      @keys="showKeys = true"
      @help="showShortcuts = true"
    />
    <main class="workspace">
      <div class="panes" :data-pane-count="sessions.panes.length">
        <PaneView
          v-for="p in sessions.panes"
          :key="p.id"
          :pane="p"
          @activate="sessions.activePaneId = p.id"
          @close-pane="sessions.closePane(p.id)"
        />
      </div>
    </main>

    <HostDialog
      v-if="editingHost"
      :host="editingHost"
      @save="saveHost"
      @cancel="editingHost = null"
    />

    <SettingsDialog v-if="showSettings" @close="showSettings = false" />
    <ShortcutHelpDialog v-if="showShortcuts" @close="showShortcuts = false" />
    <KeysDialog v-if="showKeys" @close="showKeys = false" />
    <QuickConnectDialog
      v-if="showQuick"
      @connect="quickConnect"
      @cancel="showQuick = false"
    />
    <ImportConfigDialog
      v-if="showImport"
      @imported="onImported"
      @close="showImport = false"
    />
    <ImportHostsDialog
      v-if="hostsImportPreview"
      :preview="hostsImportPreview"
      @imported="onHostsImported"
      @close="hostsImportPreview = null"
    />
    <HostKeyDialog />

    <CloseConfirmDialog
      v-if="closeConfirmCount !== null"
      :active-count="closeConfirmCount"
      @cancel="closeConfirmCount = null"
      @confirm="confirmCloseProceed"
    />

    <!-- Drop overlay -->
    <Transition name="fade">
      <div v-if="dropOverlay" class="drop-overlay" :class="{ reject: !dropReady }">
        <div class="drop-card">
          <svg width="42" height="42" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <path v-if="dropReady" d="M12 19V5M5 12l7-7 7 7" />
            <path v-else d="M12 9v4M12 17h.01M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z" />
          </svg>
          <template v-if="dropReady">
            <div class="drop-title">释放以上传</div>
            <div class="drop-sub">目标：<span class="mono">{{ dropTargetDir }}</span></div>
          </template>
          <template v-else>
            <div class="drop-title">无法上传</div>
            <div class="drop-sub">请先打开并连接一个 SSH 会话</div>
          </template>
        </div>
      </div>
    </Transition>

    <!-- Upload progress toast -->
    <Transition name="toast">
      <div v-if="dropProgress" class="toast">
        <div class="toast-bar"><div class="toast-fill" :style="{ width: dropProgress.pct + '%' }" /></div>
        <div class="toast-text">
          上传中 {{ dropProgress.pct }}% — <span class="mono">{{ dropProgress.current }}</span>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  height: 100%;
  width: 100%;
  overflow: hidden;
}
.workspace {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.panes {
  flex: 1;
  display: grid;
  gap: 1px;
  background: var(--border);
  min-height: 0;
}
.panes[data-pane-count="1"] {
  grid-template-columns: 1fr;
}
.panes[data-pane-count="2"] {
  grid-template-columns: 1fr 1fr;
}
.panes[data-pane-count="3"] {
  grid-template-columns: 1fr 1fr 1fr;
}
.panes[data-pane-count="4"] {
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
}

.drop-overlay {
  position: fixed;
  inset: 0;
  z-index: 200;
  background: color-mix(in oklab, var(--accent) 14%, rgba(0, 0, 0, 0.5));
  backdrop-filter: blur(6px);
  display: grid;
  place-items: center;
  pointer-events: none;
}
.drop-card {
  text-align: center;
  padding: 28px 36px;
  background: var(--bg-elev);
  border: 2px dashed var(--accent);
  border-radius: 10px;
  color: var(--fg);
  box-shadow: var(--shadow-lg);
}
.drop-card svg {
  color: var(--accent);
  margin-bottom: 8px;
}
.drop-overlay.reject .drop-card {
  border-color: var(--danger);
}
.drop-overlay.reject .drop-card svg {
  color: var(--danger);
}
.drop-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
}
.drop-sub {
  font-size: 12px;
  color: var(--fg-muted);
}

.toast {
  position: fixed;
  bottom: 16px;
  right: 16px;
  z-index: 150;
  min-width: 280px;
  max-width: 420px;
  padding: 10px 12px;
  background: var(--bg-elev-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
}
.toast-bar {
  height: 4px;
  background: var(--bg-active);
  border-radius: 2px;
  overflow: hidden;
  margin-bottom: 6px;
}
.toast-fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.2s ease;
}
.toast-text {
  font-size: 12px;
  color: var(--fg-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mono {
  font-family: var(--mono, monospace);
  color: var(--fg);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
