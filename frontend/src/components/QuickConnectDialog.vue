<script setup lang="ts">
import { reactive, ref } from "vue";
import type { QuickConnectParams } from "../wails.d";

const emit = defineEmits<{
  (e: "connect", params: QuickConnectParams, remember: boolean): void;
  (e: "cancel"): void;
}>();

const form = reactive<QuickConnectParams>({
  address: "",
  port: 22,
  user: "",
  authType: "password",
  password: "",
  keyPath: "",
  passphrase: "",
});
const remember = ref(false);

async function pickKey() {
  const p = await window.go.main.App.PickPrivateKey();
  if (p) form.keyPath = p;
}

function connect() {
  if (!form.address || !form.user) {
    alert("地址和用户名不能为空");
    return;
  }
  emit("connect", { ...form }, remember.value);
}
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="emit('cancel')">
    <div class="modal">
      <div class="modal-header">快速连接</div>
      <div class="modal-body">
        <div class="field-row">
          <div class="field" style="flex: 3">
            <label>地址</label>
            <input v-model="form.address" placeholder="example.com 或 10.0.0.1" autofocus />
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
            <option value="key">外部密钥文件</option>
          </select>
        </div>
        <div v-if="form.authType === 'password'" class="field">
          <label>密码</label>
          <input v-model="form.password" type="password" placeholder="临时密码，不会保存（除非勾选记住主机）" />
        </div>
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
            <input v-model="form.passphrase" type="password" placeholder="临时口令，不会保存（除非勾选记住主机）" />
          </div>
        </template>
        <div class="field">
          <label class="ck">
            <input type="checkbox" v-model="remember" />
            <span>记住此主机（保存到主机列表，密码将加密存储）</span>
          </label>
        </div>
      </div>
      <div class="modal-footer">
        <button type="button" class="ghost" @click="emit('cancel')">取消</button>
        <button type="button" class="primary" @click="connect">连接</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ck {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  color: var(--fg);
  text-transform: none;
  letter-spacing: 0;
  cursor: pointer;
}
.ck input {
  margin: 0;
  accent-color: var(--accent);
}
</style>
