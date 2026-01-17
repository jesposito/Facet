<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';

	interface RequestData {
		valid: boolean;
		request_id: string;
		label: string;
		custom_message: string;
		recipient_name: string;
		profile_name: string;
		profile_headline: string;
		profile_avatar: string;
		error?: string;
	}

	let requestData: RequestData | null = $state(null);
	let loading = $state(true);
	let error = $state('');
	let submitting = $state(false);
	let submitted = $state(false);
	let testimonialId = $state<string | null>(null);

	let authorName = $state('');
	let authorTitle = $state('');
	let authorCompany = $state('');
	let content = $state('');
	let relationship = $state('');
	let verificationEmail = $state('');
	let showEmailVerification = $state(false);
	let sendingVerification = $state(false);
	let verificationSent = $state(false);

	const relationships = [
		{ value: '', label: 'Select relationship...' },
		{ value: 'client', label: 'Client' },
		{ value: 'colleague', label: 'Colleague' },
		{ value: 'manager', label: 'Manager' },
		{ value: 'report', label: 'Direct Report' },
		{ value: 'mentor', label: 'Mentor/Mentee' },
		{ value: 'other', label: 'Other' }
	];

	onMount(async () => {
		const token = $page.params.token;
		try {
			const response = await fetch(`/api/testimonials/request/${token}`);
			const data = await response.json();
			
			if (!data.valid) {
				error = 'This link is invalid or has expired.';
			} else {
				requestData = data;
				if (data.recipient_name) {
					authorName = data.recipient_name;
				}
			}
		} catch {
			error = 'Failed to load request. Please try again.';
		} finally {
			loading = false;
		}
	});

	async function handleSubmit() {
		if (!authorName.trim() || !content.trim()) {
			return;
		}

		submitting = true;
		try {
			const response = await fetch('/api/testimonials/submit', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					request_token: $page.params.token,
					author_name: authorName,
					author_title: authorTitle,
					author_company: authorCompany,
					content,
					relationship
				})
			});

			if (response.ok) {
				const data = await response.json();
				testimonialId = data.id;
				submitted = true;
			} else {
				error = 'Failed to submit. Please try again.';
			}
		} catch {
			error = 'Failed to submit. Please try again.';
		} finally {
			submitting = false;
		}
	}

	async function sendVerificationEmail() {
		if (!verificationEmail.trim() || !testimonialId) return;

		sendingVerification = true;
		try {
			const response = await fetch('/api/testimonials/verify/email', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					testimonial_id: testimonialId,
					email: verificationEmail
				})
			});

			if (response.ok) {
				verificationSent = true;
			}
		} catch {
			error = 'Failed to send verification email.';
		} finally {
			sendingVerification = false;
		}
	}
</script>

<svelte:head>
	<title>Leave a Testimonial{requestData?.profile_name ? ` for ${requestData.profile_name}` : ''}</title>
