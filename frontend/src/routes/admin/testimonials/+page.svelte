<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';
	import { scheduleTestimonialsRefresh } from '$lib/stores/testimonials';

	interface Testimonial {
		id: string;
		content: string;
		relationship: string;
		project: string;
		author_name: string;
		author_title: string;
		author_company: string;
		author_website: string;
		author_photo: string;
		verification_method: string;
		verification_identifier: string;
		verified_at: string;
		status: string;
		submitted_at: string;
		approved_at: string;
		featured: boolean;
		sort_order: number;
		created: string;
	}

	let testimonials: Testimonial[] = $state([]);
	let loading = $state(true);
	let statusFilter = $state('all');
	let actionLoading = $state<string | null>(null);

	const statusOptions = [
		{ value: 'all', label: 'All' },
		{ value: 'pending', label: 'Pending' },
		{ value: 'approved', label: 'Approved' },
		{ value: 'rejected', label: 'Rejected' }
	];

	onMount(loadTestimonials);

	async function loadTestimonials() {
		loading = true;
		try {
			const options: { sort: string; filter?: string } = { sort: '-id' };
			if (statusFilter !== 'all') {
				options.filter = `status = "${statusFilter}"`;
			}
			const result = await pb.collection('testimonials').getList<Testimonial>(1, 100, options);
			testimonials = result.items;
		} catch (err) {
			const errorDetails = err instanceof Error ? err.message : JSON.stringify(err);
			console.error('Failed to load testimonials:', errorDetails, err);
			toasts.error('Failed to load testimonials');
		} finally {
			loading = false;
		}
	}

	async function approveTestimonial(id: string) {
		actionLoading = id;
		try {
			const headers: Record<string, string> = {
				'Content-Type': 'application/json',
				...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
			};
			
			const response = await fetch(`/api/testimonials/${id}/approve`, {
				method: 'POST',
				headers
			});
			
			if (response.ok) {
				toasts.success('Testimonial approved');
				await loadTestimonials();
				scheduleTestimonialsRefresh();
			} else {
				toasts.error('Failed to approve');
			}
		} catch (err) {
			toasts.error('Failed to approve');
		} finally {
			actionLoading = null;
		}
	}

	async function rejectTestimonial(id: string) {
		const confirmed = await confirm({
			title: 'Reject Testimonial',
			message: 'Are you sure you want to reject this testimonial?',
			confirmText: 'Reject',
			danger: true
		});
		
		if (!confirmed) return;
		
		actionLoading = id;
		try {
			const headers: Record<string, string> = {
				'Content-Type': 'application/json',
				...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
			};
			
			const response = await fetch(`/api/testimonials/${id}/reject`, {
				method: 'POST',
				headers,
				body: JSON.stringify({})
			});
			
			if (response.ok) {
				toasts.success('Testimonial rejected');
				await loadTestimonials();
				scheduleTestimonialsRefresh();
			} else {
				toasts.error('Failed to reject');
			}
		} catch (err) {
			toasts.error('Failed to reject');
		} finally {
			actionLoading = null;
		}
	}

	async function deleteTestimonial(id: string) {
		const confirmed = await confirm({
			title: 'Delete Testimonial',
			message: 'This cannot be undone. Delete this testimonial?',
			confirmText: 'Delete',
			danger: true
		});
		
		if (!confirmed) return;
		
		actionLoading = id;
		try {
			const headers: Record<string, string> = pb.authStore.isValid
				? { Authorization: `Bearer ${pb.authStore.token}` }
				: {};
			
			const response = await fetch(`/api/testimonials/${id}`, {
				method: 'DELETE',
				headers
			});
			
			if (response.ok) {
				toasts.success('Testimonial deleted');
				await loadTestimonials();
				scheduleTestimonialsRefresh();
			} else {
				toasts.error('Failed to delete');
			}
		} catch (err) {
			toasts.error('Failed to delete');
		} finally {
			actionLoading = null;
		}
	}

	async function toggleFeatured(testimonial: Testimonial) {
		try {
			const headers: Record<string, string> = {
				'Content-Type': 'application/json',
				...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
			};
			
			const response = await fetch(`/api/testimonials/${testimonial.id}`, {
				method: 'PATCH',
				headers,
				body: JSON.stringify({ featured: !testimonial.featured })
			});
			
			if (response.ok) {
				toasts.success(testimonial.featured ? 'Removed from featured' : 'Added to featured');
				await loadTestimonials();
			}
		} catch (err) {
			toasts.error('Failed to update');
		}
	}

	function getVerificationBadge(method: string, identifier: string) {
		if (method === 'email') return `Verified via email`;
		if (method === 'github') return `Verified via GitHub · ${identifier}`;
		if (method === 'twitter') return `Verified via Twitter · ${identifier}`;
		if (method === 'linkedin') return `Verified via LinkedIn`;
		return 'Unverified';
	}

	function getStatusBadgeClass(status: string) {
		if (status === 'approved') return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
		if (status === 'pending') return 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200';
		if (status === 'rejected') return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
		return 'bg-gray-100 text-gray-800';
	}

	$effect(() => {
		loadTestimonials();
	});
</script>

<svelte:head>
	<title>Testimonials | Admin</title>
</svelte:head>

