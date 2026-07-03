<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import type {
  AdvancedSSH,
  DynamicForward,
  HostRecord,
  ManagedKey,
  PortForward,
} from "../wails.d";

const props = defineProps<{ host: HostRecord }>();
const emit = defineEmits<{
  (e: "save", h: HostRecord): void;
  (e: "cancel"): void;
}>();

const form = reactive<HostRecord>({ ...props.host });
const managedKeys = ref<ManagedKey[]>([]);
const groups = ref<string[]>([]);
const otherHosts = ref<HostRecord[]>([]);
const showAdvanced = ref(false);

// Ensure a fully-shaped (non-secret) advanced object so template bindings are
// stable. Kept local; pruned back to undefined on save when nothing is set.
const adv = reactive<AdvancedSSH>({
  proxyJump: props.host.advanced?.proxyJump
    ? { ...props.host.advanced.proxyJump }
    : undefined,
  localForwards: props.host.advanced?.localForwards
    ? props.host.advanced.localForwards.map((f) => ({ ...f }))
    : [],
  remoteForwards: props.host.advanced?.remoteForwards
    ? props.host.advanced.remoteForwards.map((f) => ({ ...f }))
    : [],
  dynamicForwards: props.host.advanced?.dynamicForwards
    ? props.host.advanced.dynamicForwards.map((f) => ({ ...f }))
    : [],
  autoReconnect: props.host.advanced?.autoReconnect
    ? { ...props.host.advanced.autoReconnect }
    : { enabled: false, maxAttempts: 3, delaySeconds: 3 },
});

const proxyEnabled = ref(!!adv.proxyJump);
function toggleProxy() {
  proxyEnabled.value = !proxyEnabled.value;
  if (proxyEnabled.value) {
    if (!adv.proxyJump) adv.proxyJump = { mode: "savedHost" };
  }
}

// Open by default only if this host already has advanced config, so casual
// users never see the extra complexity.
const hasExistingAdvanced = computed(
  () =>
    !!props.host.advanced &&
    (!!props.host.advanced.proxyJump ||
      (props.host.advanced.localForwards?.length || 0) > 0 ||
      (props.host.advanced.remoteForwards?.length || 0) > 0 ||
      (props.host.advanced.dynamicForwards?.length || 0) > 0 ||
      !!props.host.advanced.autoReconnect?.enabled)
);

const wildcardWarning = computed(() => {
  const wild = (h?: string) => h === "0.0.0.0" || h === "::" || h === "*";
  const local = (adv.localForwards || []).some((f) => f.enabled && wild(f.localHost));
  const dyn = (adv.dynamicForwards || []).some((f) => f.enabled && wild(f.localHost));
  return local || dyn;
});

function newForward(): PortForward {
  return { name: "", localHost: "127.0.0.1", localPort: undefined, remoteHost: "", remotePort: undefined, enabled: true };
}
function newDynamic(): DynamicForward {
  return { name: "", localHost: "127.0.0.1", localPort: undefined, enabled: true };
}

function addLocal() {
  adv.localForwards = adv.localForwards || [];
  adv.localForwards.push(newForward());
}
function addRemote() {
  adv.remoteForwards = adv.remoteForwards || [];
  adv.remoteForwards.push(newForward());
}
function addDynamic() {
  adv.dynamicForwards = adv.dynamicForwards || [];
  adv.dynamicForwards.push(newDynamic());
}

async function pickKey() {
  const p = await window.go.main.App.PickPrivateKey();
  if (p) form.keyPath = p;
}
async function pickProxyKey() {
  const p = await window.go.main.App.PickPrivateKey();
  if (p && adv.proxyJump) adv.proxyJump.keyPath = p;
}

// buildAdvanced returns the pruned advanced object, or undefined when nothing
// meaningful is configured — keeps hosts.json clean for simple hosts.
function buildAdvanced(): AdvancedSSH | undefined {
  const out: AdvancedSSH = {};
  if (proxyEnabled.value && adv.proxyJump) out.proxyJump = adv.proxyJump;
  const nonEmptyLocal = (adv.localForwards || []).filter((f) => f.enabled || f.localPort || f.remotePort);
  const nonEmptyRemote = (adv.remoteForwards || []).filter((f) => f.enabled || f.localPort || f.remotePort);
  const nonEmptyDyn = (adv.dynamicForwards || []).filter((f) => f.enabled || f.localPort);
  if (nonEmptyLocal.length) out.localForwards = nonEmptyLocal;
  if (nonEmptyRemote.length) out.remoteForwards = nonEmptyRemote;
  if (nonEmptyDyn.length) out.dynamicForwards = nonEmptyDyn;
  if (adv.autoReconnect?.enabled) out.autoReconnect = adv.autoReconnect;
  const empty =
    !out.proxyJump &&
    !out.localForwards &&
    !out.remoteForwards &&
    !out.dynamicForwards &&
    !out.autoReconnect;
  return empty ? undefined : out;
}

