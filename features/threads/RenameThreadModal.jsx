import { ModalBackdrop, ModalContainer, ModalHeader, ModalContent, ModalFooter, closeModal } from '../../commons/components/Modal.jsx';
import Input from '../../commons/components/Input.jsx';
import Button from '../../commons/components/Button.jsx';
import ApiClient from '../../commons/http/ApiClient.js';

const { useState } = React;

export default function RenameThreadModal({ thread, onRenamed }) {
  const [title, setTitle] = useState(thread.title);
  const [isSaving, setIsSaving] = useState(false);

  function handleTitleChange(event) {
    setTitle(event.target.value);
  }

  function handleSubmit(event) {
    event.preventDefault();
    setIsSaving(true);
    ApiClient.renameThread(thread.threadId, title.trim())
      .then(updated => {
        closeModal();
        onRenamed(updated);
      })
      .finally(() => setIsSaving(false));
  }

  return (
    <ModalBackdrop>
      <ModalContainer>
        <form onSubmit={handleSubmit}>
          <ModalHeader title="Rename chat" />
          <ModalContent>
            <Input id="thread-title" label="Title" value={title} onChange={handleTitleChange} />
          </ModalContent>
          <ModalFooter isRightAligned>
            <Button onClick={closeModal}>Cancel</Button>
            <Button type="submit" variant="primary" isDisabled={title.trim() === "" || isSaving}>Save</Button>
          </ModalFooter>
        </form>
      </ModalContainer>
    </ModalBackdrop>
  );
}
