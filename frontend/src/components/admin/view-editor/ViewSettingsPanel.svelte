<script lang="ts">
	/**
	 * ViewSettingsPanel - Form sections for view configuration
	 * 
	 * Includes basic info, hero overrides, call-to-action, and settings.
	 * This is a "smart" component that manages form state but uses props
	 * for the initial Phase 2 extraction to avoid prop drilling.
	 */
	import { ACCENT_COLORS, ACCENT_COLOR_LIST, type AccentColor } from '$lib/colors';
	import { icon } from '$lib/icons';
	import type { Profile } from '$lib/pocketbase';

	// Props - comprehensive form state for Phase 2
	let {
		// Basic Information
		name = $bindable(),
		slug = $bindable(),
		description = $bindable(),
		visibility = $bindable(),
		password = $bindable(),
		
		// Hero Overrides
		heroHeadline = $bindable(),
		heroSummary = $bindable(),
		heroLocation = $bindable(),
		heroImageUrl = $bindable(),
		
		// Call to Action
		ctaText = $bindable(),
		ctaUrl = $bindable(),
		ctaEnabled = $bindable(),

		// Settings
		accentColor = $bindable(),
		isActive = $bindable(),
		
		// Additional props
		profile,
		onHeroImageChange,
		onRemoveHeroImage
	}: {
		// Basic Information
		name: string;
		slug: string;
		description: string;
		visibility: 'public' | 'unlisted' | 'private' | 'password';
		password: string;
		
		// Hero Overrides
		heroHeadline: string;
		heroSummary: string;
		heroLocation: string;
		heroImageUrl: string | null;
		
		// Call to Action
		ctaText: string;
		ctaUrl: string;
		ctaEnabled: boolean;

		// Settings
		accentColor: AccentColor | null;
		isActive: boolean;
		
		// Additional props
		profile: Profile | null;
		onHeroImageChange: (event: Event) => void;
		onRemoveHeroImage: () => void;
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
		if (!slug) {
			slug = generateSlug(name);
		}
	}
</script>

<!-- Basic Information -->
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