async function save() {
  if (!form.address || !form.user) {
    alert("地址和用户名不能为空");
    return;
  }
  form.advanced = buildAdvanced();
  emit("save", { ...form });
}

onMounted(async () => {
  showAdvanced.value = hasExistingAdvanced.value;
  try {
    managedKeys.value = (await window.go.main.App.ListKeys()) || [];
  } catch {}
  try {
    const all = (await window.go.main.App.ListHosts()) || [];
    groups.value = [
      ...new Set(all.map((h) => (h.group || "").trim()).filter((g) => g)),
    ].sort();
    // Jump-host candidates: any saved host other than the one being edited.
    otherHosts.value = all.filter((h) => h.id !== form.id);
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

        <!-- Advanced SSH: collapsed by default. -->
        <div class="adv">
          <button type="button" class="adv-toggle" @click="showAdvanced = !showAdvanced">
            <span class="chev" :class="{ open: showAdvanced }">▸</span>
            高级 SSH（跳板机 / 端口转发 / 自动重连）
          </button>

          <div v-if="showAdvanced" class="adv-body">
            <!-- ProxyJump -->
            <div class="adv-section">
              <label class="chk">
                <input type="checkbox" :checked="proxyEnabled" @change="toggleProxy" />
                通过跳板机连接 (ProxyJump)
              </label>
              <template v-if="proxyEnabled && adv.proxyJump">
                <div class="field">
                  <label>跳板机来源</label>
                  <select v-model="adv.proxyJump.mode">
                    <option value="savedHost">引用已保存主机（推荐，可用密码或密钥）</option>
                    <option value="manual">手动填写（仅限密钥，不支持密码）</option>
                  </select>
                </div>
                <div v-if="adv.proxyJump.mode === 'savedHost'" class="field">
                  <label>跳板机主机</label>
                  <select v-model="adv.proxyJump.jumpHostId">
                    <option value="">— 选择 —</option>
                    <option v-for="h in otherHosts" :key="h.id" :value="h.id">
                      {{ h.name || h.address }} ({{ h.user }}@{{ h.address }})
                    </option>
                  </select>
                  <small class="hint">跳板机认证沿用其已加密保存的凭据，不会复制任何密钥/密码。</small>
                </div>
                <template v-else>
                  <div class="field-row">
                    <div class="field" style="flex: 3">
                      <label>跳板机地址</label>
                      <input v-model="adv.proxyJump.address" placeholder="bastion.example.com" />
                    </div>
                    <div class="field" style="flex: 1">
                      <label>端口</label>
                      <input v-model.number="adv.proxyJump.port" type="number" min="1" max="65535" placeholder="22" />
                    </div>
                  </div>
                  <div class="field">
                    <label>跳板机用户</label>
                    <input v-model="adv.proxyJump.user" placeholder="jump" />
                  </div>
                  <div class="field">
                    <label>跳板机密钥文件（必填）</label>
                    <div style="display: flex; gap: 6px">
                      <input v-model="adv.proxyJump.keyPath" placeholder="仅路径引用，不复制私钥" style="flex: 1" />
                      <button type="button" class="ghost" @click="pickProxyKey">浏览...</button>
                    </div>
                    <small class="hint">手动跳板机仅支持密钥认证；需要密码的跳板机请改用“引用已保存主机”。</small>
                  </div>
                </template>
              </template>
            </div>

            <!-- Local forwards -->
            <div class="adv-section">
              <div class="adv-row-head">
                <span>本地端口转发</span>
                <button type="button" class="ghost sm" @click="addLocal">+ 添加</button>
              </div>
              <small class="hint">本地 localHost:localPort → 远端 remoteHost:remotePort，默认绑定 127.0.0.1。</small>
              <div v-for="(f, i) in adv.localForwards" :key="'lf' + i" class="tunnel">
                <input v-model="f.name" class="t-name" placeholder="名称" />
                <input v-model="f.localHost" class="t-host" placeholder="127.0.0.1" />
                <input v-model.number="f.localPort" class="t-port" type="number" min="1" max="65535" placeholder="本地端口" />
                <span class="arrow">→</span>
                <input v-model="f.remoteHost" class="t-host" placeholder="远端主机" />
                <input v-model.number="f.remotePort" class="t-port" type="number" min="1" max="65535" placeholder="远端端口" />
                <label class="chk sm"><input type="checkbox" v-model="f.enabled" />启用</label>
                <button type="button" class="ghost sm danger" @click="adv.localForwards!.splice(i, 1)">×</button>
              </div>
            </div>

            <!-- Dynamic (SOCKS) forwards -->
            <div class="adv-section">
              <div class="adv-row-head">
                <span>动态转发 (SOCKS5)</span>
                <button type="button" class="ghost sm" @click="addDynamic">+ 添加</button>
              </div>
              <small class="hint">在本地开启 SOCKS5 代理，例如 127.0.0.1:1080，供浏览器/调试临时使用。</small>
              <div v-for="(f, i) in adv.dynamicForwards" :key="'df' + i" class="tunnel">
                <input v-model="f.name" class="t-name" placeholder="名称" />
                <input v-model="f.localHost" class="t-host" placeholder="127.0.0.1" />
                <input v-model.number="f.localPort" class="t-port" type="number" min="1" max="65535" placeholder="本地端口" />
                <span class="socks-hint">= SOCKS5</span>
                <label class="chk sm"><input type="checkbox" v-model="f.enabled" />启用</label>
                <button type="button" class="ghost sm danger" @click="adv.dynamicForwards!.splice(i, 1)">×</button>
              </div>
            </div>

            <!-- Remote forwards -->
            <div class="adv-section">
              <div class="adv-row-head">
                <span>远程端口转发</span>
                <button type="button" class="ghost sm" @click="addRemote">+ 添加</button>
              </div>
              <small class="hint warn-text">
                远端 remoteHost:remotePort → 本地 localHost:localPort。是否能绑定取决于 SSH 服务器的 GatewayPorts 策略。
              </small>
              <div v-for="(f, i) in adv.remoteForwards" :key="'rf' + i" class="tunnel">
                <input v-model="f.name" class="t-name" placeholder="名称" />
                <input v-model="f.remoteHost" class="t-host" placeholder="127.0.0.1" />
                <input v-model.number="f.remotePort" class="t-port" type="number" min="1" max="65535" placeholder="远端端口" />
                <span class="arrow">→</span>
                <input v-model="f.localHost" class="t-host" placeholder="127.0.0.1" />
                <input v-model.number="f.localPort" class="t-port" type="number" min="1" max="65535" placeholder="本地端口" />
                <label class="chk sm"><input type="checkbox" v-model="f.enabled" />启用</label>
                <button type="button" class="ghost sm danger" @click="adv.remoteForwards!.splice(i, 1)">×</button>
              </div>
            </div>

            <!-- Auto reconnect -->
            <div class="adv-section" v-if="adv.autoReconnect">
              <label class="chk">
                <input type="checkbox" v-model="adv.autoReconnect.enabled" />
                意外断线时自动重连
              </label>
              <div v-if="adv.autoReconnect.enabled" class="field-row">
                <div class="field">
                  <label>最大重试次数 (0-10)</label>
                  <input v-model.number="adv.autoReconnect.maxAttempts" type="number" min="0" max="10" />
                </div>
                <div class="field">
                  <label>重试间隔（秒，1-60）</label>
                  <input v-model.number="adv.autoReconnect.delaySeconds" type="number" min="1" max="60" />
                </div>
              </div>
              <small class="hint">仅在连接建立后意外断开时触发；主动断开或认证失败不会自动重连。</small>
            </div>

            <div v-if="wildcardWarning" class="warn-box">
              ⚠ 有转发绑定到 0.0.0.0/::，将对局域网暴露该端口。除非确有需要，建议使用 127.0.0.1。
            </div>
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
  display: block;
}
.warn-text {
  color: var(--warning, #b45309);
}
.adv {
  margin-top: 10px;
  border-top: 1px solid var(--border);
  padding-top: 8px;
}
.adv-toggle {
  background: none;
  border: none;
  color: var(--fg);
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  padding: 4px 0;
  display: flex;
  align-items: center;
  gap: 6px;
}
.chev {
  display: inline-block;
  transition: transform 0.12s ease;
}
.chev.open {
  transform: rotate(90deg);
}
.adv-body {
  margin-top: 6px;
}
.adv-section {
  padding: 8px 0;
  border-top: 1px dashed var(--border);
}
.adv-row-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  font-weight: 600;
}
.chk {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  cursor: pointer;
}
.chk.sm {
  font-size: 11px;
}
.tunnel {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  flex-wrap: wrap;
}
.tunnel input {
  padding: 3px 6px;
  font-size: 11.5px;
  background: var(--bg);
  border: 1px solid var(--border);
  color: var(--fg);
  border-radius: var(--radius-sm);
}
.t-name {
  width: 72px;
}
.t-host {
  width: 96px;
}
.t-port {
  width: 74px;
}
.arrow,
.socks-hint {
  font-size: 11px;
  color: var(--fg-muted);
}
.ghost.sm {
  padding: 2px 8px;
  font-size: 11px;
}
.ghost.sm.danger {
  color: var(--danger);
}
.warn-box {
  margin-top: 8px;
  padding: 6px 10px;
  font-size: 11.5px;
  background: color-mix(in oklab, var(--warning, #f59e0b) 15%, transparent);
  border: 1px solid color-mix(in oklab, var(--warning, #f59e0b) 45%, transparent);
  border-radius: var(--radius-sm);
  color: var(--fg);
}
</style>
