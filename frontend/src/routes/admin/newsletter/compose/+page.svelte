<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { beforeNavigate } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';
	import { marked } from 'marked';
	import DOMPurify from 'isomorphic-dompurify';
	import {
		ALL_SUBSCRIBERS,
		buildSendPayload,
		findLinkPlaceholderProblem,
		newRequestKey,
		sendErrorMessage
	} from '$lib/newsletter/compose';
	import {
		outcomeCounts,
		resumeState,
		smtpReadiness,
		type NewsletterSend as Send,
		type SmtpReadiness
	} from '$lib/newsletter/sends';

	type Template = {
		slug: string;
		name: string;
		description: string;
	};

	type NewsletterList = {
		id: string;
		slug: string;
		name: string;
		subscriber_count: number;
		active_subscriber_count?: number;
		is_default: boolean;
		is_active: boolean;
	};

	let subject = $state('');
	let bodyMarkdown = $state('');
	let previewHTML = $state('');
	let sending = $state(false);
	let sendingTest = $state(false);
	let sends: Send[] = $state([]);
	let sendsLoading = $state(true);
	let templates: Template[] = $state([]);
	let selectedTemplate = $state('');
	// Optimistic until the status call answers, so the page doesn't flash a warning.
	let smtpStatus: SmtpReadiness = $state('ready');
	let smtpReady = $derived(smtpStatus === 'ready');
	let anyUncertain = $derived(sends.some((s) => (s.uncertain_count ?? 0) > 0));

	let lists: NewsletterList[] = $state([]);
	// '' is the explicit "All active subscribers" choice. Whatever is picked
	// here is exactly what the preview, the count and the send use.
	let selectedListId = $state(ALL_SUBSCRIBERS);
	let audienceListId = $derived(selectedListId);
	let personalize = $state(false);
	// The server refuses these links; saying so while writing beats a
	// failed send.
	let linkProblem = $derived(findLinkPlaceholderProblem(bodyMarkdown, personalize));
	let placeholderHintDismissed = $state(false);
	let placeholderPanelOpen = $state(false);

	// Saved-state snapshot powers the unsaved-changes navigation guard.
	let lastSavedSubject = $state('');
	let lastSavedBody = $state('');

	// Poll send status with exponential backoff while a send is in progress.
	let pollTimer: ReturnType<typeof setTimeout> | null = null;
	let pollInterval = 2000;
	const POLL_MIN = 2000;
	const POLL_MAX = 10000;

	// Placeholder shim — local self-hosted has no Liquid engine. We support
	// flat {{ subscriber.name }}, {{ subscriber.email }}, {{ list.name }},
	// {{ site.name }}, {{ site.url }} via plain substitution at send time
	// (server-side; this disclosure is documentation for the author).
	const PLACEHOLDERS: { path: string; desc: string }[] = [
		{ path: '{{ subscriber.name }}', desc: $t('admin.newsletter.ph_subscriber_name') },
		{ path: '{{ subscriber.email }}', desc: $t('admin.newsletter.ph_subscriber_email') },
		{ path: '{{ list.name }}', desc: $t('admin.newsletter.ph_list_name') },
		{ path: '{{ site.name }}', desc: $t('admin.newsletter.ph_site_name') },
		{ path: '{{ site.url }}', desc: $t('admin.newsletter.ph_site_url') }
	];

	// Regex picks up either {{ ... }} or {% ... %} authoring markers so we
	// can prompt the user to enable personalization. Deliberately liberal —
	// false positives prompt an opt-in, never block sending.
	let bodyHasPlaceholders = $derived(
		(subject && (subject.includes('{{') || subject.includes('{%'))) ||
			(bodyMarkdown && (bodyMarkdown.includes('{{') || bodyMarkdown.includes('{%')))
	);

	let hasUnsavedContent = $derived(subject !== lastSavedSubject || bodyMarkdown !== lastSavedBody);

	// Sanitize preview HTML with DOMPurify before {@html}.
	$effect(() => {
		if (bodyMarkdown) {
			previewHTML = DOMPurify.sanitize(marked(bodyMarkdown) as string);
		} else {
			previewHTML = '';
		}
	});

	beforeNavigate((navigation) => {
		if (navigation.shallow && navigation.type === 'goto') return;

		if (hasUnsavedContent && !sending) {
			if (!window.confirm($t('admin.newsletter.unsaved_warning'))) {
				navigation.cancel();
			}
		}
	});

	function handleBeforeUnload(e: BeforeUnloadEvent) {
		if (hasUnsavedContent && !sending) {
			e.preventDefault();
		}
	}

	function authHeaders(): Record<string, string> {
		return pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {};
	}

	function jsonHeaders(): Record<string, string> {
		return { 'Content-Type': 'application/json', ...authHeaders() };
	}

	onMount(async () => {
		window.addEventListener('beforeunload', handleBeforeUnload);
		await Promise.all([loadSends(), loadTemplates(), loadLists(), checkSmtpStatus()]);
		startPollingIfNeeded();
	});

	async function checkSmtpStatus() {
		try {
			const resp = await fetch('/api/admin/newsletter/smtp-status', { headers: authHeaders() });
			if (resp.ok) {
				smtpStatus = smtpReadiness(await resp.json());
			}
		} catch (err) {
			console.debug('Failed to check SMTP status:', err);
		}
	}

	async function loadTemplates() {
		try {
			const resp = await fetch('/api/admin/newsletter/templates', { headers: authHeaders() });
			if (resp.ok) {
				templates = await resp.json();
			}
		} catch (err) {
			console.debug('Failed to load templates:', err);
		}
	}

	async function loadLists() {
		try {
			const resp = await fetch('/api/admin/newsletter-lists', { headers: authHeaders() });
			if (!resp.ok) return;
			const data = await resp.json();
			lists = (data.data ?? []).filter((l: NewsletterList) => l.is_active);
			// A list that was picked and has since gone inactive can't be
			// sent to; fall back to the visible "All active subscribers".
			if (selectedListId && !lists.some((l) => l.id === selectedListId)) {
				selectedListId = ALL_SUBSCRIBERS;
			}
		} catch (err) {
			console.debug('Failed to load newsletter lists:', err);
		}
	}

	onDestroy(() => {
		if (typeof window !== 'undefined') {
			window.removeEventListener('beforeunload', handleBeforeUnload);
		}
		stopPolling();
	});

	function startPollingIfNeeded() {
		const hasSending = sends.some((s) => s.status === 'sending');
		if (hasSending) {
			pollInterval = POLL_MIN;
			schedulePoll();
		}
	}

	function schedulePoll() {
		stopPolling();
		pollTimer = setTimeout(async () => {
			await loadSends();
			const hasSending = sends.some((s) => s.status === 'sending');
			if (hasSending) {
				pollInterval = Math.min(pollInterval * 2, POLL_MAX);
				schedulePoll();
			}
		}, pollInterval);
	}

	function stopPolling() {
		if (pollTimer) {
			clearTimeout(pollTimer);
			pollTimer = null;
		}
	}

	async function loadSends() {
		sendsLoading = true;
		try {
			const response = await fetch('/api/admin/newsletter/sends', { headers: authHeaders() });
			if (!response.ok) throw new Error('Failed to load sends');
			const data = await response.json();
			sends = data.data || [];
		} catch (err) {
			console.error('Failed to load send history:', err);
		} finally {
			sendsLoading = false;
		}
	}

	async function copyPlaceholder(path: string) {
		try {
			await navigator.clipboard?.writeText(path);
			toasts.add('success', $t('admin.newsletter.copied_placeholder', { values: { path } }));
		} catch {
			// Clipboard API unavailable (insecure context); user can hand-copy.
		}
	}

	async function sendTest() {
		if (!subject.trim() || !bodyMarkdown.trim()) {
			toasts.add('error', $t('admin.newsletter.subject_body_required'));
			return;
		}

		sendingTest = true;
		try {
			const adminEmail = (pb.authStore.record as Record<string, unknown>)?.email as
				string | undefined;

			if (!adminEmail) {
				toasts.add('error', $t('admin.newsletter.no_admin_email'));
				return;
			}

			const response = await fetch('/api/admin/newsletter/send', {
				method: 'POST',
				headers: jsonHeaders(),
				body: JSON.stringify({
					...buildSendPayload({
						subject,
						bodyMarkdown,
						template: selectedTemplate,
						listId: audienceListId,
						personalize
					}),
					test_email: adminEmail
				})
			});

			if (!response.ok) {
				const data = await response.json().catch(() => ({}));
				throw new Error(sendErrorMessage(data, 'Failed to send test'));
			}

			toasts.add('success', $t('admin.newsletter.test_sent'));
		} catch (err) {
			console.error('Failed to send test:', err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.newsletter.error_send'));
		} finally {
			sendingTest = false;
		}
	}

	async function sendToAll() {
		if (!subject.trim() || !bodyMarkdown.trim()) {
			toasts.add('error', $t('admin.newsletter.subject_body_required'));
			return;
		}

		if (linkProblem) {
			toasts.add('error', linkProblem);
			return;
		}

		// The count comes from the same audience query the send uses: active,
		// confirmed members of the selected list.
		const audience = await loadAudiencePreview();
		if (!audience) {
			toasts.add('error', audiencePreviewError || $t('admin.newsletter.error_send'));
			return;
		}
		if (audience.recipient_count === 0) {
			toasts.add('error', $t('admin.newsletter.no_active_subscribers'));
			return;
		}

		const confirmed = await confirm({
			title: $t('admin.newsletter.send_confirm_title'),
			message: $t('admin.newsletter.send_confirm_message_count', {
				values: { count: audience.recipient_count }
			}),
			confirmText: $t('admin.newsletter.send_confirm_button'),
			danger: false
		});
		if (!confirmed) return;

		sending = true;
		try {
			// One key per Send click. If the network drops the reply, the
			// retry carries the same key and the server hands back the send
			// it already started instead of mailing everyone again.
			const payload = buildSendPayload({
				subject,
				bodyMarkdown,
				template: selectedTemplate,
				listId: audienceListId,
				personalize,
				requestKey: newRequestKey()
			});

			const post = async (body: Record<string, unknown>) => {
				const init = { method: 'POST', headers: jsonHeaders(), body: JSON.stringify(body) };
				try {
					return await fetch('/api/admin/newsletter/send', init);
				} catch {
					return await fetch('/api/admin/newsletter/send', init);
				}
			};

			let response = await post(payload);

			// The server refuses to repeat a newsletter that already went to
			// this audience. Sending it again is a separate, explicit choice,
			// and a new request.
			if (response.status === 409) {
				const conflict = await response.json().catch(() => ({}));
				if (conflict.code !== 'already_sent') {
					throw new Error(sendErrorMessage(conflict, 'Failed to send'));
				}
				const resend = await confirm({
					title: $t('admin.newsletter.send_confirm_title'),
					message: conflict.error,
					confirmText: $t('admin.newsletter.send_confirm_button'),
					danger: true
				});
				if (!resend) return;
				response = await post({ ...payload, request_key: newRequestKey(), resend: true });
			}

			if (!response.ok) {
				const data = await response.json().catch(() => ({}));
				throw new Error(sendErrorMessage(data, 'Failed to send'));
			}

			const data = await response.json();
			toasts.add(
				'success',
				$t('admin.newsletter.send_started', { values: { count: data.recipient_count } })
			);
			subject = '';
			bodyMarkdown = '';
			lastSavedSubject = '';
			lastSavedBody = '';
			await loadSends();
			pollInterval = POLL_MIN;
			schedulePoll();
		} catch (err) {
			console.error('Failed to send newsletter:', err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.newsletter.error_send'));
		} finally {
			sending = false;
		}
	}

	type AudiencePreview = {
		recipient_count: number;
		list_name: string;
		sample_email: string;
		subject: string;
		body_html: string;
	};

	let audiencePreviewError = $state('');

	async function loadAudiencePreview(): Promise<AudiencePreview | null> {
		audiencePreviewError = '';
		try {
			const resp = await fetch('/api/admin/newsletter/preview', {
				method: 'POST',
				headers: jsonHeaders(),
				body: JSON.stringify(
					buildSendPayload({ subject, bodyMarkdown, listId: audienceListId, personalize })
				)
			});
			if (!resp.ok) {
				const data = await resp.json().catch(() => ({}));
				audiencePreviewError = sendErrorMessage(data, '');
				return null;
			}
			return (await resp.json()) as AudiencePreview;
		} catch (err) {
			console.debug('Failed to load audience preview:', err);
			return null;
		}
	}

	// Personalized preview: what the first recipient of the selected list
	// will see, rendered by the server with the same values the send uses.
	let personalizedPreview: AudiencePreview | null = $state(null);
	let previewTimer: ReturnType<typeof setTimeout> | null = null;
	$effect(() => {
		// Track the inputs that change the rendered preview.
		const active = personalize && !!bodyMarkdown;
		void subject;
		void audienceListId;
		if (previewTimer) clearTimeout(previewTimer);
		if (!active) {
			personalizedPreview = null;
			return;
		}
		previewTimer = setTimeout(async () => {
			personalizedPreview = await loadAudiencePreview();
		}, 400);
	});

	// Resume retries only pending and known rejected recipients. The server
	// never re-sends to accepted or uncertain ones.
	async function resumeSend(send: Send) {
		const confirmed = await confirm({
			title: $t('admin.newsletter.resume_send_title'),
			message: $t('admin.newsletter.resume_send_message'),
			confirmText: $t('admin.newsletter.resume_send'),
			danger: false
		});
		if (!confirmed) return;
		try {
			const response = await fetch(`/api/admin/newsletter/sends/${send.id}/resume`, {
				method: 'POST',
				headers: authHeaders()
			});
			if (!response.ok) {
				const data = await response.json().catch(() => ({}));
				throw new Error(sendErrorMessage(data, $t('admin.newsletter.error_resume')));
			}
			toasts.add('success', $t('admin.newsletter.resume_started'));
			await loadSends();
			pollInterval = POLL_MIN;
			schedulePoll();
		} catch (err) {
			console.error('Failed to resume send:', err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.newsletter.error_resume'));
		}
	}

	async function deleteSend(send: Send) {
		const confirmed = await confirm({
			title: $t('admin.newsletter.delete_send_title'),
			message: $t('admin.newsletter.delete_send_message'),
			confirmText: $t('admin.newsletter.delete_confirm'),
			danger: true
		});
		if (!confirmed) return;

		try {
			const response = await fetch(`/api/admin/newsletter/sends/${send.id}`, {
				method: 'DELETE',
				headers: authHeaders()
			});
			if (!response.ok) throw new Error('Failed to delete');
			toasts.add('success', $t('admin.newsletter.send_deleted'));
			await loadSends();
		} catch (err) {
			console.error('Failed to delete send:', err);
			toasts.add('error', $t('admin.newsletter.error_delete'));
		}
	}

	function getStatusClass(status: string): string {
		switch (status) {
			case 'sent':
				return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300';
			case 'sending':
				return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300';
			case 'failed':
				return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300';
			case 'partial':
			case 'interrupted':
				return 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300';
			default:
				return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400';
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.newsletter.compose_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
				{$t('admin.newsletter.compose_title')}
			</h1>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
				{$t('admin.newsletter.compose_description')}
			</p>
		</div>
		<a href="/admin/newsletter" class="btn btn-ghost">
			{$t('admin.newsletter.back_to_newsletter')}
		</a>
	</div>

	<!-- SMTP warning -->
	{#if !smtpReady}
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
				></path>
			</svg>
			<div>
				<p class="text-sm font-medium text-amber-800 dark:text-amber-300">
					{smtpStatus === 'not_ready'
						? $t('admin.newsletter.smtp_not_ready')
						: $t('admin.newsletter.smtp_not_configured')}
				</p>
				<p class="text-sm text-amber-700 dark:text-amber-400 mt-0.5">
					{smtpStatus === 'not_ready'
						? $t('admin.newsletter.smtp_not_ready_hint')
						: $t('admin.newsletter.smtp_not_configured_hint')}
				</p>
			</div>
		</div>
	{/if}

	<!-- Template Picker -->
	{#if templates.length > 0}
		<div class="card p-4">
			<span class="label text-xs mb-2 block">{$t('admin.newsletter.template_label')}</span>
			<div class="flex flex-wrap gap-3">
				<button
					type="button"
					class="px-3 py-2 text-sm rounded-lg border transition-colors
						{selectedTemplate === ''
						? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300 dark:border-primary-500'
						: 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800'}"
					aria-pressed={selectedTemplate === ''}
					onclick={() => (selectedTemplate = '')}>{$t('admin.newsletter.template_default')}</button
				>

				{#each templates as tpl}
					<button
						type="button"
						class="px-3 py-2 text-sm rounded-lg border transition-colors
							{selectedTemplate === tpl.slug
							? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300 dark:border-primary-500'
							: 'border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-50 dark:hover:bg-gray-800'}"
						aria-pressed={selectedTemplate === tpl.slug}
						onclick={() => (selectedTemplate = tpl.slug)}
						title={tpl.description}
					>
						{tpl.name}
					</button>
				{/each}
			</div>
			{#if selectedTemplate}
				{@const activeTpl = templates.find((tpl) => tpl.slug === selectedTemplate)}
				{#if activeTpl}
					<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">{activeTpl.description}</p>
				{/if}
			{/if}
		</div>
	{/if}

	<!-- Compose Form -->
	<div class="card p-6 space-y-4">
		<div>
			<label for="newsletter-subject" class="label text-xs">
				{$t('admin.newsletter.subject_label')}
				<span aria-hidden="true"> *</span>
			</label>
			<input
				type="text"
				id="newsletter-subject"
				bind:value={subject}
				class="input text-sm w-full"
				placeholder={$t('admin.newsletter.subject_placeholder')}
				required
				aria-required="true"
			/>
		</div>

		<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
			<div>
				<label for="newsletter-body" class="label text-xs">
					{$t('admin.newsletter.body_label')}
					<span aria-hidden="true"> *</span>
				</label>
				<textarea
					id="newsletter-body"
					bind:value={bodyMarkdown}
					class="input text-sm w-full font-mono resize-y"
					style="min-height: 300px;"
					placeholder={$t('admin.newsletter.body_placeholder')}
					required
					aria-required="true"></textarea>
			</div>
			<div>
				<span class="label text-xs">{$t('admin.newsletter.preview_label')}</span>
				<div
					class="border border-gray-200 dark:border-gray-700 rounded-lg p-4 bg-white dark:bg-gray-800 min-h-[300px] overflow-auto prose prose-sm dark:prose-invert max-w-none"
					aria-label={$t('admin.newsletter.preview_label')}
				>
					{#if personalizedPreview && personalizedPreview.sample_email}
						<p
							class="not-prose text-xs text-gray-500 dark:text-gray-400 mb-2"
							data-testid="personalized-preview-recipient"
						>
							{personalizedPreview.sample_email} · {personalizedPreview.subject}
						</p>
						{@html DOMPurify.sanitize(personalizedPreview.body_html)}
					{:else if previewHTML}
						{@html previewHTML}
					{:else}
						<p class="text-gray-400 dark:text-gray-500 italic">
							{$t('admin.newsletter.preview_empty')}
						</p>
					{/if}
				</div>
			</div>
		</div>

		<!-- Audience. The choice made here is the one the send uses, and the
		     numbers are active, confirmed readers: the people who get mail. -->
		{#if lists.length > 0}
			<div class="border-t border-gray-200 dark:border-gray-700 pt-4">
				<fieldset>
					<legend class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 block">
						{$t('admin.newsletter.send_to_list')}
					</legend>
					<div class="flex flex-wrap items-center gap-3">
						<label class="inline-flex items-center gap-1.5 text-sm cursor-pointer">
							<input
								type="radio"
								name="composeList"
								value={ALL_SUBSCRIBERS}
								bind:group={selectedListId}
								class="text-primary-600"
							/>
							<span class="font-medium">
								{$t('admin.newsletter.all_active_subscribers', {
									default: 'All active subscribers'
								})}
							</span>
						</label>
						{#each lists as list (list.id)}
							<label class="inline-flex items-center gap-1.5 text-sm cursor-pointer">
								<input
									type="radio"
									name="composeList"
									value={list.id}
									bind:group={selectedListId}
									class="text-primary-600"
								/>
								<span>
									<span class="font-medium">{list.name}</span>
									<span class="ms-1 text-xs text-gray-500 dark:text-gray-400">
										({list.active_subscriber_count ?? list.subscriber_count})
									</span>
									{#if list.is_default}
										<span class="ms-1 text-xs text-gray-500 dark:text-gray-400">
											· {$t('admin.newsletter.list_default_marker')}
										</span>
									{/if}
								</span>
							</label>
						{/each}
					</div>
				</fieldset>
			</div>
		{/if}

		{#if linkProblem}
			<p
				class="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-200"
				role="alert"
				data-testid="link-placeholder-problem"
			>
				{linkProblem}
			</p>
		{/if}

		<!-- Personalization toggle + placeholder reference. The shim is
		     authoring-only: backend substitution lives in the future Liquid
		     port. We surface the toggle to keep the cloud-compatible UX. -->
		<div class="border-t border-gray-200 dark:border-gray-700 pt-4">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<label class="inline-flex items-center gap-2 text-sm cursor-pointer">
					<input
						type="checkbox"
						bind:checked={personalize}
						class="h-4 w-4 rounded border-gray-300 dark:border-gray-600"
					/>
					<span class="font-medium text-gray-700 dark:text-gray-300">
						{$t('admin.newsletter.personalize_label')}
					</span>
				</label>
				<button
					type="button"
					onclick={() => (placeholderPanelOpen = !placeholderPanelOpen)}
					aria-expanded={placeholderPanelOpen}
					aria-controls="placeholder-vars-panel"
					class="text-xs text-primary-700 hover:underline dark:text-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-500 rounded"
				>
					{placeholderPanelOpen
						? $t('admin.newsletter.hide_variables')
						: $t('admin.newsletter.show_variables')}
				</button>
			</div>
			{#if bodyHasPlaceholders && !personalize && !placeholderHintDismissed}
				<div
					role="alert"
					class="mt-2 flex items-start justify-between gap-3 rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:bg-amber-900/30 dark:text-amber-100"
				>
					<span>
						{$t('admin.newsletter.placeholder_hint')}
					</span>
					<button
						type="button"
						class="text-amber-900 underline hover:no-underline dark:text-amber-100"
						onclick={() => (personalize = true)}
						>{$t('admin.newsletter.placeholder_hint_enable')}</button
					>

					<button
						type="button"
						aria-label={$t('admin.newsletter.placeholder_hint_dismiss_aria')}
						onclick={() => (placeholderHintDismissed = true)}
						class="text-amber-900 hover:underline dark:text-amber-100"
					>
						{$t('admin.newsletter.placeholder_hint_dismiss')}
					</button>
				</div>
			{/if}
			{#if placeholderPanelOpen}
				<div
					id="placeholder-vars-panel"
					class="mt-3 rounded-md border border-gray-200 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-800/50"
				>
					<p class="mb-2 text-xs text-gray-600 dark:text-gray-400">
						{$t('admin.newsletter.placeholder_copy_hint')}
					</p>
					<ul class="space-y-1">
						{#each PLACEHOLDERS as v (v.path)}
							<li>
								<button
									type="button"
									class="group flex w-full items-start justify-between gap-3 rounded px-2 py-1 text-left hover:bg-gray-100 dark:hover:bg-gray-800"
									onclick={() => copyPlaceholder(v.path)}
									aria-label={$t('admin.newsletter.placeholder_copy_aria', {
										values: { path: v.path, desc: v.desc }
									})}
								>
									<code class="text-xs font-mono text-gray-800 dark:text-gray-100">{v.path}</code>
									<span class="text-xs text-gray-600 dark:text-gray-400">{v.desc}</span>
								</button>
							</li>
						{/each}
					</ul>
				</div>
			{/if}
		</div>

		<div class="flex gap-2 justify-end border-t border-gray-200 dark:border-gray-700 pt-4">
			<button
				type="button"
				class="btn btn-secondary text-sm"
				onclick={sendTest}
				disabled={sendingTest || !subject.trim() || !bodyMarkdown.trim() || !smtpReady}
				aria-busy={sendingTest ? 'true' : undefined}
				title={!smtpReady ? $t('admin.newsletter.smtp_not_configured') : undefined}
			>
				{sendingTest ? $t('admin.newsletter.sending_test') : $t('admin.newsletter.send_test')}
			</button>
			<button
				type="button"
				class="btn btn-primary text-sm"
				onclick={sendToAll}
				disabled={sending || !subject.trim() || !bodyMarkdown.trim() || !smtpReady}
				aria-busy={sending ? 'true' : undefined}
				title={!smtpReady ? $t('admin.newsletter.smtp_not_configured') : undefined}
			>
				{sending ? $t('admin.newsletter.sending') : $t('admin.newsletter.send_to_all')}
			</button>
		</div>
	</div>

	<!-- Send History -->
	<div class="card">
		<div class="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
				{$t('admin.newsletter.send_history')}
			</h2>
		</div>
		{#if sendsLoading}
			<div
				class="flex items-center justify-center py-8"
				role="status"
				aria-label={$t('admin.newsletter.loading_history')}
			>
				<svg
					class="animate-spin h-6 w-6 text-primary-600"
					fill="none"
					viewBox="0 0 24 24"
					aria-hidden="true"
				>
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
					></circle>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					></path>
				</svg>
			</div>
		{:else if sends.length === 0}
			<div class="text-center py-8 px-6">
				<p class="text-sm text-gray-500 dark:text-gray-400">
					{$t('admin.newsletter.no_sends')}
				</p>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-sm" aria-label={$t('admin.newsletter.send_history')}>
					<thead>
						<tr class="border-b border-gray-200 dark:border-gray-700">
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.newsletter.col_subject')}</th
							>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.newsletter.col_status')}</th
							>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.newsletter.col_recipients')}</th
							>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.newsletter.col_engagement')}</th
							>
							<th
								class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.newsletter.col_sent_at')}</th
							>
							<th
								class="text-right py-3 px-4 font-medium text-gray-500 dark:text-gray-400"
								scope="col">{$t('admin.newsletter.col_actions')}</th
							>
						</tr>
					</thead>
					<tbody>
						{#each sends as send (send.id)}
							<tr
								class="border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/50"
							>
								<td class="py-3 px-4 text-gray-900 dark:text-white font-medium">
									{send.subject}
								</td>
								<td class="py-3 px-4">
									<span
										class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium {getStatusClass(
											send.status
										)}"
									>
										{$t(`admin.newsletter.status_${send.status}`)}
									</span>
								</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400">
									{send.sent_count}/{send.recipient_count}
									<ul class="flex flex-col text-xs mt-1" data-testid="send-outcomes">
										{#each outcomeCounts(send) as { key, count } (key)}
											<li
												class={key === 'failed_count'
													? 'text-red-600 dark:text-red-400'
													: key === 'uncertain_count'
														? 'text-amber-700 dark:text-amber-400'
														: ''}
											>
												{$t(`admin.newsletter.${key}`, { values: { count } })}
											</li>
										{/each}
									</ul>
								</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400">
									{#if send.status === 'sent' && send.recipient_count > 0}
										<span class="inline-flex items-center gap-2 text-xs">
											<span title="Opens">
												{send.open_count || 0}
												{$t('admin.newsletter.opens', { values: { count: send.open_count || 0 } })}
												<span class="text-gray-400"
													>({Math.round(
														((send.open_count || 0) / send.recipient_count) * 100
													)}%)</span
												>
											</span>

											<span title="Clicks">
												{send.click_count || 0}
												{$t('admin.newsletter.clicks', {
													values: { count: send.click_count || 0 }
												})}
											</span>
										</span>
									{:else}
										<span class="text-gray-400">—</span>
									{/if}
								</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400">
									{send.sent_at ? new Date(send.sent_at).toLocaleString() : '—'}
								</td>
								<td class="py-3 px-4 text-right whitespace-nowrap">
									{#if resumeState(send, sends) !== 'hidden'}
										{@const blocked = resumeState(send, sends) === 'blocked'}
										<button
											type="button"
											class="btn btn-secondary text-xs me-2"
											onclick={() => resumeSend(send)}
											disabled={blocked || !smtpReady}
											title={blocked ? $t('admin.newsletter.send_in_progress') : undefined}
											aria-label={$t('admin.newsletter.resume_send_aria', {
												values: { subject: send.subject }
											})}
										>
											{$t('admin.newsletter.resume_send')}
										</button>
									{/if}
									<button
										type="button"
										class="p-1.5 text-gray-400 hover:text-red-500 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
										disabled={send.status === 'sending'}
										onclick={() => deleteSend(send)}
										aria-label={$t('admin.newsletter.delete_send_aria', {
											values: { subject: send.subject }
										})}
									>
										<svg
											class="w-4 h-4"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
											aria-hidden="true"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
											></path>
										</svg>
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			{#if anyUncertain}
				<p
					class="px-4 py-3 text-xs text-gray-600 dark:text-gray-400 border-t border-gray-200 dark:border-gray-700"
					data-testid="uncertain-explainer"
				>
					{$t('admin.newsletter.uncertain_hint')}
				</p>
			{/if}
		{/if}
	</div>
</div>
