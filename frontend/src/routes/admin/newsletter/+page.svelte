<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { pb } from '$lib/pocketbase';
	import { smtpReadiness, type SmtpReadiness } from '$lib/newsletter/sends';

	type Stats = {
		total_active: number;
		total_unsubscribed: number;
		total_bounced: number;
	};
	type ListSummary = {
		id: string;
		name: string;
		is_default: boolean;
		is_active: boolean;
		subscriber_count: number;
	};

	let stats: Stats | null = $state(null);
	let lists: ListSummary[] = $state([]);
	let loading = $state(true);
	let smtpStatus: SmtpReadiness = $state('checking');
	let smtpChecking = $state(false);

	function authHeaders(): Record<string, string> {
		return pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {};
	}

	onMount(async () => {
		await Promise.all([loadStats(), loadLists(), loadSmtp()]);
		loading = false;
	});

	async function loadStats() {
		try {
			const r = await fetch('/api/admin/subscribers/stats', { headers: authHeaders() });
			if (r.ok) stats = await r.json();
		} catch (err) {
			console.debug('Failed to load stats:', err);
		}
	}

	async function loadLists() {
		try {
			const r = await fetch('/api/admin/newsletter-lists', { headers: authHeaders() });
			if (r.ok) {
				const data = await r.json();
				lists = data.data ?? [];
			}
		} catch (err) {
			console.debug('Failed to load lists:', err);
		}
	}

	async function loadSmtp() {
		if (smtpChecking) return;
		smtpChecking = true;
		smtpStatus = 'checking';
		try {
			const r = await fetch('/api/admin/newsletter/smtp-status', { headers: authHeaders() });
			if (!r.ok) throw new Error(`SMTP status request failed with ${r.status}`);
			smtpStatus = smtpReadiness(await r.json());
		} catch (err) {
			smtpStatus = 'unavailable';
			console.debug('Failed to load SMTP status:', err);
		} finally {
			smtpChecking = false;
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.newsletter.hub_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
	<!-- Header -->
	<div>
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
			{$t('admin.newsletter.hub_title')}
		</h1>
		<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
			{$t('admin.newsletter.hub_description')}
		</p>
	</div>

	<!-- SMTP warning (text label, not color-only) -->
	{#if smtpStatus !== 'ready'}
		<div
			class="flex items-start gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg"
			role="alert"
		>
			<svg
				class="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				aria-hidden="true"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
				/>
			</svg>
			<div>
				<p class="text-sm font-medium text-amber-800 dark:text-amber-300">
					{smtpStatus === 'not_ready'
						? $t('admin.newsletter.smtp_not_ready')
						: smtpStatus === 'not_configured'
							? $t('admin.newsletter.smtp_not_configured')
							: smtpStatus === 'unavailable'
								? $t('admin.newsletter.smtp_unavailable')
								: $t('admin.newsletter.smtp_checking')}
				</p>
				<p class="text-sm text-amber-700 dark:text-amber-400 mt-0.5">
					{smtpStatus === 'not_ready'
						? $t('admin.newsletter.smtp_not_ready_hint')
						: smtpStatus === 'not_configured'
							? $t('admin.newsletter.smtp_not_configured_hint')
							: smtpStatus === 'unavailable'
								? $t('admin.newsletter.smtp_unavailable_hint')
								: $t('admin.newsletter.smtp_checking_hint')}
				</p>
				<button
					type="button"
					class="btn btn-secondary text-xs mt-3"
					onclick={loadSmtp}
					disabled={smtpChecking}
					aria-busy={smtpChecking ? 'true' : undefined}
				>
					{$t('admin.newsletter.smtp_retry')}
				</button>
			</div>
		</div>
	{/if}

	<!-- Stats overview -->
	{#if stats}
		<div class="grid grid-cols-3 gap-4">
			<div class="card p-4 text-center">
				<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_active}</p>
				<p class="text-xs text-gray-500 dark:text-gray-400">
					{$t('admin.subscribers.status_active')}
				</p>
			</div>
			<div class="card p-4 text-center">
				<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_unsubscribed}</p>
				<p class="text-xs text-gray-500 dark:text-gray-400">
					{$t('admin.subscribers.status_unsubscribed')}
				</p>
			</div>
			<div class="card p-4 text-center">
				<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_bounced}</p>
				<p class="text-xs text-gray-500 dark:text-gray-400">
					{$t('admin.subscribers.status_bounced')}
				</p>
			</div>
		</div>
	{/if}

	<!-- Navigation cards. Each <a> wraps the whole card; visible text inside
	     forms the accessible name — no extra aria-label needed. -->
	<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
		<a
			href="/admin/subscribers"
			class="card p-5 hover:border-primary-500 hover:shadow-sm transition focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 block"
		>
			<div class="flex items-center gap-3 mb-2">
				<svg
					class="w-6 h-6 text-primary-600 dark:text-primary-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z"
					/>
				</svg>
				<h2 class="text-base font-semibold text-gray-900 dark:text-white">
					{$t('admin.newsletter.card_subscribers_title')}
				</h2>
			</div>
			<p class="text-sm text-gray-500 dark:text-gray-400">
				{$t('admin.newsletter.card_subscribers_description')}
			</p>
		</a>

		<a
			href="/admin/newsletter/lists"
			class="card p-5 hover:border-primary-500 hover:shadow-sm transition focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 block"
		>
			<div class="flex items-center gap-3 mb-2">
				<svg
					class="w-6 h-6 text-primary-600 dark:text-primary-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M4 6h16M4 12h16M4 18h7"
					/>
				</svg>
				<h2 class="text-base font-semibold text-gray-900 dark:text-white">
					{$t('admin.newsletter.card_lists_title')}
				</h2>
			</div>
			<p class="text-sm text-gray-500 dark:text-gray-400">
				{$t('admin.newsletter.card_lists_description')}
			</p>
			{#if !loading && lists.length > 0}
				<p class="text-xs text-gray-600 dark:text-gray-300 mt-2">
					{$t('admin.newsletter.lists_count', { values: { count: lists.length } })}
				</p>
			{/if}
		</a>

		<a
			href="/admin/newsletter/compose"
			class="card p-5 hover:border-primary-500 hover:shadow-sm transition focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 block"
		>
			<div class="flex items-center gap-3 mb-2">
				<svg
					class="w-6 h-6 text-primary-600 dark:text-primary-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					aria-hidden="true"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
					/>
				</svg>
				<h2 class="text-base font-semibold text-gray-900 dark:text-white">
					{$t('admin.newsletter.card_compose_title')}
				</h2>
			</div>
			<p class="text-sm text-gray-500 dark:text-gray-400">
				{$t('admin.newsletter.card_compose_description')}
			</p>
		</a>
	</div>
</div>
