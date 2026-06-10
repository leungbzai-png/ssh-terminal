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

  return { settings, loaded, load, save };
});
