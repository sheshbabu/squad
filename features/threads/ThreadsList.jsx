import Link from '../../commons/components/Link.jsx';
import formatDate from '../../commons/utils/formatDate.js';
import './ThreadsList.css';

export default function ThreadsList({ botId, threads, selectedThreadId }) {
  if (threads.length === 0) {
    return <div className="threads-list is-empty">No chats yet</div>;
  }

  const items = threads.map(thread => {
    let statusElement = null;
    if (thread.status === "working") {
      statusElement = <span className="threads-list-status is-working">Replying</span>;
    } else if (thread.status === "failed") {
      statusElement = <span className="threads-list-status is-failed">Failed</span>;
    } else if (thread.scheduleId !== null) {
      statusElement = <span className="threads-list-status">Scheduled</span>;
    }

    return (
      <Link
        key={thread.threadId}
        className="threads-list-item"
        activeClassName="is-active"
        isActive={thread.threadId === selectedThreadId}
        to={`/bots/${botId}/threads/${thread.threadId}`}>
        <div className="threads-list-title">{thread.title}</div>
        <div className="threads-list-subtext">
          <span>{formatDate(new Date(thread.updatedAt))}</span>
          {statusElement}
        </div>
      </Link>
    );
  });

  return <div className="threads-list">{items}</div>;
}
