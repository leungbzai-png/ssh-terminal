<script setup lang="ts">
defineEmits<{ (e: "close"): void }>();

// Only real, currently-wired shortcuts and actions are listed — nothing aspirational.
const shortcuts: { keys: string; desc: string }[] = [
  { keys: "Ctrl + F", desc: "在当前终端中搜索" },
  { keys: "Enter", desc: "查找下一个匹配（搜索框内）" },
  { keys: "Shift + Enter", desc: "查找上一个匹配（搜索框内）" },
  { keys: "Esc", desc: "关闭搜索框" },
  { keys: "Ctrl + =", desc: "增大终端字号" },
  { keys: "Ctrl + -", desc: "减小终端字号" },
  { keys: "Ctrl + 0", desc: "重置终端字号" },
  { keys: "F1", desc: "打开本快捷键帮助" },
];

const actions: { how: string; desc: string }[] = [
  { how: "双击主机", desc: "在当前分屏打开并连接（侧栏）" },
  { how: "右键标签", desc: "重新连接 / 克隆会话 / 关闭标签" },
  { how: "拖入文件", desc: "上传到当前会话的 SFTP 工作目录" },
  { how: "双击文本文件", desc: "在 SFTP 面板中只读预览" },
  { how: "SFTP ★", desc: "远程路径书签：添加当前路径 / 跳转 / 删除" },
];
</script>

<template>
  <div class="modal-backdrop" @mousedown.self="$emit('close')">
    <div class="modal">
      <div class="modal-header">键盘快捷键</div>
      <div class="modal-body">
        <section>
          <h4>快捷键</h4>
          <table>
            <tbody>
              <tr v-for="s in shortcuts" :key="s.keys">
                <td class="keys"><kbd>{{ s.keys }}</kbd></td>
                <td class="desc">{{ s.desc }}</td>
              </tr>
            </tbody>
          </table>
        </section>
        <section>
          <h4>鼠标操作</h4>
          <table>
            <tbody>
              <tr v-for="a in actions" :key="a.how">
                <td class="keys"><span class="act">{{ a.how }}</span></td>
                <td class="desc">{{ a.desc }}</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>
      <div class="modal-footer">
        <button type="button" class="primary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
section {
  margin-bottom: 16px;
}
section:last-child {
  margin-bottom: 0;
}
h4 {
  margin: 0 0 8px;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--fg-subtle);
  font-weight: 600;
}
table {
  width: 100%;
  border-collapse: collapse;
}
td {
  padding: 5px 6px;
  font-size: 12.5px;
  vertical-align: middle;
}
.keys {
  width: 150px;
  white-space: nowrap;
}
kbd {
  font-family: var(--mono, monospace);
  font-size: 11px;
  padding: 2px 7px;
  border: 1px solid var(--border-strong);
  border-bottom-width: 2px;
  border-radius: 5px;
  background: var(--bg-elev-2);
  color: var(--fg);
}
.act {
  font-size: 11.5px;
  padding: 2px 7px;
  border-radius: 5px;
  background: var(--bg-elev-2);
  border: 1px solid var(--border);
  color: var(--fg-muted);
}
.desc {
  color: var(--fg);
}
</style>
