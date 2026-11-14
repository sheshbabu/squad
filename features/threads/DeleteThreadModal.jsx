import { ModalBackdrop, ModalContainer, ModalHeader, ModalContent, ModalFooter, closeModal } from '../../commons/components/Modal.jsx';
import Button from '../../commons/components/Button.jsx';
import ApiClient from '../../commons/http/ApiClient.js';

export default function DeleteThreadModal({ thread, onDeleted }) {
  function handleDeleteClick() {
    ApiClient.deleteThread(thread.threadId).then(() => {
      closeModal();
      onDeleted(thread.threadId);
    });
  }

  return (
    <ModalBackdrop>
      <ModalContainer>
        <ModalHeader title="Delete chat" />
        <ModalContent>
          <p className="modal-description">"{thread.title}" and its messages and attachments will be <b>permanently deleted</b>.</p>
        </ModalContent>
        <ModalFooter isRightAligned>
          <Button onClick={closeModal}>Cancel</Button>
          <Button variant="danger" onClick={handleDeleteClick}>Delete</Button>
        </ModalFooter>
      </ModalContainer>
    </ModalBackdrop>
  );
}
