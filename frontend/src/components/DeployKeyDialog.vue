<script setup lang="ts">
import { computed, ref } from "vue";
import type { ManagedKey, HostRecord } from "../wails.d";

const props = defineProps<{ keyRec: ManagedKey; hosts: HostRecord[] }>();
const emit = defineEmits<{ (e: "close"): void }>();

const selected = ref<string>("");
const deploying = ref(false);
const result = ref<{ ok: boolean; msg: string } | null>(null);

// Only hosts with stored credentials can be auto-deployed.
const eligible = computed(() =>
  props.hosts.filter((h) => h.authType === "password" || h.authType === "key" || h.authType === "managedKey")
);

async function deploy() {
  if (!selected.value) return;
  deploying.value = true;
  result.value = null;
  try {
    await window.go.main.App.DeployPublicKeyToHost(selected.value, props.keyRec.id);
    result.value = { ok: true, msg: "公钥已成功追加到目标主机的 ~/.ssh/authorized_keys" };
  } catch (e: any) {
    result.value = { ok: false, msg: String(e?.message || e) };
  } finally {
    deploying.value = false;
  }
}
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="emit('close')">
    <div class="modal" style="min-width: 460px; max-width: 540px">
      <div class="modal-header">部署公钥到主机</div>
      <div class="modal-body">
        <p class="hint">
          将密钥 <strong>{{ keyRec.name }}</strong> 的公钥追加到所选主机的 <span class="mono">~/.ssh/authorized_keys</span>。
          使用主机已保存的凭据完成此操作（重复运行不会重复添加）。
        </p>
        <div class="field">
          <label>目标主机</label>
          <select v-model="selected">
            <option value="">— 选择 —</option>
            <option v-for="h in eligible" :key="h.id" :value="h.id">
              {{ h.name || h.address }} ({{ h.user }}@{{ h.address }})
            </option>
          </select>
        </div>
        <div v-if="result" class="result" :class="{ ok: result.ok, err: !result.ok }">
          {{ result.msg }}
        </div>
      </div>
      <div class="modal-footer">
        <button type="button" class="ghost" @click="emit('close')">关闭</button>
        <button type="button" class="primary" :disabled="!selected || deploying" @click="deploy">
          {{ deploying ? "部署中…" : "部署" }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hint {
  margin: 0 0 14px;
  color: var(--fg-muted);
  font-size: 12.5px;
  line-height: 1.55;
}
.mono { font-family: var(--mono, monospace); color: var(--fg); }
.field { display: flex; flex-direction: column; gap: 4px; }
.field label {
  font-size: 11px;
  color: var(--fg-subtle);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.result {
  margin-top: 14px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  font-size: 12.5px;
}
.result.ok {
  background: color-mix(in oklab, var(--success) 14%, transparent);
  color: var(--success);
}
.result.err {
  background: color-mix(in oklab, var(--danger) 14%, transparent);
  color: var(--danger);
}
</style>
