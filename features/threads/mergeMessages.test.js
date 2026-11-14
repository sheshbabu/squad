import { test } from 'node:test';
import assert from 'node:assert/strict';
import mergeMessages from './mergeMessages.js';

test('merges pages and live messages without duplicates, oldest first', () => {
  const earlier = [{ messageId: 1, content: 'a' }, { messageId: 2, content: 'b' }];
  const refetched = [{ messageId: 2, content: 'b' }, { messageId: 4, content: 'd' }];
  const live = [{ messageId: 3, content: 'c' }];

  const merged = mergeMessages(mergeMessages(earlier, refetched), live);

  assert.deepEqual(merged.map(message => message.messageId), [1, 2, 3, 4]);
});

test('a refetched message replaces the stale copy', () => {
  const merged = mergeMessages([{ messageId: 1, content: 'old' }], [{ messageId: 1, content: 'new' }]);

  assert.equal(merged.length, 1);
  assert.equal(merged[0].content, 'new');
});
