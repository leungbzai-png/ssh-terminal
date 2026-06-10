<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from "vue";

interface Prompt {
  hostname: string;
  fingerprint: string;
}

const queue = ref<Prompt[]>([]);
let handler: ((p: Prompt) => void) | null = null;

onMounted(() => {
  handler = (p: Prompt) => queue.value.push(p);
  window.runtime.EventsOn("ssh:hostkey", handler);
});
onBeforeUnmount(() => window.runtime.EventsOff("ssh:hostkey"));

function decide(accept: boolean) {
  const p = queue.value.shift();
  if (p) window.go.main.App.AnswerHostKey(p.fingerprint, accept);
}
</script>

<template>
  <div v-if="queue.length" class="modal-backdrop">
    <div class="modal">
      <div class="modal-header">Verify host key</div>
      <div class="modal-body">
        <p style="margin: 0 0 12px">
          The authenticity of host <strong>{{ queue[0].hostname }}</strong> can't be established.
        </p>
        <p style="margin: 0 0 12px">SHA-256 fingerprint:</p>
        <code class="fp">{{ queue[0].fingerprint }}</code>
        <p style="margin: 16px 0 0; color: var(--fg-muted); font-size: 12px">
          Confirm the fingerprint matches the server you trust. Accepting will pin this key in
          <span class="kbd">known_hosts</span>.
        </p>
      </div>
      <div class="modal-footer">
        <button class="ghost" @click="decide(false)">Reject</button>
        <button class="primary" @click="decide(true)">Accept &amp; Pin</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fp {
  display: block;
  background: var(--bg-elev-2);
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  font-family: var(--mono, monospace);
  font-size: 12px;
  word-break: break-all;
}
</style>
