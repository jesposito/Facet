import { test, expect } from '@playwright/test';
import { readExperience } from '../src/lib/experiences/types';
import { newExperience, newBlock } from '../src/lib/experiences/templates';

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
	const mutations = [
		(d: any) => (d.design = null),
		(d: any) => (d.design = { preset: 'unknown' }),
		(d: any) => (d.design = { preset: 'playful', width: 'enormous' }),
		(d: any) => (d.design = { preset: 'playful', css: 'body{display:none}' }),
		(d: any) => (d.hero.imageNote = 'two\nlines'),
		(d: any) => (d.hero.imageNote = 'a'.repeat(201)),
		(d: any) => (d.hero.image = { src: '', alt: '', decorative: 'true' }),
		(d: any) => (d.sections[0].headingLayout = 'columns'),
		(d: any) => (d.sections[0].blocks[0].presentation = 'paper')
	];
	for (const mutate of mutations) {
		const doc = structuredClone(original);
		mutate(doc);
		expect(readExperience(doc)).toBeNull();
	}
});
