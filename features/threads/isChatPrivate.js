// Earlier replies count as well as the bot's current setting, since a chat that used a retaining provider has already sent its history there.
// Replies without a provider were routed by the gateway to any host, so they can't count as private.
export default function isChatPrivate(bot, replyEndpoints, messages, zdrEndpoints) {
  const zdrKeys = new Set(zdrEndpoints.map(endpoint => getKey(endpoint.modelId, endpoint.providerId)));

  const usedKeys = [getKey(bot.model, bot.provider)];
  for (const endpoint of replyEndpoints) {
    usedKeys.push(getKey(endpoint.modelId, endpoint.providerId));
  }
  for (const message of messages) {
    if (message.role === "assistant" && message.model !== "") {
      usedKeys.push(getKey(message.model, message.provider));
    }
  }

  return usedKeys.every(key => zdrKeys.has(key));
}

function getKey(modelId, providerId) {
  return `${modelId}|${providerId}`;
}
