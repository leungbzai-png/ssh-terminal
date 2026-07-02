<script setup lang="ts">
import { ref, computed, watch } from "vue";
import type {
  HostsImportPreview,
  HostImportPreviewEntry,
  SafeHost,
  HostsImportResult,
} from "../wails.d";

const props = defineProps<{ preview: HostsImportPreview }>();
const emit = defineEmits<{
  (e: "imported", res: HostsImportResult): void;
  (e: "close"): void;
}>();

const entries = ref<HostImportPreviewEntry[]>(props.preview.hosts || []);
const overwrite = ref(false);
const busy = ref(false);
const error = ref("");

// Duplicates default to skip (unchecked). Everything else is selected.
const checked = ref<Record<number, boolean>>({});
entries.value.forEach((e, i) => (checked.value[i] = !e.duplicate));

// When the user explicitly enables overwrite, duplicates become selectable and
// are auto-selected; turning it back off re-skips them.
watch(overwrite, (on) => {
  entries.value.forEach((e, i) => {
    if (e.duplicate) checked.value[i] = on;
  });
});

const dupCount = computed(() => entries.value.filter((e) => e.duplicate).length);
const selectedCount = computed(
  () => entries.value.filter((_, i) => checked.value[i]).length
);

async function doImport() {
  const chosen: SafeHost[] = entries.value
    .filter((_, i) => checked.value[i])
    .map((e) => ({
      name: e.name,
      address: e.address,
      port: e.port,
      user: e.user,
      authType: e.authType,
      keyPath: e.keyPath,
      managedKeyId: e.managedKeyId,
      group: e.group,
      note: e.note,
    }));
  if (chosen.length === 0) {
    emit("close");
    return;
  }
  busy.value = true;
  error.value = "";
  try {
    const res = await window.go.main.App.ImportHosts(chosen, overwrite.value);
    emit("imported", res);
    emit("close");
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="emit('close')">
    <div class="modal wide">
      <div class="modal-header">导入主机（安全文件）</div>
      <div class="modal-body">
        <div class="src">来源：<span class="mono">{{ props.preview.path }}</span></div>

        <div v-if="entries.length === 0" class="note">文件中没有可导入的主机。</div>

        <div v-else class="list">
          <div v-for="(e, i) in entries" :key="i" class="row" :class="{ dup: e.duplicate && !checked[i] }">
            <label class="ck">
              <input type="checkbox" v-model="checked[i]" :disabled="e.duplicate && !overwrite" />
            </label>
            <div class="info">
              <div class="line1">
                <span class="alias">{{ e.name || e.address }}</span>
                <span class="addr">{{ e.user ? e.user + "@" : "" }}{{ e.address }}:{{ e.port || 22 }}</span>
                <span v-if="e.group" class="badge grp">{{ e.group }}</span>
                <span v-if="e.duplicate" class="badge dup-badge">已存在</span>
                <span v-if="e.authType === 'key' && e.keyPath" class="badge" :class="e.keyExists ? 'ok' : 'warn'">
                  {{ e.keyExists ? "密钥" : "密钥缺失" }}
                </span>
                <span v-else-if="e.authType === 'managedKey'" class="badge">内置密钥</span>
                <span v-else-if="e.authType === 'password'" class="badge warn">需手动补密码</span>
              </div>
              <div v-if="e.keyPath" class="idfile">{{ e.keyPath }}</div>
            </div>
          </div>
        </div>

        <label v-if="dupCount > 0" class="overwrite">
          <input type="checkbox" v-model="overwrite" />
          覆盖已存在的重复主机（共 {{ dupCount }} 个；默认跳过）
        </label>

        <div v-if="error" class="note err">{{ error }}</div>
      </div>
      <div class="modal-footer">
        <span class="count">已选 {{ selectedCount }} / {{ entries.length }}</span>
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
.src {
  font-size: 11.5px;
  color: var(--fg-muted);
  margin-bottom: 10px;
  word-break: break-all;
}
.mono {
  font-family: var(--mono, monospace);
  color: var(--fg);
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
  opacity: 0.6;
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
.badge.grp {
  color: var(--accent);
  border-color: var(--accent);
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
.overwrite {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 12px;
  color: var(--fg-muted);
}
.overwrite input {
  accent-color: var(--accent);
  margin: 0;
}
.count {
  flex: 1;
  font-size: 11.5px;
  color: var(--fg-muted);
}
</style>
