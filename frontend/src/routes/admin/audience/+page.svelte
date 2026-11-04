<script lang="ts">
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import SectionLanding from '$components/admin/section/SectionLanding.svelte';
	import type { SectionCard } from '$components/admin/section/SectionLanding.svelte';
	import { siteFeatures } from '$lib/stores/features';

	// Cards follow Settings → Features. Contacts, testimonials and subscribers
	// stay visible whatever the switches say, so the owner can always review,
	// export or remove the people and records they already hold.
	const cards = $derived<SectionCard[]>([
		{
			href: '/admin/analytics',
			icon: 'chart',
			labelKey: 'admin.sidebar.analytics',
			hidden: !$siteFeatures.analytics
		},
		{ href: '/admin/contacts', icon: 'mail', labelKey: 'admin.sidebar.contact_methods' },
		{ href: '/admin/testimonials', icon: 'chat', labelKey: 'admin.sidebar.testimonials' },
		{ href: '/admin/testimonials/requests', icon: 'link', labelKey: 'admin.sidebar.request_links' },
		{
			href: '/admin/comments',
			icon: 'comment',
			labelKey: 'admin.sidebar.comments',
			hidden: !$siteFeatures.discussions
		},
		{
			href: '/admin/newsletter',
			icon: 'mail',
			labelKey: 'admin.sidebar.newsletter',
			hidden: !$siteFeatures.newsletter
		},
		{ href: '/admin/subscribers', icon: 'users', labelKey: 'admin.sidebar.subscribers' }
	]);
</script>

<svelte:head>
	<title>{$t('admin.section.audience.title')} | {$brandName}</title>
</svelte:head>

<SectionLanding
	titleKey="admin.section.audience.title"
	descKey="admin.section.audience.description"
	{cards}
/>
