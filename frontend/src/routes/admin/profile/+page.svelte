<script lang="ts">
	import { onMount } from 'svelte';
	import { pb } from '$lib/pocketbase';
	import { toasts } from '$lib/stores';

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

	onMount(async () => {
		try {
			const records = await pb.collection('profile').getList(1, 1);
			if (records.items.length > 0) {
				profile = records.items[0];
				name = (profile.name as string) || '';
				headline = (profile.headline as string) || '';
				location = (profile.location as string) || '';
				summary = (profile.summary as string) || '';
				contactEmail = (profile.contact_email as string) || '';
				contactLinks = (profile.contact_links as typeof contactLinks) || [];
				visibility = (profile.visibility as string) || 'public';
			}
		} catch (err) {
			console.error('Failed to load profile:', err);
		} finally {
			loading = false;
		}
	});

	async function handleSubmit() {
		saving = true;
		try {
			const data = {
				name,
				headline,
				location,
				summary,
				contact_email: contactEmail,
				contact_links: contactLinks,
				visibility
			};

			if (profile) {
				await pb.collection('profile').update(profile.id as string, data);
			} else {
				await pb.collection('profile').create(data);
			}

			toasts.add('success', 'Profile saved successfully');
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
</script>

<svelte:head>
	<title>Edit Profile | Me.yaml</title>
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
				<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Basic Information</h2>

				<div>
					<label for="name" class="label">Name *</label>
					<input type="text" id="name" bind:value={name} class="input" required />
				</div>

				<div>
					<label for="headline" class="label">Headline</label>
					<input
						type="text"
						id="headline"
						bind:value={headline}
						class="input"
						placeholder="e.g., Senior Software Engineer at Company"
					/>
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
					<label for="summary" class="label">Summary</label>
					<textarea
						id="summary"
						bind:value={summary}
						class="input min-h-[150px]"
						placeholder="Tell your story... (Markdown supported)"
					></textarea>
					<p class="text-xs text-gray-500 mt-1">Markdown formatting is supported</p>
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
						<label class="label mb-0">Contact Links</label>
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
										class="btn btn-ghost text-red-500"
										on:click={() => removeContactLink(i)}
									>
										×
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
