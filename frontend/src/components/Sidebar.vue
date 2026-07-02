<script setup lang="ts">
import { computed, ref } from "vue";
import type { HostRecord } from "../wails.d";

const props = defineProps<{ hosts: HostRecord[] }>();
const emit = defineEmits<{
  (e: "new"): void;
  (e: "quick"): void;
  (e: "edit", h: HostRecord): void;
  (e: "open", h: HostRecord): void;
  (e: "delete", id: string): void;
  (e: "settings"): void;
  (e: "keys"): void;
}>();

const query = ref("");

const grouped = computed(() => {
  const q = query.value.trim().toLowerCase();
  const filtered = q
    ? props.hosts.filter(
        (h) =>
          h.name.toLowerCase().includes(q) ||
          h.address.toLowerCase().includes(q) ||
          (h.user || "").toLowerCase().includes(q) ||
          (h.group || "").toLowerCase().includes(q)
      )
    : props.hosts;
  const map = new Map<string, HostRecord[]>();
  for (const h of filtered) {
    const g = h.group?.trim() || "Default";
    if (!map.has(g)) map.set(g, []);
    map.get(g)!.push(h);
  }
  return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0]));
});

const menuFor = ref<string | null>(null);
</script>

<template>
  <aside class="sidebar" @click="menuFor = null">
    <header>
      <div class="brand">
        <span class="dot" />
        <span class="name">SSH Terminal</span>
      </div>
      <div style="display:flex;gap:2px">
        <button class="icon-btn" title="SSH 密钥管理" @click="emit('keys')">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="7.5" cy="15.5" r="3.5" />
            <path d="M21 2l-9.6 9.6" />
            <path d="M15.5 7.5l3 3" />
          </svg>
        </button>
        <button class="icon-btn" title="设置" @click="emit('settings')">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h0a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h0a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v0a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
          </svg>
        </button>
      </div>
    </header>

    <div class="search">
      <input v-model="query" placeholder="Search hosts…" />
    </div>

    <div class="list">
      <div v-for="[group, items] in grouped" :key="group" class="group">
        <div class="group-label">{{ group }}</div>
        <div
          v-for="h in items"
          :key="h.id"
          class="host"
          @dblclick="emit('open', h)"
          @contextmenu.prevent="menuFor = h.id"
        >
          <div class="host-main" @click="emit('open', h)">
            <div class="host-name">{{ h.name || h.address }}</div>
            <div class="host-sub">{{ h.user }}@{{ h.address }}:{{ h.port }}</div>
          </div>
          <button class="icon-btn host-more" @click.stop="menuFor = menuFor === h.id ? null : h.id">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><circle cx="5" cy="12" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="19" cy="12" r="1.6"/></svg>
          </button>
          <div v-if="menuFor === h.id" class="popmenu" @click.stop>
            <button @click="emit('open', h); menuFor = null">Connect</button>
            <button @click="emit('edit', h); menuFor = null">Edit…</button>
            <button class="danger" @click="emit('delete', h.id); menuFor = null">Delete</button>
          </div>
        </div>
      </div>
      <div v-if="props.hosts.length === 0" class="empty">
        <p>No hosts yet.</p>
        <button class="primary" @click="emit('new')">Add host</button>
      </div>
    </div>

    <footer>
      <button class="primary full" @click="emit('quick')">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h7l-1 8 10-12h-7z"/></svg>
        快速连接
      </button>
      <button class="ghost full" @click="emit('new')">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></svg>
        新增主机
      </button>
    </footer>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 248px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg-elev);
  border-right: 1px solid var(--border);
  min-height: 0;
}
header {
  height: 44px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border);
}
.brand {
  display: flex;
  align-items: center;
  gap: 8px;
}
.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 8px color-mix(in oklab, var(--accent) 60%, transparent);
}
.name {
  font-weight: 600;
  letter-spacing: 0.01em;
}
.search {
  padding: 10px 10px 6px;
}
.search input {
  width: 100%;
}
.list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 6px 8px;
}
.group {
  margin-top: 6px;
}
.group-label {
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--fg-subtle);
  padding: 6px 8px 4px;
}
.host {
  position: relative;
  display: flex;
  align-items: center;
  padding: 7px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  gap: 6px;
}
.host:hover {
  background: var(--bg-hover);
}
.host-main {
  flex: 1;
  min-width: 0;
}
.host-name {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.host-sub {
  font-size: 11px;
  color: var(--fg-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.host-more {
  opacity: 0;
}
.host:hover .host-more {
  opacity: 1;
}
.popmenu {
  position: absolute;
  right: 8px;
  top: 32px;
  z-index: 10;
  background: var(--bg-elev-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  padding: 4px;
  display: flex;
  flex-direction: column;
  min-width: 120px;
}
.popmenu button {
  text-align: left;
  padding: 6px 10px;
  border-radius: var(--radius-sm);
}
.empty {
  text-align: center;
  padding: 32px 16px;
  color: var(--fg-muted);
}
.empty p {
  margin: 0 0 12px;
  font-size: 12px;
}
footer {
  padding: 10px;
  border-top: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.full {
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}
</style>
