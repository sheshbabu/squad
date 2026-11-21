const TOOL_LABELS = {
  save_memory: { group: "memory", running: () => "Saving a memory", done: "Saved a memory", detail: args => args.content },
  delete_memory: { group: "memory", running: () => "Forgetting a memory", done: "Forgot a memory" },
  list_files: { group: "files", running: () => "Listing files", done: "Listed files" },
  read_file: { group: "files", running: args => `Reading ${args.name}`, done: "Read", detail: args => args.name },
  write_file: { group: "files", running: args => `Writing ${args.name}`, done: "Wrote", detail: args => args.name },
  edit_file: { group: "files", running: args => `Editing ${args.name}`, done: "Edited", detail: args => args.name },
  delete_file: { group: "files", running: args => `Deleting ${args.name}`, done: "Deleted", detail: args => args.name },
  search_chats: { group: "chats", running: args => `Searching past chats for "${args.query}"`, done: "Searched past chats", detail: args => args.query },
  fetch_url: { group: "web", running: args => `Reading ${getHostname(args.url)}`, done: "Read", detail: args => args.url },
  web_search: { group: "web", running: args => `Searching the web for "${args.query}"`, done: "Searched the web", detail: args => args.query },
  hackernews_search: { group: "hackernews", running: args => `Searching Hacker News for "${args.query}"`, done: "Searched Hacker News", detail: args => args.query },
  hackernews_listing: { group: "hackernews", running: () => "Checking Hacker News", done: "Checked Hacker News" },
  hackernews_comments: { group: "hackernews", running: () => "Reading Hacker News comments", done: "Read Hacker News comments" },
  reddit_search: { group: "reddit", running: args => `Searching Reddit for "${args.query}"`, done: "Searched Reddit", detail: args => args.query },
  reddit_listing: { group: "reddit", running: args => `Checking r/${args.subreddits}`, done: "Checked Reddit", detail: args => `r/${args.subreddits}` },
  reddit_comments: { group: "reddit", running: () => "Reading Reddit comments", done: "Read Reddit comments" },
  zen_search_notes: { group: "zen", running: args => `Searching Zen for "${args.query}"`, done: "Searched Zen", detail: args => args.query },
  zen_get_note: { group: "zen", running: () => "Reading a Zen note", done: "Read a Zen note" },
  zen_list_tags: { group: "zen", running: () => "Checking Zen tags", done: "Checked Zen tags" },
  zen_create_note: { group: "zen", running: args => `Creating "${args.title}" in Zen`, done: "Created a Zen note", detail: args => args.title },
  zen_update_note: { group: "zen", running: () => "Updating a Zen note", done: "Updated a Zen note" },
  calendar_events: { group: "calendar", running: args => `Checking the calendar from ${args.from} to ${args.to}`, done: "Checked the calendar", detail: args => `${args.from} to ${args.to}` },
};

export function getRunningLabel(toolCall) {
  const labels = TOOL_LABELS[toolCall.function.name];
  if (labels === undefined) {
    return `Using ${humanize(toolCall.function.name)}`;
  }
  return labels.running(parseArguments(toolCall.function.arguments));
}

export function getDoneLabel(toolCall) {
  const labels = TOOL_LABELS[toolCall.function.name];
  if (labels === undefined) {
    return { label: `Used ${humanize(toolCall.function.name)}`, detail: "" };
  }
  if (labels.detail === undefined) {
    return { label: labels.done, detail: "" };
  }
  return { label: labels.done, detail: labels.detail(parseArguments(toolCall.function.arguments)) ?? "" };
}

export function getToolGroup(toolCall) {
  return TOOL_LABELS[toolCall.function.name]?.group ?? "";
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
