import Button from '../../commons/components/Button.jsx';
import Spinner from '../../commons/components/Spinner.jsx';
import { TrashIcon } from '../../commons/components/Icon.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import './MemoriesPane.css';

const { useState, useEffect } = React;

export default function MemoriesPane({ botId }) {
  const [memories, setMemories] = useState([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    ApiClient.getMemories(botId)
      .then(setMemories)
      .finally(() => setIsLoading(false));
  }, [botId]);

  function handleDeleteClick(memoryId) {
    ApiClient.deleteMemory(botId, memoryId)
      .then(() => setMemories(previous => previous.filter(memory => memory.memoryId !== memoryId)));
  }

  if (isLoading) {
    return <Spinner />;
  }

  if (memories.length === 0) {
    return <div className="memories-pane is-empty">No memories yet. Ask the bot to remember something and it will show up here.</div>;
  }

  const items = memories.map(memory => (
    <div key={memory.memoryId} className="memories-pane-item">
      <div className="memories-pane-content">{memory.content}</div>
      <Button variant="ghost" onClick={() => handleDeleteClick(memory.memoryId)} title="Forget"><TrashIcon /></Button>
    </div>
  ));

  return <div className="memories-pane">{items}</div>;
}
