import { defineStore } from "pinia";
import { ref, computed } from "vue";
import type { HostRecord } from "../wails.d";

export interface TabSession {
  id: string;
  hostId: string;
  hostName: string;
  status: "connecting" | "open" | "closed" | "error";
  error?: string;
  showSftp: boolean;
}

export interface Pane {
  id: string;
  activeTabId: string | null;
  tabIds: string[];
}

let counter = 0;
function uid(prefix = "s"): string {
  counter += 1;
  return `${prefix}_${Date.now().toString(36)}_${counter}`;
}

export const useSessions = defineStore("sessions", () => {
  const tabs = ref<Record<string, TabSession>>({});
  const panes = ref<Pane[]>([{ id: uid("p"), activeTabId: null, tabIds: [] }]);
  const activePaneId = ref<string>(panes.value[0].id);

  // SFTP shared state per tab
  const sftpCwd = ref<Record<string, string>>({});
  const sftpRefreshTick = ref<Record<string, number>>({});
  const reconnectTick = ref<Record<string, number>>({});

  const activePane = computed(() => panes.value.find((p) => p.id === activePaneId.value)!);

  function openInActivePane(host: HostRecord) {
    const tabId = uid("t");
    tabs.value[tabId] = {
      id: tabId,
      hostId: host.id,
      hostName: host.name || host.address,
      status: "connecting",
      showSftp: false,
    };
    activePane.value.tabIds.push(tabId);
    activePane.value.activeTabId = tabId;
    return tabId;
  }

  function setActiveTab(paneId: string, tabId: string) {
    const p = panes.value.find((x) => x.id === paneId);
    if (p) p.activeTabId = tabId;
    activePaneId.value = paneId;
  }

  function setTabStatus(tabId: string, status: TabSession["status"], err?: string) {
    const t = tabs.value[tabId];
    if (t) {
      t.status = status;
      if (err !== undefined) t.error = err;
    }
  }

  function toggleSftp(tabId: string) {
    const t = tabs.value[tabId];
    if (t) t.showSftp = !t.showSftp;
  }

  function closeTab(tabId: string) {
    for (const p of panes.value) {
      const i = p.tabIds.indexOf(tabId);
      if (i >= 0) {
        p.tabIds.splice(i, 1);
        if (p.activeTabId === tabId) {
          p.activeTabId = p.tabIds[Math.min(i, p.tabIds.length - 1)] || null;
        }
      }
    }
    delete tabs.value[tabId];
    delete sftpCwd.value[tabId];
    delete sftpRefreshTick.value[tabId];
    delete reconnectTick.value[tabId];
    if (panes.value.length > 1) {
      panes.value = panes.value.filter((p) => p.tabIds.length > 0);
      if (!panes.value.find((p) => p.id === activePaneId.value)) {
        activePaneId.value = panes.value[0].id;
      }
    }
  }

  function splitRight() {
    if (panes.value.length >= 4) return;
    const p: Pane = { id: uid("p"), activeTabId: null, tabIds: [] };
    panes.value.push(p);
    activePaneId.value = p.id;
  }

  function closePane(paneId: string) {
    if (panes.value.length <= 1) return;
    const p = panes.value.find((x) => x.id === paneId);
    if (!p) return;
    for (const tid of [...p.tabIds]) closeTab(tid);
    panes.value = panes.value.filter((x) => x.id !== paneId);
    activePaneId.value = panes.value[0].id;
  }

  function bumpSftpRefresh(tabId: string) {
    sftpRefreshTick.value[tabId] = (sftpRefreshTick.value[tabId] || 0) + 1;
  }

  function bumpReconnect(tabId: string) {
    reconnectTick.value[tabId] = (reconnectTick.value[tabId] || 0) + 1;
  }

  function setSftpCwd(tabId: string, cwd: string) {
    sftpCwd.value[tabId] = cwd;
  }

  return {
    tabs,
    panes,
    activePaneId,
    activePane,
    sftpCwd,
    sftpRefreshTick,
    reconnectTick,
    openInActivePane,
    setActiveTab,
    setTabStatus,
    toggleSftp,
    closeTab,
    splitRight,
    closePane,
    bumpSftpRefresh,
    bumpReconnect,
    setSftpCwd,
  };
});
