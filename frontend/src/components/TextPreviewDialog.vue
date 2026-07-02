<script setup lang="ts">
import type { TextPreview } from "../wails.d";

const props = defineProps<{ name: string; preview: TextPreview }>();
defineEmits<{ (e: "close"): void }>();

function fmtSize(n: number) {
  if (n < 1024) return n + " B";
  const u = ["KB", "MB", "GB"];
  let v = n / 1024,
    i = 0;
  while (v >= 1024 && i < u.length - 1) {
    v /= 1024;
    i++;
  }
  return v.toFixed(v < 10 ? 1 : 0) + " " + u[i];
}
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="$emit('close')">
    <div class="modal wide">
      <div class="modal-header">
        <span class="fname">{{ props.name }}</span>
        <span class="meta">{{ fmtSize(props.preview.size) }} · 只读预览</span>
      </div>
      <div class="modal-body">
        <div v-if="props.preview.tooLarge" class="note">
          文件过大（超过 512 KB），请下载后在本地查看。
        </div>
        <div v-else-if="props.preview.binary" class="note">
          该文件不是文本（或非 UTF-8 编码），无法预览。
        </div>
        <pre v-else class="content">{{ props.preview.content }}</pre>
      </div>
      <div class="modal-footer">
        <button type="button" class="primary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal.wide {
  width: min(760px, 94vw);
}
.modal-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}
.fname {
  font-family: var(--mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.meta {
  font-size: 11px;
  color: var(--fg-muted);
  flex-shrink: 0;
}
.note {
  padding: 20px 6px;
  font-size: 12.5px;
  color: var(--fg-muted);
  text-align: center;
}
.content {
  margin: 0;
  max-height: 60vh;
  overflow: auto;
  background: var(--bg-elev-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 10px 12px;
  font-family: var(--mono, monospace);
  font-size: 12px;
  line-height: 1.5;
  white-space: pre;
  color: var(--fg);
}
</style>
