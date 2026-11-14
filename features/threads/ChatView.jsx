import Button from '../../commons/components/Button.jsx';
import Markdown from '../../commons/components/Markdown.jsx';
import Spinner from '../../commons/components/Spinner.jsx';
import EmptyState from '../../commons/components/EmptyState.jsx';
import { openModal } from '../../commons/components/Modal.jsx';
import BotAvatar from '../../commons/components/BotAvatar.jsx';
import { BackIcon, RetryIcon, MaximizeIcon, MinimizeIcon, ShieldCheckIcon, ShieldXIcon } from '../../commons/components/Icon.jsx';
import { useLayout } from '../../commons/contexts/LayoutContext.jsx';
import isMobile from '../../commons/utils/isMobile.js';
import isChatPrivate from './isChatPrivate.js';
import { getRunningLabel } from './toolLabels.js';
import ApiClient from '../../commons/http/ApiClient.js';
import navigateTo from '../../commons/utils/navigateTo.js';
import useIsOnline from '../../commons/utils/useIsOnline.js';
import MessageItem, { ToolActivity } from './MessageItem.jsx';
import Composer from './Composer.jsx';
import ThreadMenu from './ThreadMenu.jsx';
import RenameThreadModal from './RenameThreadModal.jsx';
import ThreadDetailsModal from './ThreadDetailsModal.jsx';
import DeleteThreadModal from './DeleteThreadModal.jsx';
import mergeMessages from './mergeMessages.js';
import useThreadStream from './useThreadStream.js';
import './ChatView.css';

const { useState, useEffect, useRef, useLayoutEffect } = React;

