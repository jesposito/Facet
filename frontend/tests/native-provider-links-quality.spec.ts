import { test, expect, type Locator, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { loginAsAdmin } from './helpers';
import { apiBaseURL, adminEmail, adminPassword } from './config';
import {
	experienceProviderPage,
	readExperience,
	type ExperienceLesson,
	type ProjectExperience
} from '../src/lib/experiences/types';

type Design = 'original' | 'playful' | 'technical' | 'editorial';

const designs: Design[] = ['original', 'playful', 'technical', 'editorial'];
const presentations = ['stack', 'sidebar', 'spotlight'] as const;

const bunnyVideo = '0a1b2c3d-4e5f-6789-abcd-ef0123456789';
const providerPages = [
	['YouTube page', 'https://www.youtube.com/watch?v=abc123DEF_-&t=12'],
	['Vimeo page', 'https://player.vimeo.com/video/123456789?h=abcdef0123'],
	['Bunny page', `https://iframe.mediadelivery.net/embed/12345/${bunnyVideo}`]
] as const;
const lookalike = 'https://www.youtube.com.attacker.example/watch?v=abc123';

const lesson = (id: string, title: string, src: string, poster = ''): ExperienceLesson => ({
	id,
	title,
	src,
	poster,
	description: `${title} description.`,
	transcript: `Transcript for ${title}.`,
	credit: 'Provider link fixture'
});

const fixture = (design: Design, video: string, audio: string): ProjectExperience => ({
	version: 1,
	enabled: true,
	layout: 'immersive',
	...(design === 'original' ? {} : { design: { preset: design, width: 'wide' as const } }),
	hero: { intro: 'Provider link fixture.', tone: 'paper', imagePosition: 'right' },
	sections: [
		{
			id: 'watch',
			title: 'Watch',
			intro: '',
			tone: 'ink',
			layout: 'stack',
			blocks: presentations.map((presentation) => ({
				id: `${presentation}Block`,
				type: 'video' as const,
				title: `${presentation} chapters`,
				presentation,
				lessons: [
					...providerPages.map(([title, src], index) =>
						lesson(`${presentation}Provider${index}`, title, src, index ? '' : '/icon.png')
					),
					lesson(`${presentation}Video`, 'Native video', video),
					lesson(`${presentation}Audio`, 'Native audio', audio),
					lesson(`${presentation}Lookalike`, 'Lookalike page', lookalike),
					lesson(`${presentation}Empty`, 'Empty chapter', '')
				]
			}))
		}
	]
});

const unsafeDocument = (src: string) => {
	const document = fixture('playful', '/media/chapter.mp4', '/media/chapter.mp3');
	const block = document.sections[0].blocks[0];
	if (block.type === 'video') block.lessons[0].src = src;
	return document;
};

test('only exact provider pages become links', () => {
	const pages = [
		...providerPages.map(([, src]) => src),
		'https://m.youtube.com/watch?list=abc&v=abc123',
		'https://youtu.be/abc123?t=4',
		'https://www.youtube.com/shorts/abc123',
		'https://www.youtube-nocookie.com/embed/abc123',
		'https://vimeo.com/123456789',
		'https://vimeo.com/123456789/abcdef0123',
		`https://player.mediadelivery.net/play/12345/${bunnyVideo}`
	];
	for (const page of pages) expect(experienceProviderPage(page), page).toBe(page);

	const notPages = [
		'/api/files/uploads/abc/chapter.mp4',
		'/watch?v=abc123',
		'https://cdn.example.com/chapter.mp4',
		'https://player.vimeo.com/external/123456789.hd.mp4?s=abc',
		'https://player.vimeo.com/progressive_redirect/playback/123/rendition/720p/file.mp4',
		'https://vz-abc123.b-cdn.net/0a1b2c3d/play_720p.mp4',
		`https://iframe.mediadelivery.net/play/12345/${bunnyVideo}/play_720p.mp4`,
		'https://youtu.be/chapter.mp4',
		'https://www.youtube.com/watch',
		'https://www.youtube.com:8443/watch?v=abc123',
		lookalike,
		'https://attacker.example/youtube.com/watch?v=abc123',
		'http://www.youtube.com/watch?v=abc123',
		'https://user:secret@youtu.be/abc123',
		'//youtu.be/abc123',
		'javascript:alert(1)',
		'',
		null
	];
	for (const value of notPages) expect(experienceProviderPage(value), String(value)).toBe('');
});

test('the shared reader accepts provider pages and refuses unsafe sources', () => {
	const document = fixture('playful', '/media/chapter.mp4', '/media/chapter.mp3');
	expect(readExperience(document)).toEqual(document);
	for (const src of ['http://www.youtube.com/watch?v=abc123', 'javascript:alert(1)'])
		expect(readExperience(unsafeDocument(src)), src).toBeNull();
});

test.describe('provider page links in native video chapters', () => {
	test.skip(!adminEmail || !adminPassword, 'Disposable admin fixture required');
	const slugs = new Map<Design, string>();
	const ids: string[] = [];
	const uploads: string[] = [];
	let token: string;

	const project = (slug: string, experience: ProjectExperience) => ({
		title: `Provider links ${slug}`,
		slug,
		summary: 'Provider link fixture.',
		description: 'Classic project description.',
		visibility: 'public',
		is_draft: false,
		experience
	});

	test.beforeAll(async ({ request }) => {
		token = (await loginAsAdmin(request)).token;
		const run = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
		const urls: string[] = [];
		for (const [name, mimeType] of [
			['chapter.mp4', 'video/mp4'],
			['chapter.mp3', 'audio/mpeg']
		]) {
			const uploaded = await request.post(`${apiBaseURL}/api/collections/uploads/records`, {
				headers: { Authorization: token },
				multipart: {
					file: {
						name,
						mimeType,
						buffer: readFileSync(join(process.cwd(), 'tests/fixtures/experiences', name))
					}
				}
			});
			expect(uploaded.ok(), await uploaded.text()).toBeTruthy();
			const media = await uploaded.json();
			uploads.push(media.id);
			urls.push(`/api/files/${media.collectionId}/${media.id}/${media.file}`);
		}
		for (const design of designs) {
			const slug = `provider-links-${design}-${run}`;
			const res = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
				headers: { Authorization: token },
				data: project(slug, fixture(design, urls[0], urls[1]))
			});
			expect(res.ok(), await res.text()).toBeTruthy();
			ids.push((await res.json()).id);
			slugs.set(design, slug);
		}
	});
	test.afterAll(async ({ request }) => {
		for (const id of ids)
			await request.delete(`${apiBaseURL}/api/collections/projects/records/${id}`, {
				headers: { Authorization: token }
			});
		for (const upload of uploads)
			await request.delete(`${apiBaseURL}/api/collections/uploads/records/${upload}`, {
				headers: { Authorization: token }
			});
	});

	/** Fixtures never reach a real host. Returns every provider request the page attempted. */
	async function blockFixtureHosts(page: Page) {
		const contacted: string[] = [];
		await page.route(
			(url) => /(^|\.)(youtube\.com|vimeo\.com|mediadelivery\.net)$/.test(url.hostname),
			(route) => {
				contacted.push(route.request().url());
				return route.abort();
			}
		);
		await page.route(
			(url) => url.hostname.endsWith('attacker.example'),
			(route) => route.abort()
		);
		return contacted;
	}

	const videoBlock = (page: Page, presentation: string) =>
		page.locator(`.experience-video[data-presentation="${presentation}"]`);

	/** Clicking until the press registers proves the player has hydrated. */
	async function choose(chapter: Locator) {
		await expect(async () => {
			await chapter.click();
			await expect(chapter).toHaveAttribute('aria-pressed', 'true', { timeout: 1000 });
		}).toPass();
	}

	async function box(target: Locator) {
		const value = await target.boundingBox();
		expect(value).not.toBeNull();
		return value!;
	}

	test('the server refuses a provider page without HTTPS', async ({ request }) => {
		const res = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
			headers: { Authorization: token },
			data: project(
				`provider-links-http-${Date.now()}`,
				unsafeDocument('http://www.youtube.com/watch?v=abc123')
			)
		});
		if (res.ok()) ids.push((await res.json()).id);
		expect(res.ok()).toBeFalsy();
	});

	for (const design of designs) {
		test(`${design} links provider pages and keeps native media`, async ({ page }) => {
			test.setTimeout(120_000);
			const errors: string[] = [];
			page.on('pageerror', (error) => errors.push(error.message));
			const contacted = await blockFixtureHosts(page);
			await page.setViewportSize({ width: 1440, height: 900 });
			await page.goto(`/projects/${slugs.get(design)}`);

			for (const presentation of presentations) {
				const block = videoBlock(page, presentation);
				const player = block.locator('.experience-player');
				const link = player.locator('a.experience-provider-link');
				const transcript = block.locator('.experience-transcript');
				await block.locator('details > summary').click();
				await expect(player.locator('img')).toHaveAttribute('alt', '');

				for (const [title, src] of providerPages) {
					await choose(block.getByRole('button', { name: title }));
					await expect(
						player.getByRole('link', {
							name: new RegExp(`^Watch Video\\s+${title}\\s+\\(opens in new tab\\)`)
						})
					).toHaveAttribute('href', src);
					await expect(link).toHaveAttribute('target', '_blank');
					await expect(link).toHaveAttribute('rel', 'noopener noreferrer');
					await expect(player.locator('video, audio, iframe')).toHaveCount(0);
					await expect(block.getByRole('status')).toHaveCount(0);
					await expect(transcript).toBeVisible();
					await expect(transcript).toHaveText(`Transcript for ${title}.`);
				}

				await choose(block.getByRole('button', { name: 'Native video' }));
				const video = player.locator('video');
				await expect(video).toHaveAttribute('controls', '');
				await expect(link).toHaveCount(0);
				await video.evaluate(async (el: HTMLVideoElement) => {
					el.muted = true;
					await el.play();
				});
				await expect
					.poll(() => video.evaluate((el: HTMLVideoElement) => el.currentTime))
					.toBeGreaterThan(0);

				await choose(block.getByRole('button', { name: 'Native audio' }));
				await expect(player.locator('audio')).toHaveAttribute('controls', '');
				await expect(player.locator('video')).toHaveCount(0);
				await expect(link).toHaveCount(0);

				await choose(block.getByRole('button', { name: 'Lookalike page' }));
				await expect(player.locator('video')).toHaveCount(1);
				await expect(link).toHaveCount(0);
				await expect(block.getByRole('status')).toContainText("This recording couldn't load.");
				await expect(block.getByRole('status')).toContainText('You can read the transcript below.');
				await expect(transcript).toHaveText('Transcript for Lookalike page.');

				await choose(block.getByRole('button', { name: 'Empty chapter' }));
				await expect(player).toContainText('A video will appear here once it has been added.');
				await expect(link).toHaveCount(0);
				await expect(transcript).toHaveText('Transcript for Empty chapter.');
			}
			expect(contacted).toEqual([]);
			expect(errors).toEqual([]);
		});

		test(`${design} link works by keyboard, on a phone and in print`, async ({ page }) => {
			test.setTimeout(60_000);
			const contacted = await blockFixtureHosts(page);
			await page.setViewportSize({ width: 1440, height: 900 });
			await page.goto(`/projects/${slugs.get(design)}`);

			for (const presentation of presentations) {
				const block = videoBlock(page, presentation);
				const link = block.locator('a.experience-provider-link');
				await expect(link).toHaveCSS('color', 'rgb(32, 53, 50)');
				await expect(link).toHaveCSS('background-color', 'rgb(229, 239, 206)');
				await block.getByRole('button', { name: 'YouTube page' }).focus();
				await page.keyboard.press('Shift+Tab');
				await expect(link).toBeFocused();
				await expect(link).toHaveCSS('outline-color', 'rgb(230, 240, 233)');
			}

			await page.setViewportSize({ width: 390, height: 900 });
			for (const presentation of presentations) {
				const block = videoBlock(page, presentation);
				const frame = await box(block.locator('.experience-player'));
				const target = await box(block.locator('a.experience-provider-link'));
				expect(target.height).toBeGreaterThanOrEqual(44);
				expect(target.x).toBeGreaterThanOrEqual(frame.x);
				expect(target.x + target.width).toBeLessThanOrEqual(frame.x + frame.width);
			}
			expect(
				await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)
			).toBeTruthy();

			await page.emulateMedia({ media: 'print' });
			for (const presentation of presentations) {
				const block = videoBlock(page, presentation);
				const link = block.locator('a.experience-provider-link');
				await expect(block.locator('.experience-player img')).toBeHidden();
				await expect(link).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)');
				await expect(link).toHaveCSS('color', 'rgb(0, 0, 0)');
				expect(await link.evaluate((el) => getComputedStyle(el, '::after').content)).toMatch(
					/attr\(href\)|youtube\.com/
				);
			}
			expect(contacted).toEqual([]);
		});
	}
});
