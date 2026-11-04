import { get } from 'svelte/store';
import { readAllRecords } from '$lib/admin/read-all-records';
import { demoMode, getCollectionName } from '$lib/stores/demo';

type CatalogRecord = Record<string, unknown> & { id: string };

// PocketBase returns null for a Facet with no saved sections.
export function isSavedSections(value: unknown): boolean {
	return (
		value === null ||
		value === undefined ||
		(Array.isArray(value) && value.every((section) => typeof section?.section === 'string'))
	);
}

// Everything a Facet save acts on. One failure fails the whole read, so a missing list can't
// pass for an empty one.
export async function readFacetCatalog(
	sectionDefs: Record<string, { collection: string }>,
	signal: AbortSignal,
	expand?: string
) {
	const read = async (name: string, query: Record<string, string>) => {
		try {
			return await readAllRecords<CatalogRecord>(getCollectionName(name), query, signal);
		} catch (err) {
			// Not every collection has a demo_ shadow table, and a missing table holds nothing.
			if (get(demoMode) && err instanceof Error && err.message.includes('(404)')) return [];
			throw err;
		}
	};
	const keys = Object.keys(sectionDefs);
	const [custom, ...lists] = await Promise.all([
		read('custom_content', { sort: 'sort_order' }),
		...keys.map((key) =>
			read(sectionDefs[key].collection, {
				// Testimonials only show approved ones in view editor
				...(key === 'testimonials'
					? { sort: '-featured,-sort_order', filter: 'status = "approved"' }
					: { sort: '-id' }),
				...(expand ? { expand } : {})
			})
		)
	]);
	return { custom, lists: Object.fromEntries(keys.map((key, index) => [key, lists[index]])) };
}
