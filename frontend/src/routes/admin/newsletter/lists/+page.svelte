<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';

	// NewsletterList matches backend hooks/newsletter_list.go#serializeList.
	type NewsletterList = {
		id: string;
		name: string;
		slug: string;
		description: string;
		sender_name: string;
		reply_to: string;
		welcome_subject: string;
		welcome_html: string;
		is_default: boolean;
		is_active: boolean;
		subscriber_count: number;
		created: string;
		updated: string;
	};

	type ListsResponse = {
		data: NewsletterList[];
		cap: number; // 0 = unlimited (self-hosted always 0)
		count: number;
		at_limit: boolean;
	};

	let lists: NewsletterList[] = $state([]);
	let cap = $state(0);
	let atLimit = $state(false);
	let loading = $state(true);
	let liveMessage = $state('');
	let editingId = $state<string | null>(null);
	let actingId = $state<string | null>(null);
	let showNewForm = $state(false);

	// New-list form state.
	let newName = $state('');
	let newSlug = $state('');
	let newDescription = $state('');
	let newSenderName = $state('');
	let newReplyTo = $state('');
	let newError = $state('');
	let newSubmitting = $state(false);

	// Edit-form state (one row at a time).
	let editName = $state('');
	let editSlug = $state('');
	let editDescription = $state('');
	let editSenderName = $state('');
	let editReplyTo = $state('');
	let editWelcomeSubject = $state('');
	let editWelcomeHtml = $state('');
	let editIsActive = $state(true);
	let editError = $state('');
	let editSubmitting = $state(false);

	function authHeaders(): Record<string, string> {
		return pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {};
	}

	function jsonHeaders(): Record<string, string> {
		return { 'Content-Type': 'application/json', ...authHeaders() };
	}

	async function load() {
		try {
			const r = await fetch('/api/admin/newsletter-lists', { headers: authHeaders() });
			if (!r.ok) throw new Error(`status ${r.status}`);
			const data: ListsResponse = await r.json();
			lists = data.data ?? [];
			cap = data.cap ?? 0;
			atLimit = data.at_limit ?? false;
		} catch (err) {
			console.error('newsletter-lists load failed', err);
			toasts.add('error', $t('admin.newsletter_lists.error_load'));
		} finally {
			loading = false;
		}
	}

	onMount(load);

	function suggestSlug(name: string): string {
		return name
			.toLowerCase()
			.trim()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '');
	}

	$effect(() => {
		// auto-fill slug as the user types a name, unless they've typed one themselves
		if (!newSlug || newSlug === suggestSlug(newName.slice(0, newSlug.length))) {
			newSlug = suggestSlug(newName);
		}
	});

	async function createList(event: SubmitEvent) {
		event.preventDefault();
		if (newSubmitting) return;
		newError = '';

		const body = {
			name: newName.trim(),
			slug: newSlug.trim(),
			description: newDescription.trim(),
			sender_name: newSenderName.trim(),
			reply_to: newReplyTo.trim()
		};
		if (!body.name || !body.slug) {
			newError = $t('admin.newsletter_lists.error_required');
			return;
		}

		newSubmitting = true;
		try {
			const r = await fetch('/api/admin/newsletter-lists', {
				method: 'POST',
				headers: jsonHeaders(),
				body: JSON.stringify(body)
			});
			const data = await r.json().catch(() => ({}));
			if (!r.ok) {
				newError = data.error || `status ${r.status}`;
				return;
			}
			liveMessage = $t('admin.newsletter_lists.created_live', { values: { name: data.name } });
			toasts.add('success', $t('admin.newsletter_lists.created_toast', { values: { name: data.name } }));
			newName = '';
			newSlug = '';
			newDescription = '';
			newSenderName = '';
			newReplyTo = '';
			showNewForm = false;
			await load();
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			newError = msg;
			toasts.add('error', msg);
		} finally {
			newSubmitting = false;
		}
	}

	function beginEdit(list: NewsletterList) {
		editingId = list.id;
		editName = list.name;
		editSlug = list.slug;
		editDescription = list.description;
		editSenderName = list.sender_name;
		editReplyTo = list.reply_to;
		editWelcomeSubject = list.welcome_subject;
		editWelcomeHtml = list.welcome_html;
		editIsActive = list.is_active;
		editError = '';
	}

	function cancelEdit() {
		editingId = null;
		editError = '';
	}

	async function saveEdit(event: SubmitEvent, id: string) {
		event.preventDefault();
		if (editSubmitting) return;
		editError = '';

		const body = {
			name: editName.trim(),
			slug: editSlug.trim(),
			description: editDescription.trim(),
			sender_name: editSenderName.trim(),
			reply_to: editReplyTo.trim(),
			welcome_subject: editWelcomeSubject.trim(),
			welcome_html: editWelcomeHtml,
			is_active: editIsActive
		};

		editSubmitting = true;
		try {
			const r = await fetch(`/api/admin/newsletter-lists/${id}`, {
				method: 'PATCH',
				headers: jsonHeaders(),
				body: JSON.stringify(body)
			});
			const data = await r.json().catch(() => ({}));
			if (!r.ok) {
				editError = data.error || `status ${r.status}`;
				return;
			}
			liveMessage = $t('admin.newsletter_lists.saved_live', { values: { name: data.name } });
			toasts.add('success', $t('admin.newsletter_lists.saved_toast'));
			editingId = null;
			await load();
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			editError = msg;
			toasts.add('error', msg);
		} finally {
			editSubmitting = false;
		}
	}

	async function setDefault(id: string) {
		if (actingId) return;
		actingId = id;
		try {
			const r = await fetch(`/api/admin/newsletter-lists/${id}/set-default`, {
				method: 'POST',
				headers: authHeaders()
			});
			if (!r.ok) throw new Error(`status ${r.status}`);
			liveMessage = $t('admin.newsletter_lists.default_updated_live');
			toasts.add('success', $t('admin.newsletter_lists.default_updated_toast'));
			await load();
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			toasts.add('error', msg);
		} finally {
			actingId = null;
		}
	}

	async function removeList(list: NewsletterList) {
		if (actingId) return;
		if (list.is_default) {
			toasts.add('error', $t('admin.newsletter_lists.error_delete_default'));
			return;
		}
		const confirmed = await confirm({
			title: $t('admin.newsletter_lists.delete_title'),
			message: $t('admin.newsletter_lists.delete_message', {
				values: { name: list.name, count: list.subscriber_count }
			}),
			confirmText: $t('admin.newsletter_lists.delete_confirm'),
			danger: true
		});
		if (!confirmed) return;

		actingId = list.id;
		try {
			const r = await fetch(`/api/admin/newsletter-lists/${list.id}`, {
				method: 'DELETE',
				headers: authHeaders()
			});
			if (!r.ok) {
				const data = await r.json().catch(() => ({}));
				throw new Error(data.error || `status ${r.status}`);
			}
			liveMessage = $t('admin.newsletter_lists.deleted_live', { values: { name: list.name } });
			toasts.add('success', $t('admin.newsletter_lists.deleted_toast'));
			await load();
		} catch (err) {
			const msg = err instanceof Error ? err.message : String(err);
			toasts.add('error', msg);
		} finally {
			actingId = null;
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.newsletter_lists.page_title')} | Facet</title>
</svelte:head>

<div class="mx-auto max-w-5xl space-y-6">
	<header class="flex items-center justify-between gap-4">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">
				{$t('admin.newsletter_lists.page_title')}
			</h1>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
				{$t('admin.newsletter_lists.page_description')}
			</p>
		</div>
		{#if !atLimit}
			<button
				type="button"
				onclick={() => (showNewForm = !showNewForm)}
				aria-expanded={showNewForm}
				aria-controls="new-list-form"
				class="btn btn-primary"
			>
				{showNewForm
					? $t('admin.newsletter_lists.cancel')
					: $t('admin.newsletter_lists.new_list')}
			</button>
		{:else}
			<div
				class="rounded-md bg-amber-50 px-3 py-1.5 text-sm text-amber-900 dark:bg-amber-900/30 dark:text-amber-100"
				role="status"
			>
				{$t('admin.newsletter_lists.limit_reached', { values: { cap } })}
			</div>
		{/if}
	</header>

	<!-- Single page-level live region shared across CRUD actions. -->
	<div role="status" aria-live="polite" aria-atomic="true" class="sr-only">{liveMessage}</div>

	{#if showNewForm}
		<form
			id="new-list-form"
			onsubmit={createList}
			class="card p-4"
			aria-labelledby="new-list-heading"
		>
			<h2
				id="new-list-heading"
				class="mb-3 text-base font-semibold text-gray-900 dark:text-white"
			>
				{$t('admin.newsletter_lists.new_list_heading')}
			</h2>
			<div class="grid gap-3 sm:grid-cols-2">
				<label class="block">
					<span class="block text-xs font-medium text-gray-700 dark:text-gray-300">
						{$t('admin.newsletter_lists.field_name')}
					</span>
					<input
						type="text"
						bind:value={newName}
						required
						maxlength="100"
						class="input text-sm w-full mt-1"
					/>
				</label>
				<label class="block">
					<span class="block text-xs font-medium text-gray-700 dark:text-gray-300">
						{$t('admin.newsletter_lists.field_slug')}
						<span class="font-normal text-gray-500 dark:text-gray-400">
							{$t('admin.newsletter_lists.field_slug_hint')}
						</span>
					</span>
					<input
						type="text"
						bind:value={newSlug}
						required
						pattern="[a-z0-9]+(-[a-z0-9]+)*"
						maxlength="50"
						class="input text-sm w-full mt-1 font-mono"
					/>
				</label>
				<label class="block sm:col-span-2">
					<span class="block text-xs font-medium text-gray-700 dark:text-gray-300">
						{$t('admin.newsletter_lists.field_description')}
					</span>
					<input
						type="text"
						bind:value={newDescription}
						maxlength="500"
						class="input text-sm w-full mt-1"
					/>
				</label>
				<label class="block">
					<span class="block text-xs font-medium text-gray-700 dark:text-gray-300">
						{$t('admin.newsletter_lists.field_sender_name')}
						<span class="font-normal text-gray-500 dark:text-gray-400">
							{$t('admin.newsletter_lists.field_optional')}
						</span>
					</span>
					<input
						type="text"
						bind:value={newSenderName}
						maxlength="100"
						class="input text-sm w-full mt-1"
					/>
				</label>
				<label class="block">
					<span class="block text-xs font-medium text-gray-700 dark:text-gray-300">
						{$t('admin.newsletter_lists.field_reply_to')}
						<span class="font-normal text-gray-500 dark:text-gray-400">
							{$t('admin.newsletter_lists.field_optional')}
						</span>
					</span>
					<input type="email" bind:value={newReplyTo} class="input text-sm w-full mt-1" />
				</label>
			</div>
			{#if newError}
				<div
					role="alert"
					class="mt-3 rounded-md bg-red-50 px-3 py-2 text-sm text-red-900 dark:bg-red-900/30 dark:text-red-100"
				>
					{newError}
				</div>
			{/if}
			<div class="mt-4 flex items-center gap-2">
				<button
					type="submit"
					disabled={newSubmitting}
					aria-busy={newSubmitting}
					class="btn btn-primary btn-sm"
				>
					{newSubmitting
						? $t('admin.newsletter_lists.creating')
						: $t('admin.newsletter_lists.create_list')}
				</button>
				<button
					type="button"
					onclick={() => (showNewForm = false)}
					class="btn btn-secondary btn-sm"
				>
					{$t('admin.newsletter_lists.cancel')}
				</button>
			</div>
		</form>
	{/if}

	<!-- Table with region wrapper for horizontal scroll. tabindex="0" makes
	     the scrollable region keyboard-reachable for users without pointers. -->
	<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
	<div
		role="region"
		aria-label={$t('admin.newsletter_lists.table_region_label')}
		class="overflow-x-auto card"
		tabindex="0"
	>
		<table class="w-full min-w-[720px] text-sm">
			<caption class="sr-only">{$t('admin.newsletter_lists.table_caption')}</caption>
			<thead>
				<tr class="border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
					<th scope="col" class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400">
						{$t('admin.newsletter_lists.col_name')}
					</th>
					<th scope="col" class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400">
						{$t('admin.newsletter_lists.col_slug')}
					</th>
					<th scope="col" class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400">
						{$t('admin.newsletter_lists.col_subscribers')}
					</th>
					<th scope="col" class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400">
						{$t('admin.newsletter_lists.col_status')}
					</th>
					<th
						scope="col"
						class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400"
					>
						{$t('admin.newsletter_lists.col_actions')}
					</th>
				</tr>
			</thead>
			<tbody>
				{#if loading}
					<tr>
						<td class="px-4 py-8 text-center text-gray-500 dark:text-gray-400" colspan="5">
							{$t('admin.newsletter_lists.loading')}
						</td>
					</tr>
				{:else if lists.length === 0}
					<tr>
						<td class="px-4 py-8 text-center text-gray-500 dark:text-gray-400" colspan="5">
							{$t('admin.newsletter_lists.empty')}
						</td>
					</tr>
				{:else}
					{#each lists as list (list.id)}
						<tr class="border-b border-gray-100 dark:border-gray-800">
							<td class="px-4 py-3 text-gray-800 dark:text-gray-200">
								<div class="font-medium">{list.name}</div>
								{#if list.description}
									<div class="text-xs text-gray-500 dark:text-gray-400">{list.description}</div>
								{/if}
							</td>
							<td class="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">
								{list.slug}
							</td>
							<td class="px-4 py-3 tabular-nums text-gray-800 dark:text-gray-200">
								{list.subscriber_count}
							</td>
							<td class="px-4 py-3">
								{#if list.is_default}
									<span
										class="me-2 inline-block rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-800 dark:bg-primary-900/40 dark:text-primary-200"
									>
										{$t('admin.newsletter_lists.badge_default')}
									</span>
								{/if}
								{#if list.is_active}
									<span
										class="inline-block rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-900 dark:bg-green-900/40 dark:text-green-100"
									>
										{$t('admin.newsletter_lists.badge_active')}
									</span>
								{:else}
									<span
										class="inline-block rounded-full bg-gray-200 px-2 py-0.5 text-xs font-medium text-gray-700 dark:bg-gray-700 dark:text-gray-200"
									>
										{$t('admin.newsletter_lists.badge_inactive')}
									</span>
								{/if}
							</td>
							<td class="px-4 py-3">
								<div class="flex flex-wrap gap-1">
									<button
										type="button"
										onclick={() => beginEdit(list)}
										class="btn btn-ghost btn-sm"
									>
										{$t('admin.newsletter_lists.action_edit')}
									</button>
									{#if !list.is_default}
										<button
											type="button"
											onclick={() => setDefault(list.id)}
											disabled={actingId === list.id}
											aria-busy={actingId === list.id}
											aria-label={$t('admin.newsletter_lists.set_default_aria', {
												values: { name: list.name }
											})}
											class="btn btn-ghost btn-sm"
										>
											{$t('admin.newsletter_lists.action_set_default')}
										</button>
										<button
											type="button"
											onclick={() => removeList(list)}
											disabled={actingId === list.id}
											aria-busy={actingId === list.id}
											aria-label={$t('admin.newsletter_lists.delete_aria', {
												values: { name: list.name }
											})}
											class="btn btn-danger btn-sm"
										>
											{$t('admin.newsletter_lists.action_delete')}
										</button>
									{/if}
								</div>
							</td>
						</tr>
						{#if editingId === list.id}
							<tr class="bg-gray-50 dark:bg-gray-800/40">
								<td colspan="5" class="px-4 py-4">
									<form
										onsubmit={(e) => saveEdit(e, list.id)}
										aria-labelledby={`edit-heading-${list.id}`}
									>
										<h2
											id={`edit-heading-${list.id}`}
											class="mb-3 text-sm font-semibold text-gray-900 dark:text-white"
										>
											{$t('admin.newsletter_lists.edit_heading', {
												values: { name: list.name }
											})}
										</h2>
										<div class="grid gap-3 sm:grid-cols-2">
											<label class="block">
												<span
													class="block text-xs font-medium text-gray-700 dark:text-gray-300"
												>
													{$t('admin.newsletter_lists.field_name')}
												</span>
												<input
													type="text"
													bind:value={editName}
													required
													maxlength="100"
													class="input text-sm w-full mt-1"
												/>
											</label>
											<label class="block">
												<span
													class="block text-xs font-medium text-gray-700 dark:text-gray-300"
												>
													{$t('admin.newsletter_lists.field_slug')}
												</span>
												<input
													type="text"
													bind:value={editSlug}
													required
													pattern="[a-z0-9]+(-[a-z0-9]+)*"
													maxlength="50"
													class="input text-sm w-full mt-1 font-mono"
												/>
											</label>
											<label class="block sm:col-span-2">
												<span
													class="block text-xs font-medium text-gray-700 dark:text-gray-300"
												>
													{$t('admin.newsletter_lists.field_description')}
												</span>
												<input
													type="text"
													bind:value={editDescription}
													maxlength="500"
													class="input text-sm w-full mt-1"
												/>
											</label>
											<label class="block">
												<span
													class="block text-xs font-medium text-gray-700 dark:text-gray-300"
												>
													{$t('admin.newsletter_lists.field_sender_name')}
												</span>
												<input
													type="text"
													bind:value={editSenderName}
													maxlength="100"
													class="input text-sm w-full mt-1"
												/>
											</label>
											<label class="block">
												<span
													class="block text-xs font-medium text-gray-700 dark:text-gray-300"
												>
													{$t('admin.newsletter_lists.field_reply_to')}
												</span>
												<input
													type="email"
													bind:value={editReplyTo}
													class="input text-sm w-full mt-1"
												/>
											</label>
											<label class="block sm:col-span-2">
												<span
													class="block text-xs font-medium text-gray-700 dark:text-gray-300"
												>
													{$t('admin.newsletter_lists.field_welcome_subject')}
												</span>
												<input
													type="text"
													bind:value={editWelcomeSubject}
													maxlength="200"
													class="input text-sm w-full mt-1"
												/>
											</label>
											<label class="block sm:col-span-2">
												<span
													class="block text-xs font-medium text-gray-700 dark:text-gray-300"
												>
													{$t('admin.newsletter_lists.field_welcome_html')}
												</span>
												<textarea
													rows="4"
													bind:value={editWelcomeHtml}
													class="input text-xs w-full mt-1 font-mono"
												></textarea>
											</label>
											<label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
												<input
													type="checkbox"
													bind:checked={editIsActive}
													class="h-4 w-4 rounded border-gray-300 dark:border-gray-600"
												/>
												<span>{$t('admin.newsletter_lists.field_active')}</span>
											</label>
										</div>
										{#if editError}
											<div
												role="alert"
												class="mt-3 rounded-md bg-red-50 px-3 py-2 text-sm text-red-900 dark:bg-red-900/30 dark:text-red-100"
											>
												{editError}
											</div>
										{/if}
										<div class="mt-4 flex items-center gap-2">
											<button
												type="submit"
												disabled={editSubmitting}
												aria-busy={editSubmitting}
												class="btn btn-primary btn-sm"
											>
												{editSubmitting
													? $t('admin.newsletter_lists.saving')
													: $t('admin.newsletter_lists.save_changes')}
											</button>
											<button
												type="button"
												onclick={cancelEdit}
												class="btn btn-secondary btn-sm"
											>
												{$t('admin.newsletter_lists.cancel')}
											</button>
										</div>
									</form>
								</td>
							</tr>
						{/if}
					{/each}
				{/if}
			</tbody>
		</table>
	</div>

	<p class="text-xs text-gray-500 dark:text-gray-400">
		{$t('admin.newsletter_lists.footer_unlimited', { values: { count: lists.length } })}
	</p>
</div>
