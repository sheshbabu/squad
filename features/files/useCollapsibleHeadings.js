import FoldedHeadingsPreferences from "../../commons/preferences/FoldedHeadingsPreferences.js";

const { useLayoutEffect } = React;

const MAX_FOLDABLE_LEVEL = 3;

export default function useCollapsibleHeadings(contentRef, documentKey) {
  // Runs after every render because the rendered HTML is replaced whenever content changes or the view remounts.
  // A layout effect applies folds before paint, so folded sections don't flash open.
  useLayoutEffect(() => {
    applyFolds(contentRef.current, documentKey);
  });

  function handleHeadingClick(event) {
    const container = contentRef.current;
    const heading = event.target.closest("h1, h2, h3");
    if (container === null || heading === null || heading.parentElement !== container) {
      return false;
    }

    if (event.target.closest("a") !== null || !window.getSelection().isCollapsed) {
      return false;
    }

    const headingKey = getHeadingKey(heading);
    const currentHeadingKeys = getFoldableHeadings(container).map(getHeadingKey);
    // Dropping keys for headings that no longer exist keeps renamed headings from lingering in storage.
    const foldedHeadingKeys = FoldedHeadingsPreferences.getFoldedHeadings(documentKey).filter(key => key !== headingKey && currentHeadingKeys.includes(key));
    if (!heading.classList.contains("is-folded")) {
      foldedHeadingKeys.push(headingKey);
    }

    FoldedHeadingsPreferences.setFoldedHeadings(documentKey, foldedHeadingKeys);
    applyFolds(container, documentKey);
    return true;
  }

  return { handleHeadingClick };
}

function applyFolds(container, documentKey) {
  if (container === null) {
    return;
  }

  const foldedHeadingKeys = FoldedHeadingsPreferences.getFoldedHeadings(documentKey);
  let foldLevel = null;

  for (const element of container.children) {
    const level = getHeadingLevel(element);
    if (level !== null && foldLevel !== null && level <= foldLevel) {
      foldLevel = null;
    }

    element.hidden = foldLevel !== null;

    const isFolded = foldLevel === null && level !== null && level <= MAX_FOLDABLE_LEVEL && foldedHeadingKeys.includes(getHeadingKey(element));
    element.classList.toggle("is-folded", isFolded);
    if (isFolded) {
      foldLevel = level;
    }
  }
}

function getFoldableHeadings(container) {
  return Array.from(container.children).filter(element => {
    const level = getHeadingLevel(element);
    return level !== null && level <= MAX_FOLDABLE_LEVEL;
  });
}

function getHeadingLevel(element) {
  const match = element.tagName.match(/^H([1-6])$/);
  return match === null ? null : parseInt(match[1], 10);
}

function getHeadingKey(heading) {
  return `${heading.tagName}:${heading.textContent.trim()}`;
}
