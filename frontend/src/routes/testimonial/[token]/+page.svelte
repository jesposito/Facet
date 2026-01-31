<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { t } from 'svelte-i18n';

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
	let authorPhotoFile: FileList | null = $state(null);
	let authorPhotoPreview = $state('');
	let verificationEmail = $state('');
	let showEmailVerification = $state(false);
	let sendingVerification = $state(false);
	let verificationSent = $state(false);

	// Relationship options use i18n keys
	const relationshipKeys = [
		{ value: '', key: 'public.testimonials.submit.relationship_select' },
		{ value: 'client', key: 'public.testimonials.submit.relationship_client' },
		{ value: 'colleague', key: 'public.testimonials.submit.relationship_colleague' },
		{ value: 'manager', key: 'public.testimonials.submit.relationship_manager' },
		{ value: 'report', key: 'public.testimonials.submit.relationship_report' },
		{ value: 'mentor', key: 'public.testimonials.submit.relationship_mentor' },
		{ value: 'other', key: 'public.testimonials.submit.relationship_other' }
	];

	onMount(async () => {
		const token = $page.params.token;
		try {
			const response = await fetch(`/api/testimonials/request/${token}`);
			const data = await response.json();

			if (!data.valid) {
				error = $t('public.testimonials.submit.link_invalid_error');
			} else {
				requestData = data;
				if (data.recipient_name) {
					authorName = data.recipient_name;
				}
			}
		} catch {
			error = $t('public.testimonials.submit.load_error');
		} finally {
			loading = false;
		}
	});

	function handlePhotoChange(e: Event) {
		const input = e.target as HTMLInputElement;
		if (input.files && input.files[0]) {
			const file = input.files[0];
			// Validate file type
			if (!file.type.startsWith('image/')) {
				error = 'Please select an image file';
				return;
			}
			// Validate file size (max 2MB)
			if (file.size > 2 * 1024 * 1024) {
				error = 'Image must be less than 2MB';
				return;
			}
			authorPhotoFile = input.files;
			// Create preview
			const reader = new FileReader();
			reader.onload = (e) => {
				authorPhotoPreview = e.target?.result as string;
			};
			reader.readAsDataURL(file);
		}
	}

	function clearPhoto() {
		authorPhotoFile = null;
		authorPhotoPreview = '';
	}

	async function handleSubmit() {
		if (!authorName.trim() || !content.trim()) {
			return;
		}

		submitting = true;
		error = '';
		try {
			let response: Response;

			if (authorPhotoFile && authorPhotoFile[0]) {
				// Use FormData for file upload
				const formData = new FormData();
				formData.append('request_token', $page.params.token);
				formData.append('author_name', authorName);
				formData.append('author_title', authorTitle);
				formData.append('author_company', authorCompany);
				formData.append('content', content);
				formData.append('relationship', relationship);
				formData.append('author_photo', authorPhotoFile[0]);

				response = await fetch('/api/testimonials/submit', {
					method: 'POST',
					body: formData
				});
			} else {
				// Use JSON for regular submission
				response = await fetch('/api/testimonials/submit', {
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
			}

			if (response.ok) {
				const data = await response.json();
				testimonialId = data.id;
				submitted = true;
			} else {
				error = $t('public.testimonials.submit.submit_error');
			}
		} catch {
			error = $t('public.testimonials.submit.submit_error');
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
			error = $t('public.testimonials.submit.submit_error');
		} finally {
			sendingVerification = false;
		}
	}

	// Get first name for placeholders
	function getFirstName(fullName?: string): string {
		return fullName?.split(' ')[0] || 'them';
	}
</script>

<svelte:head>
	<title>{requestData?.profile_name ? $t('public.testimonials.submit.page_title_for', { values: { name: requestData.profile_name } }) : $t('public.testimonials.submit.page_title')}</title>
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
				<h1 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">{$t('public.testimonials.submit.link_invalid_title')}</h1>
				<p class="text-gray-600 dark:text-gray-400">{error}</p>
			</div>
		{:else if submitted}
			<div class="bg-white dark:bg-gray-800 rounded-xl shadow-sm p-8 text-center">
				<div class="w-16 h-16 mx-auto mb-4 bg-green-100 dark:bg-green-900 rounded-full flex items-center justify-center">
					<svg class="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				</div>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{$t('public.testimonials.submit.thank_you_title')}</h1>
				<p class="text-gray-600 dark:text-gray-400 mb-6">
					{$t('public.testimonials.submit.submitted_pending')}
				</p>

				{#if !verificationSent}
					<div class="border-t border-gray-200 dark:border-gray-700 pt-6">
						<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
							{$t('public.testimonials.submit.verify_prompt')}
						</p>

						{#if !showEmailVerification}
							<button
								type="button"
								disabled
								class="inline-flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg opacity-50 cursor-not-allowed"
							>
								<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
								</svg>
								{$t('public.testimonials.submit.verify_email_button')}
							</button>
						{:else}
							<div class="flex flex-col sm:flex-row gap-2">
								<input
									type="email"
									bind:value={verificationEmail}
									placeholder={$t('public.testimonials.submit.email_placeholder')}
									class="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
								/>
								<button
									type="button"
									onclick={sendVerificationEmail}
									disabled={sendingVerification || !verificationEmail.trim()}
									class="w-full sm:w-auto px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 disabled:opacity-50"
								>
									{sendingVerification ? $t('public.testimonials.submit.sending_verification') : $t('public.testimonials.submit.send_verification')}
								</button>
							</div>
						{/if}
					</div>
				{:else}
					<div class="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
						<p class="text-green-800 dark:text-green-200">
							{$t('public.testimonials.submit.verification_sent')}
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
						{$t('public.testimonials.submit.requesting', { values: { name: requestData.profile_name } })}
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
							{$t('public.testimonials.submit.your_name_label')} <span class="text-red-500">*</span>
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
								{$t('public.testimonials.submit.title_label')}
							</label>
							<input
								id="authorTitle"
								type="text"
								bind:value={authorTitle}
								placeholder={$t('public.testimonials.submit.title_placeholder')}
								class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							/>
						</div>
						<div>
							<label for="authorCompany" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
								{$t('public.testimonials.submit.company_label')}
							</label>
							<input
								id="authorCompany"
								type="text"
								bind:value={authorCompany}
								placeholder={$t('public.testimonials.submit.company_placeholder')}
								class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
							/>
						</div>
					</div>

					<!-- Profile Photo Upload -->
					<div>
						<label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							{$t('public.testimonials.submit.profile_photo_label')}
						</label>
						{#if authorPhotoPreview}
							<div class="flex items-center gap-4">
								<img
									src={authorPhotoPreview}
									alt="Preview"
									class="w-16 h-16 rounded-full object-cover"
								/>
								<button
									type="button"
									onclick={clearPhoto}
									class="text-sm text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300"
								>
									Remove
								</button>
							</div>
						{:else}
							<label class="flex items-center justify-center w-full h-24 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg cursor-pointer hover:border-primary-500 dark:hover:border-primary-400 transition-colors">
								<div class="flex flex-col items-center gap-1">
									<svg class="w-8 h-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
									</svg>
									<span class="text-sm text-gray-500 dark:text-gray-400">Click to upload</span>
								</div>
								<input
									type="file"
									accept="image/jpeg,image/png,image/webp"
									class="hidden"
									onchange={handlePhotoChange}
								/>
							</label>
						{/if}
						<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
							{$t('public.testimonials.submit.profile_photo_help')}
						</p>
					</div>

					<div>
						<label for="relationship" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							{$t('public.testimonials.submit.relationship_label', { values: { name: getFirstName(requestData.profile_name) } })}
						</label>
						<select
							id="relationship"
							bind:value={relationship}
							class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
						>
							{#each relationshipKeys as rel}
								<option value={rel.value}>{$t(rel.key)}</option>
							{/each}
						</select>
					</div>

					<div>
						<label for="content" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
							{$t('public.testimonials.submit.testimonial_label')} <span class="text-red-500">*</span>
						</label>
						<textarea
							id="content"
							bind:value={content}
							required
							rows="5"
							placeholder={$t('public.testimonials.submit.testimonial_placeholder', { values: { name: getFirstName(requestData.profile_name) } })}
							class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
						></textarea>
						<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
							{$t('public.testimonials.submit.character_count', { values: { count: content.length } })}
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
						{submitting ? $t('public.testimonials.submit.submitting') : $t('public.testimonials.submit.submit_button')}
					</button>
				</form>
			</div>
		{/if}
	</div>
</div>
