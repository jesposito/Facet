<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';

	// Self-hosted single-user alerts inbox. The cloud Facet edition aggregates
	// multi-tenant provisioning + capacity events from a control plane; here we
	// surface only locally-emitted events (backups, SMTP, webhook delivery, etc.)
	// via the `system_alerts` PocketBase collection.

	interface Alert {
		id: string;
		severity: 'info' | 'warning' | 'critical' | string;
		event: string;
		title: string;
		details: string | null;
		metadata: Record<string, unknown> | null;
		acknowledged_at: string | null;
		created: string;
	}

	type FilterValue = 'unacknowledged' | 'all' | 'acknowledged';

	let loading = $state(true);
	let error = $state(false);
	let alerts: Alert[] = $state([]);
	let filter = $state<FilterValue>('unacknowledged');
	let acknowledging = $state<Set<string>>(new Set());
	let acknowledgingAll = $state(false);

	onMount(() => {
		loadAlerts();
	});

	function authHeaders(): Record<string, string> {
		const token = pb.authStore.token;
		return token ? { Authorization: token } : {};
	}

	async function loadAlerts() {
		loading = true;
		error = false;
		try {
			const params = new URLSearchParams({ limit: '100' });
			if (filter === 'unacknowledged') params.set('acknowledged', 'false');
			else if (filter === 'acknowledged') params.set('acknowledged', 'true');

			const resp = await fetch(`/api/admin/system-alerts?${params.toString()}`, {
				headers: authHeaders()
			});
			if (resp.ok) {
				const data = await resp.json();
				alerts = (data.alerts as Alert[]) ?? [];
			} else {
				error = true;
			}
		} catch (err) {
			console.error('Failed to load system alerts:', err);
			error = true;
		} finally {
			loading = false;
		}
	}

	async function acknowledgeAlert(id: string) {
		acknowledging = new Set([...acknowledging, id]);
		try {
			const resp = await fetch(`/api/admin/system-alerts/${id}/acknowledge`, {
				method: 'POST',
				headers: { ...authHeaders(), 'Content-Type': 'application/json' }
			});
			if (resp.ok) {
				const data = await resp.json();
				if (filter === 'unacknowledged') {
					alerts = alerts.filter((a) => a.id !== id);
				} else {
					alerts = alerts.map((a) =>
						a.id === id ? { ...a, acknowledged_at: data.acknowledged_at ?? new Date().toISOString() } : a
					);
				}
				toasts.success($t('admin.alerts_page.toast_acknowledged'));
			} else {
				toasts.error($t('admin.alerts_page.toast_acknowledge_failed'));
			}
		} catch (err) {
			console.error('Failed to acknowledge alert:', err);
			toasts.error($t('admin.alerts_page.toast_acknowledge_failed'));
		} finally {
			const next = new Set(acknowledging);
			next.delete(id);
			acknowledging = next;
		}
	}

	async function acknowledgeAll() {
		const confirmed = await confirm({
			title: $t('admin.alerts_page.ack_all_confirm_title'),
			message: $t('admin.alerts_page.ack_all_confirm_message'),
			confirmText: $t('admin.alerts_page.acknowledge_all')
		});
		if (!confirmed) return;

		acknowledgingAll = true;
		try {
			const resp = await fetch('/api/admin/system-alerts/acknowledge-all', {
				method: 'POST',
				headers: { ...authHeaders(), 'Content-Type': 'application/json' }
			});
			if (resp.ok) {
				const data = await resp.json();
				const count = (data.acknowledged as number) ?? 0;
				toasts.success(
					$t('admin.alerts_page.toast_acknowledge_all_done', { values: { count } })
				);
				if (filter === 'unacknowledged') {
					alerts = [];
				} else {
					await loadAlerts();
				}
			} else {
				toasts.error($t('admin.alerts_page.toast_acknowledge_failed'));
			}
		} catch (err) {
			console.error('Failed to acknowledge all alerts:', err);
			toasts.error($t('admin.alerts_page.toast_acknowledge_failed'));
		} finally {
			acknowledgingAll = false;
		}
	}

	async function deleteAlert(id: string) {
		const confirmed = await confirm({
			title: $t('admin.alerts_page.delete_confirm_title'),
			message: $t('admin.alerts_page.delete_confirm_message'),
			confirmText: $t('shared.actions.delete'),
			danger: true
		});
		if (!confirmed) return;

		try {
			const resp = await fetch(`/api/admin/system-alerts/${id}`, {
				method: 'DELETE',
				headers: authHeaders()
			});
			if (resp.ok) {
				alerts = alerts.filter((a) => a.id !== id);
				toasts.success($t('admin.alerts_page.toast_deleted'));
			} else {
				toasts.error($t('admin.alerts_page.toast_delete_failed'));
			}
		} catch (err) {
			console.error('Failed to delete alert:', err);
			toasts.error($t('admin.alerts_page.toast_delete_failed'));
		}
	}

	function changeFilter(newFilter: FilterValue) {
		if (filter === newFilter) return;
		filter = newFilter;
		loadAlerts();
	}

	function severityLabel(severity: string): string {
		if (severity === 'critical') return $t('admin.alerts_page.severity_critical');
		if (severity === 'warning') return $t('admin.alerts_page.severity_warning');
		return $t('admin.alerts_page.severity_info');
	}

	function severityClasses(severity: string): string {
		if (severity === 'critical') return 'text-red-700 dark:text-red-400';
		if (severity === 'warning') return 'text-amber-700 dark:text-amber-400';
		return 'text-blue-700 dark:text-blue-400';
	}

	function severityBadgeClasses(severity: string): string {
		if (severity === 'critical') {
			return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300';
		}
		if (severity === 'warning') {
			return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300';
		}
		return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300';
	}

	function formatRelative(iso: string): string {
		if (!iso) return '';
		const d = new Date(iso.includes('T') ? iso : iso.replace(' ', 'T'));
		if (Number.isNaN(d.getTime())) return iso;
		const diffMs = Date.now() - d.getTime();
		const diffMin = Math.floor(diffMs / 60000);
		if (diffMin < 1) return $t('admin.alerts_page.time_just_now');
		if (diffMin < 60) return $t('admin.alerts_page.time_minutes_ago', { values: { count: diffMin } });
		const diffHr = Math.floor(diffMin / 60);
		if (diffHr < 24) return $t('admin.alerts_page.time_hours_ago', { values: { count: diffHr } });
		const diffDay = Math.floor(diffHr / 24);
		return $t('admin.alerts_page.time_days_ago', { values: { count: diffDay } });
	}

	const hasAlerts = $derived(alerts.length > 0);
	const showAckAll = $derived(filter === 'unacknowledged' && hasAlerts);
