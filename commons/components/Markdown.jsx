import renderMarkdown from '../utils/renderMarkdown.js';
import './Markdown.css';

const { useMemo } = React;

export default function Markdown({ text }) {
  const html = useMemo(() => renderMarkdown(text), [text]);

  return <div className="markdown" dangerouslySetInnerHTML={{ __html: html }} />;
}
