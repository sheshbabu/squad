import Sidebar from '../../commons/components/Sidebar.jsx';
import MobileNavbar from '../../commons/components/MobileNavbar.jsx';
import EmptyState from '../../commons/components/EmptyState.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import navigateTo from '../../commons/utils/navigateTo.js';
import { useAppContext } from '../../commons/contexts/AppContext.jsx';
import { LayoutProvider, useLayout } from '../../commons/contexts/LayoutContext.jsx';
import ThreadsList from '../threads/ThreadsList.jsx';
import ChatView from '../threads/ChatView.jsx';
import './InboxPage.css';

const { useState, useEffect, useCallback } = React;

export default function InboxPage({ threadId }) {
  let parsedThreadId = undefined;
  if (threadId !== undefined) {
    parsedThreadId = parseInt(threadId, 10);
  }

  return (
    <LayoutProvider>
      <InboxPageContent threadId={parsedThreadId} />
    </LayoutProvider>
  );
}

function InboxPageContent({ threadId }) {
  const { bots } = useAppContext();
  const { isChatExpanded } = useLayout();
  const [threads, setThreads] = useState([]);

  // Chats read here stay listed until the Inbox is opened again, so the open chat doesn't vanish when the list refreshes.
  const refreshThreads = useCallback(() => {
    return ApiClient.getInbox().then(unreadThreads => {
      setThreads(previous => {
        const unreadThreadIds = unreadThreads.map(item => item.threadId);
        const readThreads = previous
          .filter(item => !unreadThreadIds.includes(item.threadId))
          .map(item => ({ ...item, isUnread: false }));
        return [...unreadThreads, ...readThreads].sort(compareByUpdatedAt);
      });
    });
  }, []);

  useEffect(() => {
    refreshThreads();
  }, [refreshThreads]);

  useEffect(() => {
    function handleVisibilityChange() {
      if (document.visibilityState === "visible") {
        refreshThreads();
      }
    }

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
  }, [refreshThreads]);

  const handleThreadChange = useCallback(changedThread => {
    setThreads(previous => previous.map(item => {
      if (item.threadId === changedThread.threadId) {
        return changedThread;
      }
      return item;
    }).sort(compareByUpdatedAt));
  }, []);

  function handleThreadDeleted(deletedThreadId) {
    setThreads(previous => previous.filter(item => item.threadId !== deletedThreadId));
    navigateTo("/inbox");
  }

  const hasDetail = threadId !== undefined;
  const selectedThread = threads.find(item => item.threadId === threadId);

  let selectedBot = undefined;
  if (selectedThread !== undefined) {
    selectedBot = bots.find(item => item.botId === selectedThread.botId);
  }

  let detailContent = <EmptyState title="No chat selected" description="Pick a chat to read it." />;
  if (selectedThread !== undefined && selectedBot !== undefined) {
    detailContent = <ChatView key={threadId} bot={selectedBot} threadId={threadId} backPath="/inbox" onThreadChange={handleThreadChange} onThreadDeleted={handleThreadDeleted} />;
  }

  let mobileNavbar = null;
  if (!hasDetail) {
    mobileNavbar = <MobileNavbar />;
  }

  let pageClassName = "page-container inbox-page";
  if (isChatExpanded) {
    pageClassName += " is-chat-expanded";
  }

  return (
    <div className={pageClassName}>
      <Sidebar isInboxActive />

      <div className={`inbox-page-list ${hasDetail ? "has-detail" : ""}`}>
        <div className="inbox-page-header">Inbox</div>
        <ThreadsList threads={threads} selectedThreadId={threadId} basePath="/inbox/threads" emptyText="Nothing unread" bots={bots} />
      </div>

      <div className={`inbox-page-detail ${hasDetail ? "has-detail" : ""}`}>
        {detailContent}
      </div>

      {mobileNavbar}
    </div>
  );
}

function compareByUpdatedAt(a, b) {
  return new Date(b.updatedAt) - new Date(a.updatedAt);
}
