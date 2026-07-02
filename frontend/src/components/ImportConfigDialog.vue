<script setup lang="ts">
import { onMounted, ref, computed } from "vue";
import type { SshConfigPreviewEntry, SshConfigEntry } from "../wails.d";

const emit = defineEmits<{
  (e: "imported", count: number): void;
  (e: "close"): void;
}>();

const path = ref("");
const entries = ref<SshConfigPreviewEntry[]>([]);
const checked = ref<Record<number, boolean>>({});
const loading = ref(false);
const error = ref("");
const busy = ref(false);

async function loadPreview(p: string) {
  loading.value = true;
  error.value = "";
  entries.value = [];
  checked.value = {};
  try {
    const list = (await window.go.main.App.PreviewSshConfig(p)) || [];
    entries.value = list;
    // Default: import everything except duplicates.
    list.forEach((e, i) => (checked.value[i] = !e.duplicate));
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    loading.value = false;
  }
}

async function browse() {
  const p = await window.go.main.App.PickSshConfig();
  if (p) {
    path.value = p;
    await loadPreview(p);
  }
}

const selectedCount = computed(
  () => entries.value.filter((_, i) => checked.value[i]).length
);

async function doImport() {
  const chosen: SshConfigEntry[] = entries.value
    .filter((_, i) => checked.value[i])
    .map((e) => ({
      alias: e.alias,
      hostName: e.hostName,
      user: e.user,
      port: e.port,
      identityFile: e.identityFile,
      warnings: e.warnings || [],
    }));
  if (chosen.length === 0) {
    emit("close");
    return;
  }
  busy.value = true;
  try {
    const res = await window.go.main.App.ImportSshConfig(chosen);
    emit("imported", res.imported);
    emit("close");
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    busy.value = false;
  }
}

onMounted(async () => {
  try {
    path.value = (await window.go.main.App.DefaultSshConfigPath()) || "";
  } catch {}
  if (path.value) await loadPreview(path.value);
});
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="emit('close')">
    <div class="modal wide">
      <div class="modal-header">导入 ~/.ssh/config</div>
      <div class="modal-body">
        <div class="field">
          <label>配置文件</label>
          <div style="display: flex; gap: 6px">
            <input v-model="path" placeholder="~/.ssh/config" style="flex: 1" readonly />
            <button type="button" class="ghost" @click="browse">浏览...</button>
            <button type="button" class="ghost" @click="loadPreview(path)">重新加载</button>
          </div>
        </div>

        <div v-if="loading" class="note">正在解析…</div>
        <div v-else-if="error" class="note err">{{ error }}</div>
        <div v-else-if="entries.length === 0" class="note">未找到可导入的主机。</div>

        <div v-else class="list">
          <div v-for="(e, i) in entries" :key="i" class="row" :class="{ dup: e.duplicate }">
            <label class="ck">
              <input type="checkbox" v-model="checked[i]" />
            </label>
            <div class="info">
              <div class="line1">
                <span class="alias">{{ e.alias }}</span>
                <span class="addr">{{ e.user ? e.user + "@" : "" }}{{ e.hostName }}:{{ e.port || 22 }}</span>
                <span v-if="e.duplicate" class="badge dup-badge">已存在</span>
                <span v-if="e.identityFile" class="badge" :class="e.identityExists ? 'ok' : 'warn'">
                  {{ e.identityExists ? "密钥" : "密钥缺失" }}
                </span>
              </div>
              <div v-if="e.identityFile" class="idfile">{{ e.identityFile }}</div>
              <div v-for="(w, wi) in e.warnings" :key="wi" class="warnline">⚠ {{ w }}</div>
            </div>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <span class="count">已选 {{ selectedCount }} / {{ entries.length }}（重复项默认跳过）</span>
        <button type="button" class="ghost" @click="emit('close')">取消</button>
        <button type="button" class="primary" :disabled="busy || selectedCount === 0" @click="doImport">
          {{ busy ? "导入中…" : "导入" }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal.wide {
  width: min(640px, 92vw);
}
.note {
  padding: 16px 4px;
  font-size: 12.5px;
  color: var(--fg-muted);
}
.note.err {
  color: var(--danger, #e5484d);
}
.list {
  max-height: 46vh;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.row {
  display: flex;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border);
}
.row:last-child {
  border-bottom: none;
}
.row.dup {
  opacity: 0.7;
}
.ck {
  padding-top: 2px;
}
.ck input {
  accent-color: var(--accent);
  margin: 0;
}
.info {
  flex: 1;
  min-width: 0;
}
.line1 {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.alias {
  font-weight: 600;
  font-size: 13px;
}
.addr {
  font-size: 12px;
  color: var(--fg-muted);
}
.badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 999px;
  border: 1px solid var(--border-strong);
  color: var(--fg-muted);
}
.badge.ok {
  color: var(--accent);
  border-color: var(--accent);
}
.badge.warn,
.dup-badge {
  color: #e5a23c;
  border-color: #e5a23c;
}
.idfile {
  font-size: 11px;
  color: var(--fg-subtle);
  font-family: var(--mono, monospace);
  word-break: break-all;
  margin-top: 2px;
}
.warnline {
  font-size: 11px;
  color: #e5a23c;
  margin-top: 2px;
}
.count {
  flex: 1;
  font-size: 11.5px;
  color: var(--fg-muted);
}
</style>
