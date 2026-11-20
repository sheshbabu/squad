import { ModalBackdrop, ModalContainer, ModalHeader, ModalContent, ModalFooter, closeModal } from '../../commons/components/Modal.jsx';
import Input from '../../commons/components/Input.jsx';
import Button from '../../commons/components/Button.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import './ConnectionModal.css';

const { useState } = React;

export default function ConnectionModal({ botId, group, connection, onSaved, onDeleted }) {
  const isNew = connection === undefined;
  const [name, setName] = useState(isNew ? "" : connection.name);
  const [secret, setSecret] = useState(isNew ? "" : connection.secret);
  const [isSaving, setIsSaving] = useState(false);

  function handleSubmit(event) {
    event.preventDefault();
    setIsSaving(true);

    const payload = { groupId: group.groupId, name: name.trim(), secret: secret.trim() };
    let saveRequest = ApiClient.createConnection(botId, payload);
    if (!isNew) {
      saveRequest = ApiClient.updateConnection(botId, connection.connectionId, payload);
    }

    saveRequest
      .then(saved => {
        closeModal();
        onSaved(saved);
      })
      .finally(() => setIsSaving(false));
  }

  function handleDeleteClick() {
    setIsSaving(true);
    ApiClient.deleteConnection(botId, connection.connectionId)
      .then(() => {
        closeModal();
        onDeleted(connection.connectionId);
      })
      .finally(() => setIsSaving(false));
  }

  let title = `Add ${group.name} connection`;
  if (!isNew) {
    title = `Edit ${group.name} connection`;
  }

  const isSaveDisabled = name.trim() === "" || secret.trim() === "" || isSaving;
  const saveButtons = (
    <div className="connection-modal-buttons">
      <Button onClick={closeModal}>Cancel</Button>
      <Button type="submit" variant="primary" isDisabled={isSaveDisabled}>Save</Button>
    </div>
  );

  let footer = <ModalFooter isRightAligned>{saveButtons}</ModalFooter>;
  if (!isNew) {
    footer = (
      <ModalFooter>
        <Button variant="danger" isDisabled={isSaving} onClick={handleDeleteClick}>Delete</Button>
        {saveButtons}
      </ModalFooter>
    );
  }

  return (
    <ModalBackdrop>
      <ModalContainer className="connection-modal">
        <form onSubmit={handleSubmit}>
          <ModalHeader title={title} />
          <ModalContent>
            <p className="modal-description">{group.connection.hint}</p>
            <Input id="connection-name" label="Name" hint="The bot picks a connection by this name when it has more than one." placeholder="Shesh" value={name} onChange={event => setName(event.target.value)} />
            <Input id="connection-secret" label={group.connection.secretLabel} type="password" value={secret} onChange={event => setSecret(event.target.value)} />
          </ModalContent>
          {footer}
        </form>
      </ModalContainer>
    </ModalBackdrop>
  );
}
