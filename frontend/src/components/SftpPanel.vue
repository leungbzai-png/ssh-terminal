<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useSessions } from "../stores/sessions";
import type { FileEntry } from "../wails.d";

const props = defineProps<{ tabId: string; paneId: string }>();
const sessions = useSessions();

const cwd = ref<string>("");
const entries = ref<FileEntry[]>([]);
const loading = ref(false);
const error = ref<string>("");

// Context menu state
const ctx = ref<{ x: number; y: number; entry: FileEntry | null } | null>(null);

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
    loading.value = true;
    await window.go.main.App.SftpUploadPaths(props.tabId, files, cwd.value);
    await load(cwd.value);
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    loading.value = false;
  }
}

async function download(e: FileEntry) {
  if (e.isDir) return;
  const local = await window.go.main.App.PickSaveLocation(e.name);
  if (!local) return;
  try {
    await window.go.main.App.SftpDownload(props.tabId, e.path, local);
  } catch (err: any) {
    error.value = String(err?.message || err);
  }
}

async function remove(e: FileEntry) {
  if (!confirm(`删除 ${e.name}？`)) return;
  try {
    await window.go.main.App.SftpDelete(props.tabId, e.path);
    await load(cwd.value);
  } catch (err: any) {
    error.value = String(err?.message || err);
  }
}

async function mkdir() {
  const name = prompt("新文件夹名");
  if (!name) return;
  const p = (cwd.value.endsWith("/") ? cwd.value : cwd.value + "/") + name;
  try {
    await window.go.main.App.SftpMkdir(props.tabId, p);
    await load(cwd.value);
  } catch (err: any) {
    error.value = String(err?.message || err);
  }
}

async function rename(e: FileEntry) {
  const newName = prompt("重命名为：", e.name);
  if (!newName || newName === e.name) return;
  const dir = e.path.slice(0, e.path.lastIndexOf("/")) || "/";
  const newPath = (dir.endsWith("/") ? dir : dir + "/") + newName;
  try {
    await window.go.main.App.SftpRename(props.tabId, e.path, newPath);
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
  // Send `cd '<path>'\n` to the terminal session.
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

onMounted(() => {
  load("");
  window.addEventListener("click", closeCtx);
  window.addEventListener("contextmenu", onCtxOutside);
});
onBeforeUnmount(() => {
  window.removeEventListener("click", closeCtx);
  window.removeEventListener("contextmenu", onCtxOutside);
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
      <button class="icon-btn" title="新建文件夹" @click="mkdir">
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
      <span class="hint">双击进入 · 右键菜单 · 可拖拽文件/文件夹到窗口上传</span>
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
        <li @click="rename(ctx.entry!); closeCtx()">重命名</li>
        <li class="danger" @click="remove(ctx.entry!); closeCtx()">删除</li>
      </template>
      <template v-else>
        <li @click="uploadFiles(); closeCtx()">上传文件…</li>
        <li @click="mkdir(); closeCtx()">新建文件夹</li>
        <li @click="load(cwd); closeCtx()">刷新</li>
      </template>
    </ul>
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
