<script lang="ts">
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

<SectionLanding titleKey="admin.section.content.title" descKey="admin.section.content.description" {cards} />
