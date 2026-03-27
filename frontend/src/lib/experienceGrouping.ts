import type { Experience } from './pocketbase';

export interface CompanyGroup {
	type: 'group';
	groupId: string;
	company: string;
	companyLogo?: string;
	companyLogoUrl?: string;
	companyLogoLibraryUrl?: string;
	logoBackground?: string;
	overallStartDate?: string;
	overallEndDate?: string; // undefined means "present" (at least one position is current)
	positions: Experience[];
}

export type GroupedExperienceItem = Experience | CompanyGroup;

/**
 * Groups experiences by company_group_id (if present) or by exact company name match.
 * - Single-item groups are returned as-is (unwrapped Experience).
 * - Multi-item groups are wrapped in a CompanyGroup.
 * - Preserves original sort order: each group appears at the position of its first member.
 */
export function groupExperiencesByCompany(items: Experience[]): GroupedExperienceItem[] {
	if (!items || items.length === 0) {
		return [];
	}

	// Build an ordered list of group keys in first-seen order, and map key -> items
	const groupKeyOrder: string[] = [];
	const groupMap = new Map<string, Experience[]>();

	for (const item of items) {
		// Determine the group key: use company_group_id if set, otherwise fall back to company name
		// Items with no company name go through as singletons using their id as the key
		const groupKey = item.company_group_id
			? `id:${item.company_group_id}`
			: item.company
				? `company:${item.company}`
				: `singleton:${item.id}`;

		if (!groupMap.has(groupKey)) {
			groupKeyOrder.push(groupKey);
			groupMap.set(groupKey, []);
		}
		groupMap.get(groupKey)!.push(item);
	}

	// Build the result array
	const result: GroupedExperienceItem[] = [];

	for (const key of groupKeyOrder) {
		const group = groupMap.get(key)!;

		if (group.length === 1) {
			// Single item - return as-is
			result.push(group[0]);
		} else {
			// Multi-item group - create a CompanyGroup
			// Use company name and logo from the FIRST item (most recent by sort order)
			const first = group[0];

			// Calculate overall date range
			// - overallStartDate: earliest start_date across all positions
			// - overallEndDate: latest end_date; if ANY position has no end_date, overall has no end_date (present)
			let overallStartDate: string | undefined;
			let overallEndDate: string | undefined = undefined;
			let anyPresent = false;

			for (const pos of group) {
				// Track the earliest start date
				if (pos.start_date) {
					if (!overallStartDate || pos.start_date < overallStartDate) {
						overallStartDate = pos.start_date;
					}
				}
				// If any position has no end_date, the group is still "present"
				if (!pos.end_date) {
					anyPresent = true;
				} else if (!anyPresent) {
					// Track the latest end date only if no position is current yet
					if (!overallEndDate || pos.end_date > overallEndDate) {
						overallEndDate = pos.end_date;
					}
				}
			}

			// If any position is present, overallEndDate stays undefined
			if (anyPresent) {
				overallEndDate = undefined;
			}

			const companyGroup: CompanyGroup = {
				type: 'group',
				groupId: key,
				company: first.company,
				companyLogo: first.company_logo,
				companyLogoUrl: first.company_logo_url,
				companyLogoLibraryUrl: first.company_logo_library_url,
				logoBackground: first.logo_background,
				overallStartDate,
				overallEndDate,
				positions: group
			};

			result.push(companyGroup);
		}
	}

	return result;
}
