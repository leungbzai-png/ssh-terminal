<script setup lang="ts">
import { ref } from "vue";
import { useSettings } from "../stores/settings";

const props = defineProps<{ activeCount: number }>();
const emit = defineEmits<{ (e: "cancel"): void; (e: "confirm"): void }>();

const settings = useSettings();
const dontAsk = ref(false);

async function confirm() {
  if (dontAsk.value) {
    await settings.save({ ...settings.settings, confirmCloseWithActiveSessions: false });
  }
  emit("confirm");
}
</script>

<template>
  <div class="modal-backdrop">
    <div class="modal" style="min-width: 380px; max-width: 460px">
      <div class="modal-header">关闭应用？</div>
      <div class="modal-body">
        <p style="margin: 0 0 10px">
          你有 <strong>{{ props.activeCount }}</strong> 个活跃的 SSH 会话。关闭应用会断开所有连接。
        </p>
        <label class="dont-ask">
          <input type="checkbox" v-model="dontAsk" />
          <span>下次不再询问</span>
        </label>
      </div>
      <div class="modal-footer">
        <button type="button" class="ghost" @click="emit('cancel')">取消</button>
        <button type="button" class="primary danger-btn" @click="confirm">关闭</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dont-ask {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--fg-muted);
  cursor: pointer;
  user-select: none;
}
.dont-ask input {
  margin: 0;
  accent-color: var(--accent);
}
.danger-btn {
  background: var(--danger);
  border-color: var(--danger);
  color: white;
}
.danger-btn:hover {
  filter: brightness(1.08);
  background: var(--danger);
}
</style>
