import { test, expect } from '@playwright/test';
import { discoveryCollectionHref, discoveryDetailHref, discoverySectionHref, discoveryReturnHref, discoveryListBackHref } from '../src/lib/discovery';
import { loginAsAdmin } from './helpers';
import { apiBaseURL } from './config';

test('return destinations distinguish homepage, Facet and filtered list', () => {
  for (const kind of ['project', 'post', 'talk'] as const) {
    for (const from of ['', 'learning-design']) {
      const url = new URL(discoverySectionHref(kind, 'example', from), 'https://example.test');
      expect(discoveryReturnHref(kind, url.searchParams, from)).toBe(from ? '/learning-design' : '/');
    }
  }
  for (const [kind, filter] of [['post', 'Design & technology'], ['talk', '2026']] as const) {
    const context = { fromView: 'learning-design', filter };
    const url = new URL(discoveryDetailHref(kind, 'example', context), 'https://example.test');
    expect(discoveryReturnHref(kind, url.searchParams, context.fromView)).toBe(discoveryCollectionHref(kind, context));
  }
  expect(discoveryListBackHref()).toBe('/');
  expect(discoveryListBackHref('learning-design')).toBe('/learning-design');
  expect(discoveryReturnHref('project', new URLSearchParams())).toBe('/');
  expect(discoveryReturnHref('post', new URLSearchParams())).toBe('/posts');
  expect(discoveryReturnHref('talk', new URLSearchParams())).toBe('/talks');
});

test('untrusted return hints cannot leave the site or name unrelated routes', () => {
  for (const value of ['//evil.test', '/\\evil.test', 'https://evil.test/posts', 'javascript:alert(1)', '/admin', '/talks', '/posts#secret', '/posts?token=secret', '/posts?tag=a&tag=b', '/posts?from=x&from=y', '/x/../posts', '/posts\n', '/posts?tag=' + 'x'.repeat(600)]) {
    expect(discoveryReturnHref('post', new URLSearchParams({ return_to: value })), value).toBe('/posts');
  }
  expect(discoveryReturnHref('post', new URLSearchParams('return_to=%2F&return_to=%2Fposts'))).toBe('/posts');
  for (const from of ['admin', 'posts', '../admin', '/evil.test', 'x\\y', 'x'.repeat(101)]) {
    expect(discoveryListBackHref(from)).toBe('/');
  }
  expect(discoveryReturnHref('project', new URLSearchParams('return_to=%2Fprojects'))).toBe('/');
});

