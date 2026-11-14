const TOOL_LABELS = {
  save_memory: () => "Saving a memory",
  delete_memory: () => "Forgetting a memory",
  list_files: () => "Listing files",
  read_file: args => `Reading ${args.name}`,
  write_file: args => `Writing ${args.name}`,
  edit_file: args => `Editing ${args.name}`,
  delete_file: args => `Deleting ${args.name}`,
  search_chats: args => `Searching past chats for "${args.query}"`,
  fetch_url: args => `Reading ${getHostname(args.url)}`,
  web_search: args => `Searching the web for "${args.query}"`,
  hackernews_search: args => `Searching Hacker News for "${args.query}"`,
  hackernews_listing: () => "Checking Hacker News",
  hackernews_comments: () => "Reading Hacker News comments",
  reddit_search: args => `Searching Reddit for "${args.query}"`,
  reddit_listing: args => `Checking r/${args.subreddits}`,
  reddit_comments: () => "Reading Reddit comments",
  zen_search_notes: args => `Searching Zen for "${args.query}"`,
  zen_get_note: () => "Reading a Zen note",
  zen_list_tags: () => "Checking Zen tags",
  zen_create_note: args => `Creating "${args.title}" in Zen`,
  zen_update_note: () => "Updating a Zen note",
};

export function getRunningLabel(toolCall) {
  const label = TOOL_LABELS[toolCall.function.name];
  if (label === undefined) {
    return `Using ${humanize(toolCall.function.name)}`;
  }
  return label(parseArguments(toolCall.function.arguments));
}

export function getSummaryLabel(toolCalls) {
  if (toolCalls.length === 1) {
    return "Used a tool";
  }
  return `Used ${toolCalls.length} tools`;
}

function parseArguments(argumentsText) {
  try {
    return JSON.parse(argumentsText);
  } catch {
    return {};
  }
}

function getHostname(url) {
  try {
    return new URL(url).hostname;
  } catch {
    return "a page";
  }
}

function humanize(name) {
  return name.replaceAll("_", " ");
}
