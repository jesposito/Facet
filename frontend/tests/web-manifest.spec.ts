import { expect, test } from '@playwright/test';
import { buildWebManifest } from '../src/lib/web-manifest';

test.describe('web manifest generation', () => {
	test('uses profile name, custom theme color, avatar, and favicon when available', () => {
		const manifest = buildWebManifest(
			{
				name: 'Ada Lovelace',
				avatar_url: '/api/files/profile/ada/avatar.png',
				custom_hex_color: '#123456',
				hero_bg_color: '#f8fafc'
			},
			{ favicon: '/api/favicon' }
		);

		expect(manifest.name).toBe('Ada Lovelace');
		expect(manifest.short_name).toBe('Ada Lovelace');
		expect(manifest.theme_color).toBe('#123456');
		expect(manifest.background_color).toBe('#f8fafc');
		expect(manifest.icons.map((icon) => icon.src)).toEqual([
			'/api/files/profile/ada/avatar.png',
			'/api/favicon',
			'/icon.png',
			'/favicon.png'
		]);
	});

	test('falls back cleanly without profile or settings data', () => {
		const manifest = buildWebManifest(null, null);

		expect(manifest.name).toBe('Facet');
		expect(manifest.start_url).toBe('/');
		expect(manifest.scope).toBe('/');
		expect(manifest.display).toBe('standalone');
		expect(manifest.icons.map((icon) => icon.src)).toEqual(['/icon.png', '/favicon.png']);
	});
});
