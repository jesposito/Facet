<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { t } from 'svelte-i18n';
	import { toasts, confirm } from '$lib/stores';

	type Webhook = {
		id: string;
		label: string;
		url: string;
		events: string[];
		is_active: boolean;
		failure_count: number;
		last_delivered_at: string;
		last_failure_at: string;
		last_error: string;
		created: string;
	};

	type Delivery = {
		id: string;
		event_name: string;
		request_body: string;
		response_status: number;
		response_body: string;
		attempt_count: number;
		succeeded: boolean;
		created: string;
	};

	type TestResult = {
		status_code: number;
		response_body: string;
		succeeded: boolean;
		error?: string;
	};

	let loading = $state(true);
	let webhooks: Webhook[] = $state([]);
	let registeredEvents: string[] = $state([]);

	// New webhook form state
	let showAddForm = $state(false);
	let newLabel = $state('');
	let newUrl = $state('');
	let newEvents: string[] = $state([]);
	let newActive = $state(true);
	let urlError = $state('');
	let creating = $state(false);
	let createdSecret: string | null = $state(null);

	// Per-row state
	let testing: string | null = $state(null);
	let testResults: Record<string, TestResult> = $state({});

	// Dispatch-test (event picker → fires sample payload to ALL matching webhooks)
	let dispatchEvent = $state('');
	let dispatching = $state(false);
	let dispatchResult: { event: string; enqueued: number } | null = $state(null);
	let dispatchError = $state('');

	// Deliveries modal state
	let deliveriesOpen = $state(false);
	let deliveriesFor: Webhook | null = $state(null);
	let deliveries: Delivery[] = $state([]);
	let deliveriesLoading = $state(false);
	let deliveryPage = $state(1);
	let expandedDelivery: string | null = $state(null);
	let deliveriesPanelRef: HTMLDivElement | null = $state(null);
	let lastFocused: HTMLElement | null = null;

	onMount(async () => {
		await Promise.all([loadWebhooks(), loadEvents()]);
		loading = false;
	});

	async function loadWebhooks() {
		try {
			const res = await fetch('/api/admin/webhooks', {
				headers: { Authorization: pb.authStore.token }
			});
			if (!res.ok) throw new Error('failed');
			const json = await res.json();
			webhooks = json.data ?? [];
		} catch {
			toasts.error($t('admin.webhooks.load_error'));
			webhooks = [];
		}
	}

	async function loadEvents() {
		try {
			const res = await fetch('/api/admin/webhooks/events', {
				headers: { Authorization: pb.authStore.token }
			});
			if (!res.ok) throw new Error('failed');
			const json = await res.json();
			registeredEvents = json.events ?? [];
		} catch {
			registeredEvents = [];
		}
	}

	function validateUrl(u: string): string {
		if (!u) return $t('admin.webhooks.url_required');
		try {
			const parsed = new URL(u);
			if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
				return $t('admin.webhooks.url_https_only');
			}
		} catch {
			return $t('admin.webhooks.url_invalid');
		}
		return '';
	}

	function onUrlInput() {
		urlError = newUrl ? validateUrl(newUrl) : '';
	}

	function toggleEventInForm(ev: string) {
		newEvents = newEvents.includes(ev)
			? newEvents.filter((e) => e !== ev)
			: [...newEvents, ev];
	}

	function resetForm() {
		newLabel = '';
		newUrl = '';
		newEvents = [];
		newActive = true;
		urlError = '';
		createdSecret = null;
		showAddForm = false;
	}

	async function createWebhook(e: SubmitEvent) {
		e.preventDefault();
		urlError = validateUrl(newUrl);
		if (urlError) return;
		if (!newLabel.trim()) {
			toasts.error($t('admin.webhooks.label_required'));
			return;
		}
		if (newEvents.length === 0) {
			toasts.error($t('admin.webhooks.events_required'));
			return;
		}

		creating = true;
		try {
			const res = await fetch('/api/admin/webhooks', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({
					label: newLabel.trim(),
					url: newUrl.trim(),
					events: newEvents,
					is_active: newActive
				})
			});
			const json = await res.json();
			if (!res.ok) {
				toasts.error(json.error ?? $t('admin.webhooks.create_error'));
				return;
			}
			createdSecret = json.secret;
			toasts.success($t('admin.webhooks.created'));
			await loadWebhooks();
			newLabel = '';
			newUrl = '';
			newEvents = [];
			newActive = true;
		} catch {
			toasts.error($t('admin.webhooks.create_error'));
		} finally {
			creating = false;
		}
	}

	function copySecret() {
		if (createdSecret) {
			navigator.clipboard.writeText(createdSecret);
			toasts.success($t('admin.webhooks.secret_copied'));
		}
	}

	async function toggleActive(wh: Webhook) {
		try {
			const res = await fetch(`/api/admin/webhooks/${wh.id}`, {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json',
					Authorization: pb.authStore.token
				},
				body: JSON.stringify({ is_active: !wh.is_active })
			});
			if (!res.ok) throw new Error('failed');
			toasts.success(
				wh.is_active ? $t('admin.webhooks.disabled') : $t('admin.webhooks.enabled')
			);
			await loadWebhooks();
		} catch {
			toasts.error($t('admin.webhooks.update_error'));
		}
	}

	async function deleteWebhook(wh: Webhook) {
		const ok = await confirm({
			title: $t('admin.webhooks.delete_title'),
			message: $t('admin.webhooks.delete_message', { values: { label: wh.label } }),
			confirmText: $t('shared.actions.delete'),
			danger: true
		});
		if (!ok) return;

		try {
			const res = await fetch(`/api/admin/webhooks/${wh.id}`, {
				method: 'DELETE',
				headers: { Authorization: pb.authStore.token }
			});
			if (!res.ok) throw new Error('failed');
			toasts.success($t('admin.webhooks.deleted'));
			await loadWebhooks();
		} catch {
			toasts.error($t('admin.webhooks.delete_error'));
		}
	}

	async function runTest(wh: Webhook) {
		testing = wh.id;
		try {
			const res = await fetch(`/api/admin/webhooks/${wh.id}/test`, {
				method: 'POST',
				headers: { Authorization: pb.authStore.token }
			});
			const json: TestResult = await res.json();
			testResults[wh.id] = json;
		} catch {
			testResults[wh.id] = {
				status_code: 0,
				response_body: '',
				succeeded: false,
				error: $t('admin.webhooks.test_network_error')
			};
		} finally {
			testing = null;
		}
	}

	async function dispatchTest() {
		if (!dispatchEvent) return;
		dispatching = true;
		dispatchError = '';
		dispatchResult = null;
		try {
			const res = await fetch('/api/admin/webhooks/dispatch-test', {
				method: 'POST',
				headers: {
					Authorization: pb.authStore.token,
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ event: dispatchEvent })
			});
			const json = await res.json();
			if (!res.ok) {
				dispatchError = json.error || $t('admin.webhooks.dispatch_test_error_generic');
			} else {
				dispatchResult = { event: json.event, enqueued: json.enqueued };
			}
		} catch {
			dispatchError = $t('admin.webhooks.dispatch_test_network_error');
		} finally {
			dispatching = false;
		}
	}

	async function openDeliveries(wh: Webhook) {
		lastFocused = document.activeElement as HTMLElement;
		deliveriesFor = wh;
		deliveriesOpen = true;
		deliveryPage = 1;
		expandedDelivery = null;
		await loadDeliveries();
		// Focus the dialog after render
		setTimeout(() => deliveriesPanelRef?.focus(), 0);
	}

	async function loadDeliveries() {
		if (!deliveriesFor) return;
		deliveriesLoading = true;
		try {
			const res = await fetch(
				`/api/admin/webhooks/${deliveriesFor.id}/deliveries?page=${deliveryPage}&per_page=25`,
				{ headers: { Authorization: pb.authStore.token } }
			);
			const json = await res.json();
			deliveries = json.data ?? [];
		} catch {
			deliveries = [];
		} finally {
			deliveriesLoading = false;
		}
	}

	function closeDeliveries() {
		deliveriesOpen = false;
		deliveriesFor = null;
		deliveries = [];
		expandedDelivery = null;
		// Restore focus to the trigger
		setTimeout(() => lastFocused?.focus(), 0);
	}

	function onDialogKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.preventDefault();
			closeDeliveries();
			return;
		}
		// Simple focus trap: Tab cycles within the dialog
		if (e.key === 'Tab' && deliveriesPanelRef) {
			const focusable = deliveriesPanelRef.querySelectorAll<HTMLElement>(
				'a, button, input, select, textarea, [tabindex]:not([tabindex="-1"])'
			);
			if (focusable.length === 0) return;
			const first = focusable[0];
			const last = focusable[focusable.length - 1];
			if (e.shiftKey && document.activeElement === first) {
				e.preventDefault();
				last.focus();
			} else if (!e.shiftKey && document.activeElement === last) {
				e.preventDefault();
				first.focus();
			}
		}
	}

	function truncate(s: string, n: number) {
		if (s.length <= n) return s;
		return s.slice(0, n) + '…';
	}

	function fmtDate(s: string): string {
		if (!s) return '—';
		try {
			return new Date(s).toLocaleString();
		} catch {
			return s;
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.webhooks.page_title')}</title>
</svelte:head>

<div class="max-w-6xl mx-auto px-4 py-6">
	<header class="mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
			{$t('admin.webhooks.page_title')}
		</h1>
		<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
			{$t('admin.webhooks.page_intro')}
		</p>
	</header>

	{#if loading}
		<div class="text-center py-12 text-gray-500" role="status" aria-live="polite">
			{$t('admin.webhooks.loading')}
		</div>
	{:else}
		<section aria-labelledby="dispatch-test-heading" class="mb-8">
			<div class="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
				<h2 id="dispatch-test-heading" class="text-lg font-semibold text-gray-900 dark:text-white mb-1">
					{$t('admin.webhooks.dispatch_test_heading')}
				</h2>
				<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
					{$t('admin.webhooks.dispatch_test_intro')}
				</p>

				{#if registeredEvents.length === 0}
					<p class="text-sm text-gray-500">{$t('admin.webhooks.dispatch_test_no_events')}</p>
				{:else}
					<form onsubmit={(e) => { e.preventDefault(); dispatchTest(); }} class="flex flex-wrap items-end gap-3">
						<div class="flex-1 min-w-[240px]">
							<label for="dispatch-event-select" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
								{$t('admin.webhooks.dispatch_test_event_label')}
							</label>
							<select
								id="dispatch-event-select"
								bind:value={dispatchEvent}
								class="block w-full rounded-md border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
								required
								aria-required="true"
							>
								<option value="">{$t('admin.webhooks.dispatch_test_select_placeholder')}</option>
								{#each registeredEvents as ev}
									<option value={ev}>{ev}</option>
								{/each}
							</select>
						</div>
						<button
							type="submit"
							class="btn btn-secondary"
							disabled={dispatching || !dispatchEvent}
							aria-busy={dispatching ? 'true' : undefined}
						>
							{dispatching ? $t('admin.webhooks.dispatch_test_dispatching') : $t('admin.webhooks.dispatch_test_button')}
						</button>
					</form>

					<div aria-live="polite" aria-atomic="true" class="mt-3 min-h-[1.5rem]">
						{#if dispatchError}
							<p role="alert" class="text-sm text-red-700 dark:text-red-300 flex items-start gap-1.5">
								<svg class="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
								</svg>
								<span>{dispatchError}</span>
							</p>
						{:else if dispatchResult}
							{#if dispatchResult.enqueued === 0}
								<p role="status" class="text-sm text-amber-700 dark:text-amber-300 flex items-start gap-1.5">
									<svg class="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
									</svg>
									<span>{$t('admin.webhooks.dispatch_test_result_zero', { values: { event: dispatchResult.event } })}</span>
								</p>
							{:else}
								<p role="status" class="text-sm text-green-700 dark:text-green-300 flex items-start gap-1.5">
									<svg class="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
									</svg>
									<span>{$t('admin.webhooks.dispatch_test_result_success', { values: { event: dispatchResult.event, count: dispatchResult.enqueued } })}</span>
								</p>
							{/if}
						{/if}
					</div>
				{/if}
			</div>
		</section>

		<section aria-labelledby="webhooks-list-heading" class="mb-8">
			<div class="flex items-center justify-between mb-4">
				<h2 id="webhooks-list-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
					{$t('admin.webhooks.endpoints_heading')}
				</h2>
				<button
					type="button"
					class="btn btn-primary"
					onclick={() => {
						showAddForm = true;
						createdSecret = null;
					}}
				>
					{$t('admin.webhooks.new_webhook')}
				</button>
			</div>

			{#if webhooks.length === 0}
				<div class="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-8 text-center">
					<p class="text-gray-500 dark:text-gray-400">{$t('admin.webhooks.empty_state')}</p>
				</div>
			{:else}
				<div class="overflow-x-auto bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
					<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
						<caption class="sr-only">{$t('admin.webhooks.table_caption')}</caption>
						<thead class="bg-gray-50 dark:bg-gray-900">
							<tr>
								<th scope="col" class="px-4 py-2 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">{$t('admin.webhooks.col_label')}</th>
								<th scope="col" class="px-4 py-2 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">{$t('admin.webhooks.col_url')}</th>
								<th scope="col" class="px-4 py-2 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">{$t('admin.webhooks.col_events')}</th>
								<th scope="col" class="px-4 py-2 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">{$t('admin.webhooks.col_status')}</th>
								<th scope="col" class="px-4 py-2 text-left text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">{$t('admin.webhooks.col_last_delivered')}</th>
								<th scope="col" class="px-4 py-2 text-right text-xs font-semibold text-gray-600 dark:text-gray-300 uppercase tracking-wider">{$t('admin.webhooks.col_actions')}</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-200 dark:divide-gray-700">
							{#each webhooks as wh (wh.id)}
								<tr class="bg-white dark:bg-gray-800">
									<td class="px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">{wh.label}</td>
									<td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300 font-mono">
										<span title={wh.url}>{truncate(wh.url, 48)}</span>
									</td>
									<td class="px-4 py-3">
										<ul class="flex flex-wrap gap-1" aria-label={$t('admin.webhooks.events_for_row', { values: { label: wh.label } })}>
											{#each wh.events as ev}
												<li class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">{ev}</li>
											{/each}
										</ul>
									</td>
									<td class="px-4 py-3 text-sm">
										{#if !wh.is_active && wh.failure_count >= 10}
											<span class="inline-flex items-center gap-1.5 px-2 py-1 rounded-md text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200">
												<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/></svg>
												{$t('admin.webhooks.status_auto_disabled')}
											</span>
										{:else if wh.is_active}
											<span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">{$t('admin.webhooks.status_active')}</span>
										{:else}
											<span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200">{$t('admin.webhooks.status_disabled')}</span>
										{/if}
										{#if wh.failure_count > 0}
											<div class="mt-1">
												<span class="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200">
													<svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20" aria-hidden="true"><path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495z" clip-rule="evenodd"/></svg>
													{$t('admin.webhooks.failure_count', { values: { count: wh.failure_count } })}
												</span>
											</div>
										{/if}
									</td>
									<td class="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">{fmtDate(wh.last_delivered_at)}</td>
									<td class="px-4 py-3 text-sm text-right">
										<div class="inline-flex gap-2">
											<button
												type="button"
												class="btn btn-sm btn-secondary"
												onclick={() => runTest(wh)}
												disabled={testing === wh.id}
												aria-label={$t('admin.webhooks.test_aria', { values: { label: wh.label } })}
											>
												{testing === wh.id ? $t('admin.webhooks.testing') : $t('admin.webhooks.test')}
											</button>
											<button
												type="button"
												class="btn btn-sm btn-secondary"
												onclick={() => openDeliveries(wh)}
												aria-label={$t('admin.webhooks.deliveries_aria', { values: { label: wh.label } })}
											>
												{$t('admin.webhooks.deliveries')}
											</button>
											<button
												type="button"
												class="btn btn-sm btn-secondary"
												onclick={() => toggleActive(wh)}
											>
												{wh.is_active ? $t('admin.webhooks.disable') : $t('admin.webhooks.enable')}
											</button>
											<button
												type="button"
												class="btn btn-sm btn-danger"
												onclick={() => deleteWebhook(wh)}
												aria-label={$t('admin.webhooks.delete_aria', { values: { label: wh.label } })}
											>
												{$t('shared.actions.delete')}
											</button>
										</div>
										{#if testResults[wh.id]}
											<div
												class="mt-2 text-left p-2 rounded text-xs font-mono whitespace-pre-wrap break-all {testResults[wh.id].succeeded ? 'bg-green-50 dark:bg-green-900/30 text-green-800 dark:text-green-200' : 'bg-red-50 dark:bg-red-900/30 text-red-800 dark:text-red-200'}"
												role="status"
												aria-live="polite"
											>
												<strong>{$t('admin.webhooks.test_result_status', { values: { code: testResults[wh.id].status_code } })}</strong>
												{#if testResults[wh.id].error}
													<div>{$t('admin.webhooks.test_error_prefix')}: {testResults[wh.id].error}</div>
												{/if}
												{#if testResults[wh.id].response_body}
													<div class="mt-1">{truncate(testResults[wh.id].response_body, 300)}</div>
												{/if}
											</div>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>

		{#if showAddForm}
			<section aria-labelledby="new-webhook-heading" class="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-6">
				<h2 id="new-webhook-heading" class="text-lg font-semibold text-gray-900 dark:text-white mb-4">{$t('admin.webhooks.new_form_heading')}</h2>

				{#if createdSecret}
					<div class="mb-4 p-4 bg-amber-50 dark:bg-amber-900/30 border border-amber-200 dark:border-amber-800 rounded" role="status" aria-live="polite">
						<h3 class="font-semibold text-amber-900 dark:text-amber-100">{$t('admin.webhooks.secret_heading')}</h3>
						<p class="text-sm text-amber-800 dark:text-amber-200 mt-1">{$t('admin.webhooks.secret_warning')}</p>
						<div class="mt-2 flex items-center gap-2">
							<code class="flex-1 p-2 bg-white dark:bg-gray-900 rounded text-xs font-mono break-all">{createdSecret}</code>
							<button type="button" class="btn btn-sm btn-secondary" onclick={copySecret}>{$t('shared.actions.copy')}</button>
						</div>
						<button type="button" class="mt-3 btn btn-secondary" onclick={resetForm}>{$t('shared.actions.close')}</button>
					</div>
				{:else}
					<form onsubmit={createWebhook} novalidate>
						<div class="mb-4">
							<label for="wh-label" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{$t('admin.webhooks.label_field')}</label>
							<input
								id="wh-label"
								type="text"
								bind:value={newLabel}
								required
								maxlength="200"
								class="input w-full"
								placeholder={$t('admin.webhooks.label_placeholder')}
							/>
						</div>

						<div class="mb-4">
							<label for="wh-url" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{$t('admin.webhooks.url_field')}</label>
							<input
								id="wh-url"
								type="url"
								bind:value={newUrl}
								oninput={onUrlInput}
								required
								class="input w-full"
								placeholder="https://example.com/hooks/facet"
								aria-invalid={urlError ? 'true' : 'false'}
								aria-describedby={urlError ? 'wh-url-error' : 'wh-url-hint'}
							/>
							<p id="wh-url-hint" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{$t('admin.webhooks.url_hint')}</p>
							{#if urlError}
								<p id="wh-url-error" class="mt-1 text-xs text-red-600 dark:text-red-400">{urlError}</p>
							{/if}
						</div>

						<fieldset class="mb-4">
							<legend class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{$t('admin.webhooks.events_field')}</legend>
							{#if registeredEvents.length === 0}
								<p class="text-sm text-gray-500">{$t('admin.webhooks.no_events_available')}</p>
							{:else}
								<div class="space-y-2">
									{#each registeredEvents as ev}
										<label class="flex items-center gap-2 text-sm">
											<input
												type="checkbox"
												checked={newEvents.includes(ev)}
												onchange={() => toggleEventInForm(ev)}
												class="rounded text-primary-600"
											/>
											<span class="font-mono text-gray-800 dark:text-gray-200">{ev}</span>
										</label>
									{/each}
								</div>
							{/if}
						</fieldset>

						<div class="mb-4">
							<label class="flex items-center gap-2 text-sm">
								<input type="checkbox" bind:checked={newActive} class="rounded text-primary-600" />
								<span class="text-gray-800 dark:text-gray-200">{$t('admin.webhooks.active_field')}</span>
							</label>
						</div>

						<div class="flex gap-2 justify-end">
							<button type="button" class="btn btn-secondary" onclick={resetForm} disabled={creating}>{$t('shared.actions.cancel')}</button>
							<button type="submit" class="btn btn-primary" disabled={creating || !!urlError}>
								{creating ? $t('admin.webhooks.creating') : $t('admin.webhooks.create')}
							</button>
						</div>
					</form>
				{/if}
			</section>
		{/if}
	{/if}
</div>

<!-- Deliveries dialog -->
{#if deliveriesOpen && deliveriesFor}
	<!--
		Inert backdrop (aria-hidden + no handlers). Click-to-close via backdrop
		was removed because a clickable <div role="presentation"> trips the
		svelte/a11y-click-events-have-key-events rule and risks accidental
		dismissal. Users dismiss the dialog via the explicit Close button or
		Escape (onDialogKeydown).
	-->
	<!-- Backdrop is purely visual (no handlers, no role); a11y lives on the
		 dialog below. The dialog handles focus trap + Esc itself. -->
	<div class="fixed inset-0 bg-black/50 z-40 flex items-center justify-center p-4">
		<div
			bind:this={deliveriesPanelRef}
			class="bg-white dark:bg-gray-800 rounded-lg max-w-4xl w-full max-h-[80vh] overflow-hidden flex flex-col focus:outline-none"
			role="dialog"
			aria-modal="true"
			aria-labelledby="deliveries-dialog-heading"
			tabindex="-1"
			onkeydown={onDialogKeydown}
		>
			<header class="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700">
				<h2 id="deliveries-dialog-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
					{$t('admin.webhooks.deliveries_for', { values: { label: deliveriesFor.label } })}
				</h2>
				<button type="button" class="btn btn-sm btn-secondary" onclick={closeDeliveries}>
					{$t('shared.actions.close')}
				</button>
			</header>
			<div class="flex-1 overflow-y-auto p-4">
				{#if deliveriesLoading}
					<p class="text-center text-gray-500" role="status">{$t('admin.webhooks.loading')}</p>
				{:else if deliveries.length === 0}
					<p class="text-center text-gray-500">{$t('admin.webhooks.no_deliveries')}</p>
				{:else}
					<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
						<thead class="bg-gray-50 dark:bg-gray-900">
							<tr>
								<th scope="col" class="px-3 py-2 text-left text-xs font-semibold uppercase">{$t('admin.webhooks.col_event')}</th>
								<th scope="col" class="px-3 py-2 text-left text-xs font-semibold uppercase">{$t('admin.webhooks.col_response_code')}</th>
								<th scope="col" class="px-3 py-2 text-left text-xs font-semibold uppercase">{$t('admin.webhooks.col_attempts')}</th>
								<th scope="col" class="px-3 py-2 text-left text-xs font-semibold uppercase">{$t('admin.webhooks.col_succeeded')}</th>
								<th scope="col" class="px-3 py-2 text-left text-xs font-semibold uppercase">{$t('admin.webhooks.col_when')}</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-200 dark:divide-gray-700">
							{#each deliveries as d (d.id)}
								<tr>
									<td colspan="5" class="p-0">
										<button
											type="button"
											class="w-full text-left px-3 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/50 grid grid-cols-5 gap-2 items-center"
											onclick={() => (expandedDelivery = expandedDelivery === d.id ? null : d.id)}
											aria-expanded={expandedDelivery === d.id}
											aria-controls={'delivery-detail-' + d.id}
										>
											<span class="text-sm font-mono">{d.event_name}</span>
											<span class="text-sm">{d.response_status || '—'}</span>
											<span class="text-sm">{d.attempt_count}</span>
											<span class="text-sm">
												{#if d.succeeded}
													<span class="text-green-700 dark:text-green-400">{$t('admin.webhooks.succeeded')}</span>
												{:else}
													<span class="text-red-700 dark:text-red-400">{$t('admin.webhooks.failed')}</span>
												{/if}
											</span>
											<span class="text-sm text-gray-500">{fmtDate(d.created)}</span>
										</button>
										{#if expandedDelivery === d.id}
											<div id={'delivery-detail-' + d.id} class="px-3 py-3 bg-gray-50 dark:bg-gray-900/50">
												<h3 class="text-sm font-semibold mb-1 text-gray-900 dark:text-white">{$t('admin.webhooks.request_body')}</h3>
												<pre class="text-xs font-mono bg-white dark:bg-gray-900 p-2 rounded overflow-x-auto whitespace-pre-wrap break-all">{d.request_body || '—'}</pre>
												<h3 class="text-sm font-semibold mt-3 mb-1 text-gray-900 dark:text-white">{$t('admin.webhooks.response_body')}</h3>
												<pre class="text-xs font-mono bg-white dark:bg-gray-900 p-2 rounded overflow-x-auto whitespace-pre-wrap break-all">{d.response_body || '—'}</pre>
											</div>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>

					<nav class="mt-4 flex items-center justify-between" aria-label={$t('admin.webhooks.pagination_aria')}>
						<button
							type="button"
							class="btn btn-sm btn-secondary"
							disabled={deliveryPage <= 1}
							onclick={async () => { deliveryPage--; await loadDeliveries(); }}
						>
							{$t('admin.webhooks.prev_page')}
						</button>
						<span class="text-sm text-gray-600 dark:text-gray-400">{$t('admin.webhooks.page_n', { values: { page: deliveryPage } })}</span>
						<button
							type="button"
							class="btn btn-sm btn-secondary"
							disabled={deliveries.length < 25}
							onclick={async () => { deliveryPage++; await loadDeliveries(); }}
						>
							{$t('admin.webhooks.next_page')}
						</button>
					</nav>
				{/if}
			</div>
		</div>
	</div>
{/if}
