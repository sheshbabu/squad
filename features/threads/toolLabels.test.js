import { test } from 'node:test';
import assert from 'node:assert/strict';
import { getRunningLabel, getDoneLabel, getSummaryLabel } from './toolLabels.js';

function toolCall(name, args) {
  return { function: { name, arguments: JSON.stringify(args) } };
}

test('running labels describe the call', () => {
  assert.equal(getRunningLabel(toolCall('hackernews_search', { query: 'rust' })), 'Searching Hacker News for "rust"');
  assert.equal(getRunningLabel(toolCall('fetch_url', { url: 'https://www.nytimes.com/a/b' })), 'Reading www.nytimes.com');
  assert.equal(getRunningLabel(toolCall('read_file', { name: 'meal-plan.md' })), 'Reading meal-plan.md');
});

test('unknown tools and broken arguments still get a label', () => {
  assert.equal(getRunningLabel(toolCall('new_tool', {})), 'Using new tool');
  assert.equal(getRunningLabel({ function: { name: 'fetch_url', arguments: '{' } }), 'Reading a page');
});

test('done labels split the action from its detail', () => {
  assert.deepEqual(getDoneLabel(toolCall('web_search', { query: 'gold rate' })), { label: 'Searched the web', detail: 'gold rate' });
  assert.deepEqual(getDoneLabel(toolCall('list_files', {})), { label: 'Listed files', detail: '' });
  assert.deepEqual(getDoneLabel(toolCall('new_tool', {})), { label: 'Used new tool', detail: '' });
});

test('summaries count the calls', () => {
  assert.equal(getSummaryLabel([toolCall('read_file', { name: 'a.md' })]), 'Used a tool');
  assert.equal(getSummaryLabel([toolCall('read_file', { name: 'a.md' }), toolCall('read_file', { name: 'b.md' })]), 'Used 2 tools');
});
