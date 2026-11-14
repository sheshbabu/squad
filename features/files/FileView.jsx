import Button from '../../commons/components/Button.jsx';
import { openModal, closeModal } from '../../commons/components/Modal.jsx';
import Spinner from '../../commons/components/Spinner.jsx';
import { BackIcon } from '../../commons/components/Icon.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import renderMarkdown from '../../commons/utils/renderMarkdown.js';
import handleCodeCopyClick from '../../commons/utils/handleCodeCopyClick.js';
import { toggleTaskAtLine } from '../../commons/utils/toggleTaskLine.js';
import navigateTo from '../../commons/utils/navigateTo.js';
import useIsOnline from '../../commons/utils/useIsOnline.js';
import useCollapsibleHeadings from './useCollapsibleHeadings.js';
import useMarkdownFormatter from './useMarkdownFormatter.js';
import useEditorKeyboardShortcuts from './useEditorKeyboardShortcuts.js';
import FormattingToolbar from './FormattingToolbar.jsx';
import TableEditorModal from './TableEditorModal.jsx';
import './FileView.css';

const { useState, useEffect, useLayoutEffect, useMemo, useRef } = React;

export default function FileView({ botId, fileName, onSaved, onDeleted }) {
  const [content, setContent] = useState("");
  const [draft, setDraft] = useState("");
  const [isLoading, setIsLoading] = useState(true);
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const isOnline = useIsOnline();
  const contentRef = useRef(null);
  const textareaRef = useRef(null);
  const { handleHeadingClick } = useCollapsibleHeadings(contentRef, `${botId}-${fileName}`);
  const html = useMemo(() => renderMarkdown(content, { hasCodeCopyButton: true, hasClickableTasks: isOnline }), [content, isOnline]);

  useEffect(() => {
    ApiClient.getFile(botId, fileName)
      .then(file => setContent(file.content))
      .finally(() => setIsLoading(false));
  }, [botId, fileName]);

  const { replaceContent, insertAtCursor, applyMarkdownFormat } = useMarkdownFormatter({ textareaRef, setContent: setDraft });

  useEditorKeyboardShortcuts({
    isEditing,
    textareaRef,
    onSave: handleSaveClick,
    onEdit: handleEditClick,
    onReplaceContent: replaceContent,
    onInsertAtCursor: insertAtCursor,
    onFormatText: applyMarkdownFormat
  });

  // The textarea grows with its content so the page scrolls as one, like the rendered view.
  useLayoutEffect(() => {
    const textarea = textareaRef.current;
    if (textarea === null) {
      return;
    }
    // scrollHeight excludes the border, so the 2px keeps the last line from scrolling.
    textarea.style.height = `${textarea.scrollHeight + 2}px`;
  }, [draft, isEditing]);

  function handleEditClick() {
    setDraft(content);
    setIsEditing(true);
  }

  function handleCancelClick() {
    setIsEditing(false);
  }

  function handleDraftChange(event) {
    setDraft(event.target.value);
  }

  function handleFormat(format, placeholder) {
    if (format !== "insertTable" && format !== "editTable") {
      applyMarkdownFormat(format, placeholder);
      return;
    }

    const textarea = textareaRef.current;
    const startPos = textarea.selectionStart;
    const endPos = textarea.selectionEnd;
    const selectedText = textarea.value.substring(startPos, endPos);
    const beforeText = textarea.value.substring(0, startPos);
    const afterText = textarea.value.substring(endPos);

    function handleConfirm(tableMarkdown) {
      closeModal();
      setDraft(beforeText + tableMarkdown + afterText);
    }

    openModal(<TableEditorModal isEditing={format === "editTable"} selectedText={selectedText} beforeText={beforeText} afterText={afterText} onConfirm={handleConfirm} onCloseClick={closeModal} />);
  }

  function handleSaveClick() {
    setIsSaving(true);
    ApiClient.saveFile(botId, fileName, draft)
      .then(file => {
        setContent(file.content);
        setIsEditing(false);
        onSaved();
      })
      .finally(() => setIsSaving(false));
  }

  function handleDeleteClick() {
    if (!window.confirm(`Delete ${fileName}?`)) {
      return;
    }
    ApiClient.deleteFile(botId, fileName).then(onDeleted);
  }

  function handleBackClick() {
    navigateTo(`/bots/${botId}/files`);
  }

  function handleTaskCheckboxClick(checkbox) {
    const lineIndex = parseInt(checkbox.getAttribute('data-line'), 10);
    const newContent = toggleTaskAtLine(content, lineIndex);
    if (newContent === null) {
      return;
    }

    const previousContent = content;
    setContent(newContent);
    ApiClient.saveFile(botId, fileName, newContent)
      .then(onSaved)
      .catch(() => setContent(previousContent));
  }

  function handleContentClick(event) {
    if (handleCodeCopyClick(event)) {
      return;
    }

    if (handleHeadingClick(event)) {
      return;
    }

    const checkbox = event.target.closest('.task-list-item.enabled .task-list-item-checkbox[data-line]');
    if (checkbox !== null) {
      handleTaskCheckboxClick(checkbox);
    }
  }

  if (isLoading) {
    return <div className="file-view is-loading"><Spinner /></div>;
  }

  let actions = (
    <>
      <Button variant="ghost" onClick={handleDeleteClick} isDisabled={!isOnline}>Delete</Button>
      <Button variant="ghost" onClick={handleEditClick} isDisabled={!isOnline}>Edit</Button>
    </>
  );

  let body = <div className="file-view-rendered has-foldable-headings" ref={contentRef} dangerouslySetInnerHTML={{ __html: html }} onClick={handleContentClick} />;
  if (content.trim() === "") {
    body = <div className="file-view-empty-text">Empty file</div>;
  }

  if (isEditing) {
    const saveButtonText = isSaving ? "Saving..." : "Save";
    actions = (
      <>
        <Button variant="ghost" onClick={handleCancelClick}>Cancel</Button>
        <Button variant="ghost" onClick={handleSaveClick} isDisabled={isSaving || !isOnline}>{saveButtonText}</Button>
      </>
    );
    body = (
      <>
        <FormattingToolbar onFormat={handleFormat} />
        <textarea className="file-view-textarea" placeholder="Write here..." spellCheck="false" ref={textareaRef} value={draft} onChange={handleDraftChange} autoFocus />
      </>
    );
  }

  return (
    <div className="file-view">
      <div className="file-view-toolbar">
        <div className="file-view-toolbar-left">
          <Button variant="ghost" className="file-view-back" onClick={handleBackClick}><BackIcon /></Button>
        </div>
        <div className="file-view-toolbar-right">{actions}</div>
      </div>
      <div className="file-view-title">{fileName}</div>
      {body}
    </div>
  );
}
