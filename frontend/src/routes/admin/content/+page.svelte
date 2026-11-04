<script lang="ts">
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import SectionLanding from '$components/admin/section/SectionLanding.svelte';
	import type { SectionCard } from '$components/admin/section/SectionLanding.svelte';

	const cards: SectionCard[] = [
		{ href: '/admin/posts', icon: 'document', labelKey: 'admin.sidebar.posts' },
		{ href: '/admin/talks', icon: 'presentation', labelKey: 'admin.sidebar.talks' },
		{ href: '/admin/custom', icon: 'puzzle', labelKey: 'admin.sidebar.custom_content' },
		{ href: '/admin/import', icon: 'sparkle', labelKey: 'admin.sidebar.import_ai' },
		{ href: '/admin/media', icon: 'image', labelKey: 'admin.sidebar.media' },
		{ href: '/admin/settings/tags', icon: 'tag', labelKey: 'admin.sidebar.admin_tags' }
	];
</script>

<svelte:head>
	<title>{$t('admin.section.content.title')} | {$brandName}</title>
</svelte:head>

<SectionLanding
	titleKey="admin.section.content.title"
	descKey="admin.section.content.description"
	{cards}
/>
