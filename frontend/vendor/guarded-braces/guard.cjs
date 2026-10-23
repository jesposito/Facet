"use strict";

// Non-configurable ceilings protect callers even when options disable an
// upstream range limit. Ordinary repository globs are far below these bounds.
const MAX_DEPTH = 64;
const MAX_NODES = 20000;
const MAX_PATTERNS = 1024;
const MAX_RESULTS = 10000;
const MAX_OUTPUT_BYTES = 1000000;

function refuse(reason) {
  const error = new Error(`Guarded braces resource limit: ${reason}`);
  error.code = "ERR_FACET_BRACES_LIMIT";
  throw error;
}

function input(pattern) {
  if (typeof pattern !== "string") return;
  // Match upstream's absolute character ceiling, irrespective of maxLength.
  if (pattern.length > 10000) refuse("input length");
  const stack = [];
  for (let index = 0; index < pattern.length; index++) {
    const char = pattern[index];
    if (char === "\\") {
      index++;
      continue;
    }
    if (char === "[") {
      let brackets = 1;
      while (++index < pattern.length && brackets > 0) {
        if (pattern[index] === "\\") {
          index++;
          continue;
        }
        if (pattern[index] === "[") brackets++;
        if (pattern[index] === "]") brackets--;
      }
      // The matching ] was consumed, not the following pattern character.
      if (brackets === 0) index--;
      continue;
    }
    if (char === '"' || char === "'" || char === "`") {
      while (++index < pattern.length) {
        if (pattern[index] === "\\") {
          index++;
          continue;
        }
        if (pattern[index] === char) break;
      }
      continue;
    }
    if (char === "{" || char === "(") {
      stack.push(char);
      if (stack.length > MAX_DEPTH) refuse("pattern nesting depth");
    } else if (
      (char === "}" && stack.at(-1) === "{") ||
      (char === ")" && stack.at(-1) === "(")
    ) {
      stack.pop();
    }
  }
}

function ast(root) {
  if (!root || typeof root !== "object") return;
  // Traverse only child edges: upstream ASTs intentionally contain parent and
  // previous-sibling backreferences. Parent chains are validated separately.
  const active = new Set();
  const work = [{ node: root, depth: 0, leave: false }];
  let count = 0;
  let valueBytes = 0;
  while (work.length) {
    const frame = work.pop();
    const node = frame.node;
    if (frame.leave) {
      active.delete(node);
      continue;
    }
    if (!node || typeof node !== "object" || Array.isArray(node))
      refuse("invalid AST node");
    if (++count > MAX_NODES) refuse("AST node count");
    if (frame.depth > MAX_DEPTH) refuse("AST nesting depth");
    if (active.has(node)) refuse("AST child cycle");
    active.add(node);
    work.push({ node, depth: frame.depth, leave: true });
    const parents = new Set([node]);
    let parent = node.parent;
    let parentDepth = 0;
    while (parent != null) {
      if (typeof parent !== "object" || Array.isArray(parent))
        refuse("invalid AST parent");
      if (parents.has(parent)) refuse("AST parent cycle");
      if (++parentDepth > MAX_DEPTH + 1) refuse("AST parent depth");
      parents.add(parent);
      parent = parent.parent;
    }
    if (
      node.value != null &&
      typeof node.value !== "string" &&
      typeof node.value !== "number"
    ) {
      refuse("invalid AST value");
    }
    if (typeof node.value === "string") valueBytes += node.value.length;
    if (valueBytes > MAX_OUTPUT_BYTES) refuse("AST value length");
    if (node.nodes != null) {
      if (!Array.isArray(node.nodes)) refuse("invalid AST children");
      if (node.nodes.length + count + work.length > MAX_NODES)
        refuse("AST node count");
      // Check before pushing a caller-supplied huge array onto our own stack.
      for (let index = node.nodes.length - 1; index >= 0; index--) {
        work.push({
          node: node.nodes[index],
          depth: frame.depth + 1,
          leave: false,
        });
      }
    }
  }
}

function output(value) {
  if (typeof value === "string" && value.length > MAX_OUTPUT_BYTES) {
    refuse("compiled/stringified output length");
  }
  return value;
}

// fill-range normalizes zero and negative integer steps with max(abs(step),1).
// Compile's unit-step numeric regex is compact; stepped regexes enumerate just
// like expand, so bound their work BEFORE fill-range creates arrays/strings.
function range(args, options, toRegex) {
  if (args.length > 3) refuse("invalid range argument count");
  for (const value of [...args, options.step]) {
    if (typeof value === "string" && value.length > 10000)
      refuse("range component length");
  }
  const [start, end] = args;
  const valid = (value) =>
    typeof value === "number" || (typeof value === "string" && value !== "");
  if (!valid(start) || !valid(end)) return; // Preserve upstream invalid-range errors.
  let selectedStep = args[2] || options.step || 1;
  // A nonnumeric object step is retried by fill-range with explicit step=1.
  if (
    selectedStep &&
    typeof selectedStep === "object" &&
    !Number.isInteger(+selectedStep)
  ) {
    selectedStep = 1;
    // fill-range retries an object step as its options, losing compile's
    // forced regex mode. Conservatively bound that ordinary array path.
    toRegex = false;
  }
  if (!Number.isInteger(+selectedStep)) return; // Upstream handles invalid steps.
  const step = Math.max(Math.abs(Number(selectedStep)), 1);
  const numeric = Number.isInteger(+start) && Number.isInteger(+end);
  if (numeric) {
    if (!Number.isSafeInteger(+start) || !Number.isSafeInteger(+end))
      refuse("numeric range precision");
    if (toRegex && step === 1) return;
  } else if (
    (!Number.isInteger(+start) && String(start).length > 1) ||
    (!Number.isInteger(+end) && String(end).length > 1)
  ) {
    return;
  }
  const from = numeric ? Number(start) : String(start).charCodeAt(0);
  const to = numeric ? Number(end) : String(end).charCodeAt(0);
  if (toRegex && !numeric && step === 1) return;
  const count = Math.floor(Math.abs(to - from) / step) + 1;
  if (!Number.isFinite(count) || count > MAX_RESULTS)
    refuse("enumerated range count");
  // Include explicit-step zero padding and regex delimiters in the upper bound.
  const width = numeric
    ? Math.max(
        String(start).length,
        String(end).length,
        String(selectedStep).length,
      )
    : 1;
  if (
    count * (width + (toRegex ? 2 : 0)) + (toRegex ? 8 : 0) >
    MAX_OUTPUT_BYTES
  ) {
    refuse("enumerated range output length");
  }
}

module.exports = {
  input,
  ast,
  output,
  range,
  refuse,
  MAX_DEPTH,
  MAX_NODES,
  MAX_PATTERNS,
  MAX_RESULTS,
  MAX_OUTPUT_BYTES,
};
