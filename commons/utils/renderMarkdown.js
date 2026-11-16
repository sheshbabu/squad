import mark from "../../assets/markdown-it-mark.mjs";
import tasks from "../../assets/markdown-it-task-lists.js";
import { isClipboardAvailable } from "./clipboard.js";

const COPY_ICON_SVG = '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-copy code-copy-icon"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>';

const CHECK_ICON_SVG = '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-check code-copied-icon"><path d="M20 6 9 17l-5-5"/></svg>';

// markdown-it escapes raw HTML by default, which is what keeps model output from injecting markup.
// The task list plugin keeps its options in module state, so a fresh instance is built per render.
export default function renderMarkdown(text, { hasCodeCopyButton = false, hasClickableTasks = false } = {}) {
  const markdown = window.markdownit({
    linkify: true,
    breaks: true,
    highlight: function (code, language) {
      if (language && window.hljs.getLanguage(language)) {
        try {
          return window.hljs.highlight(code, { language: language }).value;
        } catch (error) {
          console.error("Error highlighting code", error);
        }
      }
      return '';
    }
  })
    .use(mark)
    .use(tasks, { enabled: hasClickableTasks });

  // Bots mention file names like meal-plan.md, and .md is a real top-level domain, so only links with a scheme or www are made clickable.
  markdown.linkify.set({ fuzzyLink: false });

  const defaultLinkRender = markdown.renderer.rules.link_open || function (tokens, index, options, env, self) {
    return self.renderToken(tokens, index, options);
  };
  markdown.renderer.rules.link_open = function (tokens, index, options, env, self) {
    tokens[index].attrSet('target', '_blank');
    tokens[index].attrSet('rel', 'noopener noreferrer');
    return defaultLinkRender(tokens, index, options, env, self);
  };

  // A table can't scroll on its own without squeezing its columns, so a wrapper scrolls it instead.
  markdown.renderer.rules.table_open = function (tokens, index, options, env, self) {
    return `<div class="table-block">${self.renderToken(tokens, index, options)}`;
  };
  markdown.renderer.rules.table_close = function (tokens, index, options, env, self) {
    return `${self.renderToken(tokens, index, options)}</div>`;
  };

  if (hasCodeCopyButton && isClipboardAvailable()) {
    const defaultFenceRender = markdown.renderer.rules.fence || function (tokens, index, options, env, self) {
      return self.renderToken(tokens, index, options);
    };
    markdown.renderer.rules.fence = function (tokens, index, options, env, self) {
      const fenceHtml = defaultFenceRender(tokens, index, options, env, self);
      const button = `<button type="button" class="code-copy-button">${COPY_ICON_SVG}${CHECK_ICON_SVG}</button>`;
      return `<div class="code-block">${button}${fenceHtml}</div>`;
    };
  }

  return markdown.render(text);
}
