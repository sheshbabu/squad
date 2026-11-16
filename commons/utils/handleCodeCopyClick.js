import { COPIED_DURATION, copyText } from "./clipboard.js";

export default function handleCodeCopyClick(event) {
  const button = event.target.closest('.code-copy-button');
  if (button === null) {
    return false;
  }

  const code = button.closest('.code-block')?.querySelector('pre code');
  if (code == null) {
    return false;
  }

  copyText(code.textContent, "Couldn't copy code.", () => showCopiedState(button));

  return true;
}

function showCopiedState(button) {
  if (button.dataset.copiedTimeoutId != null) {
    clearTimeout(parseInt(button.dataset.copiedTimeoutId, 10));
  }

  button.classList.add("is-copied");

  const timeoutId = setTimeout(() => {
    button.classList.remove("is-copied");
    delete button.dataset.copiedTimeoutId;
  }, COPIED_DURATION);

  button.dataset.copiedTimeoutId = timeoutId;
}
