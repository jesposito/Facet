import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';
import { apiBaseURL, adminEmail, adminPassword } from './config';
import {
	readExperience,
	safeExperienceUrl,
	type ProjectExperience
} from '../src/lib/experiences/types';

const fixture: ProjectExperience = {
	version: 1,
	enabled: true,
	layout: 'immersive',
	hero: {
		intro: 'A course worth exploring.',
		tone: 'aqua',
		imagePosition: 'right',
		action: { label: 'Try the story', href: '#practice' }
	},
	sections: [
		{
			id: 'inside',
			title: 'Inside the project',
			intro: 'Follow the choices behind the work.',
			tone: 'paper',
			layout: 'split',
			blocks: [
				{
					id: 'words',
					type: 'text',
					body: '# Learning by trying\n\n<img src=x onerror="window.experienceXSS=true"><script>window.experienceXSS=true</script>Useful **explanations**.'
				},
				{ id: 'notes', type: 'disclosure', title: 'Behind the scenes', body: 'A useful detail.' },
				{
					id: 'video',
					type: 'video',
					title: 'Conversation examples',
					lessons: [
						{
							id: 'lessonOne',
							title: 'A difficult question',
							src: '',
							poster: '',
							description: 'Listen closely.',
							transcript: 'A transcript for chapter one.',
							credit: 'Original cast'
						},
						{
							id: 'lessonTwo',
							title: 'A better question',
							src: '',
							poster: '',
							description: 'A second conversation.',
							transcript: 'A transcript for chapter two.',
							credit: 'Original cast'
						}
					]
				},
				{
					id: 'code',
					type: 'code',
					title: 'Read the code',
					steps: [
						{
							id: 'stepOne',
							title: 'Start here',
							body: 'First we configure the application.',
							code: 'const message = "<script>safe</script>";',
							language: 'JavaScript'
						},
						{
							id: 'stepTwo',
							title: 'Try a change',
							body: 'Then we make a request.',
							code: 'console.log(message);',
							language: 'JavaScript'
						}
					]
				}
			]
		},
		{
			id: 'practice',
			title: 'Give it a go',
			intro: 'No experience needed.',
			tone: 'sand',
			layout: 'stack',
			blocks: [
				{
					id: 'story',
					type: 'story',
					title: 'Your own story',
					template: '{character} found a potato. [[pause]] It was enormous.',
					fields: [{ key: 'character', label: 'Character', initial: 'Sylvester' }],
					pauseMs: 500,
					credit: 'A small experiment.'
				}
			]
		}
	]
};

test('reader rejects unsafe, malformed and future documents', () => {
	expect(readExperience(fixture)).toEqual(fixture);
	for (const value of [
		null,
		{ ...fixture, version: 2 },
		{ ...fixture, enabled: false },
		{ ...fixture, hero: null },
		{ ...fixture, sections: [null] },
		{ ...fixture, extra: true }
	])
		expect(readExperience(value)).toBeNull();
	for (const url of [
		'javascript:alert(1)',
		'//evil.test',
		'/\\evil.test',
		'https://user:pass@evil.test',
		'data:text/html,bad',
		'/\npath'
	])
		expect(safeExperienceUrl(url)).toBe('');
	const copy = structuredClone(fixture);
	copy.sections[1].id = 'inside';
	expect(readExperience(copy)).toBeNull();
	const unsafe = structuredClone(fixture);
	unsafe.hero.action!.href = 'javascript:alert(1)';
	expect(readExperience(unsafe)).toBeNull();
	const missing = structuredClone(fixture);
	delete (missing.hero as any).intro;
	expect(readExperience(missing)).toBeNull();
});

