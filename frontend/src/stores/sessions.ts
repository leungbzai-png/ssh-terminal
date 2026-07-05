import { defineStore } from "pinia";
import { ref, computed } from "vue";
import type { HostRecord, QuickConnectParams } from "../wails.d";

export interface TabSession {
  id: string;
  hostId: string;
  hostName: string;
  // "idle" = restored saved-host tab, not yet connected (Ready to connect).
  status: "idle" | "connecting" | "open" | "closed" | "error";
  error?: string;
  showSftp: boolean;
  // showMonitor toggles the left VPS monitor sidebar for this tab (v1.2.0).
  // Also acts as the per-tab enable/disable: monitoring only runs while shown.
  showMonitor: boolean;
  // quick marks a Quick Connect tab whose credentials live only in memory
  // (see quickParams) and are never persisted.
  quick?: boolean;
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

  // Quick Connect ephemeral credentials, keyed by tab id. In-memory only —
  // never persisted to hosts.json. Cleared when the tab closes.
  const quickParams = ref<Record<string, QuickConnectParams>>({});

  const activePane = computed(() => panes.value.find((p) => p.id === activePaneId.value)!);

  // Persist the current set of saved-host tabs (non-secret: host id + name).
  // Debounced so rapid open/close bursts collapse into one write. Quick Connect
  // tabs (no hostId) are excluded so their secrets never touch disk.
  let persistTimer: ReturnType<typeof setTimeout> | null = null;
  function schedulePersistTabs() {
    if (persistTimer) clearTimeout(persistTimer);
    persistTimer = setTimeout(() => {
      persistTimer = null;
      const list: { hostId: string; hostName: string }[] = [];
      for (const p of panes.value) {
        for (const id of p.tabIds) {
          const t = tabs.value[id];
          if (t && t.hostId && !t.quick) {
            list.push({ hostId: t.hostId, hostName: t.hostName });
          }
        }
      }
      window.go.main.App.SaveOpenTabs(list).catch(() => {});
    }, 300);
  }

  function openInActivePane(host: HostRecord) {
    const tabId = uid("t");
    tabs.value[tabId] = {
      id: tabId,
      hostId: host.id,
      hostName: host.name || host.address,
      status: "connecting",
      showSftp: false,
      showMonitor: false,
    };
    activePane.value.tabIds.push(tabId);
    activePane.value.activeTabId = tabId;
    schedulePersistTabs();
    return tabId;
  }

  // openSavedTabIdle restores a saved-host tab WITHOUT connecting. The terminal
  // shows a "Ready to connect" prompt; connection starts only on user action.
  function openSavedTabIdle(host: HostRecord) {
    const tabId = uid("t");
    tabs.value[tabId] = {
      id: tabId,
      hostId: host.id,
      hostName: host.name || host.address,
      status: "idle",
      showSftp: false,
      showMonitor: false,
    };
    activePane.value.tabIds.push(tabId);
    if (!activePane.value.activeTabId) activePane.value.activeTabId = tabId;
    schedulePersistTabs();
    return tabId;
  }

  function openQuickInActivePane(params: QuickConnectParams) {
    const tabId = uid("t");
    tabs.value[tabId] = {
      id: tabId,
      hostId: "",
      hostName: params.address,
      status: "connecting",
      showSftp: false,
      showMonitor: false,
      quick: true,
    };
    quickParams.value[tabId] = params;
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

  function toggleMonitor(tabId: string) {
    const t = tabs.value[tabId];
    if (t) t.showMonitor = !t.showMonitor;
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
    // Drop ephemeral Quick Connect credentials so the temp password does not
    // outlive the tab.
    delete quickParams.value[tabId];
    if (panes.value.length > 1) {
      panes.value = panes.value.filter((p) => p.tabIds.length > 0);
      if (!panes.value.find((p) => p.id === activePaneId.value)) {
        activePaneId.value = panes.value[0].id;
      }
    }
    schedulePersistTabs();
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
    quickParams,
    openInActivePane,
    openSavedTabIdle,
    openQuickInActivePane,
    setActiveTab,
    setTabStatus,
    toggleSftp,
    toggleMonitor,
    closeTab,
    splitRight,
    closePane,
    bumpSftpRefresh,
    bumpReconnect,
    setSftpCwd,
  };
});
