import { CalendarIcon, DatabaseIcon, FileIcon, HistoryIcon, GlobeIcon, NewspaperIcon, NotepadIcon, ToolIcon } from './Icon.jsx';

const GROUP_ICONS = {
  memory: <DatabaseIcon />,
  files: <FileIcon />,
  chats: <HistoryIcon />,
  web: <GlobeIcon />,
  hackernews: <NewspaperIcon />,
  reddit: <NewspaperIcon />,
  zen: <NotepadIcon />,
  calendar: <CalendarIcon />,
};

export default function ToolGroupIcon({ groupId }) {
  return GROUP_ICONS[groupId] ?? <ToolIcon />;
}
