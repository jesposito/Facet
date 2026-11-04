import { pb, type ItemSelection, type ViewSection } from './pocketbase';

export function normalizeItemSelection(value: unknown): ItemSelection {
	return value === 'all_public' ? 'all_public' : 'selected';
}

export function allPublicHintKey(sectionKey: string): string {
	if (sectionKey === 'contacts') return 'admin.view_editor.sections.all_public_hint_contacts';
	if (sectionKey === 'testimonials') {
		return 'admin.view_editor.sections.all_public_hint_testimonials';
	}
	return 'admin.view_editor.sections.all_public_hint';
}

interface SectionContentItem {
	id: string;
	is_draft?: boolean;
	data: Record<string, unknown>;
}

// The order all_public publishes in. allPublicQuery in
// backend/hooks/view_helpers.go keeps the same sorts.
const ALL_PUBLIC_SORT: Record<string, string> = {
	experience: '-sort_order,-start_date,id',
	projects: '-is_featured,-sort_order,id',
	education: '-sort_order,-end_date,id',
	certifications: 'issuer,sort_order,-issue_date,id',
	awards: '-sort_order,-awarded_at,id',
	skills: 'category,sort_order,id',
	posts: '-published_at,id',
	talks: '-sort_order,-date,id',
	testimonials: '-featured,-sort_order,id',
	contacts: '-is_primary,-sort_order,id'
};

function compareBySort(a: SectionContentItem, b: SectionContentItem, sort: string): number {
	for (const term of sort.split(',')) {
		const descending = term.startsWith('-');
		const field = descending ? term.slice(1) : term;
		const left = (a.data[field] ?? '') as string;
		const right = (b.data[field] ?? '') as string;
		if (left === right) continue;
		const order = left < right ? -1 : 1;
		return descending ? -order : order;
	}
	return 0;
}

// Contacts have no visibility field. An empty scope means every page; a
// nonempty one is opt-in. This never changes the contact's stored scope.
function isContactAvailable(item: SectionContentItem, viewId: string): boolean {
	const scope = item.data.view_visibility as Record<string, boolean> | null | undefined;
	return !scope || Object.keys(scope).length === 0 || scope[viewId] === true;
}

// What all_public publishes: a per-Facet private override doesn't count as public.
export function isAllPublicItem(
	sectionKey: string,
	item: SectionContentItem,
	viewId = ''
): boolean {
	if (sectionKey === 'contacts') return isContactAvailable(item, viewId);
	if (sectionKey === 'testimonials') {
		const visibility = item.data.visibility;
		return item.data.status === 'approved' && (visibility === 'public' || !visibility);
	}
	return item.data.visibility === 'public' && !item.is_draft;
}

function isSelectedItemVisible(
	sectionKey: string,
	item: SectionContentItem,
	viewId: string,
	privateSelectionPublishes: boolean
): boolean {
	if (sectionKey === 'contacts') return isContactAvailable(item, viewId);
	if (sectionKey === 'testimonials' && item.data.status !== 'approved') return false;
	if (item.is_draft) return false;
	if (item.data.visibility !== 'private') return true;
	const override = item.data.view_visibility as Record<string, boolean> | null | undefined;
	return privateSelectionPublishes || override?.[viewId] === true;
}

// Which items a section publishes, in published order. This mirrors the
// /api/view/{slug}/data handler so the editor preview can't promise anything else.
export function resolveSectionItems<T extends SectionContentItem>(
	sectionKey: string,
	config: Pick<ViewSection, 'itemSelection' | 'list'> & { items: string[] },
	items: T[],
	viewId = '',
	privateSelectionPublishes = false
): T[] {
	const sort = ALL_PUBLIC_SORT[sectionKey] ?? 'id';
	if (sectionKey === 'testimonials' && config.list) {
		return items
			.filter((item) => isAllPublicItem(sectionKey, item) && item.data.list === config.list)
			.sort((a, b) => compareBySort(a, b, sort));
	}
	if (normalizeItemSelection(config.itemSelection) === 'all_public') {
		return items
			.filter((item) => isAllPublicItem(sectionKey, item, viewId))
			.sort((a, b) => compareBySort(a, b, sort));
	}
	const byId = new Map(items.map((item) => [item.id, item]));
	return config.items.flatMap((id) => {
		const item = byId.get(id);
		return item && isSelectedItemVisible(sectionKey, item, viewId, privateSelectionPublishes)
			? [item]
			: [];
	});
}

// Asks the server to show a saved Facet's selected private items on it. The
// server reads the saved Facet and records itself, so a stale page can't drop
// another Facet's grant. False means "not confirmed": the request failed, or it
// worked and the reply got lost. Retrying is safe either way.
export async function confirmPrivateItems(viewId: string): Promise<boolean> {
	try {
		await pb.send(`/api/view/${viewId}/private-items`, {
			method: 'POST',
			requestKey: null,
			signal: AbortSignal.timeout(30_000)
		});
		return true;
	} catch (err) {
		console.error(`Failed to confirm private items on Facet ${viewId}:`, err);
		return false;
	}
}
