// Inserts a generated release entry into CHANGELOG.md. The Unreleased section
// is dropped, except for its "### Upgrade notes", which move under the new version.
// Usage: node scripts/ci/prepare-changelog.mjs CHANGELOG.md entry.md
import { readFileSync, realpathSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

const HEADER_LINES = 4;
const HEADING = /^ {0,3}(#{1,6})[ \t]+(.*?)(?:[ \t]+#+)?\s*$/;
const FENCE = /^ {0,3}(`{3,}|~{3,})(.*)$/s;

const splitLines = (text) => text.split(/(?<=\n)/);
const lineNumbers = (headings) => headings.map((heading) => heading.index + 1).join(', ');

// Scan ATX headings and code fences, the two things that move a
// section boundary here. Add setext headings if the changelog ever uses them.
function scanHeadings(lines) {
  const headings = [];
  let fence = null;
  for (const [index, line] of lines.entries()) {
    const marker = line.match(FENCE);
    if (fence) {
      const closes =
        marker &&
        marker[1][0] === fence.char &&
        marker[1].length >= fence.length &&
        !marker[2].trim();
      if (closes) fence = null;
    } else if (marker && !(marker[1][0] === '`' && marker[2].includes('`'))) {
      fence = { char: marker[1][0], length: marker[1].length, index };
    } else {
      const heading = line.match(HEADING);
      if (heading) headings.push({ index, level: heading[1].length, text: heading[2] });
    }
  }
  return { headings, openFence: fence ? fence.index : -1 };
}

function upgradeNotes(lines, headings, start, end) {
  const section = headings.filter((heading) => heading.index > start && heading.index < end);
  const matches = section.filter(
    (heading) => heading.level === 3 && /^upgrade notes:?$/i.test(heading.text),
  );
  if (matches.length > 1) {
    throw new Error(
      `Found ${matches.length} Upgrade notes sections in Unreleased at lines ${lineNumbers(matches)}; keep one before releasing`,
    );
  }
  if (matches.length === 0) return '';

  const first = matches[0].index;
  const next = section.find((heading) => heading.index > first && heading.level <= 3);
  let last = next ? next.index : end;
  while (last > first + 1 && /^(?:---)?\s*$/.test(lines[last - 1])) last -= 1;
  return last > first + 1 ? `${lines.slice(first, last).join('').trimEnd()}\n\n` : '';
}

function addNotes(entry, notes) {
  if (!notes) return entry;
  const lines = splitLines(entry);
  const separator = lines.findLastIndex((line) => line.trimEnd() === '---');
  if (separator === -1) throw new Error('Generated entry has no --- separator');
  lines.splice(separator, 0, notes);
  return lines.join('');
}

export function prepareChangelog(changelog, entry) {
  const lines = splitLines(changelog);
  const { headings, openFence } = scanHeadings(lines);
  const unreleased = headings.filter(
    (heading) => heading.level === 2 && /^\[?unreleased\b/i.test(heading.text),
  );
  if (unreleased.length > 1) {
    throw new Error(
      `Found ${unreleased.length} Unreleased sections at lines ${lineNumbers(unreleased)}; keep one before releasing`,
    );
  }

  let notes = '';
  if (unreleased.length === 1) {
    const start = unreleased[0].index;
    const next = headings.find((heading) => heading.index > start && heading.level <= 2);
    const end = next ? next.index : lines.length;
    // An open fence hides every later heading, so the section would swallow all history.
    if (openFence >= start && openFence < end) {
      throw new Error(`Unclosed code fence at line ${openFence + 1} in Unreleased`);
    }
    notes = upgradeNotes(lines, headings, start, end);
    lines.splice(start, end - start);
  }

  return [
    ...lines.slice(0, HEADER_LINES),
    '\n',
    addNotes(entry, notes),
    ...lines.slice(HEADER_LINES),
  ].join('');
}

if (process.argv[1] && realpathSync(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const [changelogPath, entryPath] = process.argv.slice(2);
    if (!changelogPath || !entryPath) {
      throw new Error('Usage: prepare-changelog.mjs <changelog> <entry>');
    }
    const changelog = readFileSync(changelogPath, 'utf8');
    writeFileSync(changelogPath, prepareChangelog(changelog, readFileSync(entryPath, 'utf8')));
  } catch (error) {
    console.error(`::error::${error.message}`);
    process.exitCode = 1;
  }
}
