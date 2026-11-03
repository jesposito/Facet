import { test, expect, type Locator, type Page } from '@playwright/test';
import { loginAsAdmin, seedAdminSession } from './helpers';
import { apiBaseURL, adminEmail, adminPassword } from './config';
import {
	readExperience,
	type ExperienceLesson,
	type ProjectExperience
} from '../src/lib/experiences/types';
import { newExperience, newBlock } from '../src/lib/experiences/templates';

type Preset = 'playful' | 'technical' | 'editorial';
type Box = { x: number; y: number; width: number; height: number };
type Colours = { color: string; background: string } | null;

const lesson = (id: string, title: string, description: string): ExperienceLesson => ({
	id,
	title,
	src: '/media/missing-design-chapter.mp4',
	poster: '',
	description,
	transcript: `A transcript for ${title.toLowerCase()}.`,
	credit: 'Design fixture'
});

const designFixture = (preset: Preset): ProjectExperience => ({
	version: 1,
	enabled: true,
	layout: 'immersive',
	design: { preset, width: 'wide' },
	hero: {
		intro: 'Design regression fixture.',
		tone: 'paper',
		imagePosition: 'right',
		action: { label: 'Start listening', href: '#listen' }
	},
	sections: [
		{
			id: 'listen',
			title: 'Listen in',
			intro: '',
			tone: 'ink',
			layout: 'stack',
			blocks: [
				{
					id: 'inkVideo',
					type: 'video',
					title: 'Ink chapters',
					lessons: [
						lesson('inkOne', 'First ink chapter', 'The first ink description.'),
						lesson('inkTwo', 'Second ink chapter', 'The second ink description.')
					]
				}
			]
		},
		{
			id: 'feature',
			title: 'In the spotlight',
			intro: '',
			tone: 'paper',
			layout: 'stack',
			blocks: [
				{
					id: 'spotlightVideo',
					type: 'video',
					title: 'Spotlight chapters',
					presentation: 'spotlight',
					lessons: [
						lesson('spotOne', 'First spotlight chapter', 'The first spotlight description.'),
						lesson('spotTwo', 'Second spotlight chapter', 'The second spotlight description.')
					]
				}
			]
		},
		...(['aqua', 'sand', 'green', 'pink'] as const).map((tone) => ({
			id: `tone-${tone}`,
			title: `A ${tone} chapter`,
			intro: '',
			tone,
			layout: 'stack' as const,
			blocks: [{ id: `${tone}Text`, type: 'text' as const, body: `Words on ${tone}.` }]
		}))
	]
});

/** The design fixture without a preset, holding layout choices Original can't draw. */
const originalFixture = (): ProjectExperience => {
	const doc = designFixture('playful');
	delete doc.design;
	doc.hero.tone = 'aqua';
	doc.sections[0].headingLayout = 'split';
	const video = doc.sections[0].blocks[0];
	if (video.type === 'video') video.presentation = 'sidebar';
	const story = newBlock('story');
	if (story.type === 'story') story.presentation = 'paper';
	doc.sections[0].blocks.push(story);
	return doc;
};

/** Computed chapter colours keyed by part, so a published page and a preview can be compared. */
const palette = (root: Locator) =>
	root.evaluate((el) => {
		const colours = (node: Element | null): Colours => {
			if (!node) return null;
			const style = getComputedStyle(node);
			return { color: style.color, background: style.backgroundColor };
		};
		const parts: Record<string, Colours> = {
			hero: colours(el.querySelector('.experience-hero')),
			index: colours(el.querySelector('.experience-index')),
			action: colours(el.querySelector('a.experience-primary'))
		};
		for (const section of el.querySelectorAll('.experience-section')) {
			const id = section.id.replace(/^preview-/, '');
			parts[id] = colours(section);
			parts[`${id} chosen chapter`] = colours(
				section.querySelector('.experience-chapters .chosen')
			);
		}
		return parts;
	});

