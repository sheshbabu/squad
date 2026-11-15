import { SendIcon, StopIcon, PlusIcon, CloseIcon } from '../../commons/components/Icon.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import isMobile from '../../commons/utils/isMobile.js';
import { isModalOpen } from '../../commons/components/Modal.jsx';
import './Composer.css';

const { useState, useRef, useEffect, useLayoutEffect } = React;

const ACCEPTED_TYPES_BY_MODALITY = {
  image: ["image/png", "image/jpeg", "image/webp", "image/gif"],
  file: ["application/pdf"],
};

export default function Composer({ botId, inputModalities, isNewChat, isWorking, isDisabled, onSend, onStop }) {
  const [content, setContent] = useState("");
  const [attachments, setAttachments] = useState([]);
  const [isUploading, setIsUploading] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const fileInputRef = useRef(null);
  const textareaRef = useRef(null);
  const wasWorkingRef = useRef(isWorking);

  useLayoutEffect(() => {
    const textarea = textareaRef.current;
    textarea.style.height = "auto";
    textarea.style.height = `${textarea.scrollHeight}px`;
  }, [content]);

  useEffect(() => {
    if (isNewChat) {
      focusInput();
    }
  }, []);

  useEffect(() => {
    if (wasWorkingRef.current && !isWorking) {
      focusInput();
    }
    wasWorkingRef.current = isWorking;
  }, [isWorking]);

  const hasInput = content.trim() !== "" || attachments.length > 0;
  const canSend = hasInput && !isDisabled && !isWorking && !isUploading && !isSending;

  // Phones would pop the keyboard up unasked, and focus is never taken from an open modal or another field being typed in.
  function focusInput() {
    if (isMobile() || isModalOpen()) {
      return;
    }

    const activeElement = document.activeElement;
    const isTypingElsewhere = activeElement !== null && activeElement !== textareaRef.current && activeElement.matches("input, textarea, [contenteditable]");
    if (isTypingElsewhere) {
      return;
    }

    textareaRef.current.focus();
  }

  function handleContentChange(event) {
    setContent(event.target.value);
  }

  // On phones Enter adds a new line, since there is no Shift key to hold.
  function handleKeyDown(event) {
    if (event.key === "Enter" && !event.shiftKey && !isMobile()) {
      event.preventDefault();
      handleSendClick();
    }
  }

  function handleSendClick() {
    if (!canSend) {
      return;
    }

    setIsSending(true);
    onSend(content.trim(), attachments.map(attachment => attachment.attachmentId))
      .then(() => {
        setContent("");
        setAttachments([]);
      })
      .finally(() => setIsSending(false));
  }

  function handleAttachClick() {
    fileInputRef.current.click();
  }

  function handleFilesSelected(event) {
    const files = Array.from(event.target.files);
    event.target.value = "";

    setIsUploading(true);
    Promise.all(files.map(file => ApiClient.uploadAttachment(botId, file)))
      .then(uploaded => setAttachments(previous => [...previous, ...uploaded]))
      .finally(() => setIsUploading(false));
  }

  function handleRemoveAttachment(attachmentId) {
    setAttachments(previous => previous.filter(attachment => attachment.attachmentId !== attachmentId));
  }

  const attachmentChips = attachments.map(attachment => (
    <div key={attachment.attachmentId} className="composer-attachment">
      <span className="composer-attachment-name">{attachment.filename}</span>
      <CloseIcon onClick={() => handleRemoveAttachment(attachment.attachmentId)} />
    </div>
  ));

  let attachmentsElement = null;
  if (attachmentChips.length > 0 || isUploading) {
    let uploadingElement = null;
    if (isUploading) {
      uploadingElement = <div className="composer-attachment is-uploading">Uploading…</div>;
    }
    attachmentsElement = <div className="composer-attachments">{attachmentChips}{uploadingElement}</div>;
  }

  let actionButton = <button type="button" className="composer-button is-primary" onClick={handleSendClick} disabled={!canSend} title="Send"><SendIcon /></button>;
  if (isWorking) {
    actionButton = <button type="button" className="composer-button is-primary" onClick={onStop} disabled={isDisabled} title="Stop"><StopIcon /></button>;
  }

  let supportedModalities = Object.keys(ACCEPTED_TYPES_BY_MODALITY);
  if (inputModalities !== null) {
    supportedModalities = supportedModalities.filter(modality => inputModalities.includes(modality));
  }
  const acceptedTypes = supportedModalities.flatMap(modality => ACCEPTED_TYPES_BY_MODALITY[modality]);
  const canAttach = acceptedTypes.length > 0;

  let attachTitle = "Attach files";
  if (!canAttach) {
    attachTitle = "This model only reads text";
  }

  let placeholder = "Message";
  if (isDisabled) {
    placeholder = "Offline";
  }

  return (
    <div className="composer">
      <div className="composer-box">
        {attachmentsElement}
        <div className="composer-row">
          <button type="button" className="composer-button" onClick={handleAttachClick} disabled={isDisabled || !canAttach} title={attachTitle}><PlusIcon /></button>
          <input type="file" ref={fileInputRef} className="composer-file-input" accept={acceptedTypes.join(",")} multiple onChange={handleFilesSelected} />
          <textarea
            ref={textareaRef}
            className="composer-input"
            placeholder={placeholder}
            value={content}
            rows={1}
            disabled={isDisabled}
            onChange={handleContentChange}
            onKeyDown={handleKeyDown} />
          {actionButton}
        </div>
      </div>
    </div>
  );
}
