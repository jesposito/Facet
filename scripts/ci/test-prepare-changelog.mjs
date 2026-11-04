// Run: node --test scripts/ci/test-prepare-changelog.mjs
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { prepareChangelog } from './prepare-changelog.mjs';

const HEADER = '# Changelog\n\nAll notable changes to Facet will be documented in this file.\n\n';
const BODY = [
  '## v3.7.1 - November 05, 2026',
  '',
  '**Bugs Fixed:**',
  '- Fix a thing',
  '',
  '**New Features:**',
  '- Add a thing',
  '',
  '**Other Changes:**',
  '- Tidy a thing',
  '',
  '**Pull Requests:** [#600](https://github.com/jesposito/Facet/pull/600),',
  '',
  '',
].join('\n');
const ENTRY = `${BODY}---\n\n`;
const HISTORY = [
  '## v3.7.0 - November 03, 2026',
  '',
  '**New Features:**',
  '- Older feature',
  '',
  '---',
  '',
  '',
  '## v3.6.1 - November 03, 2026',
  '',
  '### Upgrade notes',
  '',
  '- Historical note stays with v3.6.1.',
  '',
  '**Pull Requests:** [#521](https://github.com/jesposito/Facet/pull/521),',
  '',
  '---',
  '',
  '',
].join('\n');
const NOTES = [
  '### Upgrade notes',
  '',
  '- Facet sections now distinguish **Choose items** from **All public content**.',
  '- Fonts are served by your own installation.',
  '- The container frontend listens on loopback behind Caddy.',
].join('\n');

const entryWith = (notes) => `${BODY}${notes}\n\n---\n\n`;
const occurrences = (text, needle) => text.split(needle).length - 1;

test('moves current Unreleased upgrade notes under the new version exactly once', () => {
  const changes = '### Changes\n\n- Hand-written change is dropped.\n\n\n';
  const output = prepareChangelog(`${HEADER}## [Unreleased]\n\n${NOTES}\n\n${changes}${HISTORY}`, ENTRY);

  assert.equal(output, `${HEADER}\n${entryWith(NOTES)}${HISTORY}`);
  assert.equal(occurrences(output, 'All public content'), 1);
  assert.equal(occurrences(output, 'Historical note'), 1);
  assert.ok(output.indexOf(NOTES) > output.indexOf('## v3.7.1'));
  assert.ok(output.indexOf(NOTES) < output.indexOf('## v3.7.0'));
});

test('keeps lists, links, multiline text and subheadings verbatim', () => {
  const notes = [
    '### Upgrade Notes:',
    '',
    '- See the [upgrade guide](https://example.com/a?b=1#c) and `$(touch /tmp/nope)`.',
    '  Continuation line keeps its indent.',
    '  - Nested item',
    '',
    '1. Back up first.',
    '',
    '**API keys:**',
    '',
    '- Rotate old keys.',
    '',
    '#### Proxy installs',
    '',
    'Set `TRUSTED_PROXIES`.',
    '',
    '---',
    '',
    'Text after a rule is still a note.',
  ].join('\n');
  const input = `${HEADER}## [Unreleased]\n\n${notes}\n\n---\n\n${HISTORY}`;

  assert.equal(prepareChangelog(input, ENTRY), `${HEADER}\n${entryWith(notes)}${HISTORY}`);
});

test('matches the old head and tail insertion without an Unreleased section', () => {
  assert.equal(prepareChangelog(`${HEADER}\n${HISTORY}`, ENTRY), `${HEADER}\n${ENTRY}\n${HISTORY}`);
});

test('adds no notes when Unreleased has none, or only an empty heading', () => {
  const expected = `${HEADER}\n${ENTRY}${HISTORY}`;
  for (const section of ['### Changes\n\n- Dropped.\n\n', '### Upgrade notes\n\n---\n\n']) {
    const output = prepareChangelog(`${HEADER}## [Unreleased]\n\n${section}${HISTORY}`, ENTRY);
    assert.equal(output, expected);
    assert.equal(occurrences(output, '### Upgrade notes'), 1);
  }
});

test('ends Unreleased at the next section of any name, or at end of file', () => {
  const legacy = '## [3.6.9] - Legacy heading\n\n- Older text stays.\n\n';
  const notes = '### Upgrade notes\n\n- Keep me.';
  assert.equal(
    prepareChangelog(`${HEADER}## [Unreleased]\n\n${notes}\n\n${legacy}${HISTORY}`, ENTRY),
    `${HEADER}\n${entryWith(notes)}${legacy}${HISTORY}`,
  );
  assert.equal(prepareChangelog(`${HEADER}## Unreleased\n\n${notes}`, ENTRY), `${HEADER}\n${entryWith(notes)}`);
});

