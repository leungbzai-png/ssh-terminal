import { ref, watch } from "vue";
import type { AppSettings } from "../wails.d";

type ThemeMode = AppSettings["theme"];

const mode = ref<ThemeMode>("system");
const resolved = ref<"light" | "dark">("dark");

function applyTheme() {
  const t =
    mode.value === "system"
      ? window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light"
      : mode.value;
  resolved.value = t;
  document.documentElement.setAttribute("data-theme", t);
}

// Module-level singleton: subscribe once for the lifetime of the app.
window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", applyTheme);
watch(mode, applyTheme);
applyTheme();

export function useTheme() {
  return {
    mode,
    resolved,
    setMode(m: ThemeMode) {
      mode.value = m;
    },
  };
}