export default function ChatView({ bot, threadId, onThreadChange, onThreadDeleted }) {
  const { isChatExpanded, toggleChatExpanded } = useLayout();
  const [thread, setThread] = useState(null);
  const [messages, setMessages] = useState([]);
  const [hasMore, setHasMore] = useState(false);
  const [partialText, setPartialText] = useState("");
  const [isLoading, setIsLoading] = useState(threadId !== undefined);
  const [isLoadingEarlier, setIsLoadingEarlier] = useState(false);
  const [replyEndpoints, setReplyEndpoints] = useState([]);
  const [zdrEndpoints, setZDREndpoints] = useState(null);
  const [inputModalities, setInputModalities] = useState(null);
  const isOnline = useIsOnline();

  const scrollRef = useRef(null);
  const isFirstLoadRef = useRef(true);
  const shouldStickToBottomRef = useRef(true);
  const scrollHeightBeforeEarlierRef = useRef(null);

  function loadLatestTurns() {
    return ApiClient.getThread(threadId)
      .then(response => {
        setThread(response.thread);
        setMessages(previous => mergeMessages(previous, response.messages));
        setReplyEndpoints(response.replyEndpoints);

        // Only the first load knows about older turns, later refetches only catch up on recent ones.
        if (isFirstLoadRef.current) {
          setHasMore(response.hasMore);
          isFirstLoadRef.current = false;
        }
      })
      .finally(() => setIsLoading(false));
  }

  useEffect(() => {
    if (threadId !== undefined) {
      loadLatestTurns();
    }
  }, [threadId]);

  useEffect(() => {
    ApiClient.getZDREndpoints(bot.gateway).then(setZDREndpoints);
  }, [bot.gateway]);

  useEffect(() => {
    ApiClient.getModels(bot.gateway).then(models => {
      const model = models.find(item => item.modelId === bot.model);
      if (model !== undefined) {
        setInputModalities(model.inputModalities);
      }
    });
  }, [bot.gateway, bot.model]);

  useEffect(() => {
    function handleKeyDown(event) {
      if ((event.metaKey || event.ctrlKey) && event.key === "\\" && !isMobile()) {
        event.preventDefault();
        toggleChatExpanded();
      }
    }

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [toggleChatExpanded]);

  useEffect(() => {
    if (thread !== null) {
      onThreadChange(thread);
    }
  }, [thread, onThreadChange]);

  useThreadStream(threadId, {
    onOpen: loadLatestTurns,
    onDelta: data => setPartialText(previous => previous + data.text),
    onReset: () => setPartialText(""),
    onMessage: message => {
      setMessages(previous => mergeMessages(previous, [message]));
      if (message.role === "assistant") {
        setPartialText("");
      }
    },
    onStatus: data => setThread(previous => updateLoadedThread(previous, { status: data.status, errorMessage: data.errorMessage, updatedAt: new Date().toISOString() })),
    onThread: updated => setThread(previous => updateLoadedThread(previous, { title: updated.title })),
  });

  useLayoutEffect(() => {
    const container = scrollRef.current;
    if (container === null) {
      return;
    }

    // Loading earlier turns grows the list upwards, so the view is shifted to keep the same message on screen.
    if (scrollHeightBeforeEarlierRef.current !== null) {
      container.scrollTop += container.scrollHeight - scrollHeightBeforeEarlierRef.current;
      scrollHeightBeforeEarlierRef.current = null;
      return;
    }

    if (shouldStickToBottomRef.current) {
      container.scrollTop = container.scrollHeight;
    }
  }, [messages, partialText, thread?.status]);

  function handleScroll() {
    const container = scrollRef.current;
    shouldStickToBottomRef.current = container.scrollHeight - container.scrollTop - container.clientHeight < 80;
  }

  function handleLoadEarlierClick() {
    setIsLoadingEarlier(true);
    ApiClient.getEarlierMessages(threadId, messages[0].messageId)
      .then(response => {
        scrollHeightBeforeEarlierRef.current = scrollRef.current.scrollHeight;
        setMessages(previous => mergeMessages(response.messages, previous));
        setHasMore(response.hasMore);
      })
      .finally(() => setIsLoadingEarlier(false));
  }

  function handleSend(content, attachmentIds) {
    shouldStickToBottomRef.current = true;

    if (threadId === undefined) {
      return ApiClient.createThread(bot.botId, { content, attachmentIds })
        .then(createdThread => {
          onThreadChange(createdThread);
          navigateTo(`/bots/${bot.botId}/threads/${createdThread.threadId}`, false, true);
        });
    }

    return ApiClient.sendMessage(threadId, { content, attachmentIds })
      .then(message => {
        setMessages(previous => mergeMessages(previous, [message]));
        setThread(previous => ({ ...previous, status: "working", errorMessage: "" }));
      });
  }

  function handleStop() {
    ApiClient.cancelThread(threadId);
  }

  function handleRetryClick() {
    ApiClient.retryThread(threadId)
      .then(() => setThread(previous => ({ ...previous, status: "working", errorMessage: "" })));
  }

  function handleRenameClick() {
    openModal(<RenameThreadModal thread={thread} onRenamed={updated => setThread(previous => updateLoadedThread(previous, { title: updated.title }))} />);
  }

  function handleDetailsClick() {
    openModal(<ThreadDetailsModal thread={thread} bot={bot} />);
  }

  function handleDeleteClick() {
    openModal(<DeleteThreadModal thread={thread} onDeleted={onThreadDeleted} />);
  }

  function handleExpandToggleClick() {
    toggleChatExpanded();
  }

  function handleBackClick() {
    navigateTo(`/bots/${bot.botId}`);
  }

  if (isLoading) {
    return <div className="chat-view is-loading"><Spinner /></div>;
  }

  const isWorking = thread !== null && thread.status === "working";

  let privacyBadge = null;
  if (zdrEndpoints !== null) {
    privacyBadge = (
      <span className="chat-view-privacy color-red" title="Replies may come from a provider that stores prompts. Pin a zero data retention provider in the bot's Model settings.">
        <ShieldXIcon />Not Private
      </span>
    );
    if (isChatPrivate(bot, replyEndpoints, messages, zdrEndpoints)) {
      privacyBadge = (
        <span className="chat-view-privacy" title="Every reply in this chat comes from a provider with zero data retention, which doesn't store prompts.">
          <ShieldCheckIcon />Private
        </span>
      );
    }
  }

  let expandButton = null;
  if (!isMobile()) {
    let expandIcon = <MaximizeIcon />;
    if (isChatExpanded) {
      expandIcon = <MinimizeIcon />;
    }
    expandButton = <Button variant="ghost" onClick={handleExpandToggleClick} title="Expand (Cmd+\)">{expandIcon}</Button>;
  }

  let heading = <div className="chat-view-title">New chat</div>;
  if (thread !== null) {
    heading = <ThreadMenu title={thread.title} onRenameClick={handleRenameClick} onDetailsClick={handleDetailsClick} onDeleteClick={handleDeleteClick} />;
  }

  let loadEarlierButton = null;
  if (hasMore) {
    loadEarlierButton = (
      <div className="chat-view-load-earlier">
        <Button variant="primary" onClick={handleLoadEarlierClick} isDisabled={isLoadingEarlier}>Load 10 more</Button>
      </div>
    );
  }

  const toolResultsById = {};
  for (const message of messages) {
    if (message.role === "tool") {
      toolResultsById[message.toolCallId] = message;
    }
  }

  // Each agent step is its own assistant message, so tool calls from consecutive steps are merged into one activity line.
  const chatItems = [];
  for (const message of messages) {
    if (message.role === "tool") {
      continue;
    }

    const hasText = message.role === "user" || message.content !== "" || message.isCancelled;
    if (hasText) {
      chatItems.push({ key: message.messageId, message: message });
    }

    if (message.role === "assistant" && message.toolCalls.length > 0) {
      const lastItem = chatItems[chatItems.length - 1];
      if (lastItem !== undefined && lastItem.toolCalls !== undefined) {
        lastItem.toolCalls.push(...message.toolCalls);
      } else {
        chatItems.push({ key: `activity-${message.messageId}`, toolCalls: [...message.toolCalls] });
      }
    }
  }

  const messageItems = chatItems.map(item => {
    if (item.toolCalls !== undefined) {
      return <ToolActivity key={item.key} toolCalls={item.toolCalls} toolResultsById={toolResultsById} />;
    }
    return <MessageItem key={item.key} message={item.message} />;
  });

  let emptyState = null;
  if (thread === null) {
    emptyState = <EmptyState icon={<span className={`color-${bot.color}`}><BotAvatar name={bot.icon} /></span>} title={`Chat with ${bot.name}`} />;
  }

  let progressElement = null;
  if (partialText !== "") {
    progressElement = (
      <div className="chat-view-partial">
        <Markdown text={partialText} />
      </div>
    );
  } else if (isWorking) {
    const lastToolCalls = messages.findLast(message => message.role === "assistant" && message.toolCalls.length > 0)?.toolCalls ?? [];
    const runningToolCall = lastToolCalls.find(toolCall => toolResultsById[toolCall.id] === undefined);

    // An error message on a working thread is the reason a retry is waiting.
    let statusText = "Thinking";
    if (thread.errorMessage !== "") {
      statusText = thread.errorMessage;
    } else if (runningToolCall !== undefined) {
      statusText = getRunningLabel(runningToolCall);
    }
    progressElement = <div className="chat-view-thinking"><span className="chat-view-thinking-text">{statusText}</span></div>;
  }

  let errorElement = null;
  if (thread !== null && thread.status === "failed") {
    errorElement = (
      <div className="chat-view-error">
        <div className="chat-view-error-message">{thread.errorMessage || "The reply failed."}</div>
        <Button onClick={handleRetryClick} isDisabled={!isOnline}><RetryIcon />Retry</Button>
      </div>
    );
  }

  return (
    <div className="chat-view">
      <div className="chat-view-header">
        <Button variant="ghost" className="chat-view-back" onClick={handleBackClick}><BackIcon /></Button>
        <div className="chat-view-heading">{heading}</div>
        {privacyBadge}
        {expandButton}
      </div>

      <div className="chat-view-messages" ref={scrollRef} onScroll={handleScroll}>
        <div className="chat-view-column">
          {loadEarlierButton}
          {emptyState}
          {messageItems}
          {progressElement}
          {errorElement}
        </div>
      </div>

      <Composer botId={bot.botId} inputModalities={inputModalities} isNewChat={threadId === undefined} isWorking={isWorking} isDisabled={!isOnline} onSend={handleSend} onStop={handleStop} />
    </div>
  );
}

// An event can arrive before the first load finishes, and that load already includes its effect.
function updateLoadedThread(thread, changes) {
  if (thread === null) {
    return thread;
  }
  return { ...thread, ...changes };
}
