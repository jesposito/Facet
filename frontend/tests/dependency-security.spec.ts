import { test, expect } from '@playwright/test';
import { parseMarkdown, sanitizeRichHtml } from '../src/lib/utils';
import { loginAsAdmin } from './helpers';

test('rich HTML sanitization removes active content and deceptive embed origins', () => {
  const result = sanitizeRichHtml('<p>Keep this</p><img src="x" onerror="alert(1)"><script>alert(1)</script><iframe src="https://www.youtube.com.attacker.example/embed/1"></iframe>');
  expect(result).toContain('<p>Keep this</p>');
  expect(result).not.toMatch(/onerror|<script|<iframe/);
  expect(parseMarkdown('First paragraph\n\n**Second paragraph**')).toContain('<strong>Second paragraph</strong>');
});

test('built public pages render on desktop and mobile', async ({ page }) => {
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: 900 });
    const response = await page.goto('/');
    expect(response?.ok()).toBe(true);
    await expect(page.locator('main')).toBeVisible();
    expect(await page.locator('body').innerText()).not.toContain('Internal Error');
  }
});

test('updated editor preserves Markdown through rich-text round trip', async ({ page, request }) => {
  const { token, record: model } = await loginAsAdmin(request);
  await page.context().addCookies([{
    name: 'pb_auth', value: JSON.stringify({ token, model }), url: process.env.PLAYWRIGHT_BASE_URL!
  }]);
  await page.addInitScript(({ token, model }) => {
    localStorage.setItem('pocketbase_auth', JSON.stringify({ token, model }));
    localStorage.setItem('facet_setup_wizard', JSON.stringify({ dismissedPermanently: true }));
  }, { token, model });
  await page.goto('/admin/homepage');
  const editor = page.locator('.ProseMirror').first();
  await expect(editor).toBeVisible();
  const toggle = page.getByRole('button', { name: 'Toggle editor mode (Ctrl+Shift+M)', exact: true }).first();
  await toggle.click();
  const source = page.locator('textarea').filter({ visible: true }).first();
  await source.fill('First paragraph\n\n**Second paragraph**');
  await toggle.click();
  await expect(editor.locator('p')).toHaveCount(2);
  await expect(editor.locator('strong')).toHaveText('Second paragraph');
  await toggle.click();
  await expect(source).toHaveValue('First paragraph\n\n**Second paragraph**');
});
