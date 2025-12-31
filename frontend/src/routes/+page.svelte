<script lang="ts">
	import type { PageData } from './$types';
	import ProfileHero from '$components/public/ProfileHero.svelte';
	import ExperienceSection from '$components/public/ExperienceSection.svelte';
	import ProjectsSection from '$components/public/ProjectsSection.svelte';
	import EducationSection from '$components/public/EducationSection.svelte';
	import SkillsSection from '$components/public/SkillsSection.svelte';
	import Footer from '$components/public/Footer.svelte';
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';

	export let data: PageData;
</script>

<svelte:head>
	<title>{data.profile?.name || 'Profile'} | Me.yaml</title>
	<meta name="description" content={data.profile?.headline || 'Personal profile and portfolio'} />
	{#if data.profile?.headline}
		<meta property="og:title" content={data.profile.name} />
		<meta property="og:description" content={data.profile.headline} />
	{/if}
</svelte:head>

<div class="min-h-screen">
	<!-- Theme toggle -->
	<div class="fixed top-4 right-4 z-40">
		<ThemeToggle />
	</div>

	<!-- Hero section -->
	<ProfileHero profile={data.profile} />

	<!-- Main content -->
	<main class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
		{#if data.experience.length > 0}
			<ExperienceSection items={data.experience} />
		{/if}

		{#if data.projects.length > 0}
			<ProjectsSection items={data.projects} />
		{/if}

		{#if data.education.length > 0}
			<EducationSection items={data.education} />
		{/if}

		{#if data.skills.length > 0}
			<SkillsSection items={data.skills} />
		{/if}
	</main>

	<Footer profile={data.profile} />
</div>
