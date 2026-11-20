import { ModalBackdrop, ModalContainer, ModalHeader, ModalContent, ModalFooter, closeModal } from './Modal.jsx';
import Button from './Button.jsx';

export default function ConfirmModal({ title, description, confirmLabel, onConfirm }) {
  function handleConfirmClick() {
    closeModal();
    onConfirm();
  }

  return (
    <ModalBackdrop>
      <ModalContainer>
        <ModalHeader title={title} />
        <ModalContent>
          <p className="modal-description">{description}</p>
        </ModalContent>
        <ModalFooter isRightAligned>
          <Button onClick={closeModal}>Cancel</Button>
          <Button variant="danger" onClick={handleConfirmClick}>{confirmLabel}</Button>
        </ModalFooter>
      </ModalContainer>
    </ModalBackdrop>
  );
}
