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
						{$t('admin.about_page.new_version_available', { values: { version: latestVersion } })}
					</span>
				{/if}
			</div>
		</div>

		<p class="text-gray-600 dark:text-gray-400 mb-6">
			{$t('admin.about_page.description')}
		</p>

		<div class="flex flex-wrap gap-3 mb-6">
			<a
				href="https://github.com/jesposito/Facet/issues/new"
				target="_blank"
				rel="noopener noreferrer"
				class="btn btn-primary inline-flex items-center gap-2"
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
				class="btn btn-primary inline-flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
				</svg>
				{$t('admin.about_page.view_github')}
			</a>
			<a
				href="https://discord.gg/eKg4MhMkVJ"
				target="_blank"
				rel="noopener noreferrer"
				class="btn btn-primary inline-flex items-center gap-2"
			>
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
					<path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/>
				</svg>
				{$t('admin.about_page.join_discord')}
			</a>
		</div>

		<!-- Support Section -->
		<div class="bg-gradient-to-r from-amber-50 to-yellow-50 dark:from-amber-900/20 dark:to-yellow-900/20 border border-amber-200 dark:border-amber-800 rounded-lg p-4 mb-8">
			<div class="flex items-start gap-4">
				<div class="flex-shrink-0 text-amber-600 dark:text-amber-400">
					<svg class="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
						<path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
					</svg>
				</div>
				<div class="flex-1">
					<h3 class="font-medium text-amber-900 dark:text-amber-100 mb-1">{$t('admin.about_page.support_title')}</h3>
					<p class="text-sm text-amber-700 dark:text-amber-300 mb-3">{$t('admin.about_page.support_description')}</p>
					<a
						href="https://buymeacoffee.com/jesposito"
						target="_blank"
						rel="noopener noreferrer"
						class="inline-flex items-center gap-2 px-4 py-2 bg-amber-500 hover:bg-amber-600 text-white font-medium rounded-lg transition-colors"
					>
						<svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
							<path d="M20.216 6.415l-.132-.666c-.119-.598-.388-1.163-1.001-1.379-.197-.069-.42-.098-.57-.241-.152-.143-.196-.366-.231-.572-.065-.378-.125-.756-.192-1.133-.057-.325-.102-.69-.25-.987-.195-.4-.597-.634-.996-.788a5.723 5.723 0 00-.626-.194c-1-.263-2.05-.36-3.077-.416a25.834 25.834 0 00-3.7.062c-.915.083-1.88.184-2.75.5-.318.116-.646.256-.888.501-.297.302-.393.77-.177 1.146.154.267.415.456.692.58.36.162.737.284 1.123.366 1.075.238 2.189.331 3.287.37 1.218.05 2.437.01 3.65-.118.299-.033.598-.073.896-.119.352-.054.578-.513.474-.834-.124-.383-.457-.531-.834-.473-.466.074-.96.108-1.382.146-1.177.08-2.358.082-3.536.006a22.228 22.228 0 01-1.157-.107c-.086-.01-.18-.025-.258-.036-.243-.036-.484-.08-.724-.13-.111-.027-.111-.185 0-.212h.005c.277-.06.557-.108.838-.147h.002c.131-.009.263-.032.394-.048a25.076 25.076 0 013.426-.12c.674.019 1.347.067 2.017.144l.228.031c.267.04.533.088.798.145.392.085.895.113 1.07.542.055.137.08.288.111.431l.319 1.484a.237.237 0 01-.199.284h-.003c-.037.006-.075.01-.112.015a36.704 36.704 0 01-4.743.295 37.059 37.059 0 01-4.699-.304c-.14-.017-.293-.042-.417-.06-.326-.048-.649-.108-.973-.161-.393-.065-.768-.032-1.123.161-.29.16-.527.404-.675.701-.154.316-.199.66-.267 1-.069.34-.176.707-.135 1.056.087.753.613 1.365 1.37 1.502a39.69 39.69 0 0011.343.376.483.483 0 01.535.53l-.071.697-1.018 9.907c-.041.41-.047.832-.125 1.237-.122.637-.553 1.028-1.182 1.171-.577.131-1.165.2-1.756.205-.656.004-1.31-.025-1.966-.022-.699.004-1.556-.06-2.095-.58-.475-.458-.54-1.174-.605-1.793l-.731-7.013-.322-3.094c-.037-.351-.286-.695-.678-.678-.336.015-.718.3-.678.679l.228 2.185.949 9.112c.147 1.344 1.174 2.068 2.446 2.272.742.12 1.503.144 2.257.156.966.016 1.942.053 2.892-.122 1.408-.258 2.465-1.198 2.616-2.657.34-3.332.683-6.663 1.024-9.995l.215-2.087a.484.484 0 01.39-.426c.402-.078.787-.212 1.074-.518.455-.488.546-1.124.385-1.766zm-1.478.772c-.145.137-.363.201-.578.233-2.416.359-4.866.54-7.308.46-1.748-.06-3.477-.254-5.207-.498-.17-.024-.353-.055-.47-.18-.22-.236-.111-.71-.054-.995.052-.26.152-.609.463-.646.484-.057 1.046.148 1.526.22.577.088 1.156.159 1.737.212 2.48.226 5.002.19 7.472-.14.45-.06.899-.13 1.345-.21.399-.072.84-.206 1.08.206.166.281.188.657.162.974a.544.544 0 01-.169.364z"/>
						</svg>
						{$t('admin.about_page.buy_coffee')}
					</a>
				</div>
			</div>
		</div>

		<!-- Changelog Section -->
		<div class="border-t border-gray-200 dark:border-gray-700 pt-6">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.about_page.changelog_title')}</h2>

			{#if changelogLoading}
				<div class="animate-pulse text-sm text-gray-500 dark:text-gray-400">{$t('admin.about_page.changelog_loading')}</div>
			{:else if changelogEntries.length === 0}
				<p class="text-sm text-gray-500 dark:text-gray-400">
					{$t('admin.about_page.changelog_empty_before_link')} <a href="https://github.com/jesposito/Facet/releases" target="_blank" rel="noopener noreferrer" class="text-primary-600 dark:text-primary-400 hover:underline">{$t('admin.about_page.changelog_empty_link')}</a> {$t('admin.about_page.changelog_empty_after_link')}
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
