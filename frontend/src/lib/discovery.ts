import { matchSlug } from '../params';

export type PublishingKind = 'project' | 'post' | 'talk';
type Search = Pick<URLSearchParams, 'get' | 'getAll'>;
type Context = { fromView?: string | null; filter?: string | null };
const paths = { project: '/projects', post: '/posts', talk: '/talks' };
const filterKeys = { project: null, post: 'tag', talk: 'year' };
const unsafeCharacters = /[\\\u0000-\u001f\u007f]/;

// Navigation hints never grant access to a Facet or its content.
export function navigationFacet(value?: string | null): string {
	return value && matchSlug(value) ? value : '';
}

function filterValue(value?: string | null): string {
	return value && value.length <= 160 && !unsafeCharacters.test(value) ? value : '';
}

export function discoveryListBackHref(fromView?: string | null): string {
	const from = navigationFacet(fromView);
	return from ? `/${from}` : '/';
}

export function discoveryCollectionHref(kind: PublishingKind, context: Context = {}): string {
	const params = new URLSearchParams();
	const from = navigationFacet(context.fromView);
	if (from) params.set('from', from);
	const key = filterKeys[kind];
	const filter = filterValue(context.filter);
	if (key && filter) params.set(key, filter);
	const query = params.toString();
	return paths[kind] + (query ? `?${query}` : '');
}

/** Cards in a list return to that list, including its current filter. */
export function discoveryDetailHref(kind: PublishingKind, slug: string, context: Context = {}): string {
	const params = new URLSearchParams({ return_to: discoveryCollectionHref(kind, context) });
	const from = navigationFacet(context.fromView);
	if (from) params.set('from', from);
	return `${paths[kind]}/${encodeURIComponent(slug)}?${params}`;
}

/** A section belongs to either the homepage or a named Facet. */
export function discoverySectionHref(kind: PublishingKind, slug: string, fromView?: string | null): string {
	const from = navigationFacet(fromView);
	const params = new URLSearchParams(from ? { from } : { return_to: '/' });
	return `${paths[kind]}/${encodeURIComponent(slug)}?${params}`;
}

export function discoveryReturnHref(kind: PublishingKind, search: Search, fromView?: string | null): string {
	const from = navigationFacet(fromView);
	// Self-hosted Facet has no projects index. Bare project links return home.
	const fallback = from ? `/${from}` : kind === 'project' ? '/' : paths[kind];
	if (search.getAll('return_to').length !== 1) return fallback;
	const raw = search.get('return_to')!;
	if (!raw || raw.length > 512 || !raw.startsWith('/') || raw.startsWith('//') || unsafeCharacters.test(raw)) return fallback;
	if (raw === '/' || (from && raw === `/${from}`)) return raw;
	if (kind === 'project') return fallback;
	try {
		const target = new URL(raw, 'https://facet.invalid');
		// Require the literal collection path, rejecting normalized dot segments.
		if (raw.split('?')[0] !== paths[kind] || target.origin !== 'https://facet.invalid' || target.hash) return fallback;
		const key = filterKeys[kind];
		for (const name of target.searchParams.keys()) {
			if ((name !== 'from' && name !== key) || target.searchParams.getAll(name).length !== 1) return fallback;
		}
		const rawFrom = target.searchParams.get('from');
		return discoveryCollectionHref(kind, {
			fromView: rawFrom === from ? from : null,
			filter: key ? target.searchParams.get(key) : null
		});
	} catch {
		return fallback;
	}
}

type Translate = (key: string, options?: { values: Record<string, string> }) => string;
export function discoveryDetailBack(kind: PublishingKind, search: Search, fromView: string | null, translate: Translate) {
	const href = discoveryReturnHref(kind, search, fromView);
	const from = navigationFacet(fromView);
	const destination = href === '/' ? translate('public.nav.home')
		: from && href === `/${from}` ? from : translate(`public.sections.${paths[kind].slice(1)}`);
	return { href, label: translate('public.nav.back_to', { values: { page: destination } }) };
}
