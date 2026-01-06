<script lang="ts">
	import { onMount } from 'svelte';
	import { pb, type View } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts } from '$lib/stores';
	import { icon } from '$lib/icons';
	import AIContentHelper from '$components/admin/AIContentHelper.svelte';

	let profile: Record<string, unknown> | null = null;
	let loading = true;
	let saving = false;

	// Form fields
	let name = '';
	let headline = '';
	let location = '';
	let summary = '';
	let contactEmail = '';
	let contactLinks: Array<{ type: string; url: string; label: string }> = [];
	let visibility = 'public';
	
	// Image fields
	let avatarUrl: string | null = null;
	let heroImageUrl: string | null = null;
	let avatarFile: File | null = null;
	let heroImageFile: File | null = null;

	// Views that override headline/summary
	let viewsOverridingHeadline: View[] = [];
	let viewsOverridingSummary: View[] = [];

	onMount(async () => {
		console.log('[PROFILE] onMount() called');
		try {
			console.log('[PROFILE] About to fetch profile data...');
			const records = await collection('profile').getList(1, 1);
			console.log('[PROFILE] Fetched records:', records);
			if (records.items.length > 0) {
				profile = records.items[0];
				console.log('[PROFILE] Loaded profile:', profile);
				name = (profile.name as string) || '';
				headline = (profile.headline as string) || '';
				location = (profile.location as string) || '';
				summary = (profile.summary as string) || '';
				contactEmail = (profile.contact_email as string) || '';
				contactLinks = (profile.contact_links as typeof contactLinks) || [];
				visibility = (profile.visibility as string) || 'public';
				
				if (profile.avatar) {
					avatarUrl = `/api/files/${profile.collectionId}/${profile.id}/${profile.avatar}`;
				}
				if (profile.hero_image) {
					heroImageUrl = `/api/files/${profile.collectionId}/${profile.id}/${profile.hero_image}`;
				}
			}

			// Check for views with overrides
			const views = await collection('views').getList(1, 100);
			viewsOverridingHeadline = (views.items as unknown as View[]).filter(v => v.hero_headline);
			viewsOverridingSummary = (views.items as unknown as View[]).filter(v => v.hero_summary);
		} catch (err) {
			console.error('[PROFILE] Failed to load profile:', err);
		} finally {
			loading = false;
		}
	});

	async function handleSubmit() {
		saving = true;
		try {
			const formData = new FormData();
			formData.append('name', name);
			formData.append('headline', headline);
			formData.append('location', location);
			formData.append('summary', summary);
			formData.append('contact_email', contactEmail);
			formData.append('contact_links', JSON.stringify(contactLinks));
			formData.append('visibility', visibility);
			
			if (avatarFile) {
				formData.append('avatar', avatarFile);
			}
			if (heroImageFile) {
				formData.append('hero_image', heroImageFile);
			}

			if (profile) {
				await collection('profile').update(profile.id as string, formData);
			} else {
				await collection('profile').create(formData);
			}

			toasts.add('success', 'Profile saved successfully');
			
			avatarFile = null;
			heroImageFile = null;
			
			const records = await collection('profile').getList(1, 1);
			if (records.items.length > 0) {
				profile = records.items[0];
				if (profile.avatar) {
					avatarUrl = `/api/files/${profile.collectionId}/${profile.id}/${profile.avatar}?${Date.now()}`;
				}
				if (profile.hero_image) {
					heroImageUrl = `/api/files/${profile.collectionId}/${profile.id}/${profile.hero_image}?${Date.now()}`;
				}
			}
		} catch (err) {
			console.error('Failed to save profile:', err);
			toasts.add('error', 'Failed to save profile');
		} finally {
			saving = false;
		}
	}

	function addContactLink() {
		contactLinks = [...contactLinks, { type: 'website', url: '', label: '' }];
	}

	function removeContactLink(index: number) {
		contactLinks = contactLinks.filter((_, i) => i !== index);
	}
	
	function handleAvatarChange(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files?.[0]) {
			avatarFile = input.files[0];
			avatarUrl = URL.createObjectURL(avatarFile);
		}
	}
	
	function handleHeroImageChange(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files?.[0]) {
			heroImageFile = input.files[0];
			heroImageUrl = URL.createObjectURL(heroImageFile);
		}
	}
	
	async function removeAvatar() {
		if (!profile) return;
		try {
			await collection('profile').update(profile.id as string, { avatar: null });
			avatarUrl = null;
			avatarFile = null;
			toasts.add('success', 'Avatar removed');
		} catch (err) {
			console.error('Failed to remove avatar:', err);
			toasts.add('error', 'Failed to remove avatar');
		}
	}
	
	async function removeHeroImage() {
		if (!profile) return;
		try {
			await collection('profile').update(profile.id as string, { hero_image: null });
			heroImageUrl = null;
			heroImageFile = null;
			toasts.add('success', 'Hero image removed');
		} catch (err) {
			console.error('Failed to remove hero image:', err);
			toasts.add('error', 'Failed to remove hero image');
		}
	}
