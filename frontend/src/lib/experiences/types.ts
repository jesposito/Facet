/** Portable version-one project experience document, shared with Facet Cloud. */
export type ExperienceTone =
	'paper' | 'aqua' | 'sand' | 'green' | 'pink' | 'ink' | 'yellow' | 'orange';
export interface ExperienceImage {
	src: string;
	alt: string;
	decorative?: boolean;
}
export interface ExperienceCaption {
	src: string;
	language: string;
	label: string;
}
export interface ExperienceLesson {
	id: string;
	title: string;
	src: string;
	description: string;
	poster: string;
	captions?: ExperienceCaption;
	transcript: string;
	credit: string;
}
export interface ExperienceStep {
	id: string;
	title: string;
	body: string;
	code: string;
	language: string;
	image?: ExperienceImage;
}
export type ExperienceBlock =
	| { id: string; type: 'text'; body: string }
	| { id: string; type: 'image'; image: ExperienceImage; caption: string }
	| {
			id: string;
			type: 'video';
			title: string;
			lessons: ExperienceLesson[];
			presentation?: 'stack' | 'sidebar' | 'spotlight';
	  }
	| { id: string; type: 'code'; title: string; steps: ExperienceStep[] }
	| { id: string; type: 'disclosure'; title: string; body: string }
	| {
			id: string;
			type: 'story';
			presentation?: 'plain' | 'paper';
			title: string;
			template: string;
			fields: { key: string; label: string; initial: string }[];
			pauseMs: number;
			credit: string;
	  };
export interface ExperienceSection {
	id: string;
	title: string;
	intro: string;
	tone: ExperienceTone;
	layout: 'stack' | 'split';
	headingLayout?: 'stack' | 'split';
	blocks: ExperienceBlock[];
}
export interface ProjectExperience {
	version: 1;
	design?: { preset: 'playful' | 'technical' | 'editorial'; width?: 'standard' | 'wide' };
	enabled: boolean;
	layout: 'immersive';
	hero: {
		intro: string;
		imageNote?: string;
		tone: ExperienceTone;
		imagePosition: 'left' | 'right';
		image?: ExperienceImage;
		action?: { label: string; href: string };
	};
	sections: ExperienceSection[];
}
export const experienceTones: ExperienceTone[] = [
	'paper',
	'aqua',
	'sand',
	'green',
	'pink',
	'ink',
	'yellow',
	'orange'
];

