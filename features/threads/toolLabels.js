const TOOL_LABELS = {
  save_memory: { present: () => "Saving a memory", past: count => plural(count, "Saved a memory", "Saved {n} memories") },
  delete_memory: { present: () => "Forgetting a memory", past: count => plural(count, "Forgot a memory", "Forgot {n} memories") },
  list_files: { present: () => "Listing files", past: () => "Listed files" },
  read_file: { present: args => `Reading ${args.name}`, past: count => plural(count, "Read a file", "Read {n} files") },
  write_file: { present: args => `Writing ${args.name}`, past: count => plural(count, "Wrote a file", "Wrote {n} files") },
  edit_file: { present: args => `Editing ${args.name}`, past: count => plural(count, "Edited a file", "Edited {n} files") },
  delete_file: { present: args => `Deleting ${args.name}`, past: count => plural(count, "Deleted a file", "Deleted {n} files") },
  fetch_url: { present: args => `Reading ${getHostname(args.url)}`, past: count => plural(count, "Read a page", "Read {n} pages") },
  web_search: { present: args => `Searching the web for "${args.query}"`, past: () => "Searched the web" },
  hackernews_search: { present: args => `Searching Hacker News for "${args.query}"`, past: () => "Searched Hacker News" },
  hackernews_listing: { present: () => "Checking Hacker News", past: () => "Checked Hacker News" },
  hackernews_comments: { present: () => "Reading Hacker News comments", past: () => "Read Hacker News comments" },
  reddit_search: { present: args => `Searching Reddit for "${args.query}"`, past: () => "Searched Reddit" },
  reddit_listing: { present: args => `Checking r/${args.subreddits}`, past: () => "Checked Reddit" },
  reddit_comments: { present: () => "Reading Reddit comments", past: () => "Read Reddit comments" },
  zen_search_notes: { present: args => `Searching Zen for "${args.query}"`, past: () => "Searched Zen" },
  zen_get_note: { present: () => "Reading a Zen note", past: count => plural(count, "Read a Zen note", "Read {n} Zen notes") },
  zen_list_tags: { present: () => "Checking Zen tags", past: () => "Checked Zen tags" },
  zen_create_note: { present: args => `Creating "${args.title}" in Zen`, past: count => plural(count, "Created a Zen note", "Created {n} Zen notes") },
  zen_update_note: { present: () => "Updating a Zen note", past: count => plural(count, "Updated a Zen note", "Updated {n} Zen notes") },
};

export function getRunningLabel(toolCall) {
  const label = TOOL_LABELS[toolCall.function.name];
  if (label === undefined) {
    return `Using ${humanize(toolCall.function.name)}`;
  }
  return label.present(parseArguments(toolCall.function.arguments));
}

export function getSummaryLabel(toolCalls) {
  const counts = new Map();
  for (const toolCall of toolCalls) {
    const name = toolCall.function.name;
    counts.set(name, (counts.get(name) ?? 0) + 1);
  }

  const parts = [];
  for (const [name, count] of counts) {
    const label = TOOL_LABELS[name];
    if (label === undefined) {
      parts.push(`Used ${humanize(name)}`);
    } else {
      parts.push(label.past(count));
    }
  }
  return parts.join(" · ");
}

function plural(count, one, many) {
  if (count === 1) {
    return one;
  }
  return many.replace("{n}", count);
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
