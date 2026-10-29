import type { ExperienceBlock, ExperienceLesson, ExperienceStep, ProjectExperience } from './types';
export const experienceId = () => 'e' + crypto.randomUUID().replaceAll('-', '').slice(0, 16);
export const newLesson = (): ExperienceLesson => ({
	id: experienceId(),
	title: '',
	src: '',
	description: '',
	poster: '',
	transcript: '',
	credit: ''
});
export const newStep = (): ExperienceStep => ({
	id: experienceId(),
	title: '',
	body: '',
	code: '',
	language: ''
});
export function newBlock(type: ExperienceBlock['type']): ExperienceBlock {
	const id = experienceId();
	switch (type) {
		case 'text':
			return { id, type, body: '' };
		case 'image':
			return { id, type, image: { src: '', alt: '' }, caption: '' };
		case 'video':
			return { id, type, title: '', lessons: [newLesson()] };
		case 'code':
			return { id, type, title: '', steps: [newStep()] };
		case 'disclosure':
			return { id, type, title: '', body: '' };
		case 'story':
			return {
				id,
				type,
				title: 'Make a story',
				template:
					'{character} found something unexpected.\n\n[[pause]]\n\nA potato. It was a very good potato.',
				fields: [{ key: 'character', label: 'Character', initial: 'Sylvester' }],
				pauseMs: 1000,
				credit: ''
			};
	}
}
export function newExperience(kind: 'video' | 'code' | 'text' = 'text'): ProjectExperience {
	return {
		version: 1,
		enabled: true,
		layout: 'immersive',
		hero: { intro: '', tone: 'paper', imagePosition: 'right' },
		sections: [
			{
				id: experienceId(),
				title:
					kind === 'video' ? 'Watch and try' : kind === 'code' ? 'Inside the code' : 'The project',
				intro: '',
				tone: kind === 'video' ? 'aqua' : 'paper',
				layout: 'stack',
				blocks: [newBlock(kind)]
			}
		]
	};
}
