import { test } from "node:test";
import assert from "node:assert/strict";
import splitHighlights from "./highlights.js";

const text = value => ({ text: value, isMatch: false });
const match = value => ({ text: value, isMatch: true });

test("splitHighlights", () => {
  const cases = [
    [undefined, []],
    ["", []],
    ["plain title", [text("plain title")]],
    ["a <mark>match</mark> here", [text("a "), match("match"), text(" here")]],
    ["<mark>one</mark> and <mark>two</mark>", [match("one"), text(" and "), match("two")]],
    ["<mark>x</mark> <img src=x onerror=alert(1)>", [match("x"), text(" <img src=x onerror=alert(1)>")]],
    ["<mark>multi\nline</mark>", [match("multi\nline")]],
  ];

  for (const [input, expected] of cases) {
    assert.deepEqual(splitHighlights(input), expected, `input: ${JSON.stringify(input)}`);
  }
});
