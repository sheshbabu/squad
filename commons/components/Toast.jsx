import { CloseIcon } from "./Icon.jsx";
import "./Toast.css";

const { useState, useEffect } = React;

let toastRoot = null;

function Toast({ message, duration }) {
  const [isOpen, setIsOpen] = useState(true);

  useEffect(() => {
    setIsOpen(true);
    if (duration > 0) {
      const timer = setTimeout(handleDismiss, duration);
      return () => clearTimeout(timer);
    }
  }, [message, duration]);

  function handleDismiss() {
    setIsOpen(false);
  }

  if (!isOpen) {
    return null;
  }

  return (
    <div className="toast-container">
      <div className="toast-message">
        {message}
      </div>
      <div className="toast-close" onClick={handleDismiss}>
        <CloseIcon />
      </div>
    </div>
  );
}

export function showToast(message, duration = 3000) {
  if (toastRoot === null) {
    toastRoot = ReactDOM.createRoot(document.querySelector('.toast-root'));
  }
  toastRoot.render(<Toast message={message} duration={duration} key={Date.now()} />);
}
