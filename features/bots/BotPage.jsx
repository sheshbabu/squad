import Sidebar from '../../commons/components/Sidebar.jsx';
import MobileNavbar from '../../commons/components/MobileNavbar.jsx';
import EmptyState from '../../commons/components/EmptyState.jsx';
import Spinner from '../../commons/components/Spinner.jsx';
import Button from '../../commons/components/Button.jsx';
import { NewIcon, FileIcon } from '../../commons/components/Icon.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import navigateTo from '../../commons/utils/navigateTo.js';
import { useAppContext } from '../../commons/contexts/AppContext.jsx';
import { LayoutProvider, useLayout } from '../../commons/contexts/LayoutContext.jsx';
import ThreadsList from '../threads/ThreadsList.jsx';
import ChatView from '../threads/ChatView.jsx';
import FilesList from '../files/FilesList.jsx';
import FileView from '../files/FileView.jsx';
import './BotPage.css';

const { useState, useEffect, useCallback } = React;

export default function BotPage({ botId, threadId, fileName }) {
  let parsedThreadId = undefined;
  if (threadId !== undefined) {
    parsedThreadId = parseInt(threadId, 10);
  }

  return (
    <LayoutProvider>
      <BotPageContent botId={parseInt(botId, 10)} threadId={parsedThreadId} fileName={fileName} />
    </LayoutProvider>
  );
}

function BotPageContent({ botId, threadId, fileName }) {
  const { bots, isBotsLoading } = useAppContext();
  const { isChatExpanded } = useLayout();
  const [threads, setThreads] = useState([]);
  const [files, setFiles] = useState([]);

  const path = window.location.pathname;
  const isFilesView = path.startsWith(`/bots/${botId}/files`);
  const isNewChat = path.startsWith(`/bots/${botId}/new`);
  const bot = bots.find(item => item.botId === botId);

  const refreshThreads = useCallback(() => {
    return ApiClient.getThreads(botId).then(setThreads);
  }, [botId]);

  const refreshFiles = useCallback(() => {
    return ApiClient.getFiles(botId).then(setFiles);
  }, [botId]);

  useEffect(() => {
    setThreads([]);
    setFiles([]);
    refreshThreads();
    refreshFiles();
  }, [refreshThreads, refreshFiles]);

  useEffect(() => {
    if (isFilesView) {
      refreshFiles();
    }
  }, [isFilesView, refreshFiles]);

  // Live updates only cover the open chat, so the lists catch up on scheduled runs and chats from other devices when the tab comes back.
  useEffect(() => {
    function handleVisibilityChange() {
      if (document.visibilityState === "visible") {
        refreshThreads();
        refreshFiles();
      }
    }

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
  }, [refreshThreads, refreshFiles]);

  const handleThreadChange = useCallback(changedThread => {
    setThreads(previous => {
      const others = previous.filter(item => item.threadId !== changedThread.threadId);
      return [changedThread, ...others].sort((a, b) => new Date(b.updatedAt) - new Date(a.updatedAt));
    });
  }, []);

  function handleThreadDeleted(deletedThreadId) {
    setThreads(previous => previous.filter(item => item.threadId !== deletedThreadId));
    navigateTo(`/bots/${botId}`);
  }

  function handleFileDeleted() {
    refreshFiles();
    navigateTo(`/bots/${botId}/files`);
  }

  function handleFilesClick() {
    if (isFilesView) {
      navigateTo(`/bots/${botId}`);
    } else {
      navigateTo(`/bots/${botId}/files`);
    }
  }

  function handleNewChatClick() {
    navigateTo(`/bots/${botId}/new`);
  }

  if (bot === undefined) {
    let content = <Spinner />;
    if (!isBotsLoading) {
      content = <EmptyState title="Bot not found" description="It may have been deleted." />;
    }
    return (
      <div className="page-container bot-page">
        <Sidebar currentBotId={botId} />
        <div className="bot-page-missing">{content}</div>
      </div>
    );
  }

  const hasDetail = threadId !== undefined || fileName !== undefined || isNewChat;

  let listContent = <ThreadsList botId={botId} threads={threads} selectedThreadId={threadId} />;
  let detailContent = <ChatView key={threadId ?? "new"} bot={bot} threadId={threadId} onThreadChange={handleThreadChange} onThreadDeleted={handleThreadDeleted} />;

  if (isFilesView) {
    listContent = <FilesList botId={botId} files={files} selectedFileName={fileName} />;
    detailContent = <EmptyState title="No file selected" description="Pick a file, or ask the bot to write one." />;
    if (fileName !== undefined) {
      detailContent = <FileView key={fileName} botId={botId} fileName={fileName} onSaved={refreshFiles} onDeleted={handleFileDeleted} />;
    }
  }

  let mobileNavbar = null;
  if (!hasDetail) {
    mobileNavbar = <MobileNavbar currentBotId={botId} />;
  }

  let pageClassName = "page-container bot-page";
  if (isChatExpanded && !isFilesView) {
    pageClassName += " is-chat-expanded";
  }

  return (
    <div className={pageClassName}>
      <Sidebar currentBotId={botId} />

      <div className={`bot-page-list ${hasDetail ? "has-detail" : ""}`}>
        <div className="bot-page-header">
          <div className="bot-page-title">
            <span className="bot-page-name">{bot.name}</span>
            <Button variant="ghost" className={isFilesView ? "is-active" : ""} onClick={handleFilesClick} title="Files"><FileIcon /></Button>
            <Button variant="ghost" onClick={handleNewChatClick} title="New chat"><NewIcon /></Button>
          </div>
        </div>
        {listContent}
      </div>

      <div className={`bot-page-detail ${hasDetail ? "has-detail" : ""}`}>
        {detailContent}
      </div>

      {mobileNavbar}
    </div>
  );
}