test.describe('built-app Back journeys', () => {
  const created: { collection: string; id: string }[] = [];
  let token: string;
  test.beforeAll(async ({ request }) => {
    ({ token } = await loginAsAdmin(request));
    for (const [collection, data] of [
      ['projects', { title: 'Navigation test project', slug: 'navigation-test-project', description: 'Navigation fixture.', summary: 'Navigation fixture.' }],
      ['posts', { title: 'Navigation test post', slug: 'navigation-test-post', content: 'Navigation fixture.', excerpt: 'Navigation fixture.', tags: ['Navigation'], published_at: '2026-10-28 00:00:00.000Z' }],
      ['talks', { title: 'Navigation test talk', slug: 'navigation-test-talk', description: 'Navigation fixture.', date: '2026-10-28 00:00:00.000Z' }]
    ] as const) {
      const response = await request.post(`${apiBaseURL}/api/collections/${collection}/records`, {
        headers: { Authorization: token }, data: { ...data, visibility: 'public', is_draft: false }
      });
      expect(response.ok(), await response.text()).toBe(true);
      created.push({ collection, id: (await response.json()).id });
    }
    const view = await request.post(`${apiBaseURL}/api/collections/views/records`, {
      headers: { Authorization: token },
      data: { name: 'Navigation test Facet', slug: 'navigation-test-facet', visibility: 'public', is_active: true,
        sections: created.map(({ collection, id }) => ({ section: collection, enabled: true, layout: 'grid-2', items: [id] })) }
    });
    expect(view.ok(), await view.text()).toBe(true);
    created.push({ collection: 'views', id: (await view.json()).id });
  });
  test.afterAll(async ({ request }) => {
    for (const { collection, id } of [...created].reverse()) {
      await request.delete(`${apiBaseURL}/api/collections/${collection}/records/${id}`, { headers: { Authorization: token } });
    }
  });

  for (const width of [1280, 390]) {
    test(`project cards return to their homepage or Facet at ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      for (const source of ['/', '/navigation-test-facet']) {
        await page.goto(source);
        const card = page.locator('a[href^="/projects/"]').first();
        await expect(card).toBeVisible();
        const target = await card.getAttribute('href');
        await card.click();
        await expect(page.locator('h1')).toBeVisible();
        const back = page.getByRole('link', { name: /^Back to / }).first();
        await expect(back).toHaveAttribute('href', source);
        await back.click();
        await expect(page).toHaveURL(new RegExp(source === '/' ? '/$' : '/navigation-test-facet$'));
        await expect(page.locator('a[href^="/projects/"]').first()).toBeVisible();
        await page.goBack();
        expect(new URL(page.url()).pathname).toBe(new URL(target!, 'https://example.test').pathname);
        await expect(page.locator('h1')).toBeVisible();
        await expect(back).toHaveAttribute('href', source);
        await page.reload();
        await expect(back).toHaveAttribute('href', source);
      }
    });

    test(`filtered posts and talks keep their list and Facet at ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 900 });
      for (const [kind, query, title] of [
        ['posts', 'from=navigation-test-facet&tag=Navigation', 'Navigation test post'],
        ['talks', 'from=navigation-test-facet&year=2026', 'Navigation test talk']
      ]) {
        await page.goto(`/${kind}?${query}`);
        await page.getByRole('link', { name: title, exact: true }).click();
        await expect(page.getByRole('heading', { level: 1, name: title })).toBeVisible();
        const back = page.getByRole('link', { name: /^Back to / }).first();
        await expect(back).toHaveAttribute('href', `/${kind}?${query}`);
        await back.click();
        await expect(page.getByRole('link', { name: title, exact: true })).toBeVisible();
        await page.getByRole('link', { name: 'Back to navigation-test-facet', exact: true }).click();
        await expect(page).toHaveURL(/\/navigation-test-facet$/);
        await page.goto(`/${kind}`);
        await page.getByRole('link', { name: 'Back to Home', exact: true }).click();
        await expect(page).toHaveURL(/\/$/);
      }
    });
  }

  test('cold and external project entries use a safe fallback, including new tabs', async ({ page, context }) => {
    await page.goto('/projects/facet', { referer: 'https://outside.example/' });
    const back = page.getByRole('link', { name: 'Back to Home', exact: true });
    await expect(back).toHaveAttribute('href', '/');
    await back.click();
    await expect(page).toHaveURL(/\/$/);
    const tab = await context.newPage();
    await tab.goto('/projects/facet?from=navigation-test-facet');
    await tab.getByRole('link', { name: 'Back to navigation-test-facet', exact: true }).click();
    await expect(tab).toHaveURL(/\/navigation-test-facet$/);
    await tab.close();
  });

  test('admin filter updates preserve router state and unrelated query context', async ({ page, request }) => {
    const { token, record: model } = await loginAsAdmin(request);
    await page.context().addCookies([{ name: 'pb_auth', value: JSON.stringify({ token, model }), url: process.env.PLAYWRIGHT_BASE_URL! }]);
    await page.addInitScript(({ token, model }) => {
      localStorage.setItem('pocketbase_auth', JSON.stringify({ token, model }));
      localStorage.setItem('facet_setup_wizard', JSON.stringify({ dismissedPermanently: true }));
    }, { token, model });
    await page.goto('/admin/projects?context=keep#list');
    const input = page.getByPlaceholder('Search projects...');
    await expect(input).toBeVisible();
    const originalState = await page.evaluate(() => history.state);
    expect(Object.keys(originalState).length).toBeGreaterThan(0);
    await input.fill('Facet');
    await expect(page).toHaveURL(/search=Facet/);
    expect(new URL(page.url()).searchParams.get('context')).toBe('keep');
    expect(new URL(page.url()).hash).toBe('#list');
    expect(await page.evaluate(() => history.state)).toEqual(originalState);
    await page.locator('a[href="/admin/content"]').first().click();
    await expect(page).toHaveURL(/\/admin\/content$/);
    await page.goBack();
    await expect(input).toBeVisible();
    await expect(input).toHaveValue('Facet');
    await page.goForward();
    await expect(page).toHaveURL(/\/admin\/content$/);
    await expect(input).toBeHidden();
    await page.locator('a[href="/admin/views"]').first().click();
    await expect(page).toHaveURL(/\/admin\/views$/);
    await page.evaluate(() => {
      const link = document.createElement('a');
      link.style.cssText = 'position:fixed;top:80px;right:20px;z-index:99999;background:white;padding:16px';
      link.href = '/admin/views/missingfacet123';
      link.textContent = 'Open missing facet';
      document.body.append(link);
    });
    await page.getByRole('link', { name: 'Open missing facet', exact: true }).click();
    await expect(page.getByText('Failed to load facet', { exact: true })).toBeVisible();
    await expect(page).toHaveURL(/\/admin\/views$/);
    await page.goBack();
    await expect(page).toHaveURL(/\/admin\/views$/);
    await page.goBack();
    await expect(page).toHaveURL(/\/admin\/content$/);
  });

  test('completed Facet creation and deletion do not leave broken history entries', async ({ page, request }) => {
    const { token, record: model } = await loginAsAdmin(request);
    await page.context().addCookies([{ name: 'pb_auth', value: JSON.stringify({ token, model }), url: process.env.PLAYWRIGHT_BASE_URL! }]);
    await page.addInitScript(({ token, model }) => {
      localStorage.setItem('pocketbase_auth', JSON.stringify({ token, model }));
      localStorage.setItem('facet_setup_wizard', JSON.stringify({ dismissedPermanently: true }));
    }, { token, model });
    for (const withShare of [false, true]) {
      await page.goto('/admin/views');
      await page.locator('a[href="/admin/views/new"]').first().click();
      await page.getByPlaceholder('Recruiter View', { exact: true }).fill(`Navigation create ${withShare}`);
      await page.getByPlaceholder('recruiter', { exact: true }).fill(`navigation-create-${withShare}`);
      if (withShare) {
        await page.locator('#visibility').selectOption('unlisted');
        await page.getByLabel('Generate share link when creating this view').check();
      }
      const saved = page.waitForResponse(r => r.url().endsWith('/api/collections/views/records') && r.request().method() === 'POST');
      await page.getByRole('button', { name: 'Create', exact: true }).click();
      const response = await saved;
      expect(response.ok()).toBe(true);
      const view = await response.json();
      created.push({ collection: 'views', id: view.id });
      await expect(page).toHaveURL(new RegExp(withShare ? `/admin/views/${view.id}$` : '/admin/views$'));
      await page.goBack();
      await expect(page).toHaveURL(/\/admin\/views$/);
      // Opening and deleting the new Facet must also replace the editor entry.
      await page.locator(`a[href="/admin/views/${view.id}"]`).filter({ visible: true }).first().click();
      await page.getByRole('button', { name: 'Delete Facet', exact: true }).click();
      await page.getByRole('alertdialog').getByRole('button', { name: 'Delete', exact: true }).click();
      await expect(page).toHaveURL(/\/admin\/views$/);
      await page.goBack();
      await expect(page).toHaveURL(/\/admin\/views$/);
      await expect(page.locator(`a[href="/admin/views/${view.id}"]`)).toHaveCount(0);
    }
  });

  test('applying or rejecting an import proposal replaces the review entry', async ({ page, request }) => {
    const { token, record: model } = await loginAsAdmin(request);
    await page.context().addCookies([{ name: 'pb_auth', value: JSON.stringify({ token, model }), url: process.env.PLAYWRIGHT_BASE_URL! }]);
    await page.addInitScript(({ token, model }) => {
      localStorage.setItem('pocketbase_auth', JSON.stringify({ token, model }));
      localStorage.setItem('facet_setup_wizard', JSON.stringify({ dismissedPermanently: true }));
    }, { token, model });
    for (const action of ['apply', 'reject']) {
      const response = await request.post(`${apiBaseURL}/api/collections/import_proposals/records`, {
        headers: { Authorization: token },
        data: { source_id: 'navigation-fixture', project_id: created.find(x => x.collection === 'projects')!.id,
          proposed_data: { summary: 'Reviewed navigation fixture' }, status: 'pending' }
      });
      expect(response.ok(), await response.text()).toBe(true);
      const proposal = await response.json();
      created.push({ collection: 'import_proposals', id: proposal.id });
      await page.goto('/admin/import');
      await page.evaluate((id) => {
        const link = document.createElement('a');
      link.style.cssText = 'position:fixed;top:80px;right:20px;z-index:99999;background:white;padding:16px';
        link.href = `/admin/review/${id}`;
        link.textContent = 'Review navigation fixture';
        document.body.append(link);
      }, proposal.id);
      await page.getByRole('link', { name: 'Review navigation fixture', exact: true }).click();
      await page.getByRole('button', { name: action === 'apply' ? 'Apply Selected Changes' : 'Reject All', exact: true }).click();
      await expect(page).toHaveURL(new RegExp(action === 'apply' ? '/admin/projects$' : '/admin/import$'));
      await page.goBack();
      await expect(page).toHaveURL(/\/admin\/import$/);
      await expect(page.getByRole('button', { name: 'Apply Selected Changes', exact: true })).toHaveCount(0);
    }
  });

});
