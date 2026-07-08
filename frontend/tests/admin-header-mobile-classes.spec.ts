import { expect, test } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const repoRoot = join(import.meta.dirname, '..');

function source(path: string): string {
	return readFileSync(join(repoRoot, path), 'utf8');
}

test.describe('admin mobile page headers', () => {
	test('content list headers stack actions on narrow screens', () => {
		const posts = source('src/routes/admin/posts/+page.svelte');

		expect(posts).toContain('flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6');
		expect(posts).toContain('flex flex-wrap items-center gap-2 w-full sm:w-auto');
	});

	test('tools/settings headers keep controls wrap-safe on narrow screens', () => {
		const analytics = source('src/routes/admin/analytics/+page.svelte');
		const tags = source('src/routes/admin/settings/tags/+page.svelte');

		expect(analytics).toContain('flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 mb-6');
		expect(analytics).toContain('w-full sm:w-auto');
		expect(tags).toContain('flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 mb-6');
		expect(tags).toContain('w-full sm:w-auto');
	});
});
