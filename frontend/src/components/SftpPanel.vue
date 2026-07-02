<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useSessions } from "../stores/sessions";
import type { FileEntry } from "../wails.d";
import ConfirmDialog from "./ConfirmDialog.vue";
import InputDialog from "./InputDialog.vue";

const props = defineProps<{ tabId: string; paneId: string }>();
const sessions = useSessions();

const cwd = ref<string>("");
const entries = ref<FileEntry[]>([]);
const loading = ref(false);
const error = ref<string>("");

// Transfer progress (upload/download) via the dedicated sftp:xfer:* events.
const xfer = ref<{ dir: string; pct: number; current: string } | null>(null);

// Context menu state
const ctx = ref<{ x: number; y: number; entry: FileEntry | null } | null>(null);

// Dialog state — separate refs to avoid TypeScript narrowing issues in templates
const pendingDeleteEntry = ref<FileEntry | null>(null);
const showMkdirDialog = ref(false);
const pendingRenameEntry = ref<FileEntry | null>(null);

async function load(dir: string) {
  loading.value = true;
  error.value = "";
  try {
    if (!dir) {
      dir = await window.go.main.App.SftpCwd(props.tabId);
    }
    cwd.value = dir;
    sessions.setSftpCwd(props.tabId, dir);
    const list = await window.go.main.App.SftpList(props.tabId, dir);
    list.sort((a, b) => {
      if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
      return a.name.localeCompare(b.name);
    });
    entries.value = list;
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    loading.value = false;
  }
}

function up() {
  if (!cwd.value || cwd.value === "/") return;
  const i = cwd.value.lastIndexOf("/");
  load(i <= 0 ? "/" : cwd.value.slice(0, i));
}

function enter(e: FileEntry) {
  if (e.isDir) load(e.path);
}

function encodeUtf8B64(s: string) {
  const bytes = new TextEncoder().encode(s);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin);
}

async function uploadFiles() {
  try {
    const files: string[] = await window.go.main.App.PickFilesToUpload();
    if (!files || files.length === 0) return;
    xfer.value = { dir: "upload", pct: 0, current: "准备…" };
    await window.go.main.App.SftpUploadTracked(props.tabId, files, cwd.value);
  } catch (e: any) {
    xfer.value = null;
    error.value = String(e?.message || e);
  }
}

async function download(e: FileEntry) {
  if (e.isDir) return;
  const local = await window.go.main.App.PickSaveLocation(e.name);
  if (!local) return;
  try {
    xfer.value = { dir: "download", pct: 0, current: e.name };
    await window.go.main.App.SftpDownloadTracked(props.tabId, e.path, local);
  } catch (err: any) {
    xfer.value = null;
    error.value = String(err?.message || err);
  }
}

function requestRemove(e: FileEntry) {
  closeCtx();
  pendingDeleteEntry.value = e;
}

async function confirmRemove() {
  const entry = pendingDeleteEntry.value;
  pendingDeleteEntry.value = null;
  if (!entry) return;
  try {
    if (entry.isDir) {
      await window.go.main.App.SftpDeleteRecursive(props.tabId, entry.path);
    } else {
      await window.go.main.App.SftpDelete(props.tabId, entry.path);
    }
    await load(cwd.value);
  } catch (err: any) {
    error.value = String(err?.message || err);
  }
}

function requestMkdir() {
  closeCtx();
  showMkdirDialog.value = true;
}

async function confirmMkdir(name: string) {
  showMkdirDialog.value = false;
  const p = (cwd.value.endsWith("/") ? cwd.value : cwd.value + "/") + name;
  try {
    await window.go.main.App.SftpMkdir(props.tabId, p);
    await load(cwd.value);
  } catch (err: any) {
    error.value = String(err?.message || err);
  }
}

function requestRename(e: FileEntry) {
  closeCtx();
  pendingRenameEntry.value = e;
}

async function confirmRename(newName: string) {
  const entry = pendingRenameEntry.value;
  pendingRenameEntry.value = null;
  if (!entry || newName === entry.name) return;
  const dir = entry.path.slice(0, entry.path.lastIndexOf("/")) || "/";
  const newPath = (dir.endsWith("/") ? dir : dir + "/") + newName;
  try {
    await window.go.main.App.SftpRename(props.tabId, entry.path, newPath);
    await load(cwd.value);
  } catch (err: any) {
    error.value = String(err?.message || err);
  }
}

function copyPath(e: FileEntry) {
  navigator.clipboard.writeText(e.path).catch(() => {});
}

