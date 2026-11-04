<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { toasts } from '$lib/stores';
	import { resolveSiteFeatures, setSiteFeatures } from '$lib/stores/features';
	import { fetchSettingsJson, isSettingsObject } from '$lib/admin/settings-load';
	import SettingsLoadError from '$components/admin/SettingsLoadError.svelte';

	let loading = $state(true);
	let loaded = $state(false);
	let loadFailed = $state(false);
	let saving = $state(false);
	let heading: HTMLHeadingElement | null = $state(null);
	let loadSequence = 0;
	let settingsRequest: AbortController | null = null;
	let mounted = false;

	let features = $state({
		analytics: true,
		courses: true,
		newsletter: true,
		discussions: true
	});

	onMount(() => {
		mounted = true;
		void loadFeatures();
	});

	onDestroy(() => {
		mounted = false;
		loadSequence += 1;
		settingsRequest?.abort();
	});

	function hasValidFeatures(data: Record<string, unknown>): boolean {
		const raw = data.enabled_features;
		if (!isSettingsObject(raw)) return false;
		return ['analytics', 'courses', 'newsletter', 'discussions'].every(
			(key) => raw[key] === undefined || typeof raw[key] === 'boolean'
		);
	}

	async function loadFeatures() {
		const sequence = ++loadSequence;
		settingsRequest?.abort();
		settingsRequest = new AbortController();
		loading = true;
		try {
			const data = await fetchSettingsJson(`/api/site-settings?_=${Date.now()}`, {
				signal: settingsRequest.signal
			});
			if (!hasValidFeatures(data)) throw new Error('Invalid feature settings response');
			if (!mounted || sequence !== loadSequence) return;
			if (!loaded) features = resolveSiteFeatures(data.enabled_features);
			loaded = true;
			loadFailed = false;
		} catch (err) {
			if (!mounted || sequence !== loadSequence) return;
			console.error('Failed to load feature settings:', err);
			loadFailed = true;
		} finally {
			if (mounted && sequence === loadSequence) {
				loading = false;
				settingsRequest = null;
			}
		}
	}

	async function retryFeatures() {
		if (loading) return;
		await loadFeatures();
		if (!loadFailed) {
			await tick();
			heading?.focus();
		}
	}

	async function saveFeatures() {
		if (!loaded || loadFailed || loading || saving) return;
		saving = true;
		try {
			const saved = await pb.send('/api/site-settings', {
				method: 'PUT',
				body: { enabled_features: { ...features } }
			});
			// Update navigation and public components straight away.
			setSiteFeatures(saved?.enabled_features ?? features);
			toasts.add('success', $t('admin.settings.features.saved'));
		} catch (err) {
			console.error('Failed to save feature settings:', err);
			toasts.add('error', $t('admin.settings.features.save_error'));
		} finally {
			saving = false;
		}
	}

	interface FeatureRow {
		key: keyof typeof features;
		nameKey: string;
		descKey: string;
	}

	const featureRows: FeatureRow[] = [
		{
			key: 'analytics',
			nameKey: 'admin.settings.features.analytics',
			descKey: 'admin.settings.features.analytics_description'
		},
		{
			key: 'courses',
			nameKey: 'admin.settings.features.courses',
			descKey: 'admin.settings.features.courses_description'
		},
		{
			key: 'newsletter',
			nameKey: 'admin.settings.features.newsletter',
			descKey: 'admin.settings.features.newsletter_description'
		},
		{
			key: 'discussions',
			nameKey: 'admin.settings.features.discussions',
			descKey: 'admin.settings.features.discussions_description'
		}
	];
</script>

<svelte:head>
	<title>{$t('admin.settings.features.title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<div class="mb-6">
		<h1
			bind:this={heading}
			tabindex="-1"
			class="text-2xl font-bold text-gray-900 dark:text-white outline-none"
		>
			{$t('admin.settings.features.title')}
		</h1>
		<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
			{$t('admin.settings.features.description')}
		</p>
	</div>

	{#if loadFailed}
		<SettingsLoadError
			message={$t('admin.settings_page.recoverycopy.load_error', {
				values: { resource: $t('admin.settings_page.recoverycopy.features') }
			})}
			pending={loading}
			onretry={retryFeatures}
		/>
	{/if}

	{#if !loaded}
		{#if loading}
			<div class="card p-6">
				<div class="animate-pulse space-y-4">
					{#each { length: 4 } as _}
						<div class="flex items-center justify-between">
							<div class="space-y-2 flex-1">
								<div class="h-4 bg-gray-200 dark:bg-gray-700 rounded w-32"></div>
								<div class="h-3 bg-gray-200 dark:bg-gray-700 rounded w-64"></div>
							</div>
							<div class="h-6 w-11 bg-gray-200 dark:bg-gray-700 rounded-full ml-4"></div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{:else}
		<div class="card p-6 space-y-6">
			<p
				class="text-sm text-gray-500 dark:text-gray-400 bg-gray-50 dark:bg-gray-800 rounded-lg p-3"
			>
				{$t('admin.settings.features.all_enabled_note')}
			</p>

			<div class="divide-y divide-gray-200 dark:divide-gray-700">
				{#each featureRows as row}
					<div class="flex items-center justify-between py-4 first:pt-0 last:pb-0">
						<div class="flex-1 pr-4">
							<p class="text-sm font-medium text-gray-900 dark:text-white">{$t(row.nameKey)}</p>
							<p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">{$t(row.descKey)}</p>
						</div>

						<!-- Toggle switch -->
						<button
							type="button"
							role="switch"
							aria-checked={features[row.key]}
							aria-label={$t(row.nameKey)}
							disabled={saving || loading || loadFailed}
							onclick={() => {
								if (loadFailed || loading) return;
								features[row.key] = !features[row.key];
							}}
							class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed
								{features[row.key] ? 'bg-primary-600' : 'bg-gray-200 dark:bg-gray-700'}"
						>
							<span class="sr-only">{$t(row.nameKey)}</span>
							<span
								class="pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow transform ring-0 transition duration-200 ease-in-out
									{features[row.key] ? 'translate-x-5' : 'translate-x-0'}"
							></span>
						</button>
					</div>
				{/each}
			</div>

			<div class="flex justify-end pt-2">
				<button
					type="button"
					class="btn btn-primary"
					onclick={saveFeatures}
					disabled={saving || loading || loadFailed}
				>
					{#if saving}
						<svg
							class="animate-spin -ml-1 mr-2 h-4 w-4 text-white inline"
							fill="none"
							viewBox="0 0 24 24"
						>
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
					{/if}
					{$t('shared.actions.save')}
				</button>
			</div>
		</div>
	{/if}
</div>
