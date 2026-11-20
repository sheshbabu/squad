import { CloseIcon } from './Icon.jsx';
import './Modal.css';

let modalRoot = null;
let modalStack = [];
let nextModalId = 1;

// Modals render into their own root so any component can open one without threading state through the page.
export function openModal(component) {
  modalStack = [...modalStack, { id: nextModalId++, component }];
  renderModals();
}

export function closeModal() {
  modalStack = modalStack.slice(0, -1);
  renderModals();
}

export function closeAllModals() {
  modalStack = [];
  renderModals();
}

export function isModalOpen() {
  return modalStack.length > 0;
}

function renderModals() {
  if (modalRoot === null) {
    modalRoot = ReactDOM.createRoot(document.querySelector('.modal-root'));
  }

  const modals = modalStack.map(modal => <React.Fragment key={modal.id}>{modal.component}</React.Fragment>);
  modalRoot.render(<>{modals}</>);
}

export function ModalBackdrop({ children, onClose = closeModal, isCentered = true }) {
  function handleBackdropClick(e) {
    if (e.target.classList.contains("modal-backdrop-container")) {
      onClose();
    }
  }

  const backdropClasses = `modal-backdrop-container ${isCentered ? 'is-centered' : ''}`;

  return (
    <div className={backdropClasses} onClick={handleBackdropClick}>
      {children}
    </div>
  );
}

export function ModalContainer({ children, className = '' }) {
  const containerClasses = `modal-content-container ${className}`.trim();

  return (
    <div className={containerClasses}>
      {children}
    </div>
  );
}

export function ModalHeader({ title, onClose = closeModal, children }) {
  let titleElement = null;
  if (title !== undefined) {
    titleElement = <h3 className="modal-title">{title}</h3>;
  }

  return (
    <div className="modal-header">
      {titleElement}
      {children}
      <CloseIcon className="modal-close-button" onClick={onClose} />
    </div>
  );
}

export function ModalContent({ children, className = '' }) {
  const contentClasses = `modal-content ${className}`.trim();

  return (
    <div className={contentClasses}>
      {children}
    </div>
  );
}

export function ModalFooter({ children, isRightAligned = false }) {
  const footerClasses = `modal-footer-container ${isRightAligned ? 'right-aligned' : ''}`.trim();

  return (
    <div className={footerClasses}>
      {children}
    </div>
  );
}
