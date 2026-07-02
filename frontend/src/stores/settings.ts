import { defineStore } from "pinia";
import { ref } from "vue";
import type { AppSettings } from "../wails.d";

const defaults: AppSettings = {
  theme: "system",
  fontFamily: `"JetBrains Mono","Cascadia Code","Fira Code",Consolas,monospace`,
  fontSize: 14,
  cursorStyle: "bar",
  cursorBlink: true,
  scrollBack: 5000,
  confirmCloseWithActiveSessions: true,
  showCommandBar: true,
  connectTimeoutSec: 15,
  keepAliveEnabled: true,
  keepAliveIntervalSec: 30,
};

export const useSettings = defineStore("settings", () => {
  const settings = ref<AppSettings>({ ...defaults });
  const loaded = ref(false);

  async function load() {
    try {
      const s = await window.go.main.App.GetSettings();
      settings.value = { ...defaults, ...s };
    } catch {
      settings.value = { ...defaults };
    }
    loaded.value = true;
  }

  async function save(s: AppSettings) {
    settings.value = s;
    await window.go.main.App.SaveSettings(s);
  }

  // Font size bounds shared with the Settings dialog and Ctrl +/-/0 shortcuts.
  const FONT_MIN = 8;
  const FONT_MAX = 32;
  const FONT_DEFAULT = 14;
  function clampFontSize(n: number): number {
    const v = Math.round(Number(n) || FONT_DEFAULT);
    return Math.max(FONT_MIN, Math.min(FONT_MAX, v));
  }
  async function setFontSize(n: number) {
    const size = clampFontSize(n);
    if (size === settings.value.fontSize) return;
    await save({ ...settings.value, fontSize: size });
  }
  async function bumpFontSize(delta: number) {
    await setFontSize(settings.value.fontSize + delta);
  }
  async function resetFontSize() {
    await setFontSize(FONT_DEFAULT);
  }

  return {
    settings,
    loaded,
    load,
    save,
    FONT_MIN,
    FONT_MAX,
    setFontSize,
    bumpFontSize,
    resetFontSize,
  };
});
