<script setup lang="ts">
import { ref, onMounted } from "vue";

const props = defineProps<{
  title: string;
  placeholder?: string;
  defaultValue?: string;
  confirmLabel?: string;
}>();

const emit = defineEmits<{
  (e: "confirm", value: string): void;
  (e: "cancel"): void;
}>();

const value = ref(props.defaultValue ?? "");
const inputEl = ref<HTMLInputElement | null>(null);

onMounted(() => {
  inputEl.value?.focus();
  inputEl.value?.select();
});

function submit() {
  const v = value.value.trim();
  if (v) emit("confirm", v);
}
</script>

<template>
  <div class="modal-backdrop">
    <div class="modal" style="min-width: 340px; max-width: 420px">
      <div class="modal-header">{{ title }}</div>
      <div class="modal-body">
        <input
          ref="inputEl"
          v-model="value"
          class="input-field"
          :placeholder="placeholder ?? ''"
          @keydown.enter="submit"
          @keydown.esc="emit('cancel')"
        />
      </div>
      <div class="modal-footer">
        <button type="button" class="ghost" @click="emit('cancel')">取消</button>
        <button type="button" class="primary" :disabled="!value.trim()" @click="submit">
          {{ confirmLabel ?? "确认" }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.input-field {
  width: 100%;
  box-sizing: border-box;
}
</style>
