// Callers render these segments as text rather than HTML, so markup in a message never runs.
export function splitHighlights(text) {
  if (text === undefined) {
    return [];
  }

  const segments = [];
  const parts = text.split(/<mark>(.*?)<\/mark>/s);
  for (let index = 0; index < parts.length; index++) {
    if (parts[index] === '') {
      continue;
    }
    segments.push({ text: parts[index], isMatch: index % 2 === 1 });
  }
  return segments;
}
