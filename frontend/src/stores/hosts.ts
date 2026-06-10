import { defineStore } from "pinia";
import { ref } from "vue";
import type { HostRecord } from "../wails.d";

export const useHosts = defineStore("hosts", () => {
  const hosts = ref<HostRecord[]>([]);
  const loading = ref(false);

  async function refresh() {
    loading.value = true;
    try {
      hosts.value = (await window.go.main.App.ListHosts()) || [];
    } finally {
      loading.value = false;
    }
  }

  async function upsert(h: HostRecord) {
    await window.go.main.App.UpsertHost(h);
    await refresh();
  }

  async function remove(id: string) {
    await window.go.main.App.DeleteHost(id);
    await refresh();
  }

  return { hosts, loading, refresh, upsert, remove };
});
