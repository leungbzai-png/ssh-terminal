<script setup lang="ts">
import { reactive } from "vue";
import { useSettings } from "../stores/settings";

const emit = defineEmits<{ (e: "close"): void }>();
const settings = useSettings();
const form = reactive({ ...settings.settings });

async function save() {
  await settings.save({ ...form });
  emit("close");
}
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="emit('close')">
    <div class="modal">
      <div class="modal-header">设置</div>
      <div class="modal-body">
        <div class="field">
          <label>外观</label>
          <div class="seg">
            <button type="button" :class="{ on: form.theme === 'light' }" @click="form.theme = 'light'">浅色</button>
            <button type="button" :class="{ on: form.theme === 'dark' }" @click="form.theme = 'dark'">深色</button>
            <button type="button" :class="{ on: form.theme === 'system' }" @click="form.theme = 'system'">跟随系统</button>
          </div>
        </div>
        <div class="field">
          <label>字体</label>
          <input v-model="form.fontFamily" />
        </div>
        <div class="field-row">
          <div class="field">
            <label>字号</label>
            <input v-model.number="form.fontSize" type="number" min="9" max="28" />
          </div>
          <div class="field">
            <label>回滚行数</label>
            <input v-model.number="form.scrollBack" type="number" min="100" max="100000" step="500" />
          </div>
        </div>
        <div class="field-row">
          <div class="field">
            <label>光标样式</label>
            <select v-model="form.cursorStyle">
              <option value="bar">竖线</option>
              <option value="block">方块</option>
              <option value="underline">下划线</option>
            </select>
          </div>
          <div class="field">
            <label>光标闪烁</label>
            <select v-model="form.cursorBlink">
              <option :value="true">开</option>
              <option :value="false">关</option>
            </select>
          </div>
        </div>
        <div class="field">
          <label>连接超时（秒）</label>
          <input v-model.number="form.connectTimeoutSec" type="number" min="5" max="120" step="5" />
        </div>
        <div class="field">
          <label class="ck">
            <input type="checkbox" v-model="form.showCommandBar" />
            <span>显示命令输入栏（终端底部）</span>
          </label>
        </div>
        <div class="field">
          <label class="ck">
            <input type="checkbox" v-model="form.confirmCloseWithActiveSessions" />
            <span>关闭应用时若有活跃会话先询问</span>
          </label>
        </div>
      </div>
      <div class="modal-footer">
        <button type="button" class="ghost" @click="emit('close')">取消</button>
        <button type="button" class="primary" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.seg {
  display: inline-flex;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
  width: fit-content;
}
.seg button {
  border-radius: 0;
  padding: 6px 14px;
  border: none;
  border-right: 1px solid var(--border);
}
.seg button:last-child {
  border-right: none;
}
.seg button.on {
  background: var(--accent);
  color: var(--accent-fg);
}
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