</script>

<svelte:head>
	<title>Edit Profile | Facet</title>
</svelte:head>

<div class="max-w-3xl mx-auto">
	<h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">Edit Profile</h1>

	{#if loading}
		<div class="card p-8 text-center">
			<div class="animate-pulse">Loading profile...</div>
		</div>
	{:else}
		<form on:submit|preventDefault={handleSubmit} class="space-y-6">
			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Images</h2>
				
				<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
					<div>
						<label class="label">Avatar</label>
						<div class="flex items-start gap-4">
							<div class="relative">
								{#if avatarUrl}
									<img 
										src={avatarUrl} 
										alt="Avatar" 
										class="w-24 h-24 rounded-full object-cover border-2 border-gray-200 dark:border-gray-700"
									/>
								<button
									type="button"
									on:click={removeAvatar}
									class="absolute -top-2 -right-2 p-1 bg-red-500 text-white rounded-full hover:bg-red-600"
									title="Remove avatar"
								>
									{@html icon('x')}
								</button>
							{:else}
								<div class="w-24 h-24 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center border-2 border-dashed border-gray-300 dark:border-gray-600 text-gray-400">
									{@html icon('image')}
								</div>
								{/if}
							</div>
							<div class="flex-1">
								<input
									type="file"
									id="avatar"
									accept="image/jpeg,image/png,image/webp,image/svg+xml"
									on:change={handleAvatarChange}
									class="hidden"
								/>
								<label for="avatar" class="btn btn-secondary btn-sm cursor-pointer">
									{avatarUrl ? 'Change' : 'Upload'} Avatar
								</label>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
									JPG, PNG, WebP or SVG. Max 5MB.
								</p>
							</div>
						</div>
					</div>
					
					<div>
						<label class="label">Hero Image</label>
						<div class="space-y-3">
							{#if heroImageUrl}
								<div class="relative">
									<img 
										src={heroImageUrl} 
										alt="Hero" 
										class="w-full h-32 object-cover rounded-lg border border-gray-200 dark:border-gray-700"
									/>
								<button
									type="button"
									on:click={removeHeroImage}
									class="absolute top-2 right-2 p-1 bg-red-500 text-white rounded-full hover:bg-red-600"
									title="Remove hero image"
								>
									{@html icon('x')}
								</button>
							</div>
						{:else}
							<div class="w-full h-32 bg-gray-100 dark:bg-gray-800 flex items-center justify-center rounded-lg border-2 border-dashed border-gray-300 dark:border-gray-600 text-gray-400">
								{@html icon('image')}
							</div>
							{/if}
							<div>
								<input
									type="file"
									id="hero_image"
									accept="image/jpeg,image/png,image/webp,image/gif"
									on:change={handleHeroImageChange}
									class="hidden"
								/>
								<label for="hero_image" class="btn btn-secondary btn-sm cursor-pointer">
									{heroImageUrl ? 'Change' : 'Upload'} Hero Image
								</label>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
									JPG, PNG, WebP or GIF. Max 10MB.
								</p>
							</div>
						</div>
					</div>
				</div>
			</div>
			
			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Basic Information</h2>

				<div>
					<label for="name" class="label">Name *</label>
					<input type="text" id="name" bind:value={name} class="input" required />
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="headline" class="label mb-0">Headline</label>
						<AIContentHelper
							fieldType="headline"
							content={headline}
							context={{ name, location }}
							on:apply={(e) => (headline = e.detail.content)}
						/>
					</div>
					<input
						type="text"
						id="headline"
						bind:value={headline}
						class="input mt-1"
						placeholder="e.g., Senior Software Engineer at Company"
					/>
					{#if viewsOverridingHeadline.length > 0}
						<div class="mt-2 p-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md">
							<p class="text-sm text-amber-800 dark:text-amber-200">
								<strong>Note:</strong> {viewsOverridingHeadline.length === 1 ? 'This view has' : 'These views have'} a custom headline that overrides this value:
								{#each viewsOverridingHeadline as view, i}
									<a href="/admin/views/{view.id}" class="underline hover:no-underline">{view.name}</a>{i < viewsOverridingHeadline.length - 1 ? ', ' : ''}
								{/each}
							</p>
						</div>
					{/if}
				</div>

				<div>
					<label for="location" class="label">Location</label>
					<input
						type="text"
						id="location"
						bind:value={location}
						class="input"
						placeholder="e.g., San Francisco, CA"
					/>
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<label for="summary" class="label mb-0">Summary</label>
						<AIContentHelper
							fieldType="summary"
							content={summary}
							context={{ name, headline, location }}
							on:apply={(e) => (summary = e.detail.content)}
						/>
					</div>
					<textarea
						id="summary"
						bind:value={summary}
						class="input min-h-[150px] mt-1"
						placeholder="Tell your story... (Markdown supported)"
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Markdown formatting is supported</p>
					{#if viewsOverridingSummary.length > 0}
						<div class="mt-2 p-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-md">
							<p class="text-sm text-amber-800 dark:text-amber-200">
								<strong>Note:</strong> {viewsOverridingSummary.length === 1 ? 'This view has' : 'These views have'} a custom summary that overrides this value:
								{#each viewsOverridingSummary as view, i}
									<a href="/admin/views/{view.id}" class="underline hover:no-underline">{view.name}</a>{i < viewsOverridingSummary.length - 1 ? ', ' : ''}
								{/each}
							</p>
						</div>
					{/if}
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Contact Information</h2>

				<div>
					<label for="email" class="label">Contact Email</label>
					<input type="email" id="email" bind:value={contactEmail} class="input" />
				</div>

				<div>
					<div class="flex items-center justify-between mb-2">
						<span class="label mb-0">Contact Links</span>
						<button type="button" class="btn btn-sm btn-secondary" on:click={addContactLink}>
							+ Add Link
						</button>
					</div>

					{#if contactLinks.length === 0}
						<p class="text-gray-500 dark:text-gray-400 text-sm">Add links to help people reach you.</p>
					{:else}
						<div class="space-y-3">
							{#each contactLinks as link, i}
								<div class="flex items-start gap-2">
									<select bind:value={link.type} class="input w-32">
										<option value="github">GitHub</option>
										<option value="linkedin">LinkedIn</option>
										<option value="twitter">Twitter</option>
										<option value="email">Email</option>
										<option value="website">Website</option>
										<option value="other">Other</option>
									</select>
									<input
										type="url"
										bind:value={link.url}
										class="input flex-1"
										placeholder="https://..."
									/>
									<input
										type="text"
										bind:value={link.label}
										class="input w-32"
										placeholder="Label"
									/>
									<button
										type="button"
										class="btn btn-ghost text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"
										on:click={() => removeContactLink(i)}
										title="Remove link"
									>
										{@html icon('x')}
									</button>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>

			<div class="card p-6 space-y-4">
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Visibility</h2>

				<div>
					<label for="visibility" class="label">Profile Visibility</label>
					<select id="visibility" bind:value={visibility} class="input">
						<option value="public">Public - Anyone can view</option>
						<option value="unlisted">Unlisted - Only accessible via direct link or views</option>
						<option value="private">Private - Only you can view</option>
					</select>
				</div>
			</div>

			<div class="flex justify-end gap-3">
				<a href="/admin" class="btn btn-secondary">Cancel</a>
				<button type="submit" class="btn btn-primary" disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					Save Profile
				</button>
			</div>
		</form>
	{/if}
</div>
