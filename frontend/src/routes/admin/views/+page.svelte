<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { pb, type View, type ShareToken } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { siteNavStore } from '$lib/stores/siteNav';
	import { icon } from '$lib/icons';
	import { formatDate } from '$lib/utils';
	import PageHelp from '$components/admin/PageHelp.svelte';
	import EmptyState from '$components/admin/EmptyState.svelte';

	type VisibilityFilter = 'all' | 'public' | 'unlisted' | 'password' | 'private';

	let loading = $state(true);
	let views: View[] = $state([]);
	let tokenCounts: Record<string, number> = $state({});
	let mounted = false;

	let search = $state('');
	let filterVisibility = $state<VisibilityFilter>('all');
	let filterInNav = $state(false);
	let filterHasShareLinks = $state(false);
	let filterInactive = $state(false);

	let searchInput: HTMLInputElement | null = $state(null);
	let focusedIndex = $state(0);
	let showKeyboardHelp = $state(false);
	let helpDialog: HTMLDivElement | null = $state(null);
	let helpTrigger: HTMLButtonElement | null = $state(null);
	let helpCloseBtn: HTMLButtonElement | null = $state(null);

	let inNavIds = $derived(new Set($siteNavStore.items.map((item) => item.viewId)));
	let siteNavEnabled = $derived($siteNavStore.enabled);

	let anyFilterActive = $derived(
		search.trim().length > 0 ||
			filterVisibility !== 'all' ||
			filterInNav ||
			filterHasShareLinks ||
			filterInactive
	);

	let filtered = $derived.by(() => {
		let out = views;
		const query = search.trim().toLowerCase();
		if (query) {
			out = out.filter((v) => {
				const name = String(v.name ?? '').toLowerCase();
				const slug = String(v.slug ?? '').toLowerCase();
				return name.includes(query) || slug.includes(query);
			});
		}
		if (filterVisibility !== 'all') {
			out = out.filter((v) => v.visibility === filterVisibility);
		}
		if (filterInNav) {
			out = out.filter((v) => inNavIds.has(v.id));
		}
		if (filterHasShareLinks) {
			out = out.filter((v) => (tokenCounts[v.id] ?? 0) > 0);
		}
		if (filterInactive) {
			out = out.filter((v) => !v.is_active);
		}
		return out;
	});

	// Keep focusedIndex in bounds whenever filtered changes
	$effect(() => {
		if (focusedIndex >= filtered.length) {
			focusedIndex = Math.max(0, filtered.length - 1);
		}
	});

	onMount(() => {
		mounted = true;
		loadViews();
		loadTokenCounts();
		window.addEventListener('keydown', handleGlobalKey);
	});

	onDestroy(() => {
		mounted = false;
		if (typeof window !== 'undefined') {
			window.removeEventListener('keydown', handleGlobalKey);
		}
	});

	function isTextEditingTarget(el: EventTarget | null): boolean {
		if (!el || !(el instanceof HTMLElement)) return false;
		const tag = el.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true;
		if (el.isContentEditable) return true;
		return false;
	}

	function handleGlobalKey(e: KeyboardEvent) {
		if (e.ctrlKey || e.metaKey || e.altKey) return;

		// '/' focuses search — only when not in an input
		if (e.key === '/') {
			if (isTextEditingTarget(e.target)) return;
			e.preventDefault();
			searchInput?.focus();
			searchInput?.select();
			return;
		}

		// '?' opens keyboard help — only when not in an input
		if (e.key === '?') {
			if (isTextEditingTarget(e.target)) return;
			e.preventDefault();
			openKeyboardHelp();
			return;
		}

		// Escape closes keyboard help modal
		if (e.key === 'Escape' && showKeyboardHelp) {
			e.preventDefault();
			closeKeyboardHelp();
		}
	}

	async function loadViews() {
		loading = true;
		try {
			const result = await collection('views').getList(1, 100, {
				sort: '-updated'
			});
			views = result.items as unknown as View[];
		} catch (err) {
			if (err instanceof Error && (err.message.includes('autocancelled') || err.name === 'AbortError')) {
				return;
			}
			console.error('Failed to load facets:', err);
			if (mounted) {
				toasts.add('error', $t('admin.views.toast_load_failed'));
			}
		} finally {
			loading = false;
		}
	}

	async function loadTokenCounts() {
		try {
			const tokens = await collection('share_tokens').getFullList({
				fields: 'view_id'
			});
			const counts: Record<string, number> = {};
			for (const tok of tokens as unknown as ShareToken[]) {
				const vid = tok.view_id;
				if (vid) counts[vid] = (counts[vid] ?? 0) + 1;
			}
			tokenCounts = counts;
		} catch (err) {
			if (err instanceof Error && (err.message.includes('autocancelled') || err.name === 'AbortError')) {
				return;
			}
			console.debug('[views list] share_tokens count fetch skipped:', err);
		}
	}

	async function toggleViewActive(view: View) {
		try {
			await collection('views').update(view.id, {
				is_active: !view.is_active
			});
			await loadViews();
		} catch (err) {
			toasts.add('error', $t('admin.views.toast_update_failed'));
		}
	}

	async function deleteView(view: View) {
		const confirmed = await confirm({
			title: $t('admin.views.delete_confirm_title'),
			message: $t('admin.views.delete_confirm_message'),
			confirmText: $t('admin.views.delete_confirm_button'),
			danger: true
		});
		if (!confirmed) return;
		try {
			await collection('views').delete(view.id);
			toasts.add('success', $t('admin.views.toast_deleted'));
			await loadViews();
		} catch (err) {
			toasts.add('error', $t('admin.views.toast_delete_failed'));
		}
	}

	function getViewUrl(slug: string): string {
		return `${window.location.origin}/${slug}`;
	}

	function copyViewUrl(slug: string) {
		navigator.clipboard.writeText(getViewUrl(slug));
		toasts.add('success', $t('admin.views.toast_url_copied'));
	}

	function clearFilters() {
		search = '';
		filterVisibility = 'all';
		filterInNav = false;
		filterHasShareLinks = false;
		filterInactive = false;
	}

	function visibilityPillClass(v: string): string {
		switch (v) {
			case 'public':
				return 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300';
			case 'unlisted':
				return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300';
			case 'password':
				return 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300';
			default:
				return 'bg-gray-200 text-gray-700 dark:bg-gray-700 dark:text-gray-300';
		}
	}

	function rowAriaLabel(view: View): string {
		const active = view.is_active ? '' : ', inactive';
		const inNav = inNavIds.has(view.id) ? ', in nav' : '';
		const tokenCount = tokenCounts[view.id] ?? 0;
		const sharePart = tokenCount === 0 ? '' : `, ${tokenCount} share links`;
		const viewCount = (view as unknown as { view_count?: number }).view_count ?? 0;
		return `${view.name}, /${view.slug}, ${view.visibility}${active}${inNav}${sharePart}, ${viewCount.toLocaleString()} views`;
	}

	function getViewCount(view: View): number {
		return (view as unknown as { view_count?: number }).view_count ?? 0;
	}

	function getLastViewedAt(view: View): string | undefined {
		return (view as unknown as { last_viewed_at?: string }).last_viewed_at;
	}

	function focusRow(index: number) {
		focusedIndex = index;
		// Focus the actual row element after Svelte updates the DOM
		queueMicrotask(() => {
			const row = document.querySelector<HTMLTableRowElement>(`tr[data-row-index="${index}"]`);
			row?.focus();
		});
	}

	function handleTableKeydown(e: KeyboardEvent) {
		if (filtered.length === 0) return;
		const target = e.target as HTMLElement | null;
		if (target && target.tagName === 'BUTTON') {
			// Let button click handlers run normally on Enter/Space
			if (e.key === 'Enter' || e.key === ' ') return;
		}
		if (target && target.tagName === 'A') {
			if (e.key === 'Enter') return;
		}

		switch (e.key) {
			case 'j':
			case 'ArrowDown': {
				e.preventDefault();
				const next = Math.min(focusedIndex + 1, filtered.length - 1);
				focusRow(next);
				break;
			}
			case 'k':
			case 'ArrowUp': {
				e.preventDefault();
				const prev = Math.max(focusedIndex - 1, 0);
				focusRow(prev);
				break;
			}
			case 'Enter':
			case 'e': {
				const view = filtered[focusedIndex];
				if (view) {
					e.preventDefault();
					goto(`/admin/views/${view.id}`);
				}
				break;
			}
			case 'x': {
				const view = filtered[focusedIndex];
				if (view) {
					e.preventDefault();
					deleteView(view);
				}
				break;
			}
		}
	}

	function openKeyboardHelp() {
		helpTrigger = (document.activeElement as HTMLButtonElement) ?? helpTrigger;
		showKeyboardHelp = true;
		queueMicrotask(() => {
			helpCloseBtn?.focus();
		});
	}

	function closeKeyboardHelp() {
		showKeyboardHelp = false;
		queueMicrotask(() => {
			helpTrigger?.focus();
		});
	}

	function handleHelpKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			closeKeyboardHelp();
			return;
		}
		// Simple focus trap: only one tabbable element (close button), so trap Tab
		if (e.key === 'Tab') {
			e.preventDefault();
			helpCloseBtn?.focus();
		}
	}

	const kbdClass = 'px-2 py-0.5 rounded border border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-900 font-mono text-xs';
	let shortcutsList = $derived([
		{ label: $t('admin.views.keyboard_shortcut_focus_search'), keys: ['/'] },
		{ label: $t('admin.views.keyboard_shortcut_next_row'), keys: ['j', '↓'] },
		{ label: $t('admin.views.keyboard_shortcut_prev_row'), keys: ['k', '↑'] },
		{ label: $t('admin.views.keyboard_shortcut_open_row'), keys: ['Enter', 'e'] },
		{ label: $t('admin.views.keyboard_shortcut_delete_row'), keys: ['x'] },
		{ label: $t('admin.views.keyboard_shortcut_help'), keys: ['?'] }
	]);