test('accepts bracketed and unbracketed Unreleased headings', () => {
  const headings = [
    '## [Unreleased]',
    '## Unreleased',
    '## [Unreleased] - 2026-11-04',
    '## Unreleased - November 04, 2026',
    '## unreleased',
  ];
  for (const heading of headings) {
    assert.equal(
      prepareChangelog(`${HEADER}${heading}\n\n${NOTES}\n\n${HISTORY}`, ENTRY),
      `${HEADER}\n${entryWith(NOTES)}${HISTORY}`,
    );
  }
});

test('ignores headings inside fenced code', () => {
  const notes = [
    '### Upgrade notes',
    '',
    '```md',
    '## v9.9.9',
    '### Changes',
    '---',
    '```',
    '',
    '~~~',
    '## [Unreleased]',
    '~~~',
    '',
    '- After the fences.',
  ].join('\n');
  assert.equal(
    prepareChangelog(`${HEADER}## [Unreleased]\n\n${notes}\n\n### Changes\n\n- Dropped.\n\n${HISTORY}`, ENTRY),
    `${HEADER}\n${entryWith(notes)}${HISTORY}`,
  );

  const fencedHistory = '## v3.7.0 - Old\n\n```md\n## [Unreleased]\n\n### Upgrade notes\n\n- Sample.\n```\n\n---\n\n';
  const kept = '### Upgrade notes\n\n- Keep me.';
  assert.equal(
    prepareChangelog(`${HEADER}## [Unreleased]\n\n${kept}\n\n${fencedHistory}`, ENTRY),
    `${HEADER}\n${entryWith(kept)}${fencedHistory}`,
  );
});

test('fails clearly instead of guessing', () => {
  assert.throws(
    () => prepareChangelog(`${HEADER}## [Unreleased]\n\n- One\n\n## Unreleased\n\n- Two\n\n${HISTORY}`, ENTRY),
    /Found 2 Unreleased sections at lines 5, 9/,
  );
  assert.throws(
    () => prepareChangelog(`${HEADER}## [Unreleased]\n\n${NOTES}\n\n${NOTES}\n\n${HISTORY}`, ENTRY),
    /Found 2 Upgrade notes sections/,
  );
  assert.throws(
    () => prepareChangelog(`${HEADER}## [Unreleased]\n\n### Upgrade notes\n\n\`\`\`sh\nmake backup\n\n${HISTORY}`, ENTRY),
    /Unclosed code fence at line 9/,
  );
  assert.throws(
    () => prepareChangelog(`${HEADER}## [Unreleased]\n\n${NOTES}\n\n${HISTORY}`, BODY),
    /no --- separator/,
  );
});

test('tolerates an unclosed fence in older history', () => {
  const history = `${HISTORY}\`\`\`\nold sample\n`;
  assert.equal(prepareChangelog(`${HEADER}\n${history}`, ENTRY), `${HEADER}\n${ENTRY}\n${history}`);
});

test('CLI rewrites the changelog, and leaves it alone on failure', (t) => {
  const directory = mkdtempSync(join(tmpdir(), 'facet-changelog-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const changelog = join(directory, 'CHANGELOG.md');
  const entry = join(directory, 'entry.md');
  const script = fileURLToPath(new URL('./prepare-changelog.mjs', import.meta.url));
  const run = () => spawnSync(process.execPath, [script, changelog, entry], { encoding: 'utf8' });
  writeFileSync(entry, ENTRY);

  writeFileSync(changelog, `${HEADER}## [Unreleased]\n\n${NOTES}\n\n${HISTORY}`);
  assert.equal(run().status, 0);
  assert.equal(readFileSync(changelog, 'utf8'), `${HEADER}\n${entryWith(NOTES)}${HISTORY}`);

  const ambiguous = `${HEADER}## [Unreleased]\n\n## Unreleased\n\n${HISTORY}`;
  writeFileSync(changelog, ambiguous);
  const failed = run();
  assert.equal(failed.status, 1);
  assert.match(failed.stderr, /::error::Found 2 Unreleased sections/);
  assert.equal(readFileSync(changelog, 'utf8'), ambiguous);
});

test('release workflow uses the helper and keeps the no-changelog fallback', () => {
  const workflow = readFileSync(new URL('../../.github/workflows/auto-tag.yml', import.meta.url), 'utf8');
  assert.ok(workflow.includes('node scripts/ci/prepare-changelog.mjs CHANGELOG.md "$ENTRY_FILE"'));
  assert.ok(!workflow.includes('head -n 4 CHANGELOG.md'));
  assert.ok(workflow.includes('echo "# Changelog"'));
});