/** WCAG contrast of an element's text against its composited background. */
const textContrast = (target: Locator) =>
	target.evaluate((el) => {
		const parse = (value: string) => {
			const match = value.match(/^rgba?\(([^)]+)\)$/);
			if (!match) return null;
			const [r, g, b, a = 1] = match[1]
				.split(/[\s,/]+/)
				.filter(Boolean)
				.map(Number);
			return { r, g, b, a };
		};
		const layers: Array<{ r: number; g: number; b: number; a: number }> = [];
		for (let node: Element | null = el; node; node = node.parentElement) {
			const bg = parse(getComputedStyle(node).backgroundColor);
			if (bg && bg.a > 0) layers.push(bg);
			if (bg && bg.a >= 1) break;
		}
		let base = { r: 255, g: 255, b: 255 };
		for (const layer of layers.reverse())
			base = {
				r: layer.r * layer.a + base.r * (1 - layer.a),
				g: layer.g * layer.a + base.g * (1 - layer.a),
				b: layer.b * layer.a + base.b * (1 - layer.a)
			};
		const fg = parse(getComputedStyle(el).color);
		if (!fg) return 0;
		const text = {
			r: fg.r * fg.a + base.r * (1 - fg.a),
			g: fg.g * fg.a + base.g * (1 - fg.a),
			b: fg.b * fg.a + base.b * (1 - fg.a)
		};
		const luminance = ({ r, g, b }: { r: number; g: number; b: number }) => {
			const [lr, lg, lb] = [r, g, b].map((c) => {
				const s = c / 255;
				return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
			});
			return 0.2126 * lr + 0.7152 * lg + 0.0722 * lb;
		};
		const [hi, lo] = [luminance(text), luminance(base)].sort((a, b) => b - a);
		return (hi + 0.05) / (lo + 0.05);
	});

const overlaps = (a: Box, b: Box) =>
	a.x < b.x + b.width - 0.5 &&
	b.x < a.x + a.width - 0.5 &&
	a.y < b.y + b.height - 0.5 &&
	b.y < a.y + a.height - 0.5;

async function box(target: Locator): Promise<Box> {
	await expect(target).toBeVisible();
	const result = await target.boundingBox();
	expect(result).not.toBeNull();
	return result!;
}

test('design options remain additive and reject malformed presentation data', () => {
	const original = newExperience('video');
	expect(readExperience(original)).toEqual(original);
	for (const preset of ['playful', 'technical', 'editorial'] as const) {
		const doc = structuredClone(original);
		doc.design = { preset, width: 'wide' };
		doc.hero.imageNote = 'A useful detail';
		doc.hero.image = { src: '/photo.jpg', alt: '', decorative: true };
		doc.hero.tone = 'orange';
		doc.sections[0].tone = 'yellow';
		doc.sections[0].headingLayout = 'split';
		const video = doc.sections[0].blocks[0];
		if (video.type === 'video') video.presentation = 'sidebar';
		const story = newBlock('story');
		if (story.type === 'story') story.presentation = 'paper';
		doc.sections[0].blocks.push(story);
		expect(readExperience(doc)).toEqual(doc);
	}
	const mutations: Array<(document: ProjectExperience) => void> = [
		(d) => Reflect.set(d, 'design', null),
		(d) => Reflect.set(d, 'design', { preset: 'unknown' }),
		(d) => Reflect.set(d, 'design', { preset: 'playful', width: 'enormous' }),
		(d) => Reflect.set(d, 'design', { preset: 'playful', css: 'body{display:none}' }),
		(d) => Reflect.set(d.hero, 'imageNote', 'two\nlines'),
		(d) => Reflect.set(d.hero, 'imageNote', 'a'.repeat(201)),
		(d) => Reflect.set(d.hero, 'image', { src: '', alt: '', decorative: 'true' }),
		(d) => Reflect.set(d.sections[0], 'headingLayout', 'columns'),
		(d) => Reflect.set(d.sections[0].blocks[0], 'presentation', 'paper')
	];
	for (const mutate of mutations) {
		const doc = structuredClone(original);
		mutate(doc);
		expect(readExperience(doc)).toBeNull();
	}
});

