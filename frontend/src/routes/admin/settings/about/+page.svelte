<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';
	import { browser } from '$app/environment';

	// App version from Vite config
	const appVersion = __APP_VERSION__;

	// Version check state
	let newVersionAvailable = $state(false);
	let latestVersion = $state('');

	// Check for new version (cached daily)
	async function checkForNewVersion() {
		if (!browser) return;

		const CACHE_KEY = 'facet_version_check';
		const CACHE_DURATION = 24 * 60 * 60 * 1000; // 24 hours

		try {
			// Check cache first
			const cached = localStorage.getItem(CACHE_KEY);
			if (cached) {
				const { timestamp, latest, hasUpdate } = JSON.parse(cached);
				if (Date.now() - timestamp < CACHE_DURATION) {
					latestVersion = latest;
					newVersionAvailable = hasUpdate;
					return;
				}
			}

			// Fetch latest release from GitHub
			const response = await fetch('https://api.github.com/repos/jesposito/Facet/releases/latest', {
				headers: { 'Accept': 'application/vnd.github.v3+json' }
			});

			if (!response.ok) return;

			const data = await response.json();
			const latest = data.tag_name || '';

			// Compare versions (strip 'v' prefix for comparison)
			const currentClean = appVersion.replace(/^v/, '').split('-')[0];
			const latestClean = latest.replace(/^v/, '');

			const hasUpdate = latestClean && currentClean !== latestClean &&
				compareVersions(latestClean, currentClean) > 0;

			// Cache result
			localStorage.setItem(CACHE_KEY, JSON.stringify({
				timestamp: Date.now(),
				latest,
				hasUpdate
			}));

			latestVersion = latest;
			newVersionAvailable = hasUpdate;
		} catch {
			// Silently fail - version check is non-critical
		}
	}

	// Compare semver versions: returns 1 if a > b, -1 if a < b, 0 if equal
	function compareVersions(a: string, b: string): number {
		const partsA = a.split('.').map(Number);
		const partsB = b.split('.').map(Number);

		for (let i = 0; i < Math.max(partsA.length, partsB.length); i++) {
			const numA = partsA[i] || 0;
			const numB = partsB[i] || 0;
			if (numA > numB) return 1;
			if (numA < numB) return -1;
		}
		return 0;
	}

	// Changelog state
	interface ChangelogEntry {
		version: string;
		date: string;
		bugs: string[];
		features: string[];
		other: string[];
		prLinks: string;
	}
	let allChangelogEntries: ChangelogEntry[] = $state([]);
	let visibleChangelogCount = $state(3);
	let changelogLoading = $state(true);

	// Derived: visible changelog entries
	let changelogEntries = $derived(allChangelogEntries.slice(0, visibleChangelogCount));
	let hasMoreChangelog = $derived(visibleChangelogCount < allChangelogEntries.length);

	function showMoreChangelog() {
		visibleChangelogCount += 3;
	}

	onMount(async () => {
		await loadChangelog();
		checkForNewVersion();
	});

	async function loadChangelog() {
		try {
			const response = await fetch('/CHANGELOG.md');
			if (!response.ok) {
				changelogLoading = false;
				return;
			}
			const text = await response.text();
			allChangelogEntries = parseChangelog(text);
		} catch (err) {
			console.error('Failed to load changelog:', err);
		} finally {
			changelogLoading = false;
		}
	}

	function parseChangelog(markdown: string): ChangelogEntry[] {
		const entries: ChangelogEntry[] = [];
		// Split by version headers (## vX.X.X - Date or ## Unreleased - Date)
		const sections = markdown.split(/^## /m).filter(s => s.trim());

		for (const section of sections) {
			if (section.startsWith('Changelog') || section.startsWith('#')) continue;

			const lines = section.split('\n');
			const headerLine = lines[0]?.trim() || '';

			// Parse header: "v1.0.0 - January 26, 2026" or "Unreleased - January 26, 2026"
			const headerMatch = headerLine.match(/^(v?\d+\.\d+\.\d+|Unreleased)\s*-\s*(.+)$/i);
			if (!headerMatch) continue;

			const entry: ChangelogEntry = {
				version: headerMatch[1],
				date: headerMatch[2].trim(),
				bugs: [],
				features: [],
				other: [],
				prLinks: ''
			};

			let currentSection = '';
			for (let i = 1; i < lines.length; i++) {
				const line = lines[i].trim();
				if (!line || line === '---') continue;

				if (line.startsWith('**Bugs Fixed:**')) {
					currentSection = 'bugs';
				} else if (line.startsWith('**New Features:**')) {
					currentSection = 'features';
				} else if (line.startsWith('**Other Changes:**')) {
					currentSection = 'other';
				} else if (line.startsWith('**Pull Requests:**')) {
					entry.prLinks = line.replace('**Pull Requests:**', '').trim();
					currentSection = '';
				} else if (line.startsWith('- ') && currentSection) {
					const item = line.substring(2).trim();
					if (currentSection === 'bugs') entry.bugs.push(item);
					else if (currentSection === 'features') entry.features.push(item);
					else if (currentSection === 'other') entry.other.push(item);
				}
			}

			if (entry.bugs.length > 0 || entry.features.length > 0 || entry.other.length > 0) {
				entries.push(entry);
			}
		}

		return entries;
	}
</script>

<svelte:head>
	<title>{$t('admin.about_page.page_title')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto p-6">
	<div class="card p-6">
		<div class="flex items-center justify-between mb-4">
			<h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{$t('admin.about_page.title')}</h1>
			<div class="flex items-center gap-2">
				<span class="px-3 py-1 text-sm font-mono bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg">
					{appVersion}
				</span>
				{#if newVersionAvailable}
					<span class="inline-flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-full bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300">
						<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
						</svg>
						{latestVersion} available
					</span>
				{/if}
			</div>
		</div>

		<p class="text-gray-600 dark:text-gray-400 mb-6">
			{$t('admin.about_page.description')}
		</p>

		<div class="flex flex-wrap gap-3 mb-8">
			<a
				href="https://github.com/jesposito/Facet/issues/new"
				target="_blank"
				rel="noopener noreferrer"
				class="btn btn-secondary inline-flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
				</svg>
				{$t('admin.about_page.report_issue')}
			</a>
			<a
				href="https://github.com/jesposito/Facet"
				target="_blank"
				rel="noopener noreferrer"
				class="btn btn-ghost inline-flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
				</svg>
				{$t('admin.about_page.view_github')}
			</a>
		</div>

		<!-- Changelog Section -->
		<div class="border-t border-gray-200 dark:border-gray-700 pt-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.about_page.changelog_title')}</h2>

			{#if changelogLoading}
				<div class="animate-pulse text-sm text-gray-500 dark:text-gray-400">{$t('admin.about_page.changelog_loading')}</div>
			{:else if changelogEntries.length === 0}
				<p class="text-sm text-gray-500 dark:text-gray-400">
					No changelog available yet. Check <a href="https://github.com/jesposito/Facet/releases" target="_blank" rel="noopener noreferrer" class="text-primary-600 dark:text-primary-400 hover:underline">GitHub releases</a> for updates.
				</p>
			{:else}
				<div class="space-y-4">
					{#each changelogEntries as entry}
						<div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
							<div class="flex items-center justify-between mb-2">
								<span class="font-medium text-gray-900 dark:text-white">{entry.version}</span>
								<span class="text-xs text-gray-500 dark:text-gray-400">{entry.date}</span>
							</div>

							{#if entry.bugs.length > 0}
								<div class="mb-2">
									<span class="text-xs font-semibold text-red-600 dark:text-red-400 uppercase">{$t('admin.about_page.bug_fixes')}</span>
									<ul class="mt-1 text-sm text-gray-600 dark:text-gray-400 space-y-0.5">
										{#each entry.bugs as bug}
											<li class="flex items-start gap-2">
												<span class="text-red-500 mt-0.5">•</span>
												<span>{bug}</span>
											</li>
										{/each}
									</ul>
								</div>
							{/if}

							{#if entry.features.length > 0}
								<div class="mb-2">
									<span class="text-xs font-semibold text-green-600 dark:text-green-400 uppercase">{$t('admin.about_page.new_features')}</span>
									<ul class="mt-1 text-sm text-gray-600 dark:text-gray-400 space-y-0.5">
										{#each entry.features as feature}
											<li class="flex items-start gap-2">
												<span class="text-green-500 mt-0.5">•</span>
												<span>{feature}</span>
											</li>
										{/each}
									</ul>
								</div>
							{/if}

							{#if entry.other.length > 0}
								<div class="mb-2">
									<span class="text-xs font-semibold text-blue-600 dark:text-blue-400 uppercase">{$t('admin.about_page.other_changes')}</span>
									<ul class="mt-1 text-sm text-gray-600 dark:text-gray-400 space-y-0.5">
										{#each entry.other as item}
											<li class="flex items-start gap-2">
												<span class="text-blue-500 mt-0.5">•</span>
												<span>{item}</span>
											</li>
										{/each}
									</ul>
								</div>
							{/if}
						</div>
					{/each}
				</div>

				<div class="mt-6 flex flex-col items-center gap-2">
					{#if hasMoreChangelog}
						<button
							onclick={showMoreChangelog}
							class="btn btn-ghost text-sm"
						>
							{$t('admin.about_page.show_more', { values: { count: allChangelogEntries.length - visibleChangelogCount } })}
						</button>
					{/if}
					<a
						href="https://github.com/jesposito/Facet/blob/main/CHANGELOG.md"
						target="_blank"
						rel="noopener noreferrer"
						class="text-sm text-primary-600 dark:text-primary-400 hover:underline"
					>
						{$t('admin.about_page.view_full_changelog')}
					</a>
				</div>
			{/if}
		</div>
	</div>
</div>
