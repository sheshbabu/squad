export default function mergeMessages(existing, incoming) {
  const byId = new Map();

  for (const message of existing) {
    byId.set(message.messageId, message);
  }
  for (const message of incoming) {
    byId.set(message.messageId, message);
  }

  return Array.from(byId.values()).sort((a, b) => a.messageId - b.messageId);
}
