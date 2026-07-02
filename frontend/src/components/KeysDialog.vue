<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import type { ManagedKey, HostRecord } from "../wails.d";
import DeployKeyDialog from "./DeployKeyDialog.vue";

const emit = defineEmits<{ (e: "close"): void }>();

const keys = ref<ManagedKey[]>([]);
const loading = ref(false);
const generating = ref(false);
const error = ref("");

const form = reactive({
  name: "",
  comment: "",
  type: "ed25519" as "ed25519" | "rsa",
  rsaBits: 4096,
  passphrase: "",
});

const importForm = reactive({
  name: "",
  comment: "",
  path: "",
  passphrase: "",
});
const importing = ref(false);

const showPubFor = ref<string | null>(null);
const deployFor = ref<ManagedKey | null>(null);
const hostsCache = ref<HostRecord[]>([]);

async function refresh() {
  loading.value = true;
  error.value = "";
  try {
    keys.value = (await window.go.main.App.ListKeys()) || [];
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    loading.value = false;
  }
}

async function generate() {
  if (!form.name.trim()) {
    alert("请填写密钥名称");
    return;
  }
  generating.value = true;
  error.value = "";
  try {
    await window.go.main.App.GenerateKey(
      form.name.trim(),
      form.comment.trim() || `${form.name}@ssh-terminal`,
      form.type,
      form.type === "rsa" ? form.rsaBits : 0,
      form.passphrase
    );
    form.name = "";
    form.comment = "";
    form.passphrase = "";
    await refresh();
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    generating.value = false;
  }
}

async function pickImportKey() {
  try {
    const p = await window.go.main.App.PickPrivateKey();
    if (p) importForm.path = p;
  } catch {}
}

async function importKey() {
  if (!importForm.name.trim()) {
    alert("请填写密钥名称");
    return;
  }
  if (!importForm.path.trim()) {
    alert("请选择私钥文件");
    return;
  }
  importing.value = true;
  error.value = "";
  try {
    await window.go.main.App.ImportPrivateKey(
      importForm.name.trim(),
      importForm.comment.trim(),
      importForm.path.trim(),
      importForm.passphrase
    );
    importForm.name = "";
    importForm.comment = "";
    importForm.path = "";
    importForm.passphrase = "";
    await refresh();
  } catch (e: any) {
    error.value = String(e?.message || e);
  } finally {
    importing.value = false;
  }
}

async function remove(k: ManagedKey) {
  if (!confirm(`删除密钥 "${k.name}"？\n注意：使用此密钥的主机连接会失败。`)) return;
  try {
    await window.go.main.App.DeleteKey(k.id);
    await refresh();
  } catch (e: any) {
    alert(String(e?.message || e));
  }
}

async function copyPub(k: ManagedKey) {
  try {
    await navigator.clipboard.writeText(k.publicKey);
    flash(k.id);
  } catch {
    // fallback: select text
    showPubFor.value = k.id;
  }
}

const flashed = ref<Set<string>>(new Set());
function flash(id: string) {
  flashed.value.add(id);
  setTimeout(() => {
    flashed.value.delete(id);
    flashed.value = new Set(flashed.value);
  }, 1200);
}

async function loadHosts() {
  try {
    hostsCache.value = (await window.go.main.App.ListHosts()) || [];
  } catch {}
}

function openDeploy(k: ManagedKey) {
  deployFor.value = k;
}