async function cdInTerminal(e: FileEntry) {
  const target = e.isDir ? e.path : e.path.slice(0, e.path.lastIndexOf("/")) || "/";
  const safe = target.replace(/'/g, `'\\''`);
  await window.go.main.App.WriteSession(props.tabId, encodeUtf8B64(`cd '${safe}'\n`));
}

function fmtSize(n: number) {
  if (n < 1024) return n + " B";
  const u = ["KB", "MB", "GB", "TB"];
  let v = n / 1024,
    i = 0;
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024;
    i++;
  }
  return v.toFixed(v < 10 ? 1 : 0) + " " + u[i];
}

function deleteMessage(entry: FileEntry): string {
  if (entry.isDir) {
    return "将递归删除 " + entry.name + " 及其全部内容，此操作不可撤销。";
  }
  return "确认删除文件 " + entry.name + "？";
}

function openCtx(e: MouseEvent, entry: FileEntry | null) {
  e.preventDefault();
  ctx.value = { x: e.clientX, y: e.clientY, entry };
}
function closeCtx() {
  ctx.value = null;
}

function onCtxOutside(e: MouseEvent) {
  if (ctx.value && !(e.target as HTMLElement).closest(".sftp")) closeCtx();
}

const progressEvt = `sftp:xfer:progress:${props.tabId}`;
const doneEvt = `sftp:xfer:done:${props.tabId}`;
function onXferProgress(p: { transferred: number; total: number; current: string; direction: string }) {
  const pct = p.total > 0 ? Math.floor((p.transferred / p.total) * 100) : 0;
  xfer.value = { dir: p.direction, pct, current: p.current || "" };
}
function onXferDone(r: { ok: boolean; err: string; direction: string }) {
  if (r.ok) {
    xfer.value = { dir: r.direction, pct: 100, current: "完成" };
    setTimeout(() => (xfer.value = null), 700);
    if (r.direction === "upload") load(cwd.value);
  } else {
    xfer.value = null;
    error.value = "传输失败: " + r.err;
  }
}

onMounted(() => {
  load("");
  window.addEventListener("click", closeCtx);
  window.addEventListener("contextmenu", onCtxOutside);
  window.runtime.EventsOn(progressEvt, onXferProgress);
  window.runtime.EventsOn(doneEvt, onXferDone);
});
onBeforeUnmount(() => {
  window.removeEventListener("click", closeCtx);
  window.removeEventListener("contextmenu", onCtxOutside);
  window.runtime.EventsOff(progressEvt);
  window.runtime.EventsOff(doneEvt);
});
watch(() => props.tabId, () => load(""));
watch(
  () => sessions.sftpRefreshTick[props.tabId],
  () => load(cwd.value)
);
</script>

<template>
  <aside class="sftp" @contextmenu="openCtx($event, null)">
    <header>
      <button class="icon-btn" title="上一级" @click="up">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M5 12l7-7 7 7M12 5v14"/></svg>
      </button>
      <input class="path" :value="cwd" @change="(ev) => load((ev.target as HTMLInputElement).value)" />
      <button class="icon-btn" title="刷新" @click="load(cwd)">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M21 12a9 9 0 1 1-3-6.7L21 8"/><path d="M21 3v5h-5"/></svg>
      </button>
      <button class="icon-btn" title="上传文件 (也可直接拖入窗口)" @click="uploadFiles">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M12 19V5M5 12l7-7 7 7"/></svg>
      </button>
      <button class="icon-btn" title="新建文件夹" @click="requestMkdir">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><path d="M12 11v6M9 14h6"/></svg>
      </button>
    </header>
    <div class="body">
      <div v-if="error" class="error">{{ error }}</div>
      <div v-if="loading" class="loading">加载中…</div>
      <ul v-else class="entries">
        <li
          v-for="e in entries"
          :key="e.path"
          class="entry"
          :class="{ dir: e.isDir }"
          @dblclick="enter(e)"
          @contextmenu.stop="openCtx($event, e)"
        >
          <span class="ic">
            <svg v-if="e.isDir" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6"/></svg>
          </span>
          <span class="name">{{ e.name }}</span>
          <span class="size">{{ e.isDir ? "" : fmtSize(e.size) }}</span>
          <button v-if="!e.isDir" class="icon-btn dl" title="下载" @click.stop="download(e)">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M12 5v14M5 12l7 7 7-7"/></svg>
          </button>
        </li>
      </ul>
    </div>
    <footer>
      <div v-if="xfer" class="xfer">
        <div class="xfer-bar"><div class="xfer-fill" :style="{ width: xfer.pct + '%' }" /></div>
        <div class="xfer-text">
          {{ xfer.dir === "download" ? "下载" : "上传" }} {{ xfer.pct }}% —
          <span class="mono">{{ xfer.current }}</span>
        </div>
      </div>
      <span v-else class="hint">双击进入 · 右键菜单 · 可拖拽文件/文件夹到窗口上传</span>
    </footer>

    <!-- Context menu -->
    <ul
      v-if="ctx"
      class="ctxmenu"
      :style="{ left: ctx.x + 'px', top: ctx.y + 'px' }"
      @click.stop
    >
      <template v-if="ctx.entry">
        <li v-if="ctx.entry.isDir" @click="cdInTerminal(ctx.entry!); closeCtx()">
          在终端打开 (cd)
        </li>
        <li v-else @click="cdInTerminal(ctx.entry!); closeCtx()">
          在终端打开所在目录
        </li>
        <li @click="copyPath(ctx.entry!); closeCtx()">复制远程路径</li>
        <li v-if="!ctx.entry.isDir" @click="download(ctx.entry!); closeCtx()">下载…</li>
        <li class="sep" />
        <li @click="requestRename(ctx.entry!)">重命名</li>
        <li class="danger" @click="requestRemove(ctx.entry!)">删除</li>
      </template>
      <template v-else>
        <li @click="uploadFiles(); closeCtx()">上传文件…</li>
        <li @click="requestMkdir()">新建文件夹</li>
        <li @click="load(cwd); closeCtx()">刷新</li>
      </template>
    </ul>

    <!-- Delete confirm dialog -->
    <ConfirmDialog
      v-if="pendingDeleteEntry"
      :title="pendingDeleteEntry.isDir ? '删除文件夹？' : '删除文件？'"
      :message="deleteMessage(pendingDeleteEntry)"
      confirmLabel="删除"
      :danger="true"
      @confirm="confirmRemove"
      @cancel="pendingDeleteEntry = null"
    />

    <!-- New folder dialog -->
    <InputDialog
      v-if="showMkdirDialog"
      title="新建文件夹"
      placeholder="文件夹名"
      confirmLabel="创建"
      @confirm="confirmMkdir"
      @cancel="showMkdirDialog = false"
    />

    <!-- Rename dialog -->
    <InputDialog
      v-if="pendingRenameEntry"
      title="重命名"
      placeholder="新名称"
      :defaultValue="pendingRenameEntry.name"
      confirmLabel="重命名"
      @confirm="confirmRename"
      @cancel="pendingRenameEntry = null"
    />
  </aside>
</template>

<style scoped>
.sftp {
  position: relative;
  display: flex;
  flex-direction: column;
  border-left: 1px solid var(--border);
  background: var(--bg-elev);
  min-width: 0;
  min-height: 0;
}
header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.path {
  flex: 1;
  font-family: var(--mono, monospace);
  font-size: 11.5px;
  padding: 4px 7px;
}
.body {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}
.error {
  padding: 12px;
  color: var(--danger);
  font-size: 12px;
}
.loading {
  padding: 16px;
  color: var(--fg-muted);
  font-size: 12px;
}
.entries {
  list-style: none;
  margin: 0;
  padding: 4px;
}
.entry {
  display: grid;
  grid-template-columns: 18px 1fr auto 22px;
  align-items: center;
  gap: 8px;
  padding: 5px 7px;
  border-radius: var(--radius-sm);
  font-size: 12.5px;
  cursor: default;
}
.entry:hover {
  background: var(--bg-hover);
}
.entry.dir {
  color: var(--accent);
}
.entry .name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.entry .size {
  font-size: 11px;
  color: var(--fg-muted);
  font-variant-numeric: tabular-nums;
}
.entry .dl {
  opacity: 0;
}
.entry:hover .dl {
  opacity: 1;
}
.ic {
  display: grid;
  place-items: center;
  color: var(--fg-muted);
}
.entry.dir .ic {
  color: var(--accent);
}
footer {
  padding: 6px 10px;
  border-top: 1px solid var(--border);
  font-size: 10.5px;
  color: var(--fg-subtle);
}
.xfer-bar {
  height: 4px;
  background: var(--bg-active);
  border-radius: 2px;
  overflow: hidden;
  margin-bottom: 4px;
}
.xfer-fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.2s ease;
}
.xfer-text {
  font-size: 10.5px;
  color: var(--fg-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.xfer-text .mono {
  font-family: var(--mono, monospace);
  color: var(--fg);
}

/* context menu */
.ctxmenu {
  position: fixed;
  z-index: 200;
  background: var(--bg-elev-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  padding: 4px;
  min-width: 180px;
  list-style: none;
  margin: 0;
}
.ctxmenu li {
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12.5px;
}
.ctxmenu li:hover {
  background: var(--bg-hover);
}
.ctxmenu li.sep {
  margin: 4px 0;
  padding: 0;
  border-top: 1px solid var(--border);
  pointer-events: none;
}
.ctxmenu li.danger {
  color: var(--danger);
}
.ctxmenu li.danger:hover {
  background: color-mix(in oklab, var(--danger) 12%, transparent);
}
</style>
