import { expect, test } from '@playwright/test';
import {
	internalApiHeaders,
	parseMediaRefs,
	parsePublicPosts,
	parsePublicTalks,
	publicCacheHeaders
} from '../src/lib/server/public-api';
import { navigationFacet } from '../src/lib/discovery';
import { GET as getSitemap } from '../src/routes/sitemap.xml/+server';

test('internal API auth forwards only purchase and TOTP cookies', () => {
	const request = new Request('http://localhost/posts', {
		headers: {
			cookie: 'session=do-not-forward; facet_purchases=purchase.jwt; facet_totp_session=totp.jwt'
		}
	});

	expect(internalApiHeaders(request)).toEqual({
		'X-Internal': 'true',
		Cookie: 'facet_purchases=purchase.jwt; facet_totp_session=totp.jwt'
	});
});

test('internal API auth omits absent, empty, and malformed credential cookies', () => {
	for (const cookie of [
		'',
		'unrelated=secret',
		'facet_purchases=; facet_totp_session=',
		'facet_purchases=bad value; facet_totp_session=still-valid',
		'facet_purchases="bad.value"; facet_totp_session=still-valid'
	]) {
		const headers = internalApiHeaders(
			new Request('http://localhost/posts', { headers: { cookie } })
		);
		expect(headers['X-Internal']).toBe('true');
		expect(headers.Cookie).toBe(
			cookie.includes('bad') ? 'facet_totp_session=still-valid' : undefined
		);
		expect(headers.Cookie ?? '').not.toContain('unrelated');
	}
});

test('authorized public responses vary by cookie and cannot enter shared caches', () => {
	const anonymous = new Request('http://localhost/posts');
	expect(publicCacheHeaders(anonymous)).toEqual({ Vary: 'Cookie' });

	for (const cookie of [
		'facet_purchases=purchase.jwt',
		'facet_totp_session=totp.jwt',
		'pb_auth=admin.jwt',
		'me_share_token=share.jwt',
		'me_password_token=password.jwt'
	]) {
		expect(
			publicCacheHeaders(new Request('http://localhost/posts', { headers: { cookie } }))
		).toEqual({
			Vary: 'Cookie',
			'Cache-Control': 'private, no-store'
		});
	}
});

test('public response validators preserve typed content and discard malformed entries', () => {
	expect(
		parseMediaRefs([
			{ id: 'media-1', url: '/media one.jpg', title: 'One', provider: 'upload' },
			{ id: 'media-2', title: 'Missing URL' },
			'not-a-record'
		])
	).toEqual([{ id: 'media-1', url: '/media one.jpg', title: 'One', provider: 'upload' }]);

	expect(
		parsePublicPosts([
			{
				id: 'post-1',
				title: 'Post',
				slug: 'post',
				visibility: 'public',
				is_draft: false,
				cover_image_url: '/cover.jpg',
				tags: ['news']
			},
			{ title: 'Missing identity' }
		])
	).toEqual([
		{
			id: 'post-1',
			title: 'Post',
			slug: 'post',
			visibility: 'public',
			is_draft: false,
			cover_image_url: '/cover.jpg',
			tags: ['news']
		}
	]);

	expect(
		parsePublicTalks([
			{ id: 'talk-1', title: 'Talk', slug: 'talk', video_url: 'https://example.test/talk' },
			{ id: 'talk-2' }
		])
	).toEqual([
		{ id: 'talk-1', title: 'Talk', slug: 'talk', video_url: 'https://example.test/talk' }
	]);
});

test('Facet navigation hints accept one valid slug and reject ambiguous or unsafe values', () => {
	expect(navigationFacet('recruiter-facet')).toBe('recruiter-facet');
	for (const hint of ['../admin', '/recruiter-facet', 'x'.repeat(101), '']) {
		expect(navigationFacet(hint)).toBe('');
	}
});

test('response cache policy preserves Vary tokens and overrides loader policy only for auth', async () => {
	const { applyPublicCacheHeaders } = await import('../src/lib/server/public-api');

	const anonymous = new Headers({
		Vary: 'Accept-Encoding, Accept',
		'Cache-Control': 'public, max-age=60'
	});
	applyPublicCacheHeaders(new Request('http://localhost/posts'), anonymous);
	expect(anonymous.get('Vary')).toBe('Accept-Encoding, Accept, Cookie');
	expect(anonymous.get('Cache-Control')).toBe('public, max-age=60');

	const cookieAuth = new Headers({
		Vary: 'Accept-Encoding, cookie',
		'Cache-Control': 'public, max-age=60'
	});
	applyPublicCacheHeaders(
		new Request('http://localhost/posts', { headers: { cookie: 'facet_purchases=purchase.jwt' } }),
		cookieAuth
	);
	expect(cookieAuth.get('Vary')).toBe('Accept-Encoding, cookie');
	expect(cookieAuth.get('Cache-Control')).toBe('private, no-store');

	const tokenAuth = new Headers({ 'Cache-Control': 'public, max-age=60' });
	applyPublicCacheHeaders(new Request('http://localhost/__data.json'), tokenAuth, true);
	expect(tokenAuth.get('Vary')).toBe('Cookie');
	expect(tokenAuth.get('Cache-Control')).toBe('private, no-store');
});

test('sitemap uses public API data without unlisted, draft, or nonexistent collection URLs', async () => {
	const responses: Record<string, unknown> = {
		'/api/homepage': {
			projects: [
				{ slug: 'public-project', visibility: 'public', is_draft: false, updated: '2026-11-01' },
				{ slug: 'unlisted-project', visibility: 'unlisted', is_draft: false, updated: '2026-11-01' }
			]
		},
		'/api/posts': {
			posts: [
				{ slug: 'public-post', visibility: 'public', is_draft: false, updated: '2026-11-01' },
				{ slug: 'draft-post', visibility: 'public', is_draft: true, updated: '2026-11-01' }
			]
		},
		'/api/talks': {
			talks: [
				{ slug: 'public-talk', visibility: 'public', is_draft: false, updated: '2026-11-01' },
				{ slug: 'unlisted-talk', visibility: 'unlisted', is_draft: false, updated: '2026-11-01' }
			]
		},
		'/api/site-nav': {
			items: [{ slug: 'public-facet' }]
		}
	};
	const fetch: typeof globalThis.fetch = async (input) => {
		const path = new URL(String(input)).pathname;
		return Response.json(responses[path] ?? {});
	};
	const origin = new URL(
		process.env.PUBLIC_APP_URL || process.env.APP_URL || 'http://localhost:8080'
	).origin;
	const response = await getSitemap({
		fetch,
		url: new URL(`${origin}/sitemap.xml`)
	} as unknown as Parameters<typeof getSitemap>[0]);
	const xml = await response.text();

	for (const path of [
		'/projects/public-project',
		'/posts/public-post',
		'/talks/public-talk',
		'/public-facet'
	]) {
		expect(xml).toContain(`<loc>${origin}${path}</loc>`);
	}
	for (const excluded of ['unlisted-', 'draft-', `<loc>${origin}/projects</loc>`]) {
		expect(xml).not.toContain(excluded);
	}
});
