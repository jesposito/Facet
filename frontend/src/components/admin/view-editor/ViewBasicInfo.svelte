<script lang="ts">
	/**
	 * ViewBasicInfo - Basic information form section
	 * 
	 * Handles name, slug, description, visibility, and password fields.
	 */

	// Props
	let {
		name = $bindable(),
		slug = $bindable(), 
		description = $bindable(),
		visibility = $bindable(),
		password = $bindable(),
		view
	}: {
		name: string;
		slug: string;
		description: string;
		visibility: 'public' | 'unlisted' | 'private' | 'password';
		password: string;
		view: any;
	} = $props();

	/**
	 * Generate a URL slug from a name
	 */
	function generateSlug(text: string): string {
		return text
			.toLowerCase()
			.replace(/[^a-z0-9\s-]/g, '')
			.trim()
			.replace(/\s+/g, '-')
			.replace(/-+/g, '-')
			.replace(/^-|-$/g, '');
	}

	/**
	 * Auto-generate slug when name changes (if no existing slug)
	 */
	function handleNameChange() {
		if (!view?.slug) {
			slug = generateSlug(name);
		}
	}
</script>

<div class="card p-4 sm:p-6 space-y-4">
	<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Basic Information</h2>

	<div>
		<label for="name" class="label">Name *</label>
		<input
			type="text"
			id="name"
			bind:value={name}
			oninput={handleNameChange}
			class="input"
			placeholder="Recruiter View"
			required
		/>
		<p class="text-xs text-gray-500 mt-1">Internal name for this view</p>
	</div>

	<div>
		<label for="slug" class="label">URL Slug *</label>
		<div class="flex items-center gap-2">
			<span class="text-gray-500 text-sm">/</span>
			<input
				type="text"
				id="slug"
				bind:value={slug}
				class="input flex-1"
				placeholder="recruiter"
				required
			/>
		</div>
		<p class="text-xs text-gray-500 mt-1">Public URL will be: /{slug}</p>
	</div>

	<div>
		<label for="description" class="label">Description</label>
		<textarea
			id="description"
			bind:value={description}
			class="input min-h-[80px]"
			placeholder="Internal notes about this view..."
		></textarea>
		<p class="text-xs text-gray-500 mt-1">Private notes (not shown publicly)</p>
	</div>

	<div>
		<label for="visibility" class="label">Visibility *</label>
		<select id="visibility" bind:value={visibility} class="input">
			<option value="public">Public - Anyone can access</option>
			<option value="unlisted">Unlisted - Only with share token</option>
			<option value="password">Password - Requires password</option>
			<option value="private">Private - Admin only</option>
		</select>
		<p class="text-xs text-gray-500 mt-1">Controls who can access this view</p>
	</div>

	{#if visibility === 'password'}
		<div>
			<label for="password" class="label">
				{password ? 'Change Password' : 'Set Password *'}
			</label>
			<input
				type="password"
				id="password"
				bind:value={password}
				class="input"
				placeholder={password ? 'Enter new password to change' : 'Enter password for this view'}
				autocomplete="new-password"
			/>
			<p class="text-xs text-gray-500 mt-1">
				{password ? 'Leave blank to keep current password' : 'Visitors will need this password to access this view'}
			</p>
		</div>
	{/if}
</div>