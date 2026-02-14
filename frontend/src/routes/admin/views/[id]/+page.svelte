<script lang="ts">
	import { preventDefault, createBubbler, stopPropagation, self } from 'svelte/legacy';

	const bubble = createBubbler();
	import { onMount, onDestroy, tick } from 'svelte';
	import { page } from '$app/stores';
	import { goto, afterNavigate } from '$app/navigation';
	import { pb, type View, type ViewSection, type ItemConfig, type Profile, type SectionWidth, type ShareToken, OVERRIDABLE_FIELDS, VALID_LAYOUTS, VALID_WIDTHS, getValidWidthsForLayout, isWidthValidForLayout } from '$lib/pocketbase';
	import { collection } from '$lib/stores/demo';
	import { toasts, confirm } from '$lib/stores';
	import { icon } from '$lib/icons';

	import { ACCENT_COLORS, ACCENT_COLOR_LIST, type AccentColor } from '$lib/colors';
	import { flip } from 'svelte/animate';
	import ViewPreview from '$components/admin/ViewPreview.svelte';
	import ViewOverrideEditor from '$components/admin/view-editor/ViewOverrideEditor.svelte';
	import ViewResumeGenerator from '$components/admin/view-editor/ViewResumeGenerator.svelte';
	import ViewBasicInfo from '$components/admin/view-editor/ViewBasicInfo.svelte';
	import ViewSectionManager from '$components/admin/view-editor/ViewSectionManager.svelte';
	import AdminTagBadge from '$components/admin/AdminTagBadge.svelte';
	import { t } from 'svelte-i18n';
	import type { OverrideEditorState, ResumeGenerationConfig, ExportRecord } from '$lib/view-editor/types';

	// Default section definitions - used to initialize and provide labels
	const SECTION_DEFS: Record<string, { label: string; collection: string }> = {
		experience: { label: 'Experience', collection: 'experience' },
		projects: { label: 'Projects', collection: 'projects' },
		education: { label: 'Education', collection: 'education' },
		certifications: { label: 'Certifications', collection: 'certifications' },
		awards: { label: 'Awards', collection: 'awards' },
		skills: { label: 'Skills', collection: 'skills' },
		posts: { label: 'Posts', collection: 'posts' },
		talks: { label: 'Talks', collection: 'talks' },
		contacts: { label: 'Contact Methods', collection: 'contact_methods' },
		testimonials: { label: 'Testimonials', collection: 'testimonials' }
	};

	// Default section order
	const DEFAULT_SECTION_ORDER = ['experience', 'projects', 'education', 'certifications', 'awards', 'skills', 'posts', 'talks', 'testimonials', 'contacts'];

	// Custom content items (each becomes its own section)
	// Includes full data for preview rendering
	let customContentItems: Array<{
		id: string;
		title: string;
		is_draft: boolean;
		visibility: 'public' | 'unlisted' | 'private';
		sort_order?: number;
		content?: string;
		cover_image?: string;
		cover_image_url?: string;
		media?: string[];
		media_urls?: string[];
	}> = $state([]);

	// Dynamic section definitions (includes custom content)
	let allSectionDefs = $derived(() => {
		const defs: Record<string, { label: string; collection: string; isCustom?: boolean }> = { ...SECTION_DEFS };
		for (const item of customContentItems) {
			defs[`custom:${item.id}`] = {
				label: item.title,
				collection: 'custom_content',
				isCustom: true
			};
		}
		return defs;
	});

	// Helper to check if a section key is for custom content
	function isCustomSection(sectionKey: string): boolean {
		return sectionKey.startsWith('custom:');
	}

	// Helper to get custom content ID from section key
	function getCustomContentId(sectionKey: string): string {
		return sectionKey.replace('custom:', '');
	}

	let loading = $state(true);
	let saving = $state(false);
	let deleting = $state(false);
	let view: View | null = $state(null);
	let mounted = false;

	// Profile data for preview and showing overrides
	let profile: Profile | null = $state(null);

	// Preview panel state
	let showPreview = $state(true);
	let previewMode: 'desktop' | 'mobile' = $state('desktop'); // Phase 6.2.2

	// Form fields
	let name = $state('');
	let slug = $state('');
	let description = $state('');
	let visibility: 'public' | 'unlisted' | 'private' | 'password' = $state('public');
	let password = $state(''); // For password-protected views (only set when changing)
	let heroHeadline = $state('');
	let heroSummary = $state('');
	let heroLocation = $state('');
	let ctaText = $state('');
	let ctaUrl = $state('');
	let ctaButtonText = $state('');
	let ctaEnabled = $state(true); // Per-view CTA toggle (defaults to true)
	let isActive = $state(true);
	let accentColor: AccentColor | null = $state(null);
	let heroImageUrl: string | null = $state(null);
	let heroImageFile: File | null = null;
	let heroBlobUrl: string | null = null;

	// Sections configuration with itemConfig, layout, and width support
	let sections: Record<string, {
		enabled: boolean;
		items: string[];
		expanded: boolean;
		layout: string;
		width: SectionWidth;
		itemConfig: Record<string, ItemConfig>;
		categoryOrder?: string[];
		disabledCategories?: string[];
		categoryDisplayModes?: Record<string, string>;
		featuredId?: string;
	}> = $state({});

	// Section order for drag-drop (array of section keys with unique ids for dndzone)
	let sectionOrder: Array<{ id: string; key: string }> = $state([]);
	const flipDurationMs = 200;

	// Available items for each section (full data for override editing)
	let sectionItems: Record<string, Array<{
		id: string;
		label: string;
		visibility: string;
		is_draft?: boolean;
		data: Record<string, unknown>;
		expand?: {
			admin_tags?: Array<{ id: string; name: string; color: string }>;
		};
	}>> = $state({});

	// Override editor state
	let showOverrideEditor = $state(false);
	let editingOverride: OverrideEditorState | null = $state(null);

	// AI Print state
	let showGenerateModal = $state(false);
	let aiPrintStatus = $state({
		available: false,
		pandoc_installed: false,
		ai_configured: false
	});
	let exports: ExportRecord[] = $state([]);

	// Share token generation state
	let viewTokens: ShareToken[] = $state([]);
	let generatingToken = $state(false);
	let showTokenForm = $state(false);
	let newTokenName = $state('');
	let newTokenExpires = $state('');
	let newTokenMaxUses = $state(0);
	let createdTokenUrl: string | null = $state(null);



	onMount(async () => {
		mounted = true;
		if (!viewId) {
			toasts.add('error', 'Invalid facet ID');
			goto('/admin/views');
			return;
		}
		// Load custom content FIRST since initializeSections() needs it
		await loadCustomContentItems();
		await Promise.all([
			loadView(),
			loadSectionItems(),
			loadProfile(),
			checkAIPrintStatus(),
			loadViewTokens()
		]);
		await tick();
		applySavedItemOrder();
		await loadExports();
	});

	onDestroy(() => {
		mounted = false;
	});

	// Reload view data when navigating between different views (e.g., clicking facets in sidebar)
	afterNavigate(({ from, to }) => {
		const fromId = from?.params?.id;
		const toId = to?.params?.id;
		// Only reload if navigating between different view IDs
		if (fromId && toId && fromId !== toId) {
			loading = true;
			// Load custom content FIRST since initializeSections() needs it
			loadCustomContentItems().then(() => {
				return Promise.all([
					loadView(),
					loadSectionItems(),
					loadViewTokens()
				]);
			}).then(async () => {
				// Wait for Svelte to process reactive updates before reordering
				await tick();
				applySavedItemOrder();
			}).finally(() => {
				loading = false;
			});
		}
	});


	async function loadProfile() {
		try {
			const records = await collection('profile').getList(1, 1);
			if (records.items.length > 0) {
				const record = records.items[0] as unknown as Profile & { collectionId: string };
				// Resolve avatar URL if exists
				if (record.avatar) {
					const avatarUrl = `/api/files/${record.collectionId}/${record.id}/${record.avatar}`;
					profile = { ...record, avatar: avatarUrl };
				} else {
					profile = record;
				}
			}
		} catch (err) {
			console.error('Failed to load profile:', err);
			// Profile is optional for preview, don't show error
		}
	}

	// Load custom content items - each becomes its own section
	async function loadCustomContentItems() {
		try {
			const records = await collection('custom_content').getList(1, 100, {
				sort: 'sort_order'
			});
			customContentItems = records.items.map(item => {
				const record = item as any;
				const collectionId = record.collectionId || 'custom_content';

				// Build cover image URL if exists
				let cover_image_url: string | undefined;
				if (record.cover_image) {
					cover_image_url = `/api/files/${collectionId}/${record.id}/${record.cover_image}`;
				}

				// Build media URLs if exist
				let media_urls: string[] | undefined;
				if (record.media && Array.isArray(record.media) && record.media.length > 0) {
					media_urls = record.media.map((filename: string) =>
						`/api/files/${collectionId}/${record.id}/${filename}`
					);
				}

				return {
					id: record.id,
					title: record.title || 'Untitled',
					is_draft: record.is_draft || false,
					// Default to 'private' if visibility is not set - safer than assuming public
					visibility: (record.visibility || 'private') as 'public' | 'unlisted' | 'private',
					sort_order: record.sort_order ?? 0,
					content: record.content,
					cover_image: record.cover_image,
					cover_image_url,
					media: record.media,
					media_urls
				};
			});
		} catch (err) {
			console.error('Failed to load custom content:', err);
			customContentItems = [];
		}
	}

	// AI Print functions
	async function checkAIPrintStatus() {
		try {
			console.log('[AI-PRINT] Checking status...');
			const response = await fetch('/api/ai-print/status', {
				headers: { Authorization: pb.authStore.token || '' }
			});
			console.log('[AI-PRINT] Status response:', response.status);
			if (response.ok) {
				const data = await response.json();
				console.log('[AI-PRINT] Status result:', data);
				aiPrintStatus = {
					available: data.available,
					pandoc_installed: data.pandoc_installed,
					ai_configured: data.ai_configured
				};
			}
		} catch (err) {
			console.error('[AI-PRINT] Failed to check status:', err);
		}
	}

	async function loadExports() {
		if (!slug) return;
		try {
			const response = await fetch(`/api/view/${slug}/exports`, {
				headers: { Authorization: pb.authStore.token || '' }
			});
			if (response.ok) {
				const data = await response.json();
				exports = data.exports || [];
			}
		} catch (err) {
			console.error('Failed to load exports:', err);
		}
	}

	/**
	 * Handle resume generation from ViewResumeGenerator component.
	 * The component manages its own loading state; we handle the API call and state updates.
	 */
	async function handleResumeGenerate(config: ResumeGenerationConfig): Promise<void> {
		if (!slug) return;
		
		console.log('[AI-PRINT] Starting generation for:', slug);
		const response = await fetch(`/api/view/${slug}/generate`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: pb.authStore.token || ''
			},
			body: JSON.stringify(config)
		});

		const data = await response.json();
		console.log('[AI-PRINT] Generation result:', data);

		if (!response.ok) {
			throw new Error(data.error || 'Generation failed');
		}

		toasts.add('success', 'Resume generated successfully!');
		showGenerateModal = false;

		// Add new export to list
		exports = [{
			id: data.export_id,
			format: data.format,
			status: data.status,
			generated_at: data.generated_at,
			download_url: data.download_url
		}, ...exports];

		// Auto-download the file
		if (data.download_url) {
			console.log('[AI-PRINT] Auto-downloading from:', data.download_url);
			const link = document.createElement('a');
			link.href = data.download_url;
			link.download = `resume.${config.format}`;
			document.body.appendChild(link);
			link.click();
			document.body.removeChild(link);
		}
	}

	async function deleteExport(exportId: string) {
		if (!slug) return;
		const confirmed = await confirm({
			title: 'Delete Export',
			message: 'Delete this export? This action cannot be undone.',
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) return;
		try {
			const response = await fetch(`/api/view/${slug}/exports/${exportId}`, {
				method: 'DELETE',
				headers: { Authorization: pb.authStore.token || '' }
			});
			if (response.ok) {
				exports = exports.filter(e => e.id !== exportId);
				toasts.add('success', 'Export deleted');
			} else {
				toasts.add('error', 'Failed to delete export');
			}
		} catch (err) {
			toasts.add('error', 'Failed to delete export');
		}
	}

	async function deleteView() {
		if (!viewId || deleting) return;
		const confirmed = await confirm({
			title: 'Delete Facet',
			message: `Are you sure you want to delete "${name || 'this facet'}"? This will also delete all associated share tokens and exports. This action cannot be undone.`,
			confirmText: 'Delete',
			danger: true
		});
		if (!confirmed) return;
		deleting = true;
		try {
			// Delete related share tokens first (PocketBase requires this for relation constraints)
			const tokensToDelete = await pb.collection('share_tokens').getFullList({
				filter: `view_id = "${viewId}"`
			});
			for (const token of tokensToDelete) {
				await pb.collection('share_tokens').delete(token.id);
			}

			// Now delete the view
			await collection('views').delete(viewId);
			toasts.add('success', 'Facet deleted');
			goto('/admin/views');
		} catch (err) {
			console.error('Failed to delete view:', err);
			toasts.add('error', 'Failed to delete facet');
			deleting = false;
		}
	}

	// Share token functions
	async function loadViewTokens() {
		if (!viewId) return;
		try {
			const result = await pb.collection('share_tokens').getList<ShareToken>(1, 10, {
				filter: `view_id = "${viewId}"`,
				sort: '-id'
			});
			viewTokens = result.items;
		} catch (err) {
			// Silent fail - tokens are optional
		}
	}

	async function generateShareToken() {
		if (!viewId) return;

		generatingToken = true;
		try {
			const response = await fetch('/api/share/generate', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({
					view_id: viewId,
					name: newTokenName || undefined,
					expires_at: newTokenExpires || undefined,
					max_uses: newTokenMaxUses || 0
				})
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to create token');
			}

			const data = await response.json();

			// Store the URL to show
			createdTokenUrl = `${window.location.origin}/s/${data.token}`;

			toasts.add('success', 'Share link created!');
			await loadViewTokens();
			resetTokenForm();
		} catch (err) {
			toasts.add('error', err instanceof Error ? err.message : 'Failed to create share link');
		} finally {
			generatingToken = false;
		}
	}

	async function revokeViewToken(tokenId: string) {
		const confirmed = await confirm({
			title: 'Revoke Share Link',
			message: 'Are you sure you want to revoke this share link? It will no longer be usable.',
			confirmText: 'Revoke',
			danger: true
		});
		if (!confirmed) return;

		try {
			const response = await fetch(`/api/share/revoke/${tokenId}`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${pb.authStore.token}`
				}
			});

			if (!response.ok) {
				throw new Error('Failed to revoke token');
			}

			toasts.add('success', 'Share link revoked');
			await loadViewTokens();
		} catch (err) {
			toasts.add('error', 'Failed to revoke share link');
		}
	}

	async function regenerateViewToken(token: ShareToken) {
		const confirmed = await confirm({
			title: 'Regenerate Share Link',
			message: `This will revoke the existing link "${token.name || 'Unnamed'}" and create a new one. Anyone using the old link will no longer have access. Continue?`,
			confirmText: 'Regenerate',
			danger: false
		});
		if (!confirmed) return;

		try {
			// First revoke the old token
			const revokeResponse = await fetch(`/api/share/revoke/${token.id}`, {
				method: 'POST',
				headers: {
					Authorization: `Bearer ${pb.authStore.token}`
				}
			});

			if (!revokeResponse.ok) {
				throw new Error('Failed to revoke old token');
			}

			// Then create a new token with the same settings
			const response = await fetch('/api/share/generate', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: `Bearer ${pb.authStore.token}`
				},
				body: JSON.stringify({
					view_id: viewId,
					name: token.name || undefined,
					expires_at: token.expires_at || undefined,
					max_uses: token.max_uses || 0
				})
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to create new token');
			}

			const data = await response.json();
			createdTokenUrl = `${window.location.origin}/s/${data.token}`;

			toasts.add('success', 'Share link regenerated! Copy the new link now.');
			await loadViewTokens();
		} catch (err) {
			toasts.add('error', err instanceof Error ? err.message : 'Failed to regenerate share link');
		}
	}

	function resetTokenForm() {
		newTokenName = '';
		newTokenExpires = '';
		newTokenMaxUses = 0;
		showTokenForm = false;
	}

	function copyShareUrl(text: string) {
		navigator.clipboard.writeText(text);
		toasts.add('success', 'Copied to clipboard');
	}

	function dismissCreatedToken() {
		createdTokenUrl = null;
	}

	function isTokenExpired(token: ShareToken): boolean {
		if (!token.expires_at) return false;
		return new Date(token.expires_at) < new Date();
	}

	function isTokenMaxUsesReached(token: ShareToken): boolean {
		if (!token.max_uses || token.max_uses === 0) return false;
		return token.use_count >= token.max_uses;
	}

	function getTokenStatusInfo(token: ShareToken): { label: string; class: string } {
		if (!token.is_active) {
			return { label: 'Revoked', class: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300' };
		}
		if (isTokenExpired(token)) {
			return { label: 'Expired', class: 'bg-gray-200 text-gray-600 dark:bg-gray-700 dark:text-gray-400' };
		}
		if (isTokenMaxUsesReached(token)) {
			return { label: 'Max Uses', class: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300' };
		}
		return { label: 'Active', class: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' };
	}

	onDestroy(() => {
		if (heroBlobUrl) URL.revokeObjectURL(heroBlobUrl);
	});

	function handleHeroImageChange(event: Event) {
		const input = event.target as HTMLInputElement;
		if (input.files?.[0]) {
			if (heroBlobUrl) URL.revokeObjectURL(heroBlobUrl);
			heroImageFile = input.files[0];
			heroBlobUrl = URL.createObjectURL(heroImageFile);
			heroImageUrl = heroBlobUrl;
		}
	}

	async function removeHeroImage() {
		if (!view) return;
		const confirmed = await confirm({
			title: 'Remove Hero Image',
			message: 'Are you sure you want to remove the hero image from this view?',
			confirmText: 'Remove',
			danger: true
		});
		if (!confirmed) return;
		try {
			await collection('views').update(viewId, { hero_image: null });
			heroImageUrl = null;
			heroImageFile = null;
			toasts.add('success', 'Hero image removed');
		} catch (err) {
			console.error('Failed to remove hero image:', err);
			toasts.add('error', 'Failed to remove hero image');
		}
	}

	async function loadView() {
		if (!viewId) return;
		loading = true;
		try {
			const record = await collection('views').getOne(viewId) as unknown as View & { collectionId: string; hero_image?: string };
			view = record as unknown as View;

			name = view.name;
			slug = view.slug;
			description = view.description || '';
			visibility = view.visibility;
			heroHeadline = view.hero_headline || '';
			heroSummary = view.hero_summary || '';
			heroLocation = view.hero_location || '';
			ctaText = view.cta_text || '';
			ctaUrl = view.cta_url || '';
			ctaButtonText = view.cta_button_text || '';
			ctaEnabled = (view as any).cta_enabled !== false; // Default to true if not set
			isActive = view.is_active;
			accentColor = (view.accent_color as AccentColor) || null;

			if (record.hero_image) {
				heroImageUrl = `/api/files/${record.collectionId}/${record.id}/${record.hero_image}`;
			}

			initializeSections(view.sections);
		} catch (err) {
			if (err instanceof Error && (err.message.includes('autocancelled') || err.name === 'AbortError')) {
				return;
			}
			console.error('Failed to load view:', err);
			if (mounted) {
				toasts.add('error', 'Failed to load facet');
				goto('/admin/views');
			}
		} finally {
			loading = false;
		}
	}

	function initializeSections(viewSections?: ViewSection[]) {
		for (const key of DEFAULT_SECTION_ORDER) {
			const defaultLayout = VALID_LAYOUTS[key]?.default || 'default';
			sections[key] = { enabled: false, items: [], expanded: false, layout: defaultLayout, width: 'full', itemConfig: {}, categoryOrder: undefined, disabledCategories: undefined, categoryDisplayModes: undefined, featuredId: undefined };
		}

		for (const item of customContentItems) {
			const sectionKey = `custom:${item.id}`;
			const defaultLayout = VALID_LAYOUTS['custom']?.default || 'default';
			sections[sectionKey] = { enabled: false, items: [], expanded: false, layout: defaultLayout, width: 'full', itemConfig: {} };
		}

		const customSectionKeys = customContentItems.map(item => `custom:${item.id}`);
		const allAvailableSections = [...DEFAULT_SECTION_ORDER, ...customSectionKeys];

		if (viewSections && viewSections.length > 0) {
			const savedOrder = viewSections.map(vs => vs.section);
			const remainingSections = allAvailableSections.filter(k => !savedOrder.includes(k));
			const fullOrder = [...savedOrder, ...remainingSections];

			sectionOrder = fullOrder.map(key => ({ id: `section-${key}`, key }));

			for (const vs of viewSections) {
				if (sections[vs.section] !== undefined) {
					sections[vs.section].enabled = vs.enabled;
					sections[vs.section].items = vs.items || [];
					const layoutKey = isCustomSection(vs.section) ? 'custom' : vs.section;
					sections[vs.section].layout = vs.layout || VALID_LAYOUTS[layoutKey]?.default || 'default';
					sections[vs.section].width = vs.width || 'full';
					sections[vs.section].itemConfig = vs.itemConfig || {};
					sections[vs.section].categoryOrder = vs.categoryOrder;
					sections[vs.section].disabledCategories = vs.disabledCategories;
					sections[vs.section].categoryDisplayModes = vs.categoryDisplayModes;
					sections[vs.section].featuredId = vs.featuredId;
				}
			}
		} else {
			sectionOrder = allAvailableSections.map(key => ({ id: `section-${key}`, key }));
		}
		updateSections();
	}

	async function loadSectionItems() {
		for (const key of DEFAULT_SECTION_ORDER) {
			const def = SECTION_DEFS[key];
			try {
				// Testimonials only show approved ones in view editor
				const filter = key === 'testimonials' ? 'status = "approved"' : '';
				const records = await collection(def.collection).getList(1, 100, {
					sort: key === 'testimonials' ? '-featured,-sort_order' : '-id',
					filter,
					expand: 'admin_tags'
				});

				sectionItems[key] = records.items.map((item) => ({
					id: item.id,
					label: getItemLabel(key, item),
					// Default to 'private' if visibility is not set - safer than assuming public
					visibility: (item as Record<string, unknown>).visibility as string || 'private',
					is_draft: (item as Record<string, unknown>).is_draft as boolean || false,
					data: item as Record<string, unknown>,
					expand: (item as any).expand || {}
				}));
			} catch (err) {
				console.error(`Failed to load ${key} items:`, err);
				sectionItems[key] = [];
			}
		}
		updateSectionItems();
	}



	function getItemLabel(sectionKey: string, item: Record<string, unknown>): string {
		switch (sectionKey) {
			case 'experience':
				return `${item.title} at ${item.company}`;
			case 'projects':
				return item.title as string;
			case 'education':
				return `${item.degree || 'Degree'} - ${item.institution}`;
			case 'certifications':
				return `${item.name} (${item.issuer || 'Unknown issuer'})`;
			case 'awards':
				return `${item.title}${item.issuer ? ` (${item.issuer})` : ''}`;
			case 'skills':
				return `${item.name}${item.category ? ` (${item.category})` : ''}`;
			case 'posts':
				return item.title as string;
			case 'talks':
				return `${item.title}${item.event ? ` @ ${item.event}` : ''}`;
			case 'contacts':
				return `${item.label || item.type} - ${item.value}`;
			case 'testimonials':
				return `${item.author_name}${item.author_company ? ` - ${item.author_company}` : ''}`;
			default:
				return item.title as string || item.name as string || item.id as string;
		}
	}

	// Helper to trigger reactivity by creating a new object reference
	function updateSections() {
		sections = { ...sections };
	}

	// Helper to trigger reactivity for sectionItems
	function updateSectionItems() {
		sectionItems = { ...sectionItems };
	}



	function generateSlug(value: string): string {
		return value
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 50);
	}

	async function enableItemsForView(sectionsData: ViewSection[]) {
		for (const section of sectionsData) {
			if (!section.enabled || !section.items?.length) continue;
			
			const collectionName = SECTION_DEFS[section.section]?.collection;
			if (!collectionName) continue;

			for (const itemId of section.items) {
				const item = sectionItems[section.section]?.find(i => i.id === itemId);
				if (!item) continue;

				const isPrivate = item.visibility === 'private';
				const isDraft = item.is_draft === true;
				
				if (isPrivate || isDraft) {
					const currentViewVis = (item.data.view_visibility as Record<string, boolean>) || {};
					if (!currentViewVis[viewId]) {
						await collection(collectionName).update(itemId, {
							view_visibility: { ...currentViewVis, [viewId]: true }
						});
					}
				}
			}
		}
	}

	async function handleSubmit() {
		if (!name.trim()) {
			toasts.add('error', 'Name is required');
			return;
		}
		if (!slug.trim()) {
			toasts.add('error', 'Slug is required');
			return;
		}

		saving = true;
		try {
			// Build sections array in current order with itemConfig, layout, and width
			// Important: We save ALL sections in order (enabled + disabled) so order is preserved
			const sectionsData: ViewSection[] = sectionOrder
				.map(({ key }) => {
					const sectionConfig = sections[key];
					const defaultLayout = VALID_LAYOUTS[key]?.default || 'default';
					const sectionData: ViewSection = {
						section: key,
						enabled: sectionConfig?.enabled || false,
						items: sectionConfig?.items || [],
						layout: sectionConfig?.layout || defaultLayout,
						width: sectionConfig?.width || 'full'
					};
					// Only include itemConfig if there are overrides
					const itemConfig = sectionConfig?.itemConfig;
					if (itemConfig && Object.keys(itemConfig).length > 0) {
						// Filter out empty configs
						const filteredConfig: Record<string, ItemConfig> = {};
						for (const [itemId, config] of Object.entries(itemConfig)) {
							if (config.overrides && Object.keys(config.overrides).length > 0) {
								filteredConfig[itemId] = config;
							}
						}
						if (Object.keys(filteredConfig).length > 0) {
							sectionData.itemConfig = filteredConfig;
						}
					}
					// Include skills section settings if set
					if (key === 'skills') {
						if (sectionConfig?.categoryOrder?.length) {
							sectionData.categoryOrder = sectionConfig.categoryOrder;
						}
						if (sectionConfig?.disabledCategories?.length) {
							sectionData.disabledCategories = sectionConfig.disabledCategories;
						}
						if (sectionConfig?.categoryDisplayModes && Object.keys(sectionConfig.categoryDisplayModes).length > 0) {
							sectionData.categoryDisplayModes = sectionConfig.categoryDisplayModes;
						}
					}
					// Include testimonials section settings if set
					if (key === 'testimonials' && sectionConfig?.featuredId) {
						sectionData.featuredId = sectionConfig.featuredId;
					}
					return sectionData;
				});

			const formData = new FormData();
			formData.append('name', name.trim());
			formData.append('slug', slug.trim());
			formData.append('description', description.trim());
			formData.append('visibility', visibility);
			formData.append('hero_headline', heroHeadline.trim() || '');
			formData.append('hero_summary', heroSummary.trim() || '');
			formData.append('hero_location', heroLocation.trim() || '');
			formData.append('cta_text', ctaText.trim() || '');
			formData.append('cta_url', ctaUrl.trim() || '');
			formData.append('cta_button_text', ctaButtonText.trim() || '');
			formData.append('cta_enabled', String(ctaEnabled));
			formData.append('is_active', String(isActive));
			formData.append('sections', JSON.stringify(sectionsData));
			formData.append('accent_color', accentColor || '');

			if (visibility === 'password' && password.trim()) {
				formData.append('password', password.trim());
			}

			if (heroImageFile) {
				formData.append('hero_image', heroImageFile);
			}

			await collection('views').update(viewId, formData);
			heroImageFile = null;

			await enableItemsForView(sectionsData);

			toasts.add('success', 'Facet updated successfully');
		} catch (err) {
			console.error('Failed to save view:', err);
			toasts.add('error', 'Failed to save facet');
		} finally {
			saving = false;
		}
	}

	function previewView() {
		window.open(`/${slug}`, '_blank');
	}

	// Override editor functions
	function openOverrideEditor(sectionKey: string, itemId: string) {
		const item = sectionItems[sectionKey]?.find(i => i.id === itemId);
		if (!item) return;

		const existingConfig = sections[sectionKey]?.itemConfig?.[itemId];
		editingOverride = {
			sectionKey,
			itemId,
			itemLabel: item.label,
			originalData: item.data,
			overrides: existingConfig?.overrides
				? { ...existingConfig.overrides }
				: {}
		};
		showOverrideEditor = true;
	}

	function closeOverrideEditor() {
		showOverrideEditor = false;
		editingOverride = null;
	}

	/**
	 * Handle override save from ViewOverrideEditor component.
	 * The component handles cleaning empty overrides, so we receive cleaned data.
	 */
	function handleOverrideSave(cleanedOverrides: Record<string, string | string[]>) {
		if (!editingOverride) return;

		const { sectionKey, itemId } = editingOverride;

		// Update itemConfig
		if (!sections[sectionKey].itemConfig) {
			sections[sectionKey].itemConfig = {};
		}

		if (Object.keys(cleanedOverrides).length > 0) {
			sections[sectionKey].itemConfig[itemId] = { overrides: cleanedOverrides };
		} else {
			delete sections[sectionKey].itemConfig[itemId];
		}

		updateSections();
		closeOverrideEditor();
		toasts.add('success', 'Overrides saved');
	}





	// Apply saved item order to sectionItems after loading
	// Selected items appear first (in saved order), then unselected items (in original order)
	function applySavedItemOrder() {
		let didUpdate = false;
		for (const key of Object.keys(sections)) {
			const savedOrder = sections[key]?.items || [];
			const allItems = sectionItems[key] || [];

			// Skip if no saved order or no items
			if (savedOrder.length === 0 || allItems.length === 0) continue;

			// Create a map for quick lookup
			const itemMap = new Map(allItems.map(item => [item.id, item]));

			// Build reordered array: selected items first (in saved order), then unselected
			const selectedItems: typeof allItems = [];
			for (const id of savedOrder) {
				const item = itemMap.get(id);
				if (item) {
					selectedItems.push(item);
				}
			}

			const selectedSet = new Set(savedOrder);
			const unselectedItems = allItems.filter(item => !selectedSet.has(item.id));

			sectionItems[key] = [...selectedItems, ...unselectedItems];
			didUpdate = true;
		}
		// Trigger Svelte 5 reactivity with full object reassignment
		if (didUpdate) {
			updateSectionItems();
		}
	}

	let viewId = $derived($page.params.id as string);
</script>

<svelte:head>
	<title>Edit Facet | Facet</title>
</svelte:head>

<div class="view-editor-container">
	{#if loading}
		<div class="card p-8 text-center max-w-4xl mx-auto">
			<div class="animate-pulse">Loading facet...</div>
		</div>
	{:else}
		<!-- Header -->
		<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
			<div class="flex items-center gap-4">
				<a href="/admin/views" class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" aria-label={$t('admin.views.back_to_views')}>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</a>
				<h1 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">Edit Facet</h1>
			</div>
			<div class="flex items-center gap-2 flex-wrap">
				<!-- Preview Toggle - hidden on mobile since preview is collapsible -->
				<button
					type="button"
					class="btn btn-ghost hidden lg:flex items-center gap-2"
					onclick={() => showPreview = !showPreview}
					title={showPreview ? 'Hide preview' : 'Show preview'}
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
						{#if showPreview}
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
						{:else}
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
						{/if}
					</svg>
					<span>{showPreview ? 'Hide' : 'Show'} Preview</span>
				</button>
				<button type="button" class="btn btn-secondary text-sm" onclick={previewView}>
					<span class="sm:hidden">View</span>
					<span class="hidden sm:inline">Open in Tab</span>
				</button>
				{#if aiPrintStatus.ai_configured}
					<button
						type="button"
						class="btn btn-secondary text-sm flex items-center gap-1 sm:gap-2"
						onclick={() => showGenerateModal = true}
						title={aiPrintStatus.pandoc_installed ? "Generate AI-powered resume" : "Generate Resume (Pandoc not installed)"}
					>
						<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
						</svg>
						<span class="hidden sm:inline">Generate Resume</span>
					</button>
				{/if}
				<button type="button" class="btn btn-primary text-sm" onclick={handleSubmit} disabled={saving}>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
						</svg>
					{/if}
					Save
				</button>
			</div>
		</div>

		<!-- Split Pane Layout -->
		<div class="editor-layout" class:with-preview={showPreview}>
			<!-- Editor Pane -->
			<div class="editor-pane">
		<form onsubmit={preventDefault(handleSubmit)} class="space-y-4 sm:space-y-6">
			<!-- Basic Info -->
			<div class="card p-4 sm:p-6 space-y-4">
				<ViewBasicInfo
					bind:name
					bind:slug
					bind:description
					bind:visibility
					bind:password
					{view}
				/>

				<!-- Inline Share Token Generation Panel -->
				{#if visibility === 'unlisted' || visibility === 'private'}
					<div class="mt-2 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
						<div class="flex items-center gap-2 mb-3">
							{@html icon('link')}
							<h3 class="font-medium text-gray-900 dark:text-white">Generate Share Link</h3>
						</div>
						<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
							{visibility === 'unlisted'
								? 'This view is unlisted — only people with a share link can access it.'
								: 'This view is private — generate a share link for specific access.'}
						</p>

						<!-- Newly Created Token Banner -->
						{#if createdTokenUrl}
							<div class="p-3 mb-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
								<div class="flex items-start justify-between gap-2">
									<div class="flex-1 min-w-0">
										<p class="text-sm font-medium text-green-800 dark:text-green-200 mb-2">
											{@html icon('check')} Share link created!
										</p>
										<p class="text-xs text-green-700 dark:text-green-300 mb-2">
											Copy this link now — for security, the full token won't be shown again.
										</p>
										<div class="flex items-center gap-2">
											<input
												type="text"
												readonly
												value={createdTokenUrl}
												class="flex-1 min-w-0 px-2 py-1 text-sm font-mono bg-white dark:bg-gray-800 border border-green-300 dark:border-green-700 rounded truncate"
											/>
											<button
												type="button"
												class="btn btn-sm btn-secondary shrink-0"
												onclick={() => copyShareUrl(createdTokenUrl || '')}
											>
												{@html icon('copy')} Copy
											</button>
										</div>
									</div>
									<button
										type="button"
										class="p-1 text-green-600 hover:text-green-800 dark:text-green-400 dark:hover:text-green-200"
										onclick={dismissCreatedToken}
									>
										{@html icon('x')}
									</button>
								</div>
							</div>
						{/if}

						<!-- Generate Token Form -->
						{#if showTokenForm}
							<div class="space-y-3 mb-4 p-3 bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700">
								<div>
									<label for="token_name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
										Name (optional)
									</label>
									<input
										type="text"
										id="token_name"
										bind:value={newTokenName}
										placeholder="e.g., Sent to Company X"
										class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600 text-sm"
									/>
									<p class="text-xs text-gray-500 mt-1">A label to help you remember who this link was shared with.</p>
								</div>

								<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
									<div>
										<label for="token_expires" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
											Expiration
										</label>
										<input
											type="datetime-local"
											id="token_expires"
											bind:value={newTokenExpires}
											class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600 text-sm"
										/>
									</div>
									<div>
										<label for="token_max_uses" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
											Max Uses
										</label>
										<input
											type="number"
											id="token_max_uses"
											bind:value={newTokenMaxUses}
											min="0"
											placeholder="0 = unlimited"
											class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600 text-sm"
										/>
									</div>
								</div>

								<div class="flex justify-end gap-2 pt-2">
									<button type="button" class="btn btn-sm btn-ghost" onclick={resetTokenForm}>
										Cancel
									</button>
									<button
										type="button"
										class="btn btn-sm btn-primary"
										onclick={generateShareToken}
										disabled={generatingToken}
									>
										{#if generatingToken}
											Generating...
										{:else}
											Generate Link
										{/if}
									</button>
								</div>
							</div>
						{:else}
							<button
								type="button"
								class="btn btn-primary"
								onclick={() => showTokenForm = true}
							>
								{@html icon('plus')} Generate Share Link
							</button>
						{/if}

						<!-- Existing Tokens for this View -->
						{#if viewTokens.length > 0}
							<div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
								<div class="flex items-center justify-between mb-2">
									<h4 class="text-sm font-medium text-gray-700 dark:text-gray-300">
										Existing Share Links ({viewTokens.length})
									</h4>
									<a href="/admin/tokens" class="text-xs text-primary-600 hover:underline">
										Manage all →
									</a>
								</div>
								<div class="space-y-2">
									{#each viewTokens.slice(0, 3) as token}
										{@const status = getTokenStatusInfo(token)}
										<div class="flex items-center justify-between p-2 bg-white dark:bg-gray-900 rounded border border-gray-200 dark:border-gray-700">
											<div class="flex items-center gap-2 min-w-0">
												<span class="px-1.5 py-0.5 text-xs rounded {status.class}">
													{status.label}
												</span>
												<span class="text-sm text-gray-700 dark:text-gray-300 truncate">
													{token.name || 'Unnamed link'}
												</span>
												{#if token.token_prefix}
													<code class="text-xs text-gray-500 bg-gray-100 dark:bg-gray-700 px-1 rounded">
														{token.token_prefix}...
													</code>
												{/if}
											</div>
											<div class="flex items-center gap-1 shrink-0">
												<span class="text-xs text-gray-500">
													{token.use_count} use{token.use_count !== 1 ? 's' : ''}
												</span>
												{#if token.is_active && !isTokenExpired(token) && !isTokenMaxUsesReached(token)}
													<button
														type="button"
														class="p-1 text-primary-600 hover:text-primary-700 dark:text-primary-400"
														onclick={() => regenerateViewToken(token)}
														title="Regenerate link (revokes old, creates new)"
													>
														<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2v6h-6"></path><path d="M3 12a9 9 0 0 1 15-6.7L21 8"></path><path d="M3 22v-6h6"></path><path d="M21 12a9 9 0 0 1-15 6.7L3 16"></path></svg>
													</button>
													<button
														type="button"
														class="p-1 text-yellow-600 hover:text-yellow-700 dark:text-yellow-400"
														onclick={() => revokeViewToken(token.id)}
														title="Revoke link"
													>
														{@html icon('lock')}
													</button>
												{/if}
											</div>
										</div>
									{/each}
									{#if viewTokens.length > 3}
										<p class="text-xs text-gray-500 text-center">
											+{viewTokens.length - 3} more — <a href="/admin/tokens" class="text-primary-600 hover:underline">view all</a>
										</p>
									{/if}
								</div>
							</div>
						{/if}
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
									onclick={removeHeroImage}
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
								onchange={handleHeroImageChange}
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
					<div>
						<label for="cta_text" class="label">Description</label>
						<input
							type="text"
							id="cta_text"
							bind:value={ctaText}
							class="input"
							placeholder="Ready to work together?"
						/>
						<p class="text-xs text-gray-500 mt-1">Text shown next to the button</p>
					</div>

					<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
						<div>
							<label for="cta_button_text" class="label">Button Label</label>
							<input
								type="text"
								id="cta_button_text"
								bind:value={ctaButtonText}
								class="input"
								placeholder="Get in touch"
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
					<div class="flex flex-wrap items-center gap-3" role="group" aria-label={$t('admin.view_editor.accent_color_label')}>
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

			<!-- Sections -->
			<ViewSectionManager
				bind:sections
				bind:sectionOrder
				bind:sectionItems
				{customContentItems}
				{viewId}
				onOpenOverrideEditor={openOverrideEditor}
			/>

		</form>

			<!-- Danger Zone -->
			<div class="card mt-6 border-red-200 dark:border-red-900">
				<div class="p-4 border-b border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20">
					<h2 class="text-lg font-semibold text-red-700 dark:text-red-400">Danger Zone</h2>
				</div>
				<div class="p-4">
					<div class="flex items-center justify-between">
						<div>
							<h3 class="font-medium text-gray-900 dark:text-white">Delete this facet</h3>
							<p class="text-sm text-gray-500 dark:text-gray-400">
								Permanently delete this facet and all associated share tokens and exports.
							</p>
						</div>
						<button
							type="button"
							class="btn bg-red-600 hover:bg-red-700 text-white disabled:opacity-50"
							onclick={deleteView}
							disabled={deleting}
						>
							{deleting ? 'Deleting...' : 'Delete Facet'}
						</button>
					</div>
				</div>
			</div>
			</div>

			<!-- Preview Pane -->
			{#if showPreview}
				<details class="preview-pane group" open>
					<summary class="lg:hidden flex items-center justify-between p-3 mb-4 bg-gray-100 dark:bg-gray-800 rounded-lg cursor-pointer list-none">
						<span class="font-medium text-gray-900 dark:text-white">Live Preview</span>
						<svg class="w-5 h-5 text-gray-500 transition-transform group-open:rotate-180" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
						</svg>
					</summary>

					<div class="lg:sticky lg:top-4">
						<div class="flex items-center justify-between mb-3 px-1">
							<h2 class="hidden lg:block text-sm font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide">Live Preview</h2>
							<!-- Phase 6.2.2: Desktop/Mobile toggle -->
							<div class="flex items-center gap-1 bg-gray-100 dark:bg-gray-800 rounded-lg p-0.5 ml-auto lg:ml-0">
								<button
									type="button"
									class="px-2 py-1 text-xs rounded-md transition-colors {previewMode === 'desktop' ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'}"
									onclick={() => previewMode = 'desktop'}
									title="Desktop preview"
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
									</svg>
								</button>
								<button
									type="button"
									class="px-2 py-1 text-xs rounded-md transition-colors {previewMode === 'mobile' ? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'}"
									onclick={() => previewMode = 'mobile'}
									title="Mobile preview"
								>
									<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z" />
									</svg>
								</button>
							</div>
						</div>
						<ViewPreview
							{profile}
							{heroHeadline}
							{heroSummary}
							{ctaText}
							{ctaUrl}
							{sections}
							{sectionOrder}
							{sectionItems}
							{accentColor}
							{previewMode}
							{customContentItems}
						/>
					</div>
				</details>
			{/if}
		</div>
	{/if}
</div>

<!-- Override Editor Modal -->
{#if showOverrideEditor && editingOverride}
	<ViewOverrideEditor
		{editingOverride}
		onSave={handleOverrideSave}
		onClose={closeOverrideEditor}
	/>
{/if}

<!-- Generate Resume Modal -->
{#if showGenerateModal}
	<ViewResumeGenerator
		{slug}
		authToken={pb.authStore.token || ''}
		{exports}
		onGenerate={handleResumeGenerate}
		onDeleteExport={deleteExport}
		onClose={() => showGenerateModal = false}
	/>
{/if}

<style>
	.view-editor-container {
		width: 100%;
		max-width: 100%;
		min-width: 0;
	}

	.editor-layout {
		display: flex;
		gap: 1.5rem;
		align-items: flex-start;
	}

	.editor-pane {
		flex: 1;
		min-width: 0;
		max-width: none; /* Expand to full width when preview hidden */
	}

	.editor-layout.with-preview .editor-pane {
		flex: 3;
		max-width: 48rem; /* Constrain when preview is showing */
	}

	.preview-pane {
		flex: 2;
		min-width: 0; /* Allow shrinking on mobile */
		max-width: 480px;
	}

	/* Mobile - stack and constrain */
	@media (max-width: 1024px) {
		.editor-layout {
			flex-direction: column;
		}

		.editor-pane {
			width: 100%;
			max-width: 100%;
			min-width: 0;
		}

		.preview-pane {
			width: 100%;
			max-width: 100%;
			min-width: 0;
			margin-bottom: 1rem;
		}
	}

	/* Large screens - show side by side */
	@media (min-width: 1280px) {
		.preview-pane {
			min-width: 320px; /* Only enforce min-width on large screens */
			max-width: 560px;
		}
	}
</style>
