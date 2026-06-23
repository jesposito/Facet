<script lang="ts">
	import SectionLanding from '$components/admin/section/SectionLanding.svelte';
	import type { SectionCard } from '$components/admin/section/SectionLanding.svelte';

	const cards: SectionCard[] = [
		{ href: '/admin/analytics', icon: 'chart', labelKey: 'admin.sidebar.analytics' },
		{ href: '/admin/contacts', icon: 'mail', labelKey: 'admin.sidebar.contact_methods' },
		{ href: '/admin/testimonials', icon: 'chat', labelKey: 'admin.sidebar.testimonials' },
		{ href: '/admin/testimonials/requests', icon: 'link', labelKey: 'admin.sidebar.request_links' },
		{ href: '/admin/comments', icon: 'comment', labelKey: 'admin.sidebar.comments' },
		{ href: '/admin/newsletter', icon: 'mail', labelKey: 'admin.sidebar.newsletter' },
		{ href: '/admin/subscribers', icon: 'users', labelKey: 'admin.sidebar.subscribers' }
	];
</script>

<SectionLanding titleKey="admin.section.audience.title" descKey="admin.section.audience.description" {cards} />
