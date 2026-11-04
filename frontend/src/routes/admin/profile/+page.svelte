<script lang="ts">
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import SectionLanding from '$components/admin/section/SectionLanding.svelte';
	import type { SectionCard } from '$components/admin/section/SectionLanding.svelte';

	const cards: SectionCard[] = [
		{ href: '/admin/experience', icon: 'briefcase', labelKey: 'admin.sidebar.experience' },
		{ href: '/admin/projects', icon: 'folder', labelKey: 'admin.sidebar.projects' },
		{ href: '/admin/education', icon: 'academic', labelKey: 'admin.sidebar.education' },
		{ href: '/admin/certifications', icon: 'badge', labelKey: 'admin.sidebar.certifications' },
		{ href: '/admin/awards', icon: 'star', labelKey: 'admin.sidebar.awards' },
		{ href: '/admin/skills', icon: 'chip', labelKey: 'admin.sidebar.skills' }
	];
</script>

<svelte:head>
	<title>{$t('admin.section.profile.title')} | {$brandName}</title>
</svelte:head>

<SectionLanding
	titleKey="admin.section.profile.title"
	descKey="admin.section.profile.description"
	{cards}
/>
