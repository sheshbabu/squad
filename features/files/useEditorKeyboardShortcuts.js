const { useEffect } = React;

export default function useEditorKeyboardShortcuts({ isEditing, textareaRef, onSave, onEdit, onReplaceContent, onInsertAtCursor, onFormatText }) {
  function replaceLine(textarea, lineStart, newLine, newCursorPos) {
    const lineEnd = textarea.value.indexOf('\n', lineStart);
    const textAfter = lineEnd === -1 ? "" : textarea.value.substring(lineEnd);
    onReplaceContent(textarea.value.substring(0, lineStart) + newLine + textAfter, newCursorPos);
  }

  function handleKeyDown(event) {
    const textarea = textareaRef.current;
    const isTextAreaFocused = textarea !== null && document.activeElement === textarea;

    if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
      event.preventDefault();
      if (isEditing) {
        onSave();
      } else {
        onEdit();
      }
      return;
    }

    if (!isTextAreaFocused) {
      return;
    }

    if (event.key === 'Tab') {
      const cursorPos = textarea.selectionStart;
      const lineStart = textarea.value.lastIndexOf('\n', cursorPos - 1) + 1;
      const lineEnd = textarea.value.indexOf('\n', cursorPos);
      const currentLine = textarea.value.substring(lineStart, lineEnd === -1 ? textarea.value.length : lineEnd);

      if (isListLine(currentLine)) {
        event.preventDefault();
        const indentation = currentLine.match(/^\s*/)[0];

        if (event.shiftKey) {
          const removedCount = Math.min(indentation.length, 2);
          if (removedCount === 0) {
            return;
          }
          const newCursorPos = Math.max(lineStart, cursorPos - removedCount);
          replaceLine(textarea, lineStart, currentLine.substring(removedCount), newCursorPos);
        } else {
          replaceLine(textarea, lineStart, '  ' + currentLine, cursorPos + 2);
        }
        return;
      }

      if (!event.shiftKey) {
        event.preventDefault();
        onInsertAtCursor('  ');
      }
      return;
    }

    if ((event.metaKey || event.ctrlKey) && event.shiftKey && event.key.toLowerCase() === 'h') {
      event.preventDefault();
      onFormatText("highlight");
      return;
    }

    if ((event.metaKey || event.ctrlKey) && event.key === 'b') {
      event.preventDefault();
      onFormatText("bold");
      return;
    }

    if ((event.metaKey || event.ctrlKey) && event.key === 'i') {
      event.preventDefault();
      onFormatText("italic");
      return;
    }

    if (event.key === 'Enter' && !event.metaKey && !event.ctrlKey && !event.shiftKey) {
      const cursorPos = textarea.selectionStart;
      const textBeforeCursor = textarea.value.substring(0, cursorPos);
      const lines = textBeforeCursor.split('\n');
      const currentLine = lines[lines.length - 1];

      for (const pattern of listPatterns) {
        const match = currentLine.match(pattern);
        if (match === null) {
          continue;
        }

        event.preventDefault();
        const indentation = match[1];
        let prefix = match[2];
        const lineContent = currentLine.substring(match[0].length);

        // Pressing Enter on an empty list item ends the list, as in most editors.
        if (lineContent.trim() === "") {
          const lineStart = textBeforeCursor.length - currentLine.length;
          const newContent = textarea.value.substring(0, lineStart) + "\n" + textarea.value.substring(cursorPos);
          onReplaceContent(newContent, lineStart + 1);
          return;
        }

        if (prefix === "- [x] ") {
          prefix = "- [ ] ";
        } else if (/^\d+\. $/.test(prefix)) {
          const number = parseInt(prefix.match(/^(\d+)/)[1], 10) + 1;
          prefix = `${number}. `;
        }

        onInsertAtCursor(`\n${indentation}${prefix}`);
        return;
      }
    }
  }

  // The listener is re-attached each render so it always sees the latest isEditing and callbacks.
  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  });
}

const listPatterns = [
  /^(\s*)(- \[ \] )/,
  /^(\s*)(- \[x\] )/,
  /^(\s*)(- )/,
  /^(\s*)(\* )/,
  /^(\s*)(\+ )/,
  /^(\s*)(\d+\. )/,
];

function isListLine(line) {
  return listPatterns.some(pattern => pattern.test(line));
}
