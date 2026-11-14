const FOLDED_HEADINGS_PREFIX = 'folded-headings';

function getFoldedHeadings(documentKey) {
  try {
    const value = localStorage.getItem(`${FOLDED_HEADINGS_PREFIX}-${documentKey}`);
    const headingKeys = JSON.parse(value);
    return Array.isArray(headingKeys) ? headingKeys : [];
  } catch {
    return [];
  }
}

function setFoldedHeadings(documentKey, headingKeys) {
  try {
    const key = `${FOLDED_HEADINGS_PREFIX}-${documentKey}`;
    if (headingKeys.length === 0) {
      localStorage.removeItem(key);
      return;
    }
    localStorage.setItem(key, JSON.stringify(headingKeys));
  } catch {
  }
}

export default {
  getFoldedHeadings,
  setFoldedHeadings
};
