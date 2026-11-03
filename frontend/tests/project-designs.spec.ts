import { test, expect, type Locator, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
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

const splitTones = ['paper', 'aqua', 'sand', 'green', 'pink', 'ink', 'yellow', 'orange'] as const;

const lesson = (id: string, title: string, description: string): ExperienceLesson => ({
	id,
	title,
	src: '/media/missing-design-chapter.mp4',
	poster: '',
	description,
	transcript: `A transcript for ${title.toLowerCase()}.`,
	credit: 'Design fixture'
});

const longTitle = (lead: string) =>
	`${lead} chapter with a title that keeps going long enough to wrap over several lines`;

const story = (presentation: 'paper' | 'plain') => {
	const block = newBlock('story');
	if (block.type === 'story') block.presentation = presentation;
	return block;
};

/** `video` should be a playable upload when a test needs real media; the default never loads. */
const designFixture = (
	preset: Preset,
	video = '/media/missing-design-chapter.mp4'
): ProjectExperience => ({
	version: 1,
	enabled: true,
	layout: 'immersive',
	design: { preset, width: 'wide' },
	hero: {
		intro: 'Design regression fixture.',
		tone: 'paper',
		imagePosition: 'right',
		image: { src: '/icon.png', alt: 'Project icon', decorative: false },
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
		{
			id: 'sidebar',
			title: 'Chapters beside the player',
			intro: '',
			tone: 'paper',
			layout: 'stack',
			blocks: [
				{
					id: 'sidebarVideo',
					type: 'video',
					title: 'Sidebar chapters',
					presentation: 'sidebar',
					lessons: [
						{
							...lesson('sideOne', 'First sidebar chapter', 'The first sidebar text.'),
							src: video
						},
						{
							...lesson('sideTwo', 'Second sidebar chapter', 'The second sidebar text.'),
							src: video
						}
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
		})),
		{
			id: 'chapter-labels',
			title: 'Long chapter labels',
			intro: '',
			tone: 'paper',
			layout: 'stack',
			blocks: [
				{
					id: 'labelVideo',
					type: 'video',
					title: 'Long sidebar chapters',
					presentation: 'sidebar',
					lessons: [
						{
							...lesson('posterLong', longTitle('A poster'), 'The poster chapter text.'),
							poster: '/icon.png'
						},
						lesson('plainLong', longTitle('A posterless'), 'The posterless chapter text.')
					]
				}
			]
		},
		...(['paper', 'plain'] as const).map((presentation) => ({
			id: `story-${presentation}`,
			title: `A ${presentation} story`,
			intro: '',
			tone: 'paper' as const,
			layout: 'stack' as const,
			blocks: [story(presentation)]
		})),
		...splitTones.map((tone) => ({
			id: `split-${tone}`,
			title: `Split ${tone} chapter`,
			intro: `An introduction beside the ${tone} heading.`,
			tone,
			layout: 'split' as const,
			headingLayout: 'split' as const,
			blocks: (['first', 'second'] as const).map((place) => ({
				id: `split-${tone}-${place}`,
				type: 'text' as const,
				body: `The ${place} ${tone} column.`
			}))
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
	doc.sections[0].blocks.push(story('paper'));
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

/** Column tracks and placement of each split section's heading and blocks. */
const splitLayouts = (root: Locator, prefix: string) =>
	root.evaluate(
		(el, { tones, prefix }) => {
			const find = (scope: Element, selector: string) => {
				const node = scope.querySelector(selector);
				if (!node) throw new Error(`Missing ${selector}`);
				return node;
			};
			const rect = (node: Element) => {
				const { x, y, width, height } = node.getBoundingClientRect();
				return { x, y, width, height };
			};
			const tracks = (node: Element) =>
				getComputedStyle(node).gridTemplateColumns.split(' ').filter(Boolean).length;
			return tones.map((tone) => {
				const section = find(el, `#${prefix}split-${tone}`);
				const heading = find(section, '.experience-section-heading');
				const blocks = find(section, '.experience-blocks');
				const [first, second] = blocks.querySelectorAll(':scope > .experience-block');
				if (!first || !second) throw new Error(`${tone} needs two blocks`);
				return {
					tone,
					section: rect(section),
					headingDisplay: getComputedStyle(heading).display,
					headingTracks: tracks(heading),
					title: rect(find(heading, ':scope > h2')),
					intro: rect(find(heading, ':scope > .experience-prose')),
					blockTracks: tracks(blocks),
					first: rect(first),
					second: rect(second)
				};
			});
		},
		{ tones: [...splitTones], prefix }
	);

function expectSplit(
	layouts: Awaited<ReturnType<typeof splitLayouts>>,
	where: string,
	wide: boolean
) {
	expect(layouts.map((l) => l.tone)).toEqual(splitTones);
	for (const l of layouts) {
		const at = `${where} ${l.tone}`;
		if (wide) {
			expect(l.blockTracks, `${at} block columns`).toBe(2);
			expect(l.second.x, `${at} second block beside the first`).toBeGreaterThanOrEqual(
				l.first.x + l.first.width - 0.5
			);
			expect(l.second.y, `${at} blocks share a row`).toBeLessThan(l.first.y + l.first.height);
			expect(l.headingTracks, `${at} heading columns`).toBe(2);
			expect(l.intro.x, `${at} intro beside title`).toBeGreaterThanOrEqual(
				l.title.x + l.title.width - 0.5
			);
			expect(l.intro.y, `${at} intro shares the title row`).toBeLessThan(
				l.title.y + l.title.height
			);
			continue;
		}
		expect(l.blockTracks, `${at} block columns`).toBe(1);
		expect(l.second.y, `${at} second block below the first`).toBeGreaterThanOrEqual(
			l.first.y + l.first.height - 0.5
		);
		expect(Math.abs(l.second.x - l.first.x), `${at} blocks share a left edge`).toBeLessThan(1);
		expect(l.headingDisplay, `${at} heading`).toBe('block');
		expect(l.intro.y, `${at} intro below title`).toBeGreaterThanOrEqual(
			l.title.y + l.title.height - 0.5
		);
		for (const [name, part] of Object.entries({
			first: l.first,
			second: l.second,
			intro: l.intro
		})) {
			expect(part.x, `${at} ${name} left edge`).toBeGreaterThanOrEqual(l.section.x - 0.5);
			expect(part.x + part.width, `${at} ${name} right edge`).toBeLessThanOrEqual(
				l.section.x + l.section.width + 0.5
			);
		}
	}
}

/** Each long sidebar chapter's content box, thumbnail and label, read in one browser task. */
const chapterLabels = (root: Locator, prefix: string) =>
	root.evaluate((el, prefix) => {
		const rect = (node: Element) => {
			const { x, y, width, height } = node.getBoundingClientRect();
			return { x, y, width, height };
		};
		const list = el.querySelector(`#${prefix}chapter-labels .experience-chapters`);
		if (!list) throw new Error('Missing long chapter labels');
		return {
			list: rect(list),
			buttons: [...list.querySelectorAll('button')].map((button) => {
				const style = getComputedStyle(button);
				const outer = rect(button);
				const left = outer.x + parseFloat(style.borderLeftWidth) + parseFloat(style.paddingLeft);
				const right =
					outer.x +
					outer.width -
					parseFloat(style.borderRightWidth) -
					parseFloat(style.paddingRight);
				const label = button.querySelector('.experience-chapter-label')!;
				const img = button.querySelector('img');
				return {
					title: label.textContent ?? '',
					button: outer,
					content: { left, right, width: right - left },
					tracks: style.gridTemplateColumns.split(' ').filter(Boolean).map(parseFloat),
					label: rect(label),
					labelOverflow: label.scrollWidth - label.clientWidth,
					buttonOverflow: button.scrollWidth - button.clientWidth,
					poster: img !== null,
					thumb: img && getComputedStyle(img).display !== 'none' ? rect(img) : null
				};
			})
		};
	}, prefix);

function expectChapterLabels(
	layout: Awaited<ReturnType<typeof chapterLabels>>,
	where: string,
	thumbnails: boolean
) {
	expect(layout.buttons.map((b) => b.poster)).toEqual([true, false]);
	for (const b of layout.buttons) {
		const at = `${where} ${b.poster ? 'poster' : 'posterless'} chapter`;
		expect(b.title, at).toContain('keeps going long enough to wrap');
		expect(b.labelOverflow, `${at} label overflow`).toBeLessThanOrEqual(1);
		expect(b.buttonOverflow, `${at} button overflow`).toBeLessThanOrEqual(1);
		expect(b.button.x, `${at} left edge`).toBeGreaterThanOrEqual(layout.list.x - 0.5);
		expect(b.button.x + b.button.width, `${at} right edge`).toBeLessThanOrEqual(
			layout.list.x + layout.list.width + 0.5
		);
		expect(b.label.x, `${at} label left edge`).toBeGreaterThanOrEqual(b.content.left - 0.5);
		expect(b.label.x + b.label.width, `${at} label right edge`).toBeLessThanOrEqual(
			b.content.right + 0.5
		);
		if (b.poster && thumbnails) {
			expect(b.tracks.length, `${at} columns`).toBe(2);
			expect(b.thumb, `${at} thumbnail`).not.toBeNull();
			const thumb = b.thumb!;
			expect(Math.abs(thumb.width - b.tracks[0]), `${at} thumbnail fills its column`).toBeLessThan(
				1
			);
			expect(thumb.height, `${at} thumbnail height`).toBeGreaterThan(0);
			expect(b.label.x, `${at} label beside the thumbnail`).toBeGreaterThanOrEqual(
				thumb.x + thumb.width - 0.5
			);
			expect(Math.abs(b.label.width - b.tracks[1]), `${at} label fills its column`).toBeLessThan(1);
			continue;
		}
		if (b.poster) expect(b.thumb, `${at} thumbnail hidden on narrow players`).toBeNull();
		// Nothing shares the row, so the label must take the whole content box.
		expect(b.label.width, `${at} label width`).toBeGreaterThanOrEqual(b.content.width - 1);
	}
}

/** Sizes a preset derives from the page width, read once fonts and the opening image settle. */
const sizing = (root: Locator, prefix: string) =>
	root.evaluate(async (el, prefix) => {
		await document.fonts.ready;
		const find = (selector: string) => {
			const node = el.querySelector(selector);
			if (!node) throw new Error(`Missing ${selector}`);
			return node;
		};
		const image = find('.experience-hero-grid figure img') as HTMLImageElement;
		await image.decode().catch(() => undefined);
		const px = (selector: string, property: 'fontSize' | 'paddingLeft' | 'paddingRight') =>
			parseFloat(getComputedStyle(find(selector))[property]);
		return {
			width: el.getBoundingClientRect().width,
			gutterLeft: px('.experience-hero .experience-width', 'paddingLeft'),
			gutterRight: px(`#${prefix}listen .experience-width`, 'paddingRight'),
			h1: px('h1', 'fontSize'),
			summary: px('.experience-summary', 'fontSize'),
			h2: px(`#${prefix}listen h2`, 'fontSize'),
			h3: px(`#${prefix}listen h3`, 'fontSize'),
			body: px(`#${prefix}tone-aqua .experience-prose p`, 'fontSize'),
			image: image.getBoundingClientRect().height
		};
	}, prefix);

/** Parses a computed `rgb()`/`rgba()` colour, or the first one inside a longer value. */
function rgb(value: string) {
	const match = value.match(/rgba?\(([^)]+)\)/);
	if (!match) throw new Error(`No colour in ${value}`);
	const [r, g, b] = match[1]
		.split(/[\s,/]+/)
		.filter(Boolean)
		.map(Number);
	return { r, g, b };
}

/** WCAG contrast between two opaque computed colours. */
function contrast(a: string, b: string) {
	const luminance = ({ r, g, b }: { r: number; g: number; b: number }) => {
		const [lr, lg, lb] = [r, g, b].map((c) => {
			const s = c / 255;
			return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
		});
		return 0.2126 * lr + 0.7152 * lg + 0.0722 * lb;
	};
	const [hi, lo] = [luminance(rgb(a)), luminance(rgb(b))].sort((x, y) => y - x);
	return (hi + 0.05) / (lo + 0.05);
}

/** The outline and ring a chapter button draws, with the colour of the section behind it. */
const indicator = (button: Locator) =>
	button.evaluate((el) => {
		const style = getComputedStyle(el);
		const section = el.closest('.experience-section');
		if (!section) throw new Error('Chapter outside a section');
		return {
			focusVisible: el.matches(':focus-visible'),
			outline: [style.outlineStyle, style.outlineWidth, style.outlineColor, style.outlineOffset],
			outlineColor: style.outlineColor,
			ring: style.boxShadow,
			behind: getComputedStyle(section).backgroundColor
		};
	});

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
	const uploads: string[] = [];
	let token: string;
	let originalSlug: string;
	let originalId: string;

	test.beforeAll(async ({ request }) => {
		token = (await loginAsAdmin(request)).token;
		const run = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
		const uploaded = await request.post(`${apiBaseURL}/api/collections/uploads/records`, {
			headers: { Authorization: token },
			multipart: {
				file: {
					name: 'chapter.mp4',
					mimeType: 'video/mp4',
					buffer: readFileSync(join(process.cwd(), 'tests/fixtures/experiences/chapter.mp4'))
				}
			}
		});
		expect(uploaded.ok(), await uploaded.text()).toBeTruthy();
		const media = await uploaded.json();
		uploads.push(media.id);
		const video = `/api/files/${media.collectionId}/${media.id}/${media.file}`;
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
					experience: designFixture(preset, video)
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
		for (const upload of uploads)
			await request.delete(`${apiBaseURL}/api/collections/uploads/records/${upload}`, {
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

	/** Opens a preset's full-width preview while the admin window stays wide. */
	async function openPreview(page: Page, preset: Preset) {
		await page.setViewportSize({ width: 1440, height: 900 });
		const editor = await openEditor(page, records.get(preset)!);
		await editor.getByRole('button', { name: 'Preview experience' }).click();
		const preview = editor.locator('.project-experience');
		await expect(preview).toHaveAttribute('data-design-preset', preset);
		return { editor, preview };
	}

	async function showMobile(editor: Locator) {
		await editor.getByLabel('Mobile width').setChecked(true);
		await expect(editor.locator('.experience-preview-frame')).toHaveClass(/\bmobile\b/);
	}

	/** Opens a preset's preview at mobile width while the admin window stays wide. */
	async function openMobilePreview(page: Page, preset: Preset) {
		const { editor, preview } = await openPreview(page, preset);
		await showMobile(editor);
		return preview;
	}

	/** Sets the article container to `width`; the window stays wide, so viewport rules stay off. */
	async function sizePreview(preview: Locator, width: number | null) {
		await preview.evaluate((el: HTMLElement, w) => {
			el.style.width = w === null ? '' : `${w}px`;
		}, width);
		if (width !== null) await expect(preview).toHaveCSS('width', `${width}px`);
	}

	/** Resizes the window until the published article container is `width` wide. */
	async function sizePage(page: Page, root: Locator, width: number) {
		await page.setViewportSize({ width: width + 25, height: 900 });
		const chrome = width + 25 - (await box(root)).width;
		await page.setViewportSize({ width: Math.round(width + chrome), height: 900 });
		expect(Math.abs((await box(root)).width - width), `article width ${width}`).toBeLessThan(1);
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

	test('split sections stack on narrow pages and mobile previews in every tone', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		for (const preset of presets) {
			await page.setViewportSize({ width: 1440, height: 900 });
			await page.goto(`/projects/${slugs.get(preset)}`);
			const root = page.locator('.project-experience');
			await expect(root).toHaveAttribute('data-design-preset', preset);
			expectSplit(await splitLayouts(root, ''), `${preset} public 1440`, true);
			// Splits stack at a 700px container, so widths between 650 and 700 must stack too.
			await sizePage(page, root, 675);
			expectSplit(await splitLayouts(root, ''), `${preset} public 675`, false);
			await page.setViewportSize({ width: 390, height: 900 });
			expectSplit(await splitLayouts(root, ''), `${preset} public 390`, false);

			// A wide window keeps the viewport rules off, so only container rules can stack it.
			const { editor, preview } = await openPreview(page, preset);
			await sizePreview(preview, 675);
			expectSplit(await splitLayouts(preview, 'preview-'), `${preset} preview 675`, false);
			await sizePreview(preview, null);
			await showMobile(editor);
			expect((await box(preview)).width).toBeLessThanOrEqual(390);
			expectSplit(await splitLayouts(preview, 'preview-'), `${preset} preview mobile`, false);
		}
	});

	test('sidebar chapter labels fill their row with or without a poster', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const read = async (root: Locator, prefix: string, where: string, thumbnails: boolean) => {
			const poster = root.locator(`#${prefix}chapter-labels img`);
			if (thumbnails) {
				await poster.scrollIntoViewIfNeeded();
				await expect
					.poll(
						() => poster.evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth),
						{
							message: `${where} poster`
						}
					)
					.toBeGreaterThan(0);
			}
			expectChapterLabels(await chapterLabels(root, prefix), where, thumbnails);
		};
		for (const preset of presets) {
			await page.setViewportSize({ width: 1440, height: 900 });
			await page.goto(`/projects/${slugs.get(preset)}`);
			const root = page.locator('.project-experience');
			await expect(root).toHaveAttribute('data-design-preset', preset);
			await read(root, '', `${preset} public 1440`, true);
			// Between 650 and 1000 the thumbnail column narrows but stays beside the label.
			await page.setViewportSize({ width: 900, height: 900 });
			await read(root, '', `${preset} public 900`, true);
			await page.setViewportSize({ width: 390, height: 900 });
			await read(root, '', `${preset} public 390`, false);
			expect(
				await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1),
				`${preset} public 390 page overflow`
			).toBe(true);

			const preview = await openMobilePreview(page, preset);
			await read(preview, 'preview-', `${preset} preview mobile`, false);
		}
	});

	test('printing a story keeps the personalised text and hides its controls', async ({ page }) => {
		const name = 'Marigold Quill';
		await page.setViewportSize({ width: 1440, height: 900 });
		for (const preset of presets) {
			await page.goto(`/projects/${slugs.get(preset)}`);
			await expect(page.locator('.project-experience')).toHaveAttribute(
				'data-design-preset',
				preset
			);
			for (const presentation of ['paper', 'plain'] as const) {
				const at = `${preset} ${presentation}`;
				const block = page.locator(`#story-${presentation} .experience-story`);
				await expect(block).toHaveAttribute('data-presentation', presentation);
				const output = block.locator('.experience-story-output');
				const told = `${name} found something unexpected.`;
				// Typing registers only once the activity has hydrated.
				await expect(async () => {
					await block.getByLabel('Character').fill(name);
					await expect(output).toContainText(told, { timeout: 1000 });
				}).toPass();
				const controls = {
					fields: block.locator('.experience-story-fields'),
					options: block.locator('.experience-story-options'),
					actions: block.locator('.experience-actions')
				};
				for (const [part, control] of Object.entries(controls))
					await expect(control, `${at} ${part} on screen`).toBeVisible();

				await page.emulateMedia({ media: 'print' });
				for (const [part, control] of Object.entries(controls))
					await expect(control, `${at} ${part} in print`).toBeHidden();
				await expect(output, `${at} printed story`).toBeVisible();
				await expect(output, `${at} printed story`).toContainText(told);
				expect((await box(output)).height, `${at} printed story height`).toBeGreaterThan(0);

				await page.emulateMedia({ media: 'screen' });
				await expect(controls.fields, `${at} fields back on screen`).toBeVisible();
			}
		}
	});

	test('mobile previews size gutters, type and images like a phone of the same width', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		for (const preset of presets) {
			// The admin window stays wide, so only the article's own width can size the preview.
			const preview = await openMobilePreview(page, preset);
			const previewed = await sizing(preview, 'preview-');
			expect(previewed.width, `${preset} preview width`).toBeLessThanOrEqual(390);

			await page.setViewportSize({ width: 390, height: 900 });
			await page.goto(`/projects/${slugs.get(preset)}`);
			const root = page.locator('.project-experience');
			await expect(root).toHaveAttribute('data-design-preset', preset);
			const chrome = 390 - (await box(root)).width;
			await page.setViewportSize({ width: Math.round(previewed.width + chrome), height: 900 });
			const published = await sizing(root, '');
			expect(Math.abs(published.width - previewed.width), `${preset} article widths`).toBeLessThan(
				0.5
			);
			for (const key of Object.keys(published) as Array<keyof typeof published>)
				expect(
					Math.abs(previewed[key] - published[key]),
					`${preset} ${key}: preview ${previewed[key]} vs page ${published[key]}`
				).toBeLessThanOrEqual(0.5);
		}
	});

	test('keyboard focus on the chosen spotlight chapter differs from selection alone', async ({
		page
	}) => {
		await page.setViewportSize({ width: 1440, height: 900 });
		for (const preset of presets) {
			await page.goto(`/projects/${slugs.get(preset)}`);
			await expect(page.locator('.project-experience')).toHaveAttribute(
				'data-design-preset',
				preset
			);
			const section = page.locator('#feature');
			const chapters = section.locator('.experience-chapters button');
			const [first, second] = [chapters.nth(0), chapters.nth(1)];
			await choose(second);

			for (const tone of ['paper', 'ink'] as const) {
				// The ring must read on light and ink sections, so repaint the same player in ink.
				if (tone === 'ink')
					await section.evaluate((el) =>
						el.classList.replace('experience-tone-paper', 'experience-tone-ink')
					);
				await first.focus();
				await page.keyboard.press('Tab');
				await expect(second).toBeFocused();
				await expect(second).toHaveAttribute('aria-pressed', 'true');
				const focused = await indicator(second);
				expect(focused.focusVisible, `${preset} ${tone} keyboard focus`).toBe(true);

				await page.keyboard.press('Tab');
				await expect(second).not.toBeFocused();
				await expect(second).toHaveAttribute('aria-pressed', 'true');
				const resting = await indicator(second);
				expect(resting.focusVisible).toBe(false);

				const at = `${preset} ${tone}`;
				expect(focused.outline, `${at} focus outline`).not.toEqual(resting.outline);
				expect(focused.ring, `${at} focus ring`).not.toBe(resting.ring);
				expect(focused.ring, `${at} focus ring`).not.toBe('none');
				// One of the two ring colours must stand out from the section behind the player.
				expect(
					Math.max(
						contrast(focused.outlineColor, focused.behind),
						contrast(focused.ring, focused.behind)
					),
					`${at} focus ring contrast`
				).toBeGreaterThanOrEqual(3);
			}
		}
	});

	test('narrow sidebar players keep chapters after the player in reading and tab order', async ({
		page,
		request
	}) => {
		await seedAdminSession(page, request);
		const check = async (root: Locator, prefix: string, where: string) => {
			const block = root.locator(`#${prefix}sidebar .experience-video`);
			const video = block.locator('video');
			await expect
				.poll(() => video.evaluate((v: HTMLVideoElement) => v.readyState), {
					message: `${where} metadata`
				})
				.toBeGreaterThanOrEqual(1);
			expect(await video.evaluate((v: HTMLVideoElement) => v.error?.code ?? null)).toBeNull();
			const chapters = block.locator('.experience-chapters button');
			await expect(chapters).toHaveCount(2);
			const area = await box(block);
			const player = await box(block.locator('.experience-player'));
			const list = await box(block.locator('.experience-chapters'));
			const title = await box(block.locator(':scope > h4'));
			expect(list.y, `${where} chapters below the player`).toBeGreaterThanOrEqual(
				player.y + player.height - 0.5
			);
			expect(title.y, `${where} chapter title below the chapters`).toBeGreaterThanOrEqual(
				list.y + list.height - 0.5
			);
			for (const [name, part] of Object.entries({ player, list, title })) {
				expect(part.x, `${where} ${name} left edge`).toBeGreaterThanOrEqual(area.x - 0.5);
				expect(part.x + part.width, `${where} ${name} right edge`).toBeLessThanOrEqual(
					area.x + area.width + 0.5
				);
			}

			// Native controls may hold several tab stops inside the video before the chapters.
			await video.focus();
			const isActive = (target: Locator) => target.evaluate((el) => el === document.activeElement);
			for (let i = 0; i < 12 && !(await isActive(chapters.nth(0))); i++) {
				await page.keyboard.press('Tab');
				if (await isActive(chapters.nth(0))) break;
				expect(await isActive(video), `${where} tab stays in the player until the chapters`).toBe(
					true
				);
			}
			await expect(chapters.nth(0)).toBeFocused();
			await page.keyboard.press('Tab');
			await expect(chapters.nth(1)).toBeFocused();
			// Focus can animate scrolling; read every rectangle in the same browser task.
			const stops = await block.evaluate((el) =>
				[
					el.querySelector('.experience-player')!,
					...el.querySelectorAll('.experience-chapters button')
				].map((node) => {
					const { x, y, width, height } = node.getBoundingClientRect();
					return { x, y, width, height };
				})
			);
			for (let i = 1; i < stops.length; i++) {
				const [prev, next] = [stops[i - 1], stops[i]];
				const forward =
					next.y >= prev.y + prev.height - 0.5 ||
					(Math.abs(next.y - prev.y) < 1 && next.x >= prev.x + prev.width - 0.5);
				expect(
					forward,
					`${where} tab stop ${i} ${JSON.stringify(next)} follows ${JSON.stringify(prev)}`
				).toBe(true);
			}
		};
		for (const preset of presets) {
			await page.setViewportSize({ width: 390, height: 900 });
			await page.goto(`/projects/${slugs.get(preset)}`);
			const root = page.locator('.project-experience');
			await expect(root).toHaveAttribute('data-design-preset', preset);
			await check(root, '', `${preset} public 390`);

			const preview = await openMobilePreview(page, preset);
			await check(preview, 'preview-', `${preset} preview mobile`);
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
			activity: editor.getByLabel('Activity layout').first()
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