export function safeExperienceUrl(value: unknown, link = false): string {
	if (typeof value !== 'string' || /[\u0000-\u0020\u007f\\]/.test(value) || value.length > 2048)
		return '';
	if (value.startsWith('/') && !value.startsWith('//')) return value;
	if (link && /^#[a-zA-Z][\w-]*$/.test(value)) return value;
	try {
		const u = new URL(value);
		return u.protocol === 'https:' && !u.username && !u.password ? value : '';
	} catch {
		return '';
	}
}

/** Fail closed on old, malformed or future documents; keep the classic project available. */
export function readExperience(value: unknown): ProjectExperience | null {
	const object = (v: unknown): v is Record<string, any> =>
		!!v && typeof v === 'object' && !Array.isArray(v);
	const shape = (v: unknown, keys: string[]): v is Record<string, any> =>
		object(v) && Object.keys(v).every((k) => keys.includes(k));
	const str = (v: unknown, max = 500) =>
		typeof v === 'string' && Array.from(v).length <= max && (max > 500 || !/[\r\n]/.test(v));
	const url = (v: unknown, link = false) =>
		v === '' || (typeof v === 'string' && safeExperienceUrl(v, link) !== '');
	const img = (v: unknown) =>
		shape(v, ['src', 'alt', 'decorative']) &&
		url(v.src) &&
		str(v.alt) &&
		(v.decorative === undefined || typeof v.decorative === 'boolean');
	const tone = (v: unknown) => experienceTones.includes(v as ExperienceTone);
	const list = (v: unknown, max: number): v is any[] => Array.isArray(v) && v.length <= max;
	const ids = new Set<string>();
	const id = (v: unknown) =>
		typeof v === 'string' && /^[a-zA-Z][\w-]{0,63}$/.test(v) && !ids.has(v) && !!ids.add(v);
	try {
		if (
			!shape(value, ['version', 'enabled', 'layout', 'hero', 'sections', 'design']) ||
			value.version !== 1 ||
			value.enabled !== true ||
			value.layout !== 'immersive'
		)
			return null;
		if (new TextEncoder().encode(JSON.stringify(value)).length > 500_000) return null;
		if (
			value.design !== undefined &&
			(!shape(value.design, ['preset', 'width']) ||
				!['playful', 'technical', 'editorial'].includes(value.design.preset) ||
				(value.design.width !== undefined && !['standard', 'wide'].includes(value.design.width)))
		)
			return null;
		const h = value.hero;
		if (
			!shape(h, ['intro', 'tone', 'imagePosition', 'image', 'action', 'imageNote']) ||
			!str(h.intro, 50_000) ||
			(h.imageNote !== undefined && !str(h.imageNote, 200)) ||
			!tone(h.tone) ||
			!['left', 'right'].includes(h.imagePosition) ||
			(h.image !== undefined && !img(h.image))
		)
			return null;
		if (
			h.action !== undefined &&
			(!shape(h.action, ['label', 'href']) ||
				!str(h.action.label, 200) ||
				!url(h.action.href, true))
		)
			return null;
		if (!list(value.sections, 30)) return null;
		for (const s of value.sections) {
			if (
				!shape(s, ['id', 'title', 'intro', 'tone', 'layout', 'blocks', 'headingLayout']) ||
				!id(s.id) ||
				!str(s.title, 200) ||
				!str(s.intro, 50_000) ||
				!tone(s.tone) ||
				!['stack', 'split'].includes(s.layout) ||
				(s.headingLayout !== undefined && !['stack', 'split'].includes(s.headingLayout)) ||
				!list(s.blocks, 20)
			)
				return null;
			for (const b of s.blocks) {
				if (!object(b) || !id(b.id)) return null;
				switch (b.type) {
					case 'text':
						if (!shape(b, ['id', 'type', 'body']) || !str(b.body, 50_000)) return null;
						break;
					case 'image':
						if (!shape(b, ['id', 'type', 'image', 'caption']) || !img(b.image) || !str(b.caption))
							return null;
						break;
					case 'disclosure':
						if (
							!shape(b, ['id', 'type', 'title', 'body']) ||
							!str(b.title, 200) ||
							!str(b.body, 50_000)
						)
							return null;
						break;
					case 'video':
						if (
							!shape(b, ['id', 'type', 'title', 'lessons', 'presentation']) ||
							(b.presentation !== undefined &&
								!['stack', 'sidebar', 'spotlight'].includes(b.presentation)) ||
							!str(b.title, 200) ||
							!list(b.lessons, 30)
						)
							return null;
						for (const l of b.lessons) {
							if (
								!shape(l, [
									'id',
									'title',
									'src',
									'description',
									'poster',
									'captions',
									'transcript',
									'credit'
								]) ||
								!id(l.id) ||
								!str(l.title, 200) ||
								!url(l.src) ||
								!url(l.poster) ||
								!str(l.description) ||
								!str(l.transcript, 50_000) ||
								!str(l.credit)
							)
								return null;
							if (
								l.captions !== undefined &&
								(!shape(l.captions, ['src', 'language', 'label']) ||
									!url(l.captions.src) ||
									!str(l.captions.language) ||
									!str(l.captions.label, 200))
							)
								return null;
						}
						break;
					case 'code':
						if (
							!shape(b, ['id', 'type', 'title', 'steps']) ||
							!str(b.title, 200) ||
							!list(b.steps, 30)
						)
							return null;
						for (const step of b.steps)
							if (
								!shape(step, ['id', 'title', 'body', 'code', 'language', 'image']) ||
								!id(step.id) ||
								!str(step.title, 200) ||
								!str(step.body, 50_000) ||
								!str(step.code, 50_000) ||
								!str(step.language) ||
								(step.image !== undefined && !img(step.image))
							)
								return null;
						break;
					case 'story': {
						if (
							!shape(b, [
								'id',
								'type',
								'title',
								'template',
								'fields',
								'pauseMs',
								'credit',
								'presentation'
							]) ||
							(b.presentation !== undefined && !['plain', 'paper'].includes(b.presentation)) ||
							!str(b.title, 200) ||
							!str(b.template, 50_000) ||
							!str(b.credit) ||
							!list(b.fields, 12) ||
							!Number.isFinite(b.pauseMs) ||
							b.pauseMs < 0 ||
							b.pauseMs > 5000
						)
							return null;
						const keys = new Set<string>();
						for (const f of b.fields) {
							if (
								!shape(f, ['key', 'label', 'initial']) ||
								typeof f.key !== 'string' ||
								!/^[a-zA-Z][\w-]{0,31}$/.test(f.key) ||
								keys.has(f.key) ||
								!str(f.label, 200) ||
								!str(f.initial)
							)
								return null;
							keys.add(f.key);
						}
						break;
					}
					default:
						return null;
				}
			}
		}
		return value as unknown as ProjectExperience;
	} catch {
		return null;
	}
}
