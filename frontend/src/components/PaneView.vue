<script setup lang="ts">
import { computed } from "vue";
import { useSessions, type Pane } from "../stores/sessions";
import { useSettings } from "../stores/settings";
import TabBar from "./TabBar.vue";
import Terminal from "./Terminal.vue";
import SftpPanel from "./SftpPanel.vue";
import CommandBar from "./CommandBar.vue";

const props = defineProps<{ pane: Pane }>();
const emit = defineEmits<{ (e: "activate"): void; (e: "close-pane"): void }>();

const sessions = useSessions();
const settings = useSettings();

const activeTab = computed(() => {
  const id = props.pane.activeTabId;
  return id ? sessions.tabs[id] : null;
});
</script>

<template>
  <section class="pane" @mousedown="emit('activate')">
    <TabBar :pane="pane" @close-pane="emit('close-pane')" />
    <div class="pane-body">
      <template v-if="activeTab">
        <div class="terminal-area">
          <div class="split" :data-split="activeTab.showSftp ? 'on' : 'off'">
            <div class="term-col">
              <Terminal :tab-id="activeTab.id" />
              <CommandBar
                v-if="settings.settings.showCommandBar && activeTab.status === 'open'"
                :tab-id="activeTab.id"
                :pane-id="pane.id"
              />
            </div>
            <SftpPanel v-if="activeTab.showSftp" :tab-id="activeTab.id" :pane-id="pane.id" />
          </div>
        </div>
      </template>
      <div v-else class="empty">
        <div class="empty-card">
          <h3>No session</h3>
          <p>Double-click a host in the sidebar to connect.</p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.pane {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  background: var(--bg);
}
.pane-body {
  flex: 1;
  min-height: 0;
  position: relative;
}
.terminal-area {
  height: 100%;
}
.split {
  display: grid;
  height: 100%;
  width: 100%;
}
.split[data-split="off"] {
  grid-template-columns: 1fr;
}
.split[data-split="on"] {
  grid-template-columns: 1fr 360px;
}
.term-col {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.empty {
  height: 100%;
  display: grid;
  place-items: center;
  color: var(--fg-muted);
}
.empty-card {
  text-align: center;
  padding: 32px;
}
.empty-card h3 {
  margin: 0 0 6px;
  font-weight: 600;
}
.empty-card p {
  margin: 0;
  font-size: 12px;
}
</style>