onMounted(() => {
  refresh();
  loadHosts();
});
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="emit('close')">
    <div class="modal" style="min-width: 620px; max-width: 740px">
      <div class="modal-header">
        <span>SSH 密钥</span>
      </div>
      <div class="modal-body">
        <!-- Generate form -->
        <section class="generator">
          <h4>生成新密钥</h4>
          <div class="gen-grid">
            <div class="field">
              <label>名称</label>
              <input v-model="form.name" placeholder="例如：vps-main" />
            </div>
            <div class="field">
              <label>注释 (可选)</label>
              <input v-model="form.comment" placeholder="leung@laptop" />
            </div>
            <div class="field">
              <label>类型</label>
              <select v-model="form.type">
                <option value="ed25519">Ed25519 (推荐)</option>
                <option value="rsa">RSA</option>
              </select>
            </div>
            <div class="field" v-if="form.type === 'rsa'">
              <label>位数</label>
              <select v-model.number="form.rsaBits">
                <option :value="2048">2048</option>
                <option :value="3072">3072</option>
                <option :value="4096">4096</option>
              </select>
            </div>
            <div class="field" :style="form.type === 'rsa' ? {} : { gridColumn: 'span 2' }">
              <label>口令保护 (可选)</label>
              <input v-model="form.passphrase" type="password" placeholder="留空表示不加口令" />
            </div>
          </div>
          <div class="gen-actions">
            <button type="button" class="primary" :disabled="generating" @click="generate">
              {{ generating ? "生成中…" : "生成密钥" }}
            </button>
          </div>
        </section>

        <!-- Import existing private key -->
        <section class="generator">
          <h4>导入已有私钥</h4>
          <div class="gen-grid">
            <div class="field">
              <label>名称</label>
              <input v-model="importForm.name" placeholder="例如：old-server-key" />
            </div>
            <div class="field">
              <label>注释 (可选)</label>
              <input v-model="importForm.comment" placeholder="imported" />
            </div>
            <div class="field" style="grid-column: span 2">
              <label>私钥文件</label>
              <div style="display: flex; gap: 6px">
                <input v-model="importForm.path" placeholder="选择私钥文件…" style="flex: 1" readonly />
                <button type="button" class="ghost" @click="pickImportKey">浏览...</button>
              </div>
            </div>
            <div class="field" style="grid-column: span 2">
              <label>口令 (如私钥已加密)</label>
              <input v-model="importForm.passphrase" type="password" placeholder="未加密可留空" />
            </div>
          </div>
          <small class="hint">私钥将立即加密保存为 <code>.key.enc</code>，绝不以明文写入 data/；口令仅用于校验，不会被保存。</small>
          <div class="gen-actions">
            <button type="button" class="primary" :disabled="importing" @click="importKey">
              {{ importing ? "导入中…" : "导入私钥" }}
            </button>
          </div>
        </section>

        <!-- Existing keys -->
        <section class="keylist">
          <h4>已有密钥 ({{ keys.length }})</h4>
          <div v-if="error" class="error">{{ error }}</div>
          <div v-if="loading" class="loading">加载中…</div>
          <div v-else-if="keys.length === 0" class="empty">还没有密钥，生成一个吧。</div>
          <ul v-else>
            <li v-for="k in keys" :key="k.id" class="key">
              <div class="key-head">
                <div class="key-name">
                  <span class="kname">{{ k.name }}</span>
                  <span class="ktype">{{ k.type }}</span>
                  <span v-if="k.hasPassword" class="badge" title="带口令保护">🔒</span>
                </div>
                <div class="key-actions">
                  <button type="button" class="ghost" @click="copyPub(k)">
                    {{ flashed.has(k.id) ? "已复制 ✓" : "复制公钥" }}
                  </button>
                  <button type="button" class="ghost" @click="showPubFor = showPubFor === k.id ? null : k.id">
                    {{ showPubFor === k.id ? "隐藏" : "查看" }}
                  </button>
                  <button type="button" class="ghost" @click="openDeploy(k)">部署到主机…</button>
                  <button type="button" class="ghost danger" @click="remove(k)">删除</button>
                </div>
              </div>
              <div class="fingerprint">{{ k.fingerprint }}</div>
              <div v-if="showPubFor === k.id" class="pubkey">
                <code>{{ k.publicKey }}</code>
              </div>
            </li>
          </ul>
        </section>
      </div>
      <div class="modal-footer">
        <button type="button" class="primary" @click="emit('close')">关闭</button>
      </div>
    </div>

    <DeployKeyDialog
      v-if="deployFor"
      :keyRec="deployFor"
      :hosts="hostsCache"
      @close="deployFor = null"
    />
  </div>
</template>

<style scoped>
.modal-header { display: flex; justify-content: space-between; align-items: center; }
section { margin-bottom: 18px; }
section:last-child { margin-bottom: 0; }
h4 {
  margin: 0 0 10px;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--fg-subtle);
  font-weight: 600;
}

.gen-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px 12px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.field label {
  font-size: 11px;
  color: var(--fg-subtle);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.gen-actions {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
.hint {
  display: block;
  margin-top: 10px;
  font-size: 11px;
  color: var(--fg-muted);
  line-height: 1.5;
}
.hint code {
  font-family: var(--mono, monospace);
  font-size: 10.5px;
}

.error { padding: 8px 10px; background: color-mix(in oklab, var(--danger) 12%, transparent); color: var(--danger); border-radius: var(--radius-sm); font-size: 12px; }
.loading, .empty { padding: 14px; color: var(--fg-muted); font-size: 12px; text-align: center; }

.keylist ul {
  list-style: none;
  margin: 0;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}
.key {
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
}
.key:last-child { border-bottom: none; }
.key-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}
.key-name {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.kname {
  font-weight: 500;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ktype {
  font-size: 10.5px;
  padding: 1px 6px;
  background: var(--bg-elev-2);
  border-radius: 3px;
  color: var(--fg-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.badge { font-size: 11px; }
.key-actions {
  display: flex;
  gap: 4px;
}
.key-actions button {
  padding: 4px 8px;
  font-size: 11.5px;
}
.fingerprint {
  margin-top: 4px;
  font-family: var(--mono, monospace);
  font-size: 11px;
  color: var(--fg-muted);
}
.pubkey {
  margin-top: 8px;
  background: var(--bg-elev-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 10px;
  max-height: 100px;
  overflow: auto;
}
.pubkey code {
  font-family: var(--mono, monospace);
  font-size: 11.5px;
  word-break: break-all;
  white-space: pre-wrap;
}
.danger { color: var(--danger); }
.danger:hover { background: color-mix(in oklab, var(--danger) 12%, transparent); }
</style>
