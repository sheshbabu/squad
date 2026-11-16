import Link from '../../commons/components/Link.jsx';
import formatDate from '../../commons/utils/formatDate.js';
import './FilesList.css';

export default function FilesList({ botId, files, selectedFileName }) {
  if (files.length === 0) {
    return <div className="files-list is-empty">No files yet</div>;
  }

  const items = files.map(file => (
    <Link
      key={file.name}
      className="files-list-item"
      activeClassName="is-active"
      isActive={file.name === selectedFileName}
      to={`/bots/${botId}/files/${encodeURIComponent(file.name)}`}>
      <div className="files-list-name">{file.title || file.name}</div>
      <div className="files-list-subtext">
        <span className="files-list-filename">{file.name}</span>
        <span>{formatDate(new Date(file.updatedAt))}</span>
      </div>
    </Link>
  ));

  return <div className="files-list">{items}</div>;
}
