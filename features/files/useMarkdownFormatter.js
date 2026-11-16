const FORMATS = {
  bold: { prefix: "**", suffix: "**" },
  italic: { prefix: "*", suffix: "*" },
  strikethrough: { prefix: "~~", suffix: "~~" },
  highlight: { prefix: "==", suffix: "==" },
  code: { prefix: "`", suffix: "`" },
  h1: { prefix: "# ", suffix: "" },
  h2: { prefix: "## ", suffix: "" },
  h3: { prefix: "### ", suffix: "" },
  ul: { prefix: "- ", suffix: "" },
  ol: { prefix: "1. ", suffix: "" },
  todo: { prefix: "- [ ] ", suffix: "" },
  quote: { prefix: "> ", suffix: "" },
};

export default function useMarkdownFormatter({ textareaRef, setContent }) {
  // React re-renders the textarea after setContent, so the cursor is placed in a timeout once the new value is in the DOM.
  function replaceContent(newContent, newPosition) {
    setContent(newContent);
    setTimeout(() => {
      if (textareaRef.current !== null) {
        textareaRef.current.selectionStart = newPosition;
        textareaRef.current.selectionEnd = newPosition;
        textareaRef.current.focus();
      }
    }, 0);
  }

  function insertAtCursor(text) {
    if (textareaRef.current === null) {
      return;
    }

    const textarea = textareaRef.current;
    const startPos = textarea.selectionStart;
    const beforeText = textarea.value.substring(0, startPos);
    const afterText = textarea.value.substring(textarea.selectionEnd);
    replaceContent(beforeText + text + afterText, startPos + text.length);
  }

  function applyMarkdownFormat(format, placeholder = "") {
    if (textareaRef.current === null) {
      return;
    }

    const textarea = textareaRef.current;
    const startPos = textarea.selectionStart;
    const endPos = textarea.selectionEnd;
    const beforeText = textarea.value.substring(0, startPos);
    const afterText = textarea.value.substring(endPos);
    const selectedText = textarea.value.substring(startPos, endPos);

    let formattedText = "";
    let cursorOffset = 0;

    if (format === "hr") {
      formattedText = "\n---\n";
      cursorOffset = formattedText.length;
    } else if (format === "link") {
      formattedText = `[${selectedText || placeholder}](url)`;
      // A selection becomes the link text, so the cursor goes to the url; otherwise it goes to the placeholder text.
      cursorOffset = selectedText ? formattedText.length - 4 : 1;
    } else {
      const { prefix, suffix } = FORMATS[format];
      formattedText = `${prefix}${selectedText || placeholder}${suffix}`;
      cursorOffset = selectedText ? formattedText.length : prefix.length;
    }

    replaceContent(beforeText + formattedText + afterText, startPos + cursorOffset);
  }

  return { replaceContent, insertAtCursor, applyMarkdownFormat };
}
