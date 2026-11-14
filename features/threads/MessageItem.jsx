import Markdown from '../../commons/components/Markdown.jsx';
import { ToolIcon, FileIcon, CopyIcon, CheckIcon, ChevronRightIcon } from '../../commons/components/Icon.jsx';
import { showToast } from '../../commons/components/Toast.jsx';
import formatDate from '../../commons/utils/formatDate.js';
import { getSummaryLabel } from './toolLabels.js';
import './MessageItem.css';

const { useState } = React;

export default function MessageItem({ message }) {
  if (message.role === "user") {
    return <UserMessage message={message} />;
  }

  return <AssistantMessage message={message} />;
}

// Consecutive tool calls collapse into one line so a long chain of them doesn't push the reply off screen.
export function ToolActivity({ toolCalls, toolResultsById }) {
  const [isExpanded, setIsExpanded] = useState(false);

  function handleToggleClick() {
    setIsExpanded(previous => !previous);
  }

  const finishedToolCalls = toolCalls.filter(toolCall => toolResultsById[toolCall.id] !== undefined);
  if (finishedToolCalls.length === 0) {
    return null;
  }

  let detailsElement = null;
  if (isExpanded) {
    const items = finishedToolCalls.map(toolCall => <ToolCall key={toolCall.id} toolCall={toolCall} result={toolResultsById[toolCall.id]} />);
    detailsElement = <div className="message-item-activity-details">{items}</div>;
  }

  const classNames = ["message-item", "message-item-activity"];
  if (isExpanded) {
    classNames.push("is-expanded");
  }

  return (
    <div className={classNames.join(" ")}>
      <div className="message-item-activity-summary" onClick={handleToggleClick}>
        <span>{getSummaryLabel(finishedToolCalls)}</span>
        <ChevronRightIcon />
      </div>
      {detailsElement}
    </div>
  );
}

const COPIED_DURATION = 2000;

function UserMessage({ message }) {
  const attachments = message.attachments.map(attachment => <Attachment key={attachment.attachmentId} attachment={attachment} />);

  let attachmentsElement = null;
  if (attachments.length > 0) {
    attachmentsElement = <div className="message-item-attachments">{attachments}</div>;
  }

  let contentElement = null;
  if (message.content !== "") {
    contentElement = <div className="message-item-text">{message.content}</div>;
  }

  return (
    <div className="message-item is-user">
      <div className="message-item-bubble">
        {contentElement}
        {attachmentsElement}
      </div>
      <MessageActions content={message.content} createdAt={message.createdAt} />
    </div>
  );
}

function AssistantMessage({ message }) {
  let contentElement = null;
  if (message.content !== "") {
    contentElement = <Markdown text={message.content} />;
  }

  let stoppedElement = null;
  if (message.isCancelled) {
    stoppedElement = <div className="message-item-stopped">Stopped</div>;
  }

  return (
    <div className="message-item is-assistant">
      {contentElement}
      {stoppedElement}
      <MessageActions content={message.content} createdAt={message.createdAt} />
    </div>
  );
}

// Messages with no text, such as a tool-calling step or an attachment-only message, have nothing to copy, so the row is left out.
function MessageActions({ content, createdAt }) {
  const [isCopied, setIsCopied] = useState(false);

  if (content === "") {
    return null;
  }

  function handleCopyClick() {
    navigator.clipboard.writeText(content)
      .then(() => {
        setIsCopied(true);
        setTimeout(() => setIsCopied(false), COPIED_DURATION);
      })
      .catch(() => showToast("Couldn't copy message."));
  }

  let copyButton = null;
  const isClipboardAvailable = typeof navigator.clipboard !== 'undefined';
  if (isClipboardAvailable) {
    let copyIcon = <CopyIcon />;
    if (isCopied) {
      copyIcon = <CheckIcon />;
    }
    copyButton = <button type="button" className="message-item-action" onClick={handleCopyClick} title="Copy">{copyIcon}</button>;
  }

  return (
    <div className="message-item-actions">
      <span className="message-item-time" title={new Date(createdAt).toLocaleString()}>{formatDate(new Date(createdAt))}</span>
      {copyButton}
    </div>
  );
}

function ToolCall({ toolCall, result }) {
  const [isExpanded, setIsExpanded] = useState(false);

  function handleToggleClick() {
    setIsExpanded(previous => !previous);
  }

  let detailsElement = null;
  if (isExpanded) {
    detailsElement = (
      <div className="message-item-tool-details">
        <pre>{formatArguments(toolCall.function.arguments)}</pre>
        <pre>{result.content}</pre>
      </div>
    );
  }

  return (
    <div className="message-item-tool">
      <div className="message-item-tool-summary" onClick={handleToggleClick}>
        <ToolIcon />
        <span className="message-item-tool-name">{toolCall.function.name}</span>
        <span className="message-item-tool-argument">{getArgumentSummary(toolCall.function.arguments)}</span>
      </div>
      {detailsElement}
    </div>
  );
}

function Attachment({ attachment }) {
  const url = `/api/attachments/${attachment.attachmentId}/`;

  if (attachment.mimeType.startsWith("image/")) {
    return (
      <a href={url} target="_blank" rel="noopener noreferrer" className="message-item-image">
        <img src={url} alt={attachment.filename} />
      </a>
    );
  }

  return (
    <a href={url} target="_blank" rel="noopener noreferrer" className="message-item-file">
      <FileIcon />
      {attachment.filename}
    </a>
  );
}

function getArgumentSummary(argumentsText) {
  try {
    const values = Object.values(JSON.parse(argumentsText)).filter(value => typeof value === "string");
    if (values.length > 0) {
      return values[0];
    }
  } catch (error) {
    // Arguments that aren't valid JSON are shown as they are.
  }
  return argumentsText;
}

function formatArguments(argumentsText) {
  try {
    return JSON.stringify(JSON.parse(argumentsText), null, 2);
  } catch (error) {
    return argumentsText;
  }
}