</svelte:head>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900 py-12 px-4">
	<div class="max-w-lg mx-auto">
		{#if loading}
			<div class="flex items-center justify-center py-24">
				<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
			</div>
		{:else if error && !requestData}
			<div class="text-center py-24">
				<svg class="w-16 h-16 mx-auto text-gray-400 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
				</svg>
				<h1 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">Link Invalid</h1>
				<p class="text-gray-600 dark:text-gray-400">{error}</p>
			</div>
		{:else if submitted}
			<div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm p-8 text-center">
				<div class="w-16 h-16 mx-auto mb-4 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center">
					<svg class="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</div>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Thank You!</h1>
				<p class="text-gray-600 dark:text-gray-400 mb-6">
					Your testimonial has been submitted and is pending review.
				</p>

				{#if !verificationSent}
					<div class="border-t border-gray-200 dark:border-gray-700 pt-6">
						<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
							Want to verify your testimonial? This adds credibility and links to your profile.
						</p>
						
						{#if !showEmailVerification}
							<button
								type="button"
								onclick={() => showEmailVerification = true}
								class="inline-flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								</svg>
								Verify with Email
							</button>
						{:else}
							<div class="flex flex-col sm:flex-row gap-2">
								<input
									type="email"
									bind:value={verificationEmail}
									placeholder="your@email.com"
									class="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
								/>
								<button
									type="button"
									onclick={sendVerificationEmail}
									disabled={sendingVerification || !verificationEmail.trim()}
									class="w-full sm:w-auto px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50"
								>
									{sendingVerification ? 'Sending...' : 'Send'}
								</button>
							</div>
						{/if}
					</div>
				{:else}
					<div class="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
						<p class="text-green-800 dark:text-green-200">
							Verification email sent! Check your inbox and click the link to verify.
						</p>
					</div>
				{/if}
			</div>
		{:else if requestData}
			<div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm overflow-hidden">
				<div class="p-6 border-b border-gray-200 dark:border-gray-700 text-center">
					{#if requestData.profile_avatar}
						<img
							src={requestData.profile_avatar}
							alt=""
							class="w-20 h-20 rounded-full mx-auto mb-4 object-cover"
						/>
					{:else}
						<div class="w-20 h-20 rounded-full mx-auto mb-4 bg-primary-100 dark:bg-primary-900 flex items-center justify-center">
							<span class="text-2xl font-bold text-primary-600 dark:text-primary-400">
								{requestData.profile_name?.charAt(0) || '?'}
							</span>
						</div>
					{/if}
					<h1 class="text-xl font-semibold text-gray-900 dark:text-white">
						{requestData.profile_name} is requesting a testimonial
					</h1>
					{#if requestData.profile_headline}
						<p class="text-gray-600 dark:text-gray-400 mt-1">{requestData.profile_headline}</p>
					{/if}
				</div>

				{#if requestData.custom_message}
					<div class="px-6 py-4 bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
						<p class="text-gray-700 dark:text-gray-300 italic">"{requestData.custom_message}"</p>
					</div>
				{/if}

				<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="p-6 space-y-5">
					<div>
						<label for="authorName" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							Your Name <span class="text-red-500">*</span>
						</label>
						<input
							id="authorName"
							type="text"
							bind:value={authorName}
							required
							class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
						/>
					</div>

					<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
						<div>
							<label for="authorTitle" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
								Title
							</label>
							<input
								id="authorTitle"
								type="text"
								bind:value={authorTitle}
								placeholder="e.g., CEO"
								class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							/>
						</div>
						<div>
							<label for="authorCompany" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
								Company
							</label>
							<input
								id="authorCompany"
								type="text"
								bind:value={authorCompany}
								placeholder="e.g., Acme Inc"
								class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							/>
						</div>
					</div>

					<div>
						<label for="relationship" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							How do you know {requestData.profile_name?.split(' ')[0] || 'them'}?
						</label>
						<select
							id="relationship"
							bind:value={relationship}
							class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						>
							{#each relationships as rel}
								<option value={rel.value}>{rel.label}</option>
							{/each}
						</select>
					</div>

					<div>
						<label for="content" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							Your Testimonial <span class="text-red-500">*</span>
						</label>
						<textarea
							id="content"
							bind:value={content}
							required
							rows="5"
							placeholder="Share your experience working with {requestData.profile_name?.split(' ')[0] || 'them'}..."
							class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
						></textarea>
						<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
							{content.length} characters
						</p>
					</div>

					{#if error}
						<div class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-300 text-sm">
							{error}
						</div>
					{/if}

					<button
						type="submit"
						disabled={submitting || !authorName.trim() || !content.trim()}
						class="w-full py-3 px-4 bg-primary-600 text-white rounded-lg font-medium hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
					>
						{submitting ? 'Submitting...' : 'Submit Testimonial'}
					</button>
				</form>
			</div>
		{/if}
	</div>
</div>
