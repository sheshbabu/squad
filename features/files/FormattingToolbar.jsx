import { BoldIcon, ItalicIcon, StrikethroughIcon, HighlightIcon, CodeIcon, Heading1Icon, Heading2Icon, Heading3Icon, ListIcon, ListOrderedIcon, ListTodoIcon, QuoteIcon, LinkIcon, SeparatorIcon, TableIcon, TablePropertiesIcon } from '../../commons/components/Icon.jsx';
import './FormattingToolbar.css';

const GROUPS = [
  [
    { format: "bold", placeholder: "bold text", title: "Bold (Ctrl+B)", icon: <BoldIcon /> },
    { format: "italic", placeholder: "italic text", title: "Italic (Ctrl+I)", icon: <ItalicIcon /> },
    { format: "strikethrough", placeholder: "strikethrough text", title: "Strikethrough", icon: <StrikethroughIcon /> },
    { format: "highlight", placeholder: "highlight text", title: "Highlight (Ctrl+Shift+H)", icon: <HighlightIcon /> },
    { format: "code", placeholder: "code", title: "Inline Code", icon: <CodeIcon /> },
  ],
  [
    { format: "h1", placeholder: "Heading 1", title: "Heading 1", icon: <Heading1Icon /> },
    { format: "h2", placeholder: "Heading 2", title: "Heading 2", icon: <Heading2Icon /> },
    { format: "h3", placeholder: "Heading 3", title: "Heading 3", icon: <Heading3Icon /> },
  ],
  [
    { format: "ul", placeholder: "list item", title: "Bullet List", icon: <ListIcon /> },
    { format: "ol", placeholder: "list item", title: "Numbered List", icon: <ListOrderedIcon /> },
    { format: "todo", placeholder: "todo item", title: "Todo List", icon: <ListTodoIcon /> },
  ],
  [
    { format: "quote", placeholder: "quote text", title: "Quote", icon: <QuoteIcon /> },
    { format: "link", placeholder: "link text", title: "Link", icon: <LinkIcon /> },
    { format: "hr", placeholder: "", title: "Horizontal Rule", icon: <SeparatorIcon /> },
  ],
  [
    { format: "insertTable", placeholder: "", title: "Insert Table", icon: <TableIcon /> },
    { format: "editTable", placeholder: "", title: "Edit Table", icon: <TablePropertiesIcon /> },
  ],
];

export default function FormattingToolbar({ onFormat }) {
  const groups = GROUPS.map((buttons, groupIndex) => {
    const items = buttons.map(button => (
      <button key={button.format} type="button" className="formatting-button" onClick={() => onFormat(button.format, button.placeholder)} title={button.title}>
        {button.icon}
      </button>
    ));
    return <div key={groupIndex} className="formatting-toolbar-group">{items}</div>;
  });

  return <div className="formatting-toolbar">{groups}</div>;
}