</script>

<svelte:head>
	<title>{$t('admin.views.title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<PageHelp pageKey="views">
		<p><strong>Facets</strong> are curated versions of your profile for different audiences.</p>
		<p>Create a "Recruiter" facet with your full work history, a "Conference" facet highlighting talks, or a "Consulting" facet for client work. Each facet shows exactly what you want that audience to see.</p>
		<p><strong>Tip:</strong> Your homepage (<code>/</code>) shows all public content. Facets let you create focused views at URLs like <code>/recruiter</code> with only the sections and items you choose.</p>
	</PageHelp>

	<div class="flex items-center justify-between mb-4">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.views.title')}</h1>
		<div class="flex items-center gap-2">
			{#if !loading && views.length > 0}
				<button
					type="button"
					class="btn btn-ghost btn-sm hidden md:inline-flex"
					onclick={openKeyboardHelp}
					aria-label={$t('admin.views.keyboard_help_show')}
					title={$t('admin.views.keyboard_help_show')}
				>
					<kbd class="font-mono text-xs">?</kbd>
				</button>
			{/if}
			<a href="/admin/views/new" class="btn btn-primary">{$t('admin.views.new_button')}</a>
		</div>
	</div>

	<p class="text-gray-600 dark:text-gray-400 mb-6">
		{$t('admin.views.intro')}
	</p>

	{#if !loading && views.length > 0}
		<div class="card p-3 mb-4 space-y-3">
			<div class="flex flex-col sm:flex-row gap-2 sm:items-center">
				<div class="relative flex-1 min-w-0">
					<label for="views-search" class="sr-only">{$t('admin.views.search_placeholder')}</label>
					<input
						id="views-search"
						bind:this={searchInput}
						bind:value={search}
						type="search"
						placeholder={$t('admin.views.search_placeholder')}
						autocomplete="off"
						aria-label={$t('admin.views.search_placeholder')}
						aria-keyshortcuts="/"
						class="input w-full pe-16"
					/>
					<span class="pointer-events-none absolute inset-y-0 right-2 flex items-center gap-1 text-xs text-gray-400 dark:text-gray-500">
						<kbd class="px-1.5 py-0.5 rounded border border-gray-300 dark:border-gray-600 bg-gray-50 dark:bg-gray-800 font-mono">/</kbd>
					</span>
				</div>

				<label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 shrink-0">
					<span class="sr-only sm:not-sr-only">{$t('admin.views.filter_visibility_label')}</span>
					<select bind:value={filterVisibility} class="input input-sm">
						<option value="all">{$t('admin.views.filter_visibility_all')}</option>
						<option value="public">{$t('admin.views.filter_visibility_public')}</option>
						<option value="unlisted">{$t('admin.views.filter_visibility_unlisted')}</option>
						<option value="password">{$t('admin.views.filter_visibility_password')}</option>
						<option value="private">{$t('admin.views.filter_visibility_private')}</option>
					</select>
				</label>
			</div>

			<div class="flex flex-wrap gap-2" role="group" aria-label={$t('admin.views.filter_visibility_label')}>
				{#if siteNavEnabled}
					<button
						type="button"
						class="px-3 py-1 rounded-full text-xs font-medium transition-colors border
							{filterInNav
								? 'bg-primary-600 text-white border-primary-600 dark:bg-primary-500 dark:border-primary-500'
								: 'bg-gray-100 text-gray-700 border-gray-200 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700'}"
						aria-pressed={filterInNav}
						onclick={() => (filterInNav = !filterInNav)}
					>
						{$t('admin.views.chip_in_nav')}
					</button>
				{/if}
				<button
					type="button"
					class="px-3 py-1 rounded-full text-xs font-medium transition-colors border
						{filterHasShareLinks
							? 'bg-primary-600 text-white border-primary-600 dark:bg-primary-500 dark:border-primary-500'
							: 'bg-gray-100 text-gray-700 border-gray-200 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700'}"
					aria-pressed={filterHasShareLinks}
					onclick={() => (filterHasShareLinks = !filterHasShareLinks)}
				>
					{$t('admin.views.chip_share_links')}
				</button>
				<button
					type="button"
					class="px-3 py-1 rounded-full text-xs font-medium transition-colors border
						{filterInactive
							? 'bg-primary-600 text-white border-primary-600 dark:bg-primary-500 dark:border-primary-500'
							: 'bg-gray-100 text-gray-700 border-gray-200 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700'}"
					aria-pressed={filterInactive}
					onclick={() => (filterInactive = !filterInactive)}
				>
					{$t('admin.views.chip_inactive')}
				</button>
				{#if anyFilterActive}
					<button
						type="button"
						class="ms-auto px-3 py-1 rounded-full text-xs font-medium text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100 underline-offset-2 hover:underline"
						onclick={clearFilters}
					>
						{$t('admin.views.clear_filters')}
					</button>
				{/if}
			</div>
		</div>

		<p class="text-sm text-gray-600 dark:text-gray-400 mb-3" role="status" aria-live="polite" aria-atomic="true">
			{#if anyFilterActive}
				{$t('admin.views.count_filtered', { values: { filtered: filtered.length, total: views.length } })}
			{:else}
				{$t('admin.views.count_summary', { values: { count: views.length } })}
			{/if}
		</p>
	{/if}

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">{$t('admin.views.loading')}</div>
		</div>
	{:else if views.length === 0}
		<EmptyState
			title={$t('admin.views.empty_title')}
			description={$t('admin.views.empty_subtitle')}
			icon="eye"
		>
			{#snippet actions()}
				<a href="/admin/views/new" class="btn btn-primary">{$t('admin.views.create_cta')}</a>
			{/snippet}
		</EmptyState>
	{:else if filtered.length === 0}
		<EmptyState
			title={$t('admin.views.no_matches_title')}
			description={$t('admin.views.no_matches_description')}
			icon="eye"
		>
			{#snippet actions()}
				<button type="button" class="btn btn-secondary" onclick={clearFilters}>
					{$t('admin.views.clear_filters')}
				</button>
			{/snippet}
		</EmptyState>
	{:else}
		<!-- Mobile card list (touch primary input — no keyboard shortcuts) -->
		<div class="md:hidden space-y-3">
			{#each filtered as view (view.id)}
				<article class="card p-4">
					<div class="flex items-start justify-between gap-2 mb-2">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2 flex-wrap">
								<a href="/admin/views/{view.id}" class="font-medium text-gray-900 dark:text-white hover:text-primary-600 dark:hover:text-primary-400 truncate">
									{view.name}
								</a>
								{#if !view.is_active}
									<span class="px-2 py-0.5 text-xs bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
										{$t('admin.views.pill_inactive')}
									</span>
								{/if}
							</div>
							<div class="mt-1 flex flex-wrap items-center gap-1.5">
								<code class="text-xs bg-gray-100 dark:bg-gray-700 px-1.5 py-0.5 rounded">/{view.slug}</code>
								<span class="px-2 py-0.5 text-xs rounded {visibilityPillClass(String(view.visibility))}">
									{view.visibility}
								</span>
								{#if siteNavEnabled && inNavIds.has(view.id)}
									<span class="px-2 py-0.5 text-xs rounded bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-300">
										{$t('admin.views.pill_in_nav')}
									</span>
								{/if}
								{#if (tokenCounts[view.id] ?? 0) > 0}
									<span class="px-2 py-0.5 text-xs rounded bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300">
										{$t('admin.views.pill_share_links_count', { values: { count: tokenCounts[view.id] } })}
									</span>
								{/if}
							</div>
						</div>
					</div>

					<div class="flex items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400">
						<div class="flex items-center gap-3">
							<span class="flex items-center gap-1">
								{@html icon('eye')}
								{getViewCount(view).toLocaleString()}
							</span>
							{#if getLastViewedAt(view)}
								<span title={formatDate(String(getLastViewedAt(view)))}>
									{formatDate(String(getLastViewedAt(view)), { month: 'short', day: 'numeric' })}
								</span>
							{/if}
						</div>
						<div class="flex items-center gap-1">
							<button class="btn btn-sm btn-ghost p-2" onclick={() => copyViewUrl(String(view.slug))} title={$t('admin.views.action_copy_url')} aria-label={$t('admin.views.action_copy_url')}>
								{@html icon('copy')}
							</button>
							<a href="/{view.slug}" target="_blank" class="btn btn-sm btn-ghost p-2" title={$t('admin.views.action_preview')} aria-label={$t('admin.views.action_preview')}>
								{@html icon('eye')}
							</a>
							<a href="/admin/views/{view.id}" class="btn btn-sm btn-secondary">
								{$t('admin.views.action_edit')}
							</a>
							<button class="btn btn-sm btn-ghost p-2" onclick={() => toggleViewActive(view)} title={view.is_active ? $t('admin.views.action_deactivate') : $t('admin.views.action_activate')} aria-label={view.is_active ? $t('admin.views.action_deactivate') : $t('admin.views.action_activate')}>
								{@html view.is_active ? icon('toggleOn') : icon('toggleOff')}
							</button>
							<button class="btn btn-danger-ghost btn-sm p-2" onclick={() => deleteView(view)} title={$t('admin.views.action_delete')} aria-label={$t('admin.views.action_delete')}>
								{@html icon('trash')}
							</button>
						</div>
					</div>
				</article>
			{/each}
		</div>

		<!-- Desktop table view (keyboard navigation wired) -->
		<div class="hidden md:block card overflow-x-auto">
			<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
			<table
				class="w-full text-sm"
				onkeydown={handleTableKeydown}
			>
				<caption class="sr-only">{$t('admin.views.title')}</caption>
				<thead class="bg-gray-50 dark:bg-gray-800 text-gray-600 dark:text-gray-300 text-xs uppercase tracking-wide">
					<tr>
						<th scope="col" class="text-start font-medium px-4 py-2">{$t('admin.views.column_facet')}</th>
						<th scope="col" class="text-start font-medium px-4 py-2">{$t('admin.views.column_visibility')}</th>
						{#if siteNavEnabled}
							<th scope="col" class="text-start font-medium px-4 py-2">{$t('admin.views.column_nav')}</th>
						{/if}
						<th scope="col" class="text-start font-medium px-4 py-2">{$t('admin.views.column_share_links')}</th>
						<th scope="col" class="text-end font-medium px-4 py-2">{$t('admin.views.column_views')}</th>
						<th scope="col" class="text-start font-medium px-4 py-2 whitespace-nowrap">{$t('admin.views.column_last_viewed')}</th>
						<th scope="col" class="text-end font-medium px-4 py-2">
							<span class="sr-only">{$t('admin.views.column_actions')}</span>
						</th>
					</tr>
				</thead>
				<tbody
					class="divide-y divide-gray-100 dark:divide-gray-700"
					aria-keyshortcuts="j k Enter e x"
				>
					{#each filtered as view, i (view.id)}
						<tr
							data-row-index={i}
							tabindex={i === focusedIndex ? 0 : -1}
							class="hover:bg-gray-50 dark:hover:bg-gray-800/50 outline-none focus-visible:bg-primary-50 dark:focus-visible:bg-primary-900/30 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary-500 {i === focusedIndex ? 'bg-primary-50/50 dark:bg-primary-900/10' : ''}"
							aria-label={rowAriaLabel(view)}
							onfocus={() => (focusedIndex = i)}
						>
							<td class="px-4 py-3 align-top">
								<div class="flex items-center gap-2 flex-wrap">
									<a href="/admin/views/{view.id}" class="font-medium text-gray-900 dark:text-white hover:text-primary-600 dark:hover:text-primary-400">
										{view.name}
									</a>
									{#if !view.is_active}
										<span class="px-2 py-0.5 text-xs bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded">
											{$t('admin.views.pill_inactive')}
										</span>
									{/if}
								</div>
								<code class="text-xs text-gray-500 dark:text-gray-400 break-all">/{view.slug}</code>
							</td>
							<td class="px-4 py-3 align-top">
								<span class="px-2 py-0.5 text-xs rounded {visibilityPillClass(String(view.visibility))}">
									{view.visibility}
								</span>
							</td>
							{#if siteNavEnabled}
								<td class="px-4 py-3 align-top">
									{#if inNavIds.has(view.id)}
										<span class="px-2 py-0.5 text-xs rounded bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-300">
											{$t('admin.views.pill_in_nav')}
										</span>
									{:else}
										<span class="text-gray-300 dark:text-gray-600" aria-hidden="true">—</span>
									{/if}
								</td>
							{/if}
							<td class="px-4 py-3 align-top">
								{#if (tokenCounts[view.id] ?? 0) > 0}
									<a href="/admin/tokens" class="text-xs text-purple-700 hover:underline dark:text-purple-300">
										{$t('admin.views.pill_share_links_count', { values: { count: tokenCounts[view.id] } })}
									</a>
								{:else}
									<span class="text-gray-300 dark:text-gray-600" aria-hidden="true">—</span>
								{/if}
							</td>
							<td class="px-4 py-3 align-top text-end tabular-nums text-gray-600 dark:text-gray-300">
								{getViewCount(view).toLocaleString()}
							</td>
							<td class="px-4 py-3 align-top whitespace-nowrap text-gray-500 dark:text-gray-400">
								{#if getLastViewedAt(view)}
									{formatDate(String(getLastViewedAt(view)), { month: 'short', day: 'numeric', year: 'numeric' })}
								{:else}
									<span class="text-gray-300 dark:text-gray-600">{$t('admin.views.never_viewed')}</span>
								{/if}
							</td>
							<td class="px-4 py-3 align-top">
								<div class="flex items-center justify-end gap-1">
									<button class="btn btn-sm btn-ghost p-2" onclick={() => copyViewUrl(String(view.slug))} title={$t('admin.views.action_copy_url')} aria-label={$t('admin.views.action_copy_url')}>
										{@html icon('copy')}
									</button>
									<a href="/{view.slug}" target="_blank" class="btn btn-sm btn-ghost p-2" title={$t('admin.views.action_preview')} aria-label={$t('admin.views.action_preview')}>
										{@html icon('eye')}
									</a>
									<a href="/admin/views/{view.id}" class="btn btn-sm btn-secondary">
										{$t('admin.views.action_edit')}
									</a>
									<button class="btn btn-sm btn-ghost p-2" onclick={() => toggleViewActive(view)} title={view.is_active ? $t('admin.views.action_deactivate') : $t('admin.views.action_activate')} aria-label={view.is_active ? $t('admin.views.action_deactivate') : $t('admin.views.action_activate')}>
										{@html view.is_active ? icon('toggleOn') : icon('toggleOff')}
									</button>
									<button class="btn btn-danger-ghost btn-sm p-2" onclick={() => deleteView(view)} title={$t('admin.views.action_delete')} aria-label={$t('admin.views.action_delete')}>
										{@html icon('trash')}
									</button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Share Tokens Section -->
	<div class="mt-12">
		<h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">{$t('admin.views.share_tokens_title')}</h2>
		<p class="text-gray-600 dark:text-gray-400 mb-4">
			{$t('admin.views.share_tokens_description')}
		</p>
		<a href="/admin/tokens" class="btn btn-secondary">{$t('admin.views.manage_tokens')}</a>
	</div>
</div>

{#if showKeyboardHelp}
	<!-- Keyboard shortcuts modal -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50"
		role="presentation"
		onclick={(e) => {
			if (e.target === e.currentTarget) closeKeyboardHelp();
		}}
		onkeydown={handleHelpKeydown}
	>
		<div
			bind:this={helpDialog}
			role="dialog"
			aria-modal="true"
			aria-labelledby="kbd-help-title"
			aria-describedby="kbd-help-desc"
			class="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6"
		>
			<h2 id="kbd-help-title" class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
				{$t('admin.views.keyboard_help_title')}
			</h2>
			<p id="kbd-help-desc" class="text-sm text-gray-600 dark:text-gray-400 mb-4">
				{$t('admin.views.keyboard_help_description')}
			</p>
			<dl class="space-y-2 text-sm">
				{#each shortcutsList as { label, keys }}
					<div class="flex items-center justify-between gap-4">
						<dt class="text-gray-700 dark:text-gray-300">{label}</dt>
						<dd class="flex gap-1">
							{#each keys as k}
								<kbd class={kbdClass}>{k}</kbd>
							{/each}
						</dd>
					</div>
				{/each}
			</dl>
			<div class="flex justify-end mt-6">
				<button
					bind:this={helpCloseBtn}
					type="button"
					class="btn btn-secondary"
					onclick={closeKeyboardHelp}
				>
					{$t('admin.views.keyboard_help_close')}
				</button>
			</div>
		</div>
	</div>
{/if}
