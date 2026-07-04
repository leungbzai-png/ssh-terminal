<script setup lang="ts">
// SftpPane — the LOCAL filesystem browse pane of the two-pane SFTP UI (v1.1.0).
// Read-only in this commit: browse, navigate, refresh. No upload/download,
// mkdir, rename, or delete here — those arrive with the transfer wiring (commit
// 4). Local cwd is component-local and is never persisted to disk.
//
// The `side` prop is present for future symmetry; commit 3 only implements the
// local pane. The remote pane's rich logic (bookmarks/preview/context menu)
// stays inline in SftpPanel.vue.
import { onMounted, ref } from "vue";
import type { FileEntry } from "../wails.d";

const props = withDefaults(defineProps<{ side?: "local" | "remote" }>(), {
  side: "local",
});

// Notify the parent (SftpPanel) of the current cwd and the single selection so
// it can enable/disable the transfer buttons and compute destinations.
const emit = defineEmits<{
  cwd: [{ path: string; atRoots: boolean }];
  selection: [FileEntry | null];
}>();

const cwd = ref<string>(""); // "" while showing the roots list
const atRoots = ref(false);
const entries = ref<FileEntry[]>([]);
const loading = ref(false);
const error = ref<string>("");
const selected = ref<FileEntry | null>(null);

function select(e: FileEntry) {
  selected.value = e;
  emit("selection", e);
}

// clearSelection + publish the new cwd whenever the listing changes.
function published() {
  selected.value = null;
  emit("selection", null);
  emit("cwd", { path: cwd.value, atRoots: atRoots.value });
}

function sortEntries(list: FileEntry[]): FileEntry[] {
  return [...list].sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    return a.name.localeCompare(b.name);
  });
}

async function loadDir(dir: string) {
  loading.value = true;
  error.value = "";
  try {
    const list = (await window.go.main.App.LocalList(dir)) || [];
    entries.value = sortEntries(list);
    cwd.value = dir;
    atRoots.value = false;
    published();
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    loading.value = false;
  }
}

async function loadRoots() {
  loading.value = true;
  error.value = "";
  try {
    const roots = (await window.go.main.App.LocalRoots()) || [];
    entries.value = roots.map((r) => ({
      name: r,
      path: r,
      size: 0,
      mode: "",
      modTime: 0,
      isDir: true,
      isLink: false,
    }));
    cwd.value = "";
    atRoots.value = true;
    published();
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    loading.value = false;
  }
}

async function loadHome() {
  try {
    const home = await window.go.main.App.LocalHome();
    if (home) {
      await loadDir(home);
      return;
    }
  } catch {
    /* fall through to roots */
  }
  await loadRoots();
}

// parseParent tolerates however Wails represents the Go (string, bool) return —
// an array [parent, isRoot] or an object — so navigation is robust regardless of
// the exact multi-return marshalling.
function parseParent(res: any): { parent: string; isRoot: boolean } {
  if (Array.isArray(res)) return { parent: String(res[0] ?? ""), isRoot: !!res[1] };
  if (res && typeof res === "object") {
    return {
      parent: String(res.parent ?? res.Parent ?? ""),
      isRoot: !!(res.isRoot ?? res.IsRoot),
    };
  }
  return { parent: String(res ?? ""), isRoot: false };
}

async function up() {
  if (atRoots.value) return; // already at the top
  try {
    const { parent, isRoot } = parseParent(await window.go.main.App.LocalParent(cwd.value));
    if (isRoot || !parent) {
      await loadRoots();
    } else {
      await loadDir(parent);
    }
  } catch (e: any) {
    error.value = String(e?.message || e);
  }
}

function enter(e: FileEntry) {
  if (e.isDir) loadDir(e.path);
}

function refresh() {
  if (atRoots.value) loadRoots();
  else loadDir(cwd.value);
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

onMounted(loadHome);

// Parent (SftpPanel) calls refresh() after a successful download into this cwd.
defineExpose({ refresh });
</script>

<template>
  <section class="pane" :data-side="props.side">
    <header>
      <span class="pane-tag">本地</span>
      <button class="icon-btn" title="上一级" :disabled="atRoots" @click="up">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M5 12l7-7 7 7M12 5v14"/></svg>
      </button>
      <span class="path" :title="atRoots ? '此电脑' : cwd">{{ atRoots ? "此电脑" : cwd }}</span>
      <button class="icon-btn" title="主目录" @click="loadHome">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 11l9-8 9 8"/><path d="M5 10v10h14V10"/></svg>
      </button>
      <button class="icon-btn" title="刷新" @click="refresh">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"><path d="M21 12a9 9 0 1 1-3-6.7L21 8"/><path d="M21 3v5h-5"/></svg>
      </button>
    </header>
    <div class="body">
      <div v-if="error" class="error">{{ error }}</div>
      <div v-else-if="loading" class="loading">加载中…</div>
      <ul v-else-if="entries.length" class="entries">
        <li
          v-for="e in entries"
          :key="e.path"
          class="entry"
          :class="{ dir: e.isDir, sel: selected?.path === e.path }"
          @click="select(e)"
          @dblclick="enter(e)"
        >
          <span class="ic">
            <svg v-if="e.isDir" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>
            <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><path d="M14 3v6h6"/></svg>
          </span>
          <span class="name">{{ e.name }}</span>
          <span class="size">{{ e.isDir ? "" : fmtSize(e.size) }}</span>
        </li>
      </ul>
      <div v-else class="empty">空目录</div>
    </div>
    <footer>
      <span class="hint">双击进入 · 主目录 / 上一级导航（只读）</span>
    </footer>
  </section>
</template>

<style scoped>
.pane {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--bg-elev);
}
header {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.pane-tag {
  font-size: 10.5px;
  font-weight: 600;
  color: var(--fg-muted);
  padding: 2px 6px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}
.path {
  flex: 1;
  min-width: 0;
  font-family: var(--mono, monospace);
  font-size: 11.5px;
  padding: 4px 7px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--fg-muted);
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
.loading,
.empty {
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
  grid-template-columns: 18px 1fr auto;
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
.entry.sel {
  background: var(--bg-active, var(--bg-hover));
  outline: 1px solid var(--accent);
  outline-offset: -1px;
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
</style>
