import { ModalBackdrop, ModalContainer, ModalHeader, ModalContent } from '../../commons/components/Modal.jsx';
import ApiClient from '../../commons/http/ApiClient.js';
import './ThreadDetailsModal.css';

const { useState, useEffect } = React;

export default function ThreadDetailsModal({ thread, bot }) {
  const [usage, setUsage] = useState(null);

  useEffect(() => {
    ApiClient.getThreadUsage(thread.threadId).then(setUsage);
  }, [thread.threadId]);

  const rows = [
    ["Model", bot.model],
    ["Provider", bot.provider || "Any"],
    ["Gateway", bot.gateway],
    ["Created", new Date(thread.createdAt).toLocaleString()],
    ["Updated", new Date(thread.updatedAt).toLocaleString()],
  ];

  if (usage !== null) {
    rows.push(["Messages", usage.messageCount.toLocaleString()]);
    rows.push(["Input tokens", usage.promptTokens.toLocaleString()]);
    rows.push(["Output tokens", usage.completionTokens.toLocaleString()]);
    rows.push(["Cost", `$${usage.cost.toFixed(4)}`]);
  }

  const items = rows.map(([label, value]) => (
    <div key={label} className="thread-details-row">
      <span>{label}</span>
      <span>{value}</span>
    </div>
  ));

  return (
    <ModalBackdrop>
      <ModalContainer className="thread-details-modal">
        <ModalHeader title="Chat details" />
        <ModalContent>
          <div className="thread-details-rows">{items}</div>
        </ModalContent>
      </ModalContainer>
    </ModalBackdrop>
  );
}
