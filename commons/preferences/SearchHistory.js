// Zen runs on the same localhost origin during development, so the key is prefixed to keep the two histories apart.
const SEARCH_HISTORY_KEY = 'squad-search-history';
const MAX_HISTORY_ENTRIES = 5;

function getItems() {
  try {
    const history = JSON.parse(localStorage.getItem(SEARCH_HISTORY_KEY));
    return Array.isArray(history) ? history.filter(item => typeof item?.threadId === "number") : [];
  } catch {
    return [];
  }
}

function saveItem(item) {
  try {
    let history = getItems().filter(entry => entry.threadId !== item.threadId);
    history.unshift(item);
    history = history.slice(0, MAX_HISTORY_ENTRIES);
    localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(history));
  } catch {
  }
}

export default {
  getItems,
  saveItem
};
