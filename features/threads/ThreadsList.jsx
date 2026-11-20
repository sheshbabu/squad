import Link from '../../commons/components/Link.jsx';
import formatDate from '../../commons/utils/formatDate.js';
import './ThreadsList.css';

export default function ThreadsList({ threads, selectedThreadId, basePath, emptyText, bots }) {
  if (threads.length === 0) {
    return <div className="threads-list is-empty">{emptyText}</div>;
  }

  const items = threads.map(thread => {
    let unreadDot = null;
    if (thread.isUnread) {
      unreadDot = <span className="threads-list-unread" title="Unread" />;
    }

    return (
      <Link
        key={thread.threadId}
        className="threads-list-item"
        activeClassName="is-active"
        isActive={thread.threadId === selectedThreadId}
        to={`${basePath}/${thread.threadId}`}>
        <div className="threads-list-title">
          <span className="threads-list-title-text">{thread.title}</span>
          {unreadDot}
        </div>
        <div className="threads-list-subtext">
          {renderLabel(thread, bots)}
          <span className="threads-list-time">{formatDate(new Date(thread.updatedAt))}</span>
        </div>
      </Link>
    );
  });

  return <div className="threads-list">{items}</div>;
}

function renderLabel(thread, bots) {
  if (bots !== undefined) {
    const bot = bots.find(item => item.botId === thread.botId);
    if (bot === undefined) {
      return null;
    }
    return <span className={`threads-list-label bot color-${bot.color}`}>{bot.name}</span>;
  }

  if (thread.status === "working") {
    return <span className="threads-list-label is-working">Replying</span>;
  }
  if (thread.status === "failed") {
    return <span className="threads-list-label is-failed">Failed</span>;
  }
  if (thread.scheduleId !== null) {
    return <span className="threads-list-label">Scheduled</span>;
  }
  return null;
}
