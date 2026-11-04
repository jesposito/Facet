import { test, expect } from '@playwright/test';
import { seedAdminSession } from './helpers';

// Upgrade notes on Settings → About. Only /CHANGELOG.md is mocked, so the real
// page parses and renders this fixture. The admin session comes from the test server.

const EMPTY_SELECTION_NOTE =
	'Facet sections now distinguish **Choose items** from **All public content**. Existing sections without an explicit choice use Choose items, so an empty selection shows nothing.';
const EMPTY_SELECTION_TEXT = EMPTY_SELECTION_NOTE.replaceAll('**', '');
const PROXY_NOTE =
	"Behind a reverse proxy or tunnel, set `TRUSTED_PROXIES` to your proxy's address range, or every visitor shares one rate-limit bucket. See the [configuration guide](https://github.com/jesposito/Facet/blob/main/docs/CONFIGURATION.md).";
const PROXY_TEXT =
	"Behind a reverse proxy or tunnel, set TRUSTED_PROXIES to your proxy's address range, or every visitor shares one rate-limit bucket. See the configuration guide.";
const WRAPPED_TEXT = 'Wrapped notes stay in one item when the manual text continues.';
const API_KEY_NOTE = "A key's allowed origins are now enforced for browser requests.";
const LEGACY_NOTE = 'Set SMTP_TLS explicitly if your provider needs something else.';
const HOSTILE_NOTE =
	'Hostile note <img src="data:," onerror="window.upgradeNoteXss = true"> ends here.';
const TAGGED_FEATURE = 'Tagged release feature';
const TAGGED_OTHER = 'Tagged release other change';
const LEGACY_FEATURE = 'Legacy release feature';
const OLDER_BUG = 'Older release bug fix';
const UNRELEASED_NOTE = 'Unreleased note that has not shipped';
const UNRELEASED_CHANGE = 'Unreleased change that has not shipped';

const CHANGELOG = `# Changelog

All notable changes to Facet will be documented in this file.

## [Unreleased]

### Upgrade notes

- ${UNRELEASED_NOTE}

### Changes

- ${UNRELEASED_CHANGE}

## v9.2.0 - November 04, 2026

**New Features:**
- ${TAGGED_FEATURE}

**Other Changes:**
- ${TAGGED_OTHER}

**Pull Requests:** [#900](https://github.com/jesposito/Facet/pull/900),

### Upgrade notes

- ${EMPTY_SELECTION_NOTE}
- ${PROXY_NOTE}
- Wrapped notes stay in one item
  when the manual text continues.

**API keys:**

- ${API_KEY_NOTE}

### Changes

- Change after the notes

---

## v9.1.0 - October 30, 2026

**Upgrade notes:**
- ${LEGACY_NOTE}
- ${HOSTILE_NOTE}

**New Features:**
- ${LEGACY_FEATURE}

**Pull Requests:** [#899](https://github.com/jesposito/Facet/pull/899),

---

## v9.0.0 - October 28, 2026

**Bugs Fixed:**
- ${OLDER_BUG}
`;

test.describe('Upgrade notes (About page)', () => {
	test.beforeEach(async ({ page, request }) => {
		await seedAdminSession(page, request);
		await page.route('**/CHANGELOG.md', (route) =>
			route.fulfill({ status: 200, contentType: 'text/markdown', body: CHANGELOG })
		);
		await page.goto('/admin/settings/about');
		await expect(page.getByRole('heading', { name: 'Upgrade notes' })).toHaveCount(2);
	});

	test('tagged release shows manual notes after pull request links', async ({ page }) => {
		const tagged = page.getByTestId('upgrade-notes').first();
		const items = tagged.getByRole('listitem');

		await expect(tagged.getByRole('heading', { name: 'Upgrade notes' })).toBeVisible();
		await expect(items).toHaveCount(5);
		await expect(items).toContainText([
			EMPTY_SELECTION_TEXT,
			PROXY_TEXT,
			WRAPPED_TEXT,
			'API keys:',
			API_KEY_NOTE
		]);
		await expect(tagged.locator('strong', { hasText: 'Choose items' })).toBeVisible();
		await expect(tagged.getByRole('link', { name: 'configuration guide' })).toHaveAttribute(
			'href',
			'https://github.com/jesposito/Facet/blob/main/docs/CONFIGURATION.md'
		);

		// Ordinary lists keep their items, and the notes no longer land in Other Changes.
		await expect(page.getByText(TAGGED_FEATURE)).toBeVisible();
		await expect(page.getByText(TAGGED_OTHER)).toBeVisible();
		await expect(page.getByRole('listitem').filter({ hasText: API_KEY_NOTE })).toHaveCount(1);
	});

	test('legacy bold notes stop at the next section and render sanitized', async ({ page }) => {
		const legacy = page.getByTestId('upgrade-notes').nth(1);
		const items = legacy.getByRole('listitem');

		await expect(items).toHaveCount(2);
		await expect(items).toContainText([LEGACY_NOTE, 'Hostile note ends here.']);
		await expect(legacy).not.toContainText(LEGACY_FEATURE);
		await expect(page.getByText(LEGACY_FEATURE)).toBeVisible();
		await expect(legacy.locator('[onerror]')).toHaveCount(0);
		expect(await page.evaluate(() => 'upgradeNoteXss' in window)).toBe(false);

		// The next version down keeps its own entry and has no notes to inherit.
		await expect(page.getByText(OLDER_BUG)).toBeVisible();
	});

	test('undated Unreleased notes are not shown as a release', async ({ page }) => {
		await expect(page.getByText('v9.2.0', { exact: true })).toBeVisible();
		await expect(page.getByText('Unreleased', { exact: true })).toHaveCount(0);
		await expect(page.getByText(UNRELEASED_NOTE)).toHaveCount(0);
		await expect(page.getByText(UNRELEASED_CHANGE)).toHaveCount(0);
	});
});
