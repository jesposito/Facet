<script lang="ts">
	import SectionLanding from '$components/admin/section/SectionLanding.svelte';
	import type { SectionCard } from '$components/admin/section/SectionLanding.svelte';

	const cards: SectionCard[] = [
		{ href: '/admin/settings/account', icon: 'badge', labelKey: 'admin.sidebar.account_security' },
		{ href: '/admin/settings/site', icon: 'cog', labelKey: 'admin.sidebar.site_settings' },
		{ href: '/admin/settings/features', icon: 'puzzle', labelKey: 'admin.sidebar.features' },
		{ href: '/admin/settings/integrations', icon: 'sparkle', labelKey: 'admin.sidebar.integrations' },
		{ href: '/admin/settings/webhooks', icon: 'webhook', labelKey: 'admin.sidebar.webhooks' },
		{ href: '/admin/api', icon: 'key', labelKey: 'admin.sidebar.api_keys' },
		{ href: '/admin/settings/audit', icon: 'shield', labelKey: 'admin.sidebar.audit_log' },
		{ href: '/admin/settings/about', icon: 'info', labelKey: 'admin.sidebar.about_facet' }
	];
</script>

<SectionLanding titleKey="admin.section.settings.title" descKey="admin.section.settings.description" {cards} />
