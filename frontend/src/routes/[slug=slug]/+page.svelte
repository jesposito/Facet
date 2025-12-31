<script lang="ts">
	import type { PageData } from './$types';
	import { enhance } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import ProfileHero from '$components/public/ProfileHero.svelte';
	import ExperienceSection from '$components/public/ExperienceSection.svelte';
	import ProjectsSection from '$components/public/ProjectsSection.svelte';
	import EducationSection from '$components/public/EducationSection.svelte';
	import SkillsSection from '$components/public/SkillsSection.svelte';
	import PostsSection from '$components/public/PostsSection.svelte';
	import TalksSection from '$components/public/TalksSection.svelte';
	import Footer from '$components/public/Footer.svelte';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
	import PasswordPrompt from '$components/public/PasswordPrompt.svelte';

	export let data: PageData;

	// Hidden form ref for setting password token cookie
	let passwordForm: HTMLFormElement;
	let tokenInput: HTMLInputElement;
	let maxAgeInput: HTMLInputElement;

	async function handlePasswordVerified(event: CustomEvent<{ token: string; expiresIn: number }>) {
		const { token, expiresIn } = event.detail;

		// Set form values and submit to set cookie via server action
		tokenInput.value = token;
		maxAgeInput.value = String(expiresIn);
		passwordForm.requestSubmit();
	}
</script>

<svelte:head>
	<title>{data.view?.name || 'View'} | {data.profile?.name || 'Profile'}</title>
	<meta name="description" content={data.view?.hero_headline || data.profile?.headline || ''} />
	<!-- Canonical URL is /<slug> -->
	<link rel="canonical" href="/{data.view?.slug}" />
</svelte:head>

<!-- Hidden form for setting password token cookie -->
<form
	bind:this={passwordForm}
	method="POST"
	action="?/setPasswordToken"
	class="hidden"
	use:enhance={() => {
		return async ({ result }) => {
			if (result.type === 'success') {
				// Reload page to fetch data with the new token
				await invalidateAll();
			}
		};
	}}
>
	<input bind:this={tokenInput} type="hidden" name="token" value="" />
	<input bind:this={maxAgeInput} type="hidden" name="maxAge" value="3600" />
</form>

{#if data.requiresPassword}
	<PasswordPrompt
		viewId={data.view?.id || ''}
		on:verified={handlePasswordVerified}
	/>
{:else if !data.view}
	<div class="min-h-screen flex items-center justify-center">
		<div class="text-center">
			<h1 class="text-4xl font-bold text-gray-900 dark:text-white mb-4">Not Found</h1>
			<p class="text-gray-600 dark:text-gray-400">This page doesn't exist.</p>
			<a href="/" class="mt-4 inline-block btn btn-primary">Go Home</a>
		</div>
	</div>
{:else}
	<div class="min-h-screen">
		<div class="fixed top-4 right-4 z-40">
			<ThemeToggle />
		</div>

		<!-- Modified hero with view overrides -->
		<ProfileHero
			profile={{
				...data.profile,
				headline: data.view?.hero_headline || data.profile?.headline,
				summary: data.view?.hero_summary || data.profile?.summary
			}}
		/>

		<!-- CTA banner if configured -->
		{#if data.view?.cta_text && data.view?.cta_url}
			<div class="bg-primary-600 text-white py-4">
				<div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center justify-between">
					<span class="font-medium">{data.view.cta_text}</span>
					<a
						href={data.view.cta_url}
						target="_blank"
						rel="noopener noreferrer"
						class="btn bg-white text-primary-600 hover:bg-gray-100"
					>
						Learn More
					</a>
				</div>
			</div>
		{/if}

		<main class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
			{#if data.sections?.experience?.length > 0}
				<ExperienceSection items={data.sections.experience} />
			{/if}

			{#if data.sections?.projects?.length > 0}
				<ProjectsSection items={data.sections.projects} />
			{/if}

			{#if data.sections?.education?.length > 0}
				<EducationSection items={data.sections.education} />
			{/if}

			{#if data.sections?.skills?.length > 0}
				<SkillsSection items={data.sections.skills} />
			{/if}

			{#if data.sections?.posts?.length > 0}
				<PostsSection items={data.sections.posts} />
			{/if}

			{#if data.sections?.talks?.length > 0}
				<TalksSection items={data.sections.talks} />
			{/if}
		</main>

		<Footer profile={data.profile} />
	</div>
{/if}
