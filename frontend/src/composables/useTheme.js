import { ref } from "vue";

const STORAGE_KEY = "ash_theme_v1";
const THEMES = ["auto", "light", "dark"];

function readStoredTheme() {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    return THEMES.includes(value) ? value : "auto";
  } catch {
    return "auto";
  }
}

function applyTheme(theme) {
  document.documentElement.dataset.theme = theme;
  const meta = document.querySelector('meta[name="color-scheme"]');
  if (meta) {
    meta.content = theme === "auto" ? "light dark" : theme;
  }
}

export function useTheme() {
  const theme = ref(readStoredTheme());
  applyTheme(theme.value);

  function setTheme(value) {
    if (!THEMES.includes(value)) {
      return;
    }
    theme.value = value;
    try {
      localStorage.setItem(STORAGE_KEY, value);
    } catch {
      // localStorage can be unavailable in private browsing
    }
    applyTheme(value);
  }

  return { theme, setTheme };
}