<!-- Hero Overrides -->
<div class="card p-4 sm:p-6 space-y-4">
	<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Hero Overrides</h2>
	<p class="text-sm text-gray-500 -mt-2">Override your profile's hero image, headline and summary for this view</p>

	<div>
		<span class="label">Hero Image</span>
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
						onclick={onRemoveHeroImage}
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
					id="view_hero_image"
					accept="image/jpeg,image/png,image/webp"
					onchange={onHeroImageChange}
					class="hidden"
				/>
				<label for="view_hero_image" class="btn btn-secondary btn-sm cursor-pointer">
					{heroImageUrl ? 'Change' : 'Upload'} Hero Image
				</label>
				<p class="text-xs text-gray-500 mt-1">Leave empty to use profile's hero image. JPG, PNG or WebP. Max 10MB.</p>
			</div>
		</div>
	</div>

	<div>
		<div class="flex items-center justify-between">
			<label for="hero_headline" class="label">Custom Headline</label>
			{#if heroHeadline}
				<button
					type="button"
					class="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400"
					onclick={() => heroHeadline = ''}
				>
					Use profile value
				</button>
			{/if}
		</div>
		<input
			type="text"
			id="hero_headline"
			bind:value={heroHeadline}
			class="input"
			placeholder="Leave empty to use profile headline"
		/>
		{#if profile?.headline}
			<p class="text-xs text-gray-500 mt-1">
				Profile value: <span class="text-gray-700 dark:text-gray-300">{profile.headline}</span>
			</p>
		{/if}
	</div>

	<div>
		<div class="flex items-center justify-between">
			<label for="hero_summary" class="label">Custom Summary</label>
			{#if heroSummary}
				<button
					type="button"
					class="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400"
					onclick={() => heroSummary = ''}
				>
					Use profile value
				</button>
			{/if}
		</div>
		<textarea
			id="hero_summary"
			bind:value={heroSummary}
			class="input min-h-[120px]"
			placeholder="Leave empty to use profile summary (Markdown supported)"
		></textarea>
	{#if profile?.summary}
		<p class="text-xs text-gray-500 mt-1">
			Profile value: <span class="text-gray-700 dark:text-gray-300">{profile.summary.length > 100 ? profile.summary.substring(0, 100) + '...' : profile.summary}</span>
		</p>
	{/if}
	</div>

	<div>
		<div class="flex items-center justify-between">
			<label for="hero_location" class="label">Custom Location</label>
			{#if heroLocation}
				<button
					type="button"
					class="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400"
					onclick={() => heroLocation = ''}
				>
					Use profile value
				</button>
			{/if}
		</div>
		<input
			type="text"
			id="hero_location"
			bind:value={heroLocation}
			class="input"
			placeholder="e.g., Wellington, NZ | US Citizen | W-2 or EOR-ready"
		/>
		{#if profile?.location}
			<p class="text-xs text-gray-500 mt-1">
				Profile value: <span class="text-gray-700 dark:text-gray-300">{profile.location}</span>
			</p>
		{/if}
	</div>
</div>

<!-- Call to Action -->
<div class="card p-4 sm:p-6 space-y-4">
	<div class="flex items-start justify-between gap-4">
		<div>
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Call to Action</h2>
			<p class="text-sm text-gray-500">Add a prominent button to this view</p>
		</div>
		<label class="relative inline-flex items-center cursor-pointer">
			<input
				type="checkbox"
				class="sr-only peer"
				bind:checked={ctaEnabled}
			/>
			<div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"></div>
		</label>
	</div>

	{#if ctaEnabled}
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			<div>
				<label for="cta_text" class="label">Button Text</label>
				<input
					type="text"
					id="cta_text"
					bind:value={ctaText}
					class="input"
					placeholder="Download Resume"
				/>
			</div>
			<div>
				<label for="cta_url" class="label">Button URL</label>
				<input
					type="url"
					id="cta_url"
					bind:value={ctaUrl}
					class="input"
					placeholder="https://..."
				/>
			</div>
		</div>
	{:else}
		<p class="text-sm text-gray-500 dark:text-gray-400 italic">
			CTA button is disabled for this view. Enable the toggle above to configure it.
		</p>
	{/if}
</div>

<!-- Settings -->
<div class="card p-4 sm:p-6 space-y-4">
	<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Settings</h2>

	<!-- Accent Color Override -->
	<div class="pt-2">
		<span class="label mb-3 block">Accent Color</span>
		<div class="flex flex-wrap items-center gap-3" role="group" aria-label="Select accent color">
			<!-- Use Global Option -->
			<button
				type="button"
				class="flex items-center gap-2 px-3 py-2 rounded-lg border transition-all
					{accentColor === null
					? 'border-gray-900 dark:border-white bg-gray-100 dark:bg-gray-800'
					: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
				onclick={() => accentColor = null}
			>
				<div class="w-5 h-5 rounded-full bg-gradient-to-r from-primary-400 to-primary-600 border-2 border-white shadow-sm"></div>
				<span class="text-sm font-medium text-gray-700 dark:text-gray-300">Use global</span>
				{#if accentColor === null}
					<svg class="w-4 h-4 text-gray-900 dark:text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
						<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
					</svg>
				{/if}
			</button>

			<!-- Color Swatches -->
			{#each ACCENT_COLOR_LIST as color}
				{@const colorInfo = ACCENT_COLORS[color]}
				<button
					type="button"
					class="relative group"
					onclick={() => accentColor = color}
					title="{colorInfo.label} - {colorInfo.description}"
				>
					<div
						class="w-10 h-10 rounded-lg transition-all duration-200 ring-offset-2 ring-offset-white dark:ring-offset-gray-900
							{accentColor === color
							? 'ring-2 ring-gray-900 dark:ring-white scale-110'
							: 'hover:scale-105'}"
						style="background-color: {colorInfo.scale[500]}"
					>
						{#if accentColor === color}
							<div class="absolute inset-0 flex items-center justify-center">
								<svg class="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="3">
									<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
								</svg>
							</div>
						{/if}
					</div>
				</button>
			{/each}
		</div>
		<p class="text-xs text-gray-500 mt-2">
			{#if accentColor}
				Using <strong>{ACCENT_COLORS[accentColor].label}</strong> for this view
			{:else}
				Inherits from global profile setting
			{/if}
		</p>
	</div>

	<div class="flex flex-col gap-3 pt-2">
		<label class="flex items-center gap-3">
			<input
				type="checkbox"
				bind:checked={isActive}
				class="w-4 h-4 text-primary-600 rounded border-gray-300"
			/>
			<div>
				<span class="text-sm font-medium text-gray-700 dark:text-gray-300">Active</span>
				<p class="text-xs text-gray-500">Inactive views are not accessible publicly</p>
			</div>
		</label>
	</div>
</div>