import showToast from "../components/Toast.jsx";

export const COPIED_DURATION = 2000;

// The clipboard API only exists on secure origins, so copy buttons are hidden over plain HTTP.
export function isClipboardAvailable() {
  return typeof navigator.clipboard !== 'undefined';
}

export function copyText(text, failureMessage, onCopied) {
  navigator.clipboard.writeText(text)
    .then(onCopied)
    .catch(() => showToast(failureMessage));
}
