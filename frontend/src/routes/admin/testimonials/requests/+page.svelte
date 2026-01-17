<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';

	interface TestimonialRequest {
		id: string;
		label: string;
		custom_message: string;
		recipient_name: string;
		recipient_email: string;
		expires_at: string;
		max_uses: number;
		use_count: number;
		is_active: boolean;
		created: string;
	}

	let requests: TestimonialRequest[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let saving = $state(false);
	let newToken = $state<string | null>(null);

	let label = $state('');
	let customMessage = $state('');
	let recipientName = $state('');
	let recipientEmail = $state('');
	let expiresAt = $state('');
	let maxUses = $state(0);

	onMount(loadRequests);

	async function loadRequests() {
		loading = true;
		try {
			const result = await pb.collection('testimonial_requests').getList<TestimonialRequest>(1, 100, {
				sort: '-id'
			});
			requests = result.items;
		} catch (err) {
			const errorDetails = err instanceof Error ? err.message : JSON.stringify(err);
			console.error('Failed to load requests:', errorDetails, err);
			toasts.error('Failed to load request links');
		} finally {
			loading = false;
		}
	}

	async function createRequest() {
		saving = true;
		try {
			const headers: Record<string, string> = {
				'Content-Type': 'application/json',
				...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
			};
			
			const response = await fetch('/api/testimonials/requests', {
				method: 'POST',
				headers,
				body: JSON.stringify({
					label,
					custom_message: customMessage,
					recipient_name: recipientName,
					recipient_email: recipientEmail,
					expires_at: expiresAt || null,
					max_uses: maxUses
				})
			});
			
			if (response.ok) {
				const data = await response.json();
				newToken = data.token;
				toasts.success('Request link created');
				resetForm();
				await loadRequests();
			} else {
				toasts.error('Failed to create request link');
			}
		} catch (err) {
			toasts.error('Failed to create request link');
		} finally {
			saving = false;
		}
	}

	async function deleteRequest(id: string) {
		const confirmed = await confirm({
			title: 'Delete Request Link',
			message: 'This will invalidate the link. Continue?',
			confirmText: 'Delete',
			danger: true
		});
		
		if (!confirmed) return;
		
		try {
			const headers: Record<string, string> = pb.authStore.isValid
				? { Authorization: `Bearer ${pb.authStore.token}` }
				: {};
			
			const response = await fetch(`/api/testimonials/requests/${id}`, {
				method: 'DELETE',
				headers
			});
			
			if (response.ok) {
				toasts.success('Request link deleted');
				await loadRequests();
			} else {
				toasts.error('Failed to delete');
			}
		} catch (err) {
			toasts.error('Failed to delete');
		}
	}

	function resetForm() {
		showForm = false;
		label = '';
		customMessage = '';
		recipientName = '';
		recipientEmail = '';
		expiresAt = '';
		maxUses = 0;
	}

	function getRequestUrl(token: string) {
		return `${window.location.origin}/testimonial/${token}`;
	}

	async function copyToClipboard(text: string) {
		try {
			await navigator.clipboard.writeText(text);
			toasts.success('Copied to clipboard');
		} catch {
			toasts.error('Failed to copy');
		}
	}
</script>

<svelte:head>
	<title>Request Links | Testimonials | Admin</title>
</svelte:head>

<div class="max-w-4xl mx-auto">
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
		<div>
			<div class="flex items-center gap-2 mb-1 text-sm sm:text-base">
				<a href="/admin/testimonials" class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
					Testimonials
				</a>
				<span class="text-gray-400">/</span>
				<span class="text-gray-900 dark:text-white">Request Links</span>
			</div>
			<p class="text-sm text-gray-600 dark:text-gray-400">
				Generate and manage links for collecting testimonials
			</p>
		</div>
		<button
			type="button"
			onclick={() => showForm = !showForm}
			class="inline-flex items-center justify-center gap-2 w-full sm:w-auto px-4 py-2 rounded-lg bg-primary-600 text-white hover:bg-primary-700 transition-colors"
		>
			<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
			</svg>
			<span>New Request Link</span>
		</button>
	</div>

	{#if newToken}
		<div class="mb-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
			<div class="flex items-start gap-3">
				<svg class="w-5 h-5 text-green-600 dark:text-green-400 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
					<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
				</svg>
				<div class="flex-1">
					<h3 class="font-medium text-green-800 dark:text-green-200">Request link created!</h3>
					<p class="text-sm text-green-700 dark:text-green-300 mt-1 mb-3">
						Copy this link and share it. The full token is only shown once.
					</p>
					<div class="flex items-center gap-2">
						<input
							type="text"
							readonly
							value={getRequestUrl(newToken)}
							class="flex-1 px-3 py-2 bg-white dark:bg-gray-800 border border-green-300 dark:border-green-700 rounded-lg text-sm"
						/>
						<button
							type="button"
							onclick={() => copyToClipboard(getRequestUrl(newToken!))}
							class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 text-sm font-medium"
						>
							Copy
						</button>
					</div>
				</div>
				<button
					type="button"
					onclick={() => newToken = null}
					class="text-green-600 dark:text-green-400 hover:text-green-800"
					aria-label="Dismiss"
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>
	{/if}

	{#if showForm}
		<div class="mb-6 p-6 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
			<h3 class="font-medium text-gray-900 dark:text-white mb-4">Create Request Link</h3>
			<form onsubmit={(e) => { e.preventDefault(); createRequest(); }} class="space-y-4">
				<div>
					<label for="label" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						Label (internal)
					</label>
					<input
						id="label"
						type="text"
						bind:value={label}
						placeholder="e.g., Project X Clients"
						class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
					/>
				</div>
				<div>
					<label for="customMessage" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
						Custom Message (shown on form)
					</label>
					<textarea
						id="customMessage"
						bind:value={customMessage}
						rows="3"
						placeholder="Thanks for working with me! I'd love to feature your feedback..."
						class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
					></textarea>
				</div>
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label for="expiresAt" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							Expires At (optional)
						</label>
						<input
							id="expiresAt"
							type="datetime-local"
							bind:value={expiresAt}
							class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						/>
					</div>
					<div>
						<label for="maxUses" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							Max Uses (0 = unlimited)
						</label>
						<input
							id="maxUses"
							type="number"
							bind:value={maxUses}
							min="0"
							class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						/>
					</div>
				</div>
				<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-2 pt-2">
					<button
						type="button"
						onclick={resetForm}
						class="w-full sm:w-auto px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={saving}
						class="w-full sm:w-auto px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50"
					>
						{saving ? 'Creating...' : 'Create Link'}
					</button>
				</div>
			</form>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
		</div>
	{:else if requests.length === 0}
		<div class="text-center py-12 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No request links yet</h3>
			<p class="text-gray-600 dark:text-gray-400 mb-4">
				Create a link to start collecting testimonials.
			</p>
			<button
				type="button"
				onclick={() => showForm = true}
				class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary-600 text-white hover:bg-primary-700"
			>
				Create First Link
			</button>
		</div>
	{:else}
		<div class="space-y-3">
			{#each requests as request}
				<div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
					<div class="flex items-start justify-between gap-4">
						<div class="flex-1 min-w-0">
							<div class="flex flex-wrap items-center gap-2 mb-1">
								<span class="font-medium text-gray-900 dark:text-white">
									{request.label || 'Untitled Request'}
								</span>
								{#if !request.is_active}
									<span class="px-2 py-0.5 text-xs font-medium rounded-full bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400">
										Inactive
									</span>
								{/if}
							</div>
							<div class="flex flex-wrap items-center gap-2 sm:gap-4 text-sm text-gray-500 dark:text-gray-400">
								<span>{request.use_count} submissions</span>
								{#if request.max_uses > 0}
									<span>Max: {request.max_uses}</span>
								{/if}
								{#if request.expires_at}
									<span>Expires: {new Date(request.expires_at).toLocaleDateString()}</span>
								{/if}
							</div>
							{#if request.custom_message}
								<p class="text-sm text-gray-600 dark:text-gray-400 mt-2 line-clamp-2">
									{request.custom_message}
								</p>
							{/if}
						</div>
						<button
							type="button"
							onclick={() => deleteRequest(request.id)}
							class="shrink-0 p-2 text-red-600 hover:text-red-700 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20 rounded-lg"
							aria-label="Delete request"
						>
							<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
							</svg>
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
