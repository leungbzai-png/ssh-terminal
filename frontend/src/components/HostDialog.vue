<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import type { HostRecord, ManagedKey } from "../wails.d";

const props = defineProps<{ host: HostRecord }>();
const emit = defineEmits<{
  (e: "save", h: HostRecord): void;
  (e: "cancel"): void;
}>();

const form = reactive<HostRecord>({ ...props.host });
const managedKeys = ref<ManagedKey[]>([]);
const groups = ref<string[]>([]);

async function pickKey() {
  const p = await window.go.main.App.PickPrivateKey();
  if (p) form.keyPath = p;
}

async function save() {
  if (!form.address || !form.user) {
    alert("地址和用户名不能为空");
    return;
  }
  emit("save", { ...form });
}

onMounted(async () => {
  try {
    managedKeys.value = (await window.go.main.App.ListKeys()) || [];
  } catch {}
  try {
    const all = (await window.go.main.App.ListHosts()) || [];
    groups.value = [
      ...new Set(all.map((h) => (h.group || "").trim()).filter((g) => g)),
    ].sort();
  } catch {}
});
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="emit('cancel')">
    <div class="modal">
      <div class="modal-header">{{ form.id ? "编辑主机" : "新增主机" }}</div>
      <div class="modal-body">
        <div class="field">
          <label>名称</label>
          <input v-model="form.name" placeholder="例如：production-web" />
        </div>
        <div class="field-row">
          <div class="field" style="flex: 3">
            <label>地址</label>
            <input v-model="form.address" placeholder="example.com 或 10.0.0.1" />
          </div>
          <div class="field" style="flex: 1">
            <label>端口</label>
            <input v-model.number="form.port" type="number" min="1" max="65535" />
          </div>
        </div>
        <div class="field">
          <label>用户</label>
          <input v-model="form.user" placeholder="root" />
        </div>
        <div class="field">
          <label>认证方式</label>
          <select v-model="form.authType">
            <option value="password">密码</option>
            <option value="managedKey">内置密钥 (推荐)</option>
            <option value="key">外部密钥文件</option>
          </select>
        </div>
        <div v-if="form.authType === 'password'" class="field">
          <label>密码</label>
          <input v-model="form.password" type="password" placeholder="留空表示保留已有密码" />
        </div>
        <template v-else-if="form.authType === 'managedKey'">
          <div class="field">
            <label>选择密钥</label>
            <select v-model="form.managedKeyId">
              <option value="">— 选择 —</option>
              <option v-for="k in managedKeys" :key="k.id" :value="k.id">
                {{ k.name }} ({{ k.type }}{{ k.hasPassword ? " 🔒" : "" }})
              </option>
            </select>
            <small v-if="managedKeys.length === 0" class="hint">
              还没有密钥。点击左侧栏顶部钥匙图标先生成一个。
            </small>
          </div>
          <div class="field" v-if="form.managedKeyId && managedKeys.find(k => k.id === form.managedKeyId)?.hasPassword">
            <label>密钥口令</label>
            <input v-model="form.passphrase" type="password" placeholder="留空表示保留已有口令" />
          </div>
        </template>
        <template v-else>
          <div class="field">
            <label>密钥文件路径</label>
            <div style="display: flex; gap: 6px">
              <input v-model="form.keyPath" placeholder="C:\Users\you\.ssh\id_ed25519" style="flex: 1" />
              <button type="button" class="ghost" @click="pickKey">浏览...</button>
            </div>
          </div>
          <div class="field">
            <label>口令 (如有加密)</label>
            <input v-model="form.passphrase" type="password" placeholder="留空表示保留已有口令" />
          </div>
        </template>
        <div class="field-row">
          <div class="field">
            <label>分组</label>
            <input v-model="form.group" list="host-groups" placeholder="Production（留空为 Ungrouped）" />
            <datalist id="host-groups">
              <option v-for="g in groups" :key="g" :value="g" />
            </datalist>
          </div>
          <div class="field">
            <label>备注</label>
            <input v-model="form.note" placeholder="可选" />
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <button type="button" class="ghost" @click="emit('cancel')">取消</button>
        <button type="button" class="primary" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.hint {
  font-size: 11px;
  color: var(--fg-muted);
  margin-top: 4px;
}
</style>
