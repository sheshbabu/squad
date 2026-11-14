// Zen runs on the same localhost origin during development, so the key is prefixed to keep the two preferences apart.
const THEME_PREFERENCE_KEY = 'squad-theme-preference';
const DEFAULT_THEME = 'system';
const DARK_BG_COLOR = "#121212";
const LIGHT_BG_COLOR = "#FFF";

function getPreference() {
  try {
    return localStorage.getItem(THEME_PREFERENCE_KEY) || DEFAULT_THEME;
  } catch {
    return DEFAULT_THEME;
  }
}

function setPreference(themeId) {
  try {
    localStorage.setItem(THEME_PREFERENCE_KEY, themeId);
  } catch {
  }
}

function applyTheme() {
  const themeId = getPreference();

  let isDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  if (themeId === 'system') {
    document.documentElement.removeAttribute('data-theme');
  } else {
    document.documentElement.setAttribute('data-theme', themeId);
    isDark = themeId === 'dark';
  }

  const themeColor = isDark ? DARK_BG_COLOR : LIGHT_BG_COLOR;
  document.querySelector("meta[name=theme-color]").setAttribute("content", themeColor);
  document.querySelector("meta[name=background-color]").setAttribute("content", themeColor);
}

export default {
  getPreference,
  setPreference,
  applyTheme
};