<div class="max-w-5xl mx-auto">
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
		<div>
			<h1 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">Testimonials</h1>
			<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
				Manage testimonials from clients and colleagues
			</p>
		</div>
		<a
			href="/admin/testimonials/requests"
			class="inline-flex items-center justify-center gap-2 w-full sm:w-auto px-4 py-2 rounded-lg bg-primary-600 text-white hover:bg-primary-700 transition-colors"
		>
			<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
			</svg>
			<span>Request Links</span>
		</a>
	</div>

	<div class="mb-6">
		<div class="flex flex-wrap gap-2">
			{#each statusOptions as option}
				<button
					type="button"
					onclick={() => statusFilter = option.value}
					class="px-3 sm:px-4 py-2 rounded-lg text-sm font-medium transition-colors {statusFilter === option.value
						? 'bg-primary-100 text-primary-700 dark:bg-primary-900 dark:text-primary-300'
						: 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'}"
				>
					{option.label}
					{#if option.value === 'pending'}
						{@const pendingCount = testimonials.filter(t => t.status === 'pending').length}
						{#if pendingCount > 0 && statusFilter === 'all'}
							<span class="ml-1 px-1.5 py-0.5 text-xs rounded-full bg-amber-500 text-white">{pendingCount}</span>
						{/if}
					{/if}
				</button>
			{/each}
		</div>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
		</div>
	{:else if testimonials.length === 0}
		<div class="text-center py-12 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
			<svg class="w-12 h-12 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
			</svg>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No testimonials yet</h3>
			<p class="text-gray-600 dark:text-gray-400 mb-4">
				Share your request link to start collecting testimonials.
			</p>
			<a
				href="/admin/testimonials/requests"
				class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary-600 text-white hover:bg-primary-700"
			>
				Get Request Link
			</a>
		</div>
	{:else}
		<div class="space-y-4">
			{#each testimonials as testimonial}
				<div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
				<div class="flex flex-col sm:flex-row sm:items-start gap-4">
					<div class="w-12 h-12 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center text-gray-500 dark:text-gray-400 shrink-0">
						{#if testimonial.author_photo}
							<img src={testimonial.author_photo} alt="" class="w-12 h-12 rounded-full object-cover" />
						{:else}
							<span class="text-lg font-medium">{testimonial.author_name.charAt(0).toUpperCase()}</span>
						{/if}
					</div>
					<div class="flex-1 min-w-0">
						<div class="flex flex-wrap items-center gap-1 sm:gap-2 mb-1">
							<span class="font-medium text-gray-900 dark:text-white">{testimonial.author_name}</span>
							{#if testimonial.author_title || testimonial.author_company}
								<span class="hidden sm:inline text-gray-500 dark:text-gray-400">·</span>
								<span class="text-sm text-gray-600 dark:text-gray-400 w-full sm:w-auto">
									{testimonial.author_title}{testimonial.author_title && testimonial.author_company ? ', ' : ''}{testimonial.author_company}
								</span>
							{/if}
						</div>
							<div class="flex items-center gap-2 mb-3">
								<span class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full {getStatusBadgeClass(testimonial.status)}">
									{testimonial.status}
								</span>
								{#if testimonial.verification_method && testimonial.verification_method !== 'none'}
									<span class="inline-flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
										<svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
											<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
										</svg>
										{getVerificationBadge(testimonial.verification_method, testimonial.verification_identifier)}
									</span>
								{/if}
								{#if testimonial.featured}
									<span class="inline-flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
										<svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
											<path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
										</svg>
										Featured
									</span>
								{/if}
							</div>
							<p class="text-gray-700 dark:text-gray-300 whitespace-pre-wrap">{testimonial.content}</p>
							{#if testimonial.relationship}
								<p class="text-sm text-gray-500 dark:text-gray-400 mt-2">
									Relationship: {testimonial.relationship}
								</p>
							{/if}
						</div>
					</div>
				<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
					<span class="text-sm text-gray-500 dark:text-gray-400">
						Submitted {new Date(testimonial.submitted_at || testimonial.created).toLocaleDateString()}
					</span>
					<div class="flex flex-wrap items-center gap-2">
						{#if testimonial.status === 'pending'}
							<button
								type="button"
								onclick={() => approveTestimonial(testimonial.id)}
								disabled={actionLoading === testimonial.id}
								class="flex-1 sm:flex-none px-3 py-1.5 text-sm font-medium rounded-lg bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900 dark:text-green-300 dark:hover:bg-green-800 disabled:opacity-50"
							>
								Approve
							</button>
							<button
								type="button"
								onclick={() => rejectTestimonial(testimonial.id)}
								disabled={actionLoading === testimonial.id}
								class="flex-1 sm:flex-none px-3 py-1.5 text-sm font-medium rounded-lg bg-red-100 text-red-700 hover:bg-red-200 dark:bg-red-900 dark:text-red-300 dark:hover:bg-red-800 disabled:opacity-50"
							>
								Reject
							</button>
						{:else}
							<button
								type="button"
								onclick={() => toggleFeatured(testimonial)}
								class="flex-1 sm:flex-none px-3 py-1.5 text-sm font-medium rounded-lg {testimonial.featured ? 'bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-300' : 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'} hover:opacity-80"
							>
								{testimonial.featured ? 'Unfeature' : 'Feature'}
							</button>
						{/if}
						<button
							type="button"
							onclick={() => deleteTestimonial(testimonial.id)}
							disabled={actionLoading === testimonial.id}
							class="flex-1 sm:flex-none px-3 py-1.5 text-sm font-medium rounded-lg text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20 disabled:opacity-50"
						>
							Delete
						</button>
					</div>
				</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
