/**
 * Shared filter state helper for admin list pages
 * 
 * Provides reactive state for:
 * - Search query
 * - Visibility filter (all, public, unlisted, private)
 * - Draft filter (all, published, drafts)
 * - Tag filter (selected admin tag IDs)
 * 
 * With URL synchronization for bookmarkable filter states
 */

import { page } from '$app/stores';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';

export interface FilterState {
	searchQuery: string;
	visibilityFilter: 'all' | 'public' | 'unlisted' | 'private';
	draftFilter: 'all' | 'published' | 'drafts';
	selectedTagIds: string[];
}

export interface FilterConfig {
	enableSearch?: boolean;
	enableVisibilityFilter?: boolean;
	enableDraftFilter?: boolean;
	enableTagFilter?: boolean;
	searchPlaceholder?: string;
}

export function createFilterState(config: FilterConfig = {}) {
	const {
		enableSearch = true,
		enableVisibilityFilter = true,
		enableDraftFilter = true,
		enableTagFilter = true,
		searchPlaceholder = 'Search...'
	} = config;

	// Initialize state from URL params if available
	let state = $state<FilterState>({
		searchQuery: '',
		visibilityFilter: 'all',
		draftFilter: 'all',
		selectedTagIds: []
	});

	// Load state from URL on creation (browser only)
	if (browser) {
		loadFromUrl();
	}

	function loadFromUrl() {
		const urlParams = new URLSearchParams(window.location.search);
		
		state.searchQuery = urlParams.get('search') || '';
		state.visibilityFilter = (urlParams.get('visibility') as FilterState['visibilityFilter']) || 'all';
		state.draftFilter = (urlParams.get('draft') as FilterState['draftFilter']) || 'all';
		
		const tagParam = urlParams.get('tags');
		state.selectedTagIds = tagParam ? tagParam.split(',').filter(Boolean) : [];
	}

	function updateUrl() {
		if (!browser) return;
		
		const urlParams = new URLSearchParams();
		
		if (state.searchQuery) urlParams.set('search', state.searchQuery);
		if (state.visibilityFilter !== 'all') urlParams.set('visibility', state.visibilityFilter);
		if (state.draftFilter !== 'all') urlParams.set('draft', state.draftFilter);
		if (state.selectedTagIds.length > 0) urlParams.set('tags', state.selectedTagIds.join(','));
		
		const newUrl = urlParams.toString() ? `?${urlParams.toString()}` : window.location.pathname;
		window.history.replaceState({}, '', newUrl);
	}

	function setSearchQuery(query: string) {
		state.searchQuery = query;
		updateUrl();
	}

	function setVisibilityFilter(filter: FilterState['visibilityFilter']) {
		state.visibilityFilter = filter;
		updateUrl();
	}

	function setDraftFilter(filter: FilterState['draftFilter']) {
		state.draftFilter = filter;
		updateUrl();
	}

	function setSelectedTagIds(tagIds: string[]) {
		state.selectedTagIds = [...tagIds];
		updateUrl();
	}

	function toggleTag(tagId: string) {
		const index = state.selectedTagIds.indexOf(tagId);
		if (index === -1) {
			state.selectedTagIds = [...state.selectedTagIds, tagId];
		} else {
			state.selectedTagIds = state.selectedTagIds.filter(id => id !== tagId);
		}
		updateUrl();
	}

	function clearAllFilters() {
		state.searchQuery = '';
		state.visibilityFilter = 'all';
		state.draftFilter = 'all';
		state.selectedTagIds = [];
		updateUrl();
	}

	function hasActiveFilters(): boolean {
		return !!(
			state.searchQuery ||
			state.visibilityFilter !== 'all' ||
			state.draftFilter !== 'all' ||
			state.selectedTagIds.length > 0
		);
	}

	/**
	 * Filter function for array of items
	 * @param items Array of items to filter
	 * @param getSearchableText Function to extract searchable text from item
	 * @returns Filtered array
	 */
	function filterItems<T>(
		items: T[],
		getSearchableText: (item: T) => string,
		getVisibility: (item: T) => string = () => 'public',
		getIsDraft: (item: T) => boolean = () => false,
		getTagIds: (item: T) => string[] = () => []
	): T[] {
		return items.filter(item => {
			// Search filter
			if (enableSearch && state.searchQuery) {
				const searchText = getSearchableText(item).toLowerCase();
				const query = state.searchQuery.toLowerCase();
				if (!searchText.includes(query)) return false;
			}

			// Visibility filter
			if (enableVisibilityFilter && state.visibilityFilter !== 'all') {
				const itemVisibility = getVisibility(item);
				if (itemVisibility !== state.visibilityFilter) return false;
			}

			// Draft filter
			if (enableDraftFilter && state.draftFilter !== 'all') {
				const isDraft = getIsDraft(item);
				if (state.draftFilter === 'published' && isDraft) return false;
				if (state.draftFilter === 'drafts' && !isDraft) return false;
			}

			// Tag filter
			if (enableTagFilter && state.selectedTagIds.length > 0) {
				const itemTagIds = getTagIds(item);
				const hasMatchingTag = state.selectedTagIds.some(tagId => itemTagIds.includes(tagId));
				if (!hasMatchingTag) return false;
			}

			return true;
		});
	}

	return {
		// State (read-only getters)
		get searchQuery() { return state.searchQuery; },
		get visibilityFilter() { return state.visibilityFilter; },
		get draftFilter() { return state.draftFilter; },
		get selectedTagIds() { return state.selectedTagIds; },
		
		// Actions
		setSearchQuery,
		setVisibilityFilter,
		setDraftFilter,
		setSelectedTagIds,
		toggleTag,
		clearAllFilters,
		loadFromUrl,
		
		// Utilities
		hasActiveFilters,
		filterItems,
		
		// Config
		config: {
			enableSearch,
			enableVisibilityFilter,
			enableDraftFilter,
			enableTagFilter,
			searchPlaceholder
		}
	};
}

export type FilterStateStore = ReturnType<typeof createFilterState>;