import { test } from 'node:test';
import assert from 'node:assert/strict';
import isChatPrivate from './isChatPrivate.js';

const zdrEndpoints = [{ modelId: 'anthropic/claude-sonnet-5.5', providerId: 'amazon-bedrock' }];
const privateBot = { model: 'anthropic/claude-sonnet-5.5', provider: 'amazon-bedrock' };

test('a new chat follows the bot', () => {
  assert.equal(isChatPrivate(privateBot, [], [], zdrEndpoints), true);
  assert.equal(isChatPrivate({ ...privateBot, provider: 'anthropic' }, [], [], zdrEndpoints), false);
});

test('any provider is not private, since the gateway picks the host', () => {
  assert.equal(isChatPrivate({ ...privateBot, provider: '' }, [], [], zdrEndpoints), false);
});

test('an earlier reply from another provider makes the chat not private', () => {
  const replyEndpoints = [{ modelId: 'anthropic/claude-sonnet-5.5', providerId: 'anthropic' }];
  assert.equal(isChatPrivate(privateBot, replyEndpoints, [], zdrEndpoints), false);
});

test('a live reply counts too, while user messages are ignored', () => {
  const messages = [
    { role: 'user', model: '', provider: '' },
    { role: 'assistant', model: 'anthropic/claude-sonnet-5.5', provider: '' },
  ];
  assert.equal(isChatPrivate(privateBot, [], messages.slice(0, 1), zdrEndpoints), true);
  assert.equal(isChatPrivate(privateBot, [], messages, zdrEndpoints), false);
});
