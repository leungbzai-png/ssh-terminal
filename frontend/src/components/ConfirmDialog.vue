<script setup lang="ts">
defineProps<{
  title: string;
  message: string;
  confirmLabel?: string;
  danger?: boolean;
}>();

const emit = defineEmits<{
  (e: "confirm"): void;
  (e: "cancel"): void;
}>();
</script>

<template>
  <div class="modal-backdrop">
    <div class="modal" style="min-width: 340px; max-width: 420px">
      <div class="modal-header">{{ title }}</div>
      <div class="modal-body">
        <p style="margin: 0">{{ message }}</p>
      </div>
      <div class="modal-footer">
        <button type="button" class="ghost" @click="emit('cancel')">取消</button>
        <button
          type="button"
          :class="['primary', { 'danger-btn': danger }]"
          @click="emit('confirm')"
        >{{ confirmLabel ?? '确认' }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
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
