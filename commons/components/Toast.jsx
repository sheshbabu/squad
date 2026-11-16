import { CloseIcon } from "./Icon.jsx";
import "./Toast.css";

const { useState, useEffect } = React;

let toastRoot = null;

export default function showToast(message, duration = 3000) {
  if (toastRoot === null) {
    toastRoot = ReactDOM.createRoot(document.querySelector('.toast-root'));
  }
  toastRoot.render(<Toast message={message} duration={duration} key={Date.now()} />);
}

function Toast({ message, duration }) {
  const [isOpen, setIsOpen] = useState(true);

  useEffect(() => {
    setIsOpen(true);
    if (duration > 0) {
      const timer = setTimeout(() => setIsOpen(false), duration);
      return () => clearTimeout(timer);
    }
  }, [message, duration]);

  function handleCloseClick() {
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
      <div className="toast-close" onClick={handleCloseClick}>
        <CloseIcon />
      </div>
    </div>
  );
}
