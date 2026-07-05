import { ref } from "vue";

// useWorkspaceLayout holds the user-adjustable widths of the workspace side
// panels (VPS monitor on the left, SFTP on the right) as a module-level
// singleton, so every pane shares one consistent width and the choice survives
// reloads. Only two non-secret integers (pixel widths) are persisted, in
// localStorage — no paths, hosts, credentials, monitor samples, or listings.
//
// Stored values are the user's *desired* width (clamped to absolute min/max);
// PaneView further scales the rendered width down when the window is too narrow
// so the terminal keeps a usable minimum and the layout never overflows.

export const MON_MIN = 180;
export const MON_MAX = 360;
export const MON_DEFAULT = 240;
export const SFTP_MIN = 360;
export const SFTP_MAX = 900;
export const SFTP_DEFAULT = 460;
export const TERM_MIN = 360;
export const SPLITTER_W = 6;

const K_MON = "ssh-terminal.monitorWidth";
const K_SFTP = "ssh-terminal.sftpWidth";

function clampNum(n: number, lo: number, hi: number): number {
  return Math.min(hi, Math.max(lo, n));
}

function loadWidth(key: string, def: number, lo: number, hi: number): number {
  try {
    const raw = localStorage.getItem(key);
    if (raw == null) return def;
    const n = parseInt(raw, 10);
    return Number.isFinite(n) ? clampNum(n, lo, hi) : def;
  } catch {
    return def;
  }
}

function saveWidth(key: string, v: number) {
  try {
    localStorage.setItem(key, String(Math.round(v)));
  } catch {
    // localStorage may be unavailable (private mode / disabled); layout still
    // works for the session, it just won't persist.
  }
}

const monitorWidth = ref<number>(loadWidth(K_MON, MON_DEFAULT, MON_MIN, MON_MAX));
const sftpWidth = ref<number>(loadWidth(K_SFTP, SFTP_DEFAULT, SFTP_MIN, SFTP_MAX));

function setMonitorWidth(px: number) {
  monitorWidth.value = Math.round(clampNum(px, MON_MIN, MON_MAX));
  saveWidth(K_MON, monitorWidth.value);
}
function setSftpWidth(px: number) {
  sftpWidth.value = Math.round(clampNum(px, SFTP_MIN, SFTP_MAX));
  saveWidth(K_SFTP, sftpWidth.value);
}
function resetMonitorWidth() {
  setMonitorWidth(MON_DEFAULT);
}
function resetSftpWidth() {
  setSftpWidth(SFTP_DEFAULT);
}

export function useWorkspaceLayout() {
  return {
    monitorWidth,
    sftpWidth,
    setMonitorWidth,
    setSftpWidth,
    resetMonitorWidth,
    resetSftpWidth,
  };
}