test.describe('native project experiences', () => {
	test.skip(!adminEmail || !adminPassword, 'Disposable admin fixture required');
	let token: string;
	let model: unknown;
	let id: string;
	let slug: string;
	const uploads: string[] = [];
	test.beforeAll(async ({ request }) => {
		const auth = await loginAsAdmin(request);
		token = auth.token;
		model = auth.record;
		slug = 'experience-' + Date.now();
		const urls: string[] = [];
		for (const [name, mime] of [
			['chapter.mp4', 'video/mp4'],
			['chapter.vtt', 'text/vtt'],
			['chapter.mp3', 'audio/mpeg']
		]) {
			const uploaded = await request.post(`${apiBaseURL}/api/collections/uploads/records`, {
				headers: { Authorization: token },
				multipart: {
					file: {
						name,
						mimeType: mime,
						buffer: readFileSync(join(process.cwd(), 'tests/fixtures/experiences', name))
					}
				}
			});
			expect(uploaded.ok(), await uploaded.text()).toBeTruthy();
			const media = await uploaded.json();
			uploads.push(media.id);
			urls.push(`/api/files/${media.collectionId}/${media.id}/${media.file}`);
		}
		const video = fixture.sections[0].blocks.find((b) => b.type === 'video');
		if (video?.type === 'video')
			for (const lesson of video.lessons) {
				lesson.src = urls[0];
				lesson.captions = { src: urls[1], language: 'en', label: 'English' };
			}

		if (video?.type === 'video')
			video.lessons.push({
				id: 'audioLesson',
				title: 'An audio example',
				src: urls[2],
				poster: '',
				description: 'Hear the example.',
				transcript: 'A short tone.',
				credit: 'Playback fixture'
			});
		const res = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
			headers: { Authorization: token },
			data: {
				title: 'A native learning project',
				slug,
				summary: 'Watch, explore and have a go.',
				description: 'Classic project description.',
				visibility: 'public',
				is_draft: false,
				experience: fixture
			}
		});
		expect(res.ok(), await res.text()).toBeTruthy();
		id = (await res.json()).id;
	});
	test.afterAll(async ({ request }) => {
		if (id)
			await request.delete(`${apiBaseURL}/api/collections/projects/records/${id}`, {
				headers: { Authorization: token }
			});
		for (const upload of uploads)
			await request.delete(`${apiBaseURL}/api/collections/uploads/records/${upload}`, {
				headers: { Authorization: token }
			});
	});
	test('renders every block safely and works at desktop and mobile widths', async ({ page }) => {
		const errors: string[] = [];
		page.on('pageerror', (e) => errors.push(e.message));
		await page.goto(`/projects/${slug}`);
		await expect(page.locator('h1')).toHaveCount(1);
		await expect(page.locator('.project-experience')).toBeVisible();
		const player = page.locator('video');
		await player.evaluate(async (el: HTMLVideoElement) => {
			el.muted = true;
			await el.play();
		});
		await expect
			.poll(() => player.evaluate((el: HTMLVideoElement) => el.currentTime))
			.toBeGreaterThan(0);
		await expect
			.poll(() => player.evaluate((el: HTMLVideoElement) => el.textTracks[0]?.cues?.length || 0))
			.toBeGreaterThan(0);
		await page.getByRole('button', { name: 'A better question' }).click();
		expect(await player.evaluate((el: HTMLVideoElement) => el.paused)).toBeTruthy();
		await page.getByText('Read the transcript', { exact: true }).click();
		await expect(page.getByText('A transcript for chapter two.')).toBeVisible();
		await page.getByRole('button', { name: 'An audio example' }).click();
		await page.locator('audio').evaluate(async (el: HTMLAudioElement) => {
			el.muted = true;
			await el.play();
		});
		await expect
			.poll(() => page.locator('audio').evaluate((el: HTMLAudioElement) => el.currentTime))
			.toBeGreaterThan(0);
		await page.getByRole('button', { name: 'A better question' }).click();
		await page.getByRole('button', { name: 'Try a change' }).click();
		await expect(page.locator('pre')).toHaveText('console.log(message);');
		await page.getByText('Behind the scenes', { exact: true }).click();
		await expect(page.getByText('A useful detail.')).toBeVisible();
		const character = page.getByLabel('Character', { exact: true });
		await character.fill('<img src=x onerror=alert(1)>');
		await expect(page.locator('.experience-story-output')).toContainText(
			'<img src=x onerror=alert(1)>'
		);
		await expect(page.locator('.experience-story-output img')).toHaveCount(0);
		await page.getByRole('button', { name: 'Reset', exact: true }).click();
		await expect(character).toHaveValue('Sylvester');
		expect(await page.evaluate(() => (window as any).experienceXSS)).toBeUndefined();
		for (const dark of [false, true])
			for (const width of [1440, 768, 390]) {
				await page.setViewportSize({ width, height: 900 });
				await page.evaluate((d) => document.documentElement.classList.toggle('dark', d), dark);
				expect(
					await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)
				).toBeTruthy();
				await page.waitForTimeout(250);
				await page.screenshot({
					path: `test-results/experience-${width}-${dark ? 'dark' : 'light'}.png`,
					fullPage: true,
					animations: 'disabled'
				});
			}
		expect(errors).toEqual([]);
	});
	test('admin editor previews, saves and reopens the document', async ({ page, request }) => {
		await page.addInitScript(
			({ token, model }) => {
				localStorage.setItem('pocketbase_auth', JSON.stringify({ token, model }));
				localStorage.setItem('facet_setup_wizard', JSON.stringify({ dismissedPermanently: true }));
			},
			{ token, model }
		);
		await page.goto('/admin/projects');
		const row = page.locator(`[data-row-id="${id}"]`);
		await row.getByTitle('Edit', { exact: true }).click();
		const editor = page.locator('.experience-editor');
		await editor
			.getByLabel('Introduction', { exact: true })
			.first()
			.fill('A revised introduction.');
		await editor.getByRole('button', { name: 'Preview experience' }).click();
		await expect(editor.locator('.project-experience')).toContainText('A revised introduction.');
		await editor.getByLabel('Mobile width').check();
		expect(
			await editor
				.locator('.experience-preview-frame')
				.evaluate((el) => el.getBoundingClientRect().width)
		).toBeLessThanOrEqual(390);
		await editor.getByRole('button', { name: 'Return to editor' }).click();
		await page.getByRole('button', { name: /update project|save project|save changes/i }).click();
		const saved = await (
			await request.get(`${apiBaseURL}/api/collections/projects/records/${id}`, {
				headers: { Authorization: token }
			})
		).json();
		expect(saved.experience.hero.intro).toBe('A revised introduction.');
		await page.reload();
		await page.locator(`[data-row-id="${id}"]`).getByTitle('Edit', { exact: true }).click();
		await expect(
			page.locator('.experience-editor').getByLabel('Introduction', { exact: true }).first()
		).toHaveValue('A revised introduction.');
	});

	test('export and duplication retain the saved experience', async ({ request }) => {
		const headers = { Authorization: token };
		const original = await (
			await request.get(`${apiBaseURL}/api/collections/projects/records/${id}`, { headers })
		).json();
		const exported = await request.get(`${apiBaseURL}/api/export?format=json`, { headers });
		expect(exported.ok()).toBeTruthy();
		const data = await exported.json();
		expect(data.projects.find((p: { id: string }) => p.id === id).experience).toEqual(
			original.experience
		);
		const result = await request.post(`${apiBaseURL}/api/admin/duplicate/projects/${id}`, {
			headers
		});
		expect(result.ok(), await result.text()).toBeTruthy();
		const copy = (await result.json()).record;
		try {
			expect(copy.experience).toEqual(original.experience);
		} finally {
			await request.delete(`${apiBaseURL}/api/collections/projects/records/${copy.id}`, {
				headers
			});
		}
	});
	test('disabled content falls back; private content stays private', async ({ request, page }) => {
		const patch = async (data: unknown) => {
			const r = await request.patch(`${apiBaseURL}/api/collections/projects/records/${id}`, {
				headers: { Authorization: token },
				data
			});
			expect(r.ok(), await r.text()).toBeTruthy();
		};
		await patch({ experience: { ...fixture, enabled: false } });
		await page.goto(`/projects/${slug}`);
		await expect(page.locator('.project-experience')).toHaveCount(0);
		await expect(page.getByText('Classic project description.')).toBeVisible();
		await patch({ experience: fixture, visibility: 'private' });
		expect((await request.get(`${apiBaseURL}/api/project/${slug}`)).status()).toBe(404);
		await patch({ visibility: 'public', is_draft: true });
		expect((await request.get(`${apiBaseURL}/api/project/${slug}`)).status()).toBe(404);
		await patch({ is_draft: false });
	});
});