</script>

<svelte:head>
	<title>{$t('admin.alerts_page.title')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<div class="mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.alerts_page.title')}</h1>
		<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
			{$t('admin.alerts_page.description')}
		</p>
	</div>

	<!-- Filter segmented control (a11y: aria-pressed pattern, not tabs — no arrow-key handler needed) -->
	<div class="flex flex-wrap items-center gap-2 mb-4">
		<div
			class="inline-flex items-center gap-1 rounded-md border border-gray-200 dark:border-gray-700 p-1"
			role="group"
			aria-label={$t('admin.alerts_page.filter_label')}
		>
			<button
				type="button"
				aria-pressed={filter === 'unacknowledged'}
				onclick={() => changeFilter('unacknowledged')}
				class="px-3 py-1.5 text-sm rounded transition-colors {filter === 'unacknowledged'
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}"
			>
				{$t('admin.alerts_page.filter_unacknowledged')}
			</button>
			<button
				type="button"
				aria-pressed={filter === 'all'}
				onclick={() => changeFilter('all')}
				class="px-3 py-1.5 text-sm rounded transition-colors {filter === 'all'
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}"
			>
				{$t('admin.alerts_page.filter_all')}
			</button>
			<button
				type="button"
				aria-pressed={filter === 'acknowledged'}
				onclick={() => changeFilter('acknowledged')}
				class="px-3 py-1.5 text-sm rounded transition-colors {filter === 'acknowledged'
					? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
					: 'text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'}"
			>
				{$t('admin.alerts_page.filter_acknowledged')}
			</button>
		</div>

		{#if showAckAll}
			<button
				type="button"
				onclick={acknowledgeAll}
				disabled={acknowledgingAll}
				class="ms-auto px-3 py-1.5 text-sm rounded-md bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
			>
				{acknowledgingAll
					? $t('admin.alerts_page.acknowledging')
					: $t('admin.alerts_page.acknowledge_all')}
			</button>
		{/if}
	</div>

	<!-- Alert list -->
	{#if loading}
		<div
			class="flex justify-center py-12 text-gray-500 dark:text-gray-400"
			role="status"
			aria-live="polite"
		>
			<svg class="w-6 h-6 animate-spin text-primary-600" fill="none" viewBox="0 0 24 24" aria-hidden="true">
				<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
				></path>
			</svg>
			<span class="sr-only">{$t('admin.alerts_page.loading')}</span>
		</div>
	{:else if error}
		<div class="text-center py-12" role="alert">
			<p class="text-red-600 dark:text-red-400 mb-3">{$t('admin.alerts_page.error_loading')}</p>
			<button
				type="button"
				class="px-4 py-2 text-sm rounded-md bg-primary-600 text-white hover:bg-primary-700 transition-colors"
				onclick={loadAlerts}
			>
				{$t('admin.alerts_page.retry')}
			</button>
		</div>
	{:else if !hasAlerts}
		<div class="text-center py-12 text-gray-500 dark:text-gray-400" role="status">
			<svg
				class="w-12 h-12 mx-auto mb-3 text-green-400"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				aria-hidden="true"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.5"
					d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
			<p>
				{filter === 'unacknowledged'
					? $t('admin.alerts_page.empty_unacknowledged')
					: filter === 'acknowledged'
						? $t('admin.alerts_page.empty_acknowledged')
						: $t('admin.alerts_page.empty_all')}
			</p>
		</div>
	{:else}
		<ul class="space-y-2" aria-label={$t('admin.alerts_page.list_label')}>
			{#each alerts as alert (alert.id)}
				<li
					class="flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 {alert.acknowledged_at
						? 'opacity-70'
						: ''}"
				>
					<div class="flex-1 min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<span
								class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {severityBadgeClasses(
									alert.severity
								)}"
							>
								{severityLabel(alert.severity)}
							</span>
							<span class="text-sm font-semibold {severityClasses(alert.severity)} truncate">
								{alert.title}
							</span>
							<span
								class="text-xs text-gray-500 dark:text-gray-400 font-mono"
								title={alert.event}
							>
								{alert.event}
							</span>
							<time
								class="text-xs text-gray-500 dark:text-gray-400 ms-auto"
								datetime={alert.created}
							>
								{formatRelative(alert.created)}
							</time>
						</div>
						{#if alert.details}
							<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{alert.details}</p>
						{/if}
						{#if alert.acknowledged_at}
							<p class="mt-1 text-xs text-gray-500 dark:text-gray-500">
								{$t('admin.alerts_page.acknowledged_at_prefix')}
								<time datetime={alert.acknowledged_at}>{formatRelative(alert.acknowledged_at)}</time>
							</p>
						{/if}
					</div>
					<div class="flex flex-col gap-1 shrink-0">
						{#if !alert.acknowledged_at}
							<button
								type="button"
								onclick={() => acknowledgeAlert(alert.id)}
								disabled={acknowledging.has(alert.id)}
								class="px-2 py-1 text-xs rounded bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
							>
								{acknowledging.has(alert.id)
									? $t('admin.alerts_page.acknowledging')
									: $t('admin.alerts_page.acknowledge')}
							</button>
						{/if}
						<button
							type="button"
							onclick={() => deleteAlert(alert.id)}
							class="px-2 py-1 text-xs rounded text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400 transition-colors"
							aria-label={$t('admin.alerts_page.delete_aria', { values: { title: alert.title } })}
						>
							{$t('shared.actions.delete')}
						</button>
					</div>
				</li>
			{/each}
		</ul>
	{/if}
</div>