test.describe('project design presentation in the browser', () => {
	test.skip(!adminEmail || !adminPassword, 'Disposable admin fixture required');
	const presets: Preset[] = ['playful', 'technical', 'editorial'];
	const slugs = new Map<Preset, string>();
	const records = new Map<Preset, string>();
	const ids: string[] = [];
	let token: string;
	let originalSlug: string;
	let originalId: string;

	test.beforeAll(async ({ request }) => {
		token = (await loginAsAdmin(request)).token;
		const run = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
		for (const preset of presets) {
			const slug = `design-${preset}-${run}`;
			const res = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
				headers: { Authorization: token },
				data: {
					title: `Design fixture ${preset}`,
					slug,
					summary: 'Design regression fixture.',
					description: 'Classic project description.',
					visibility: 'public',
					is_draft: false,
					experience: designFixture(preset)
				}
			});
			expect(res.ok(), await res.text()).toBeTruthy();
			const id = (await res.json()).id;
			ids.push(id);
			records.set(preset, id);
			slugs.set(preset, slug);
		}
		originalSlug = `design-original-${run}`;
		const res = await request.post(`${apiBaseURL}/api/collections/projects/records`, {
			headers: { Authorization: token },
			data: {
				title: 'Design fixture original',
				slug: originalSlug,
				summary: 'Design regression fixture.',
				description: 'Classic project description.',
				visibility: 'public',
				is_draft: false,
				experience: originalFixture()
			}
		});
		expect(res.ok(), await res.text()).toBeTruthy();
		originalId = (await res.json()).id;
		ids.push(originalId);
	});
	test.afterAll(async ({ request }) => {
		for (const id of ids)
			await request.delete(`${apiBaseURL}/api/collections/projects/records/${id}`, {
				headers: { Authorization: token }
			});
	});

	/** Clicking until the press registers proves the player has hydrated. */
	async function choose(chapter: Locator) {
		await expect(async () => {
			await chapter.click();
			await expect(chapter).toHaveAttribute('aria-pressed', 'true', { timeout: 1000 });
		}).toPass();
	}

	test('technical ink chapters keep selected and unselected text readable', async ({ page }) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto(`/projects/${slugs.get('technical')}`);
		await expect(page.locator('.project-experience')).toHaveAttribute(
			'data-design-preset',
			'technical'
		);
		const chapters = page.locator('#listen .experience-chapters button');
		await expect(chapters).toHaveCount(2);
		for (const dark of [false, true]) {
			await page.evaluate((d) => document.documentElement.classList.toggle('dark', d), dark);
			for (const selected of [1, 0]) {
				await choose(chapters.nth(selected));
				await expect(chapters.nth(1 - selected)).toHaveAttribute('aria-pressed', 'false');
				for (const index of [0, 1]) {
					const label = chapters.nth(index).locator('.experience-chapter-label');
					const state = index === selected ? 'selected' : 'unselected';
					for (const text of [label, label.locator('span').first()])
						expect(
							await textContrast(text),
							`${dark ? 'dark' : 'light'} theme, chapter ${index + 1} ${state}`
						).toBeGreaterThanOrEqual(4.5);
				}
			}
		}
	});

	test('media that fails before hydration still reports its status', async ({ page }) => {
		let releaseJS!: () => void;
		const blockedJS = new Promise<void>((resolve) => (releaseJS = resolve));
		let held = 0;
		await page.route(/\/_app\/immutable\/.*\.js(?:\?|$)/, async (route) => {
			held++;
			await blockedJS;
			await route.continue();
		});
		const block = page.locator('#listen .experience-video');
		try {
			await page.goto(`/projects/${slugs.get('technical')}`, { waitUntil: 'commit' });
			await expect.poll(() => held).toBeGreaterThan(0);
			await page.waitForFunction(
				() => document.querySelector<HTMLVideoElement>('#listen video')?.error != null
			);
			await expect(block.getByRole('status')).toHaveCount(0);
		} finally {
			releaseJS();
		}
		expect(await block.locator('video').evaluate((v: HTMLVideoElement) => v.error?.code)).toBe(4);
		await expect(block.getByRole('status')).toContainText("This recording couldn't load.");
		await expect(block.getByRole('status')).toContainText('You can read the transcript below.');
		await block.getByRole('button', { name: 'First ink chapter' }).click();
		expect(await block.locator('video').evaluate((v: HTMLVideoElement) => v.error?.code)).toBe(4);
		await expect(block.getByRole('status')).toContainText("This recording couldn't load.");
		await block.locator('details > summary').click();
		await expect(block.locator('.experience-transcript')).toHaveText(
			'A transcript for first ink chapter.'
		);
	});

	test('spotlight media failure status never overlaps the chapter description', async ({
		page
	}) => {
		for (const preset of presets) {
			await page.setViewportSize({ width: 1440, height: 900 });
			await page.goto(`/projects/${slugs.get(preset)}`);
			await expect(page.locator('.project-experience')).toHaveAttribute(
				'data-design-preset',
				preset
			);
			const block = page.locator('#feature .experience-video');
			await choose(block.getByRole('button', { name: 'Second spotlight chapter' }));
			// A missing file can fail before hydration, so raise the failure on the live player.
			await block.locator('video').dispatchEvent('error');
			const status = block.getByRole('status');
			await expect(status).toContainText("This recording couldn't load.");
			const parts = {
				heading: block.locator(':scope > h4'),
				status,
				description: block.getByText('The second spotlight description.', { exact: true }),
				credit: block.locator(':scope > .experience-credit'),
				transcript: block.locator(':scope > details')
			};
			for (const layout of ['desktop', 'mobile'] as const) {
				if (layout === 'mobile') await page.setViewportSize({ width: 390, height: 900 });
				await expect(block).toHaveCSS('display', layout === 'desktop' ? 'grid' : 'block');
				const boxes: Array<[string, Box]> = [];
				for (const [name, part] of Object.entries(parts)) boxes.push([name, await box(part)]);
				if (layout === 'desktop') {
					boxes.push(['player', await box(block.locator('.experience-player'))]);
					boxes.push(['chapters', await box(block.locator('.experience-chapters'))]);
				}
				for (let i = 0; i < boxes.length; i++)
					for (let j = i + 1; j < boxes.length; j++) {
						const [a, b] = [boxes[i], boxes[j]];
						expect(
							overlaps(a[1], b[1]),
							`${preset} ${layout}: ${a[0]} ${JSON.stringify(a[1])} overlaps ${b[0]} ${JSON.stringify(b[1])}`
						).toBe(false);
					}
				if (layout === 'mobile') {
					for (const [name, rect] of boxes) {
						expect(rect.x, `${preset} mobile ${name} left edge`).toBeGreaterThanOrEqual(-0.5);
						expect(rect.x + rect.width, `${preset} mobile ${name} right edge`).toBeLessThanOrEqual(
							390.5
						);
					}
					expect(
						await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1)
					).toBeTruthy();
					for (const text of [status, parts.description])
						expect(await textContrast(text), `${preset} mobile text`).toBeGreaterThanOrEqual(4.5);
				}
			}
		}
	});

	async function openEditor(page: Page, id: string) {
		await page.goto('/admin/projects');
		await page.locator(`[data-row-id="${id}"]`).getByTitle('Edit', { exact: true }).click();
		const editor = page.locator('.experience-editor');
		await expect(editor).toBeVisible();
		return editor;
	}

	test('editor previews show each preset palette exactly as published', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		await page.setViewportSize({ width: 1440, height: 900 });
		for (const preset of presets) {
			const published: Record<string, Record<string, Colours>> = {};
			await page.goto(`/projects/${slugs.get(preset)}`);
			const root = page.locator('.project-experience');
			await expect(root).toHaveAttribute('data-design-preset', preset);
			for (const dark of [false, true]) {
				await page.evaluate((d) => document.documentElement.classList.toggle('dark', d), dark);
				published[dark ? 'dark' : 'light'] = await palette(root);
			}
			// The fixture reaches the tones and button the Original light preview used to repaint.
			for (const part of ['tone-aqua', 'tone-sand', 'tone-green', 'tone-pink', 'action', 'index'])
				expect(published.light[part], `${preset} ${part}`).not.toBeNull();

			const editor = await openEditor(page, records.get(preset)!);
			await editor.getByRole('button', { name: 'Preview experience' }).click();
			const preview = editor.locator('.project-experience');
			await expect(preview).toHaveAttribute('data-design-preset', preset);
			for (const theme of ['light', 'dark'] as const) {
				await editor.getByLabel('Dark preview').setChecked(theme === 'dark');
				await expect(editor.locator('.experience-preview-frame')).toHaveAttribute(
					'data-experience-theme',
					theme
				);
				expect(await palette(preview), `${preset} ${theme} preview`).toEqual(published[theme]);
			}
		}
	});

	test('Original disables the layouts it cannot draw and keeps their choices', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto(`/projects/${originalSlug}`);
		const published = page.locator('.project-experience');
		await expect(published).toBeVisible();
		await expect(published).not.toHaveAttribute('data-design-preset');
		await page.evaluate(() => document.documentElement.classList.remove('dark'));
		const publishedLight = await palette(published);

		const editor = await openEditor(page, originalId);
		const requirement = editor.locator('#experience-layout-requirement');
		const note = 'Needs a built-in Page design: Playful, Technical or Editorial.';
		const controls = {
			heading: editor.getByLabel('Section introduction').first(),
			player: editor.getByLabel('Player layout').first(),
			activity: editor.getByLabel('Activity layout')
		};
		type Layouts = Record<keyof typeof controls, string>;
		const expectLayouts = async (enabled: boolean, values: Layouts) => {
			if (enabled) await expect(requirement).toHaveCount(0);
			else await expect(requirement).toContainText('need a built-in Page design');
			for (const name of Object.keys(controls) as Array<keyof typeof controls>) {
				const control = controls[name];
				await expect(control, name).toHaveValue(values[name]);
				if (enabled) {
					await expect(control, name).toBeEnabled();
					await expect(control, name).not.toHaveAttribute('aria-describedby');
				} else {
					await expect(control, name).toBeDisabled();
					await expect(control, name).toHaveAccessibleDescription(note);
				}
			}
		};

		await expect(editor.getByRole('button', { name: /^Original/ })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
		await expectLayouts(false, { heading: 'split', player: 'sidebar', activity: 'paper' });
		await editor.getByRole('button', { name: /^Playful/ }).click();
		await expectLayouts(true, { heading: 'split', player: 'sidebar', activity: 'paper' });
		await controls.heading.selectOption('stack');
		await controls.player.selectOption('spotlight');
		await controls.activity.selectOption('plain');
		await editor.getByRole('button', { name: /^Original/ }).click();
		await expectLayouts(false, { heading: 'stack', player: 'spotlight', activity: 'plain' });

		// A light preview inside a dark admin keeps the Original palette and its contrast.
		await page.evaluate(() => document.documentElement.classList.add('dark'));
		await editor.getByRole('button', { name: 'Preview experience' }).click();
		const preview = editor.locator('.project-experience');
		await expect(preview).not.toHaveAttribute('data-design-preset');
		await expect(editor.locator('.experience-preview-frame')).toHaveAttribute(
			'data-experience-theme',
			'light'
		);
		expect(await palette(preview)).toEqual(publishedLight);
		for (const text of [
			preview.locator('h1'),
			preview.locator('.experience-hero .experience-prose p')
		])
			expect(await textContrast(text)).toBeGreaterThanOrEqual(4.5);
		await editor.getByRole('button', { name: 'Return to editor' }).click();
		await page.evaluate(() => document.documentElement.classList.remove('dark'));

		const saveResponse = page.waitForResponse(
			(response) =>
				response.request().method() === 'PATCH' &&
				new URL(response.url()).pathname === `/api/collections/projects/records/${originalId}`
		);
		await page.getByRole('button', { name: /update project|save project|save changes/i }).click();
		expect((await saveResponse).ok()).toBeTruthy();
		const saved: ProjectExperience = (
			await (
				await request.get(`${apiBaseURL}/api/collections/projects/records/${originalId}`, {
					headers: { Authorization: token }
				})
			).json()
		).experience;
		expect(saved.design).toBeUndefined();
		const [listen] = saved.sections;
		expect(listen.headingLayout).toBe('stack');
		expect(listen.blocks.map((b) => ('presentation' in b ? b.presentation : null))).toEqual([
			'spotlight',
			'plain'
		]);
	});
});
