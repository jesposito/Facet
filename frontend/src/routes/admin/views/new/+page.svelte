<script lang="ts">
	import { preventDefault, createBubbler, stopPropagation } from 'svelte/legacy';

	const bubble = createBubbler();
	import { onDestroy, onMount, tick } from 'svelte';
	import { goto } from '$app/navigation';
	import { browser } from '$app/env';
	import ReorderAnnouncer from '$lib/components/admin/ReorderAnnouncer.svelte';
	import {
		pb,
		type ViewSection,
		type Profile,
		type SectionWidth,
		VALID_LAYOUTS,
		VALID_WIDTHS,
		getValidWidthsForLayout,
		isWidthValidForLayout,
		type ItemSelection
	} from '$lib/pocketbase';
	import {
		allPublicHintKey,
		confirmPrivateItems,
		isAllPublicItem,
		normalizeItemSelection
	} from '$lib/facet-content';
	import { collection } from '$lib/stores/demo';
	import { toasts } from '$lib/stores';
	import { icon } from '$lib/icons';
	import { flip } from 'svelte/animate';
	import ViewPreview from '$components/admin/ViewPreview.svelte';
	import ResumeImportModal from '$components/admin/ResumeImportModal.svelte';
	import { ACCENT_COLORS, ACCENT_COLOR_LIST, type AccentColor } from '$lib/colors';
	import { FONT_PACKS, FONT_PACK_LIST, type FontPack } from '$lib/fonts';
	import { t } from 'svelte-i18n';
	import { brandName } from '$lib/stores/plan';
	import VisibilitySelector from '$components/admin/VisibilitySelector.svelte';
	import { handoffCreatedShareLink, isValidMaxUses, localExpiryToISO } from '$lib/share-links';
	import { copyToClipboard } from '$lib/share';
	import { createListLoader } from '$lib/admin/list-load';
	import { readFacetCatalog } from '$lib/admin/editor-load';
	import { guardUnsaved } from '$lib/admin/unsaved';
	import {
		DraftReadError,
		findDraftRecord,
		mayHaveLanded,
		newRecordId
	} from '$lib/admin/create-record';

	// Import DnD safely - only in browser
	let dndzone: any = $state((node: HTMLElement, params?: any) => ({ destroy: () => {} }));
	let SHADOW_PLACEHOLDER_ITEM_ID = '';

	// Shared screen-reader announcer for keyboard + drag reorder (DRY).
	let reorderAnnouncer = $state<ReorderAnnouncer | undefined>(undefined);

	// Load DnD functionality when in browser
	onMount(async () => {
		if (browser) {
			const { dndzone: dnd, SHADOW_PLACEHOLDER_ITEM_ID: shadowId } =
				await import('svelte-dnd-action');
			dndzone = dnd;
			SHADOW_PLACEHOLDER_ITEM_ID = shadowId;
		}
	});

	// Default section definitions - used to initialize and provide labels
	const SECTION_DEFS: Record<string, { label: string; collection: string }> = {
		experience: { label: 'Experience', collection: 'experience' },
		projects: { label: 'Projects', collection: 'projects' },
		education: { label: 'Education', collection: 'education' },
		certifications: { label: 'Certifications', collection: 'certifications' },
		skills: { label: 'Skills', collection: 'skills' },
		awards: { label: 'Awards', collection: 'awards' },
		posts: { label: 'Posts', collection: 'posts' },
		talks: { label: 'Talks', collection: 'talks' },
		testimonials: { label: 'Testimonials', collection: 'testimonials' },
		contacts: { label: 'Contact Methods', collection: 'contact_methods' }
	};

	// Custom content items (each becomes its own section)
	let customContentItems: Array<{
		id: string;
		title: string;
		is_draft: boolean;
		visibility: 'public' | 'unlisted' | 'private';
	}> = $state([]);

	// Helper to check if a section key is for custom content
	function isCustomSection(sectionKey: string): boolean {
		return sectionKey.startsWith('custom:');
	}

	// Get custom section title
	function getCustomSectionTitle(sectionKey: string): string {
		const customId = sectionKey.replace('custom:', '');
		const item = customContentItems.find((c) => c.id === customId);
		return item ? `Custom: ${item.title}` : `Custom: ${customId.slice(0, 8)}...`;
	}

	// Default section order
	const DEFAULT_SECTION_ORDER = [
		'experience',
		'projects',
		'education',
		'certifications',
		'awards',
		'skills',
		'posts',
		'talks',
		'testimonials',
		'contacts'
	];

	const editorLoad = createListLoader('facet editor');
	let canSave = $derived($editorLoad.loaded && !$editorLoad.failed && !$editorLoad.loading);
	let saving = $state(false);
	let showImportModal = $state(false);
	let hasAiProvider = $state(false);

	// Profile data for preview
	let profile: Profile | null = $state(null);

	// Preview panel state
	let showPreview = $state(true);
	let previewMode: 'desktop' | 'mobile' = $state('desktop'); // Phase 6.2.2

	// Form fields
	let name = $state('');
	let slug = $state('');
	let description = $state('');
	let visibility: 'public' | 'unlisted' | 'private' | 'password' = $state('public');
	let password = $state(''); // For password-protected views
	let heroHeadline = $state('');
	let heroSummary = $state('');
	let heroLocation = $state('');
	let ctaText = $state('');
	let ctaUrl = $state('');
	let ctaButtonText = $state('');
	let ctaEnabled = $state(true); // Per-view CTA toggle (defaults to true)
	let isActive = $state(true);
	let accentColor: AccentColor | null = $state(null); // null = use global profile setting
	let fontPack: FontPack | null = $state(null); // null = use global profile setting
	let heroLayout: string | null = $state(null); // null = use global profile setting

	// Share token generation state (for after view creation)
	let generateTokenAfterCreate = $state(false);
	let newTokenName = $state('');
	let newTokenExpires = $state('');
	let tokenExpiresInput: HTMLInputElement | null = $state(null);
	let newTokenMaxUses = $state(0);
	let createdTokenUrl: string | null = $state(null);
	let generatingToken = $state(false);

	// Sections configuration with layout and width support (itemConfig empty for new views)
	let sections: Record<
		string,
		{
			enabled: boolean;
			items: string[];
			itemSelection?: ItemSelection;
			expanded: boolean;
			layout: string;
			width: SectionWidth;
			itemConfig: Record<string, { overrides?: Record<string, string | string[]> }>;
			categoryOrder?: string[];
			disabledCategories?: string[];
			categoryDisplayModes?: Record<string, string>;
		}
	> = $state({});

	// Section order for drag-drop
	let sectionOrder: Array<{ id: string; key: string }> = $state([]);
	const flipDurationMs = 200;

	// Available items for each section (with full data for preview)
	let sectionItems: Record<
		string,
		Array<{
			id: string;
			label: string;
			visibility: string;
			is_draft?: boolean;
			data: Record<string, unknown>;
		}>
	> = $state({});

	// Names the record this form will create, so a lost response can be looked up, not repeated.
	const draftId = newRecordId();
	// The body of the last create. Once set, a new press looks the record up before creating.
	let createSentBody: string | null = null;
	// An earlier version landed and the owner has been told, so the next press updates it.
	let updateLanded = false;
	let createIssue: '' | 'unknown' | 'differs' = $state('');
	let created = false;
	let left = false;

	// A blank form is the baseline. Sections join it once the content read allows them.
	const savedFields = fieldsSnapshot();
	let savedSections: string | null = null;

	guardUnsaved(
		() =>
			!created &&
			(fieldsSnapshot() !== savedFields ||
				(savedSections !== null && sectionsSnapshot() !== savedSections)),
		() => $t('admin.view_editor.page.unsaved_warning')
	);

	onDestroy(() => {
		left = true;
	});

	// Simple pattern - admin layout handles auth
	onMount(async () => {
		await Promise.all([loadCatalog(), loadProfile(), checkAiProviders()]);
	});

	// Custom content and section items load together, because Create acts on both.
	function loadCatalog() {
		return editorLoad.load(
			(_requestKey, signal) => readFacetCatalog(SECTION_DEFS, signal),
			({ custom, lists }) => {
				customContentItems = custom.map((item) => {
					const record = item as any;
					return {
						id: record.id,
						title: record.title || 'Untitled',
						is_draft: record.is_draft || false,
						visibility: (record.visibility || 'private') as 'public' | 'unlisted' | 'private'
					};
				});
				for (const [key, records] of Object.entries(lists)) {
					sectionItems[key] = records.map((item) => ({
						id: item.id,
						label: getItemLabel(key, item),
						// Contact methods have no visibility field; view_visibility is their scope.
						// Otherwise default to 'private' if not set - safer than assuming public
						visibility: key === 'contacts' ? 'public' : (item.visibility as string) || 'private',
						is_draft: (item.is_draft as boolean) || false,
						data: item
					}));
				}
				if (savedSections === null) {
					initializeSections();
					savedSections = sectionsSnapshot();
				}
			}
		);
	}

	function checkAiProviders() {
		return editorLoad.loadAncillary(
			'ai_providers',
			(requestKey, signal) =>
				collection('ai_providers')
					.getList(1, 1, {
						filter: 'is_active = true',
						requestKey: requestKey('ai_providers'),
						signal
					})
					// A list 404 means the collection isn't there, as in demo mode.
					.catch((err) => {
						if (err?.status === 404) return { totalItems: 0 };
						throw err;
					}),
			(result) => {
				hasAiProvider = result.totalItems > 0;
			}
		);
	}

	function loadProfile() {
		return editorLoad.loadAncillary(
			'profile',
			(requestKey, signal) =>
				collection('profile').getList(1, 1, { requestKey: requestKey('profile'), signal }),
			(records) => {
				if (records.items.length > 0) {
					const record = records.items[0] as unknown as Profile & { collectionId: string };
					if (record.avatar) {
						const avatarUrl = `/api/files/${record.collectionId}/${record.id}/${record.avatar}`;
						profile = { ...record, avatar: avatarUrl };
					} else {
						profile = record;
					}
				}
			}
		);
	}

	function initializeSections() {
		// Start with all sections enabled by default for new views, with default layout and full width
		for (const key of DEFAULT_SECTION_ORDER) {
			const defaultLayout = VALID_LAYOUTS[key]?.default || 'default';
			sections[key] = {
				enabled: true,
				items: [],
				itemSelection: 'all_public',
				expanded: false,
				layout: defaultLayout,
				width: 'full',
				itemConfig: {},
				categoryOrder: undefined,
				disabledCategories: undefined,
				categoryDisplayModes: undefined
			};
		}

		// Add custom content sections (each custom content item is its own section)
		for (const item of customContentItems) {
			const sectionKey = `custom:${item.id}`;
			const defaultLayout = VALID_LAYOUTS['custom']?.default || 'default';
			// Custom content sections start disabled by default for new views
			sections[sectionKey] = {
				enabled: false,
				items: [],
				expanded: false,
				layout: defaultLayout,
				width: 'full',
				itemConfig: {}
			};
		}

		// Initialize section order (standard sections first, then custom content)
		const customSectionKeys = customContentItems.map((item) => `custom:${item.id}`);
		const allSections = [...DEFAULT_SECTION_ORDER, ...customSectionKeys];
		sectionOrder = allSections.map((key) => ({ id: `section-${key}`, key }));
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
				return (item.title as string) || (item.name as string) || (item.id as string);
		}
	}

	// Helper to trigger reactivity by creating a new object reference
	function updateSections() {
		sections = { ...sections };
	}

	function toggleSection(key: string) {
		sections[key].enabled = !sections[key].enabled;
		updateSections();
	}

	function toggleSectionExpand(key: string) {
		sections[key].expanded = !sections[key].expanded;
		updateSections();
	}

	// Picks stay in place across a mode change, so going back restores them.
	function setItemSelection(key: string, mode: ItemSelection) {
		sections[key].itemSelection = mode;
		updateSections();
	}

	function toggleItem(sectionKey: string, itemId: string) {
		const idx = sections[sectionKey].items.indexOf(itemId);
		if (idx === -1) {
			sections[sectionKey].items.push(itemId);
		} else {
			sections[sectionKey].items.splice(idx, 1);
		}
		updateSections();
	}

	// Bulk selection adds public, published items after the current picks, so
	// private items stay a per-item choice and existing order is kept.
	function selectAllItems(sectionKey: string) {
		const selected = new Set(sections[sectionKey].items);
		const additions = (sectionItems[sectionKey] || [])
			.filter((i) => i.visibility === 'public' && !i.is_draft && !selected.has(i.id))
			.map((i) => i.id);
		sections[sectionKey].items = [...sections[sectionKey].items, ...additions];
		updateSections();
	}

	function clearAllItems(sectionKey: string) {
		sections[sectionKey].items = [];
		updateSections();
	}

	function updateSectionWidth(sectionKey: string, width: string) {
		sections[sectionKey].width = width as SectionWidth;
		updateSections();
	}

	function updateSectionLayout(sectionKey: string, layout: string) {
		sections[sectionKey].layout = layout;
		// Auto-reset width to 'full' if current width is not valid for new layout
		const layoutKey = isCustomSection(sectionKey) ? 'custom' : sectionKey;
		if (!isWidthValidForLayout(layoutKey, layout, sections[sectionKey].width)) {
			sections[sectionKey].width = 'full';
		}
		updateSections();
	}

	function generateSlug(value: string): string {
		return value
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 50);
	}

	function handleNameInput() {
		slug = generateSlug(name);
	}

	// Drag-drop handlers for section reordering
	function moveSection(sectionId: string, direction: 'up' | 'down') {
		const idx = sectionOrder.findIndex((s) => s.id === sectionId);
		if (idx === -1) return;
		if (direction === 'up' && idx === 0) return;
		if (direction === 'down' && idx === sectionOrder.length - 1) return;
		const newOrder = [...sectionOrder];
		const targetIdx = direction === 'up' ? idx - 1 : idx + 1;
		[newOrder[idx], newOrder[targetIdx]] = [newOrder[targetIdx], newOrder[idx]];
		sectionOrder = newOrder;
		updateSections();
	}

	function handleSectionDndConsider(e: CustomEvent<{ items: typeof sectionOrder }>) {
		sectionOrder = e.detail.items.filter((item) => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
	}

	function handleSectionDndFinalize(e: CustomEvent<{ items: typeof sectionOrder }>) {
		sectionOrder = e.detail.items.filter((item) => item.id !== SHADOW_PLACEHOLDER_ITEM_ID);
	}

	// Drag-drop handlers for item reordering within a section
	function handleItemDndConsider(
		sectionKey: string,
		e: CustomEvent<{
			items: Array<{
				id: string;
				label: string;
				visibility: string;
				is_draft?: boolean;
				data: Record<string, unknown>;
			}>;
		}>
	) {
		// Only update visual state during consider - don't commit selection changes
		sectionItems[sectionKey] = e.detail.items;
	}

	function handleItemDndFinalize(
		sectionKey: string,
		e: CustomEvent<{
			items: Array<{
				id: string;
				label: string;
				visibility: string;
				is_draft?: boolean;
				data: Record<string, unknown>;
			}>;
			info?: { id?: string };
		}>
	) {
		sectionItems[sectionKey] = e.detail.items;
		// Only commit order changes on finalize (drag complete)
		updateItemsOrderFromDisplay(sectionKey);
		// Announce the dropped item's new position for screen-reader users.
		const droppedId = e.detail.info?.id;
		const list = sectionItems[sectionKey] || [];
		const droppedIdx = list.findIndex((i) => i.id === droppedId);
		if (droppedIdx !== -1) {
			announceItemMove(list[droppedIdx].label, droppedIdx + 1, list.length);
		}
	}

	function updateItemsOrderFromDisplay(sectionKey: string) {
		const displayOrder = sectionItems[sectionKey]?.map((i) => i.id) || [];
		const selectedSet = new Set(sections[sectionKey].items);
		sections[sectionKey].items = displayOrder.filter((id) => selectedSet.has(id));
		updateSections();
	}

	// Announce a reorder change for screen-reader users (position is 1-based).
	function announceItemMove(label: string, position: number, total: number) {
		reorderAnnouncer?.announce(
			$t('admin.view_editor.sections.item_reorder_announce', {
				values: { item: label, position, total }
			})
		);
	}

	// Keyboard-accessible item reordering within a section (complements drag).
	async function moveSectionItem(
		sectionKey: string,
		index: number,
		direction: 'up' | 'down',
		btn: HTMLButtonElement
	) {
		const list = sectionItems[sectionKey] || [];
		const targetIdx = direction === 'up' ? index - 1 : index + 1;
		if (targetIdx < 0 || targetIdx >= list.length) return;
		const moved = list[index];
		const next = [...list];
		[next[index], next[targetIdx]] = [next[targetIdx], next[index]];
		sectionItems[sectionKey] = next;
		updateItemsOrderFromDisplay(sectionKey);
		announceItemMove(moved.label, targetIdx + 1, list.length);

		await tick();
		if (btn.disabled) {
			const opposite = direction === 'up' ? 'down' : 'up';
			btn.parentElement?.querySelector<HTMLButtonElement>(`[data-move="${opposite}"]`)?.focus();
		}
	}

	// Apply imported items to sections after resume import
	async function applyImportedToSections(imported: Record<string, string[]>) {
		// Map from import response keys to section keys (they match in this case)
		const importToSectionMap: Record<string, string> = {
			experience: 'experience',
			education: 'education',
			skills: 'skills',
			certifications: 'certifications',
			projects: 'projects',
			awards: 'awards',
			talks: 'talks'
		};

		// Apply imported items to each section
		for (const [importKey, sectionKey] of Object.entries(importToSectionMap)) {
			const importedIds = imported[importKey];
			if (importedIds && importedIds.length > 0 && sections[sectionKey]) {
				// Enable the section
				sections[sectionKey].enabled = true;
				// Expand it so user can see selections
				sections[sectionKey].expanded = true;
				// Imported items are explicit picks, so this section now shows those.
				sections[sectionKey].itemSelection = 'selected';
				// Add imported IDs to items (union with existing, maintaining order)
				const existingSet = new Set(sections[sectionKey].items);
				for (const id of importedIds) {
					if (!existingSet.has(id)) {
						sections[sectionKey].items.push(id);
					}
				}
			}
		}

		// Select the imported IDs before refreshing the item lists. The refresh
		// makes one request per section, and a Create click during it would
		// otherwise save the facet without the imported items.
		updateSections();
		toasts.add('success', 'Imported items have been selected in your facet.');

		// Refresh section items so the newly imported records show up in the lists
		await loadCatalog();
	}

	// Token generation functions
	function resetTokenForm() {
		newTokenName = '';
		newTokenExpires = '';
		newTokenMaxUses = 0;
		generateTokenAfterCreate = false;
	}

	async function copyShareUrl(text: string) {
		const copied = await copyToClipboard(text);
		toasts.add(
			copied ? 'success' : 'error',
			$t(copied ? 'admin.tokens.toast_copied' : 'admin.tokens.toast_copy_failed')
		);
	}

	function dismissCreatedToken() {
		createdTokenUrl = null;
	}

	async function generateShareToken(viewId: string) {
		const expiresAt = tokenExpiresInput?.validity.badInput
			? null
			: localExpiryToISO(newTokenExpires);
		if (expiresAt === null || !isValidMaxUses(newTokenMaxUses)) return;
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
					expires_at: expiresAt || undefined,
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
			handoffCreatedShareLink(viewId, createdTokenUrl);

			toasts.add('success', 'Share link created!');
			resetTokenForm();
		} catch (err) {
			toasts.add('error', err instanceof Error ? err.message : 'Failed to create share link');
		} finally {
			generatingToken = false;
		}
	}

	// Build sections array in current order with layout and width
	function buildSectionsData(): ViewSection[] {
		return sectionOrder.map(({ key }) => {
			const sectionData: ViewSection = {
				section: key,
				enabled: sections[key]?.enabled || false,
				items: sections[key]?.items || [],
				layout: sections[key]?.layout || VALID_LAYOUTS[key]?.default || 'default',
				width: sections[key]?.width || 'full'
			};
			if (!isCustomSection(key)) {
				sectionData.itemSelection = normalizeItemSelection(sections[key]?.itemSelection);
			}
			// Include skills section settings if set
			if (key === 'skills') {
				if (sections[key]?.categoryOrder?.length) {
					sectionData.categoryOrder = sections[key].categoryOrder;
				}
				if (sections[key]?.disabledCategories?.length) {
					sectionData.disabledCategories = sections[key].disabledCategories;
				}
				if (
					sections[key]?.categoryDisplayModes &&
					Object.keys(sections[key].categoryDisplayModes).length > 0
				) {
					sectionData.categoryDisplayModes = sections[key].categoryDisplayModes;
				}
			}
			return sectionData;
		});
	}

	function sectionsSnapshot() {
		return JSON.stringify(buildSectionsData());
	}

	function viewFields() {
		return {
			name: name.trim(),
			slug: slug.trim(),
			description: description.trim(),
			visibility,
			password: visibility === 'password' ? password.trim() : null,
			hero_headline: heroHeadline.trim() || null,
			hero_summary: heroSummary.trim() || null,
			hero_location: heroLocation.trim() || null,
			cta_text: ctaText.trim() || null,
			cta_url: ctaUrl.trim() || null,
			cta_button_text: ctaButtonText.trim() || null,
			cta_enabled: ctaEnabled,
			is_active: isActive,
			is_default: false, // New views are never default - only the system-created Default view is
			accent_color: accentColor || null,
			font_pack: fontPack || null,
			hero_layout: heroLayout || null
		};
	}

	function fieldsSnapshot() {
		return JSON.stringify({
			...viewFields(),
			generateTokenAfterCreate,
			newTokenName,
			newTokenExpires,
			newTokenMaxUses
		});
	}

	// Creates this form's Facet once. A create that may have landed is found by its ID, so a
	// repeat press can't add a second Facet or trip over its own slug.
	async function createDraftView(data: Record<string, unknown>): Promise<{ id: string } | null> {
		const records = collection('views');
		const body = JSON.stringify(data);
		if (createSentBody) {
			const stored = await findDraftRecord<{ id: string }>(records, draftId);
			if (stored) {
				if (body === createSentBody) return stored;
				if (updateLanded) {
					return records.update(draftId, data, {
						requestKey: null,
						signal: AbortSignal.timeout(30_000)
					});
				}
				updateLanded = true;
				createIssue = 'differs';
				return null;
			}
		}

		createSentBody = body;
		try {
			return await records.create(
				{ ...data, id: draftId },
				{ requestKey: null, signal: AbortSignal.timeout(30_000) }
			);
		} catch (err) {
			if (!mayHaveLanded(err)) {
				createSentBody = null;
				throw err;
			}
			const stored = await findDraftRecord<{ id: string }>(records, draftId);
			if (!stored) throw err;
			return stored;
		}
	}

	async function handleSubmit() {
		if (saving) return;
		if (!canSave) {
			toasts.add('error', $t('admin.view_editor.page.save_paused'));
			return;
		}
		if (!name.trim()) {
			toasts.add('error', 'Name is required');
			return;
		}
		if (!slug.trim()) {
			toasts.add('error', 'Slug is required');
			return;
		}
		if (visibility === 'password' && !password.trim()) {
			toasts.add('error', 'Password is required for password-protected views');
			return;
		}
		if (generateTokenAfterCreate && visibility === 'unlisted' && isActive) {
			if (tokenExpiresInput?.validity.badInput || localExpiryToISO(newTokenExpires) === null) {
				toasts.add('error', $t('admin.tokens.toast_invalid_expiry'));
				return;
			}
			if (!isValidMaxUses(newTokenMaxUses)) {
				toasts.add('error', $t('admin.tokens.toast_invalid_max_uses'));
				return;
			}
		}

		// Check for reserved slugs
		const reservedSlugs = [
			'admin',
			'api',
			's',
			'v',
			'_app',
			'_',
			'assets',
			'static',
			'favicon.ico',
			'robots.txt',
			'sitemap.xml',
			'health',
			'healthz',
			'ready',
			'login',
			'logout',
			'auth',
			'oauth',
			'callback',
			'home',
			'index',
			'default',
			'profile',
			'projects',
			'posts',
			'new'
		];
		if (reservedSlugs.includes(slug.toLowerCase())) {
			toasts.add('error', `"${slug}" is a reserved slug. Please choose another.`);
			return;
		}

		saving = true;
		createIssue = '';
		try {
			const newView = await createDraftView({ ...viewFields(), sections: buildSectionsData() });
			if (!newView) return;
			created = true;

			// The Facet exists from here on. An unconfirmed grant reports a partial
			// save and opens it, so the retry is Save there and never a second Create.
			const grantsConfirmed = await confirmPrivateItems(newView.id);
			if (grantsConfirmed) {
				toasts.add('success', 'View created successfully');
			} else {
				toasts.add('error', $t('admin.view_editor.sections.private_grants_unconfirmed'));
			}

			if (left) return;
			// Generate share token if requested
			if (generateTokenAfterCreate && visibility === 'unlisted' && isActive) {
				await generateShareToken(newView.id);
				// Redirect to edit page so user can see/copy the token
				goto(`/admin/views/${newView.id}`, { replace: true });
			} else if (!grantsConfirmed) {
				goto(`/admin/views/${newView.id}`, { replace: true });
			} else {
				goto('/admin/views', { replace: true });
			}
		} catch (err) {
			if (err instanceof DraftReadError) {
				createIssue = 'unknown';
				return;
			}
			console.error('Failed to create view:', err);
			const message = err instanceof Error ? err.message : 'Failed to create view';
			if (message.includes('slug')) {
				toasts.add('error', 'A view with this slug already exists');
			} else {
				toasts.add('error', message);
			}
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head>
	<title>{$t('admin.views.create_page_title')} | {$brandName}</title>
</svelte:head>

<div class="view-editor-container">
	{#if !$editorLoad.loaded && !$editorLoad.failed}
		<div class="card p-8 text-center max-w-4xl mx-auto">
			<div class="animate-pulse">Loading...</div>
		</div>
	{:else}
		<!-- Header -->
		<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
			<div class="flex items-center gap-4">
				<a
					href="/admin/views"
					class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
					aria-label={$t('admin.views.back_to_views')}
				>
					<svg
						class="w-5 h-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M15 19l-7-7 7-7"
						/>
					</svg>
				</a>
				<h1 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">Create Facet</h1>
			</div>
			<div class="flex items-center gap-2 flex-wrap">
				<!-- Preview Toggle - hidden on mobile -->
				<button
					type="button"
					class="btn btn-ghost hidden lg:flex items-center gap-2"
					onclick={() => (showPreview = !showPreview)}
					title={showPreview ? 'Hide preview' : 'Show preview'}
				>
					<svg
						class="w-5 h-5"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						aria-hidden="true"
					>
						{#if showPreview}
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
							/>
						{:else}
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							/>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
							/>
						{/if}
					</svg>
					<span>{showPreview ? 'Hide' : 'Show'} Preview</span>
				</button>
				{#if hasAiProvider && $editorLoad.loaded}
					<button
						type="button"
						class="btn btn-secondary text-sm flex items-center gap-2"
						onclick={() => (showImportModal = true)}
						title="Import from resume"
					>
						<svg
							class="w-4 h-4"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
							aria-hidden="true"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
							/>
						</svg>
						<span class="hidden sm:inline">Import Resume</span>
						<span class="sm:hidden">Import</span>
					</button>
				{/if}
				<button
					type="button"
					class="btn btn-primary text-sm"
					onclick={handleSubmit}
					disabled={saving || !canSave}
				>
					{#if saving}
						<svg class="animate-spin -ml-1 mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24">
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
					{/if}
					Create
				</button>
			</div>
		</div>

		{#if $editorLoad.failed}
			<div role="alert" class="card p-4 mb-4 border-red-300 dark:border-red-700">
				<p class="text-red-700 dark:text-red-300">
					{$t(
						$editorLoad.loaded
							? 'admin.view_editor.page.content_refresh_failed'
							: 'admin.view_editor.page.content_load_failed'
					)}
				</p>
				<button
					type="button"
					class="btn btn-secondary mt-3"
					onclick={loadCatalog}
					disabled={$editorLoad.loading}
				>
					{$t('admin.dashboard.retry')}
				</button>
			</div>
		{/if}
		{#if createIssue}
			<div role="alert" class="card p-4 mb-4 border-red-300 dark:border-red-700">
				<p class="text-red-700 dark:text-red-300">
					{$t(`admin.view_editor.page.create_${createIssue}`)}
				</p>
			</div>
		{/if}
		{#if $editorLoad.ancillaryFailed}
			<p role="status" class="mb-2 text-sm text-gray-700 dark:text-gray-300">
				{$t('admin.view_editor.page.supporting_failed')}
			</p>
			<button
				type="button"
				class="btn btn-secondary btn-sm mb-4"
				onclick={editorLoad.retryAncillary}
			>
				{$t('admin.list_status.retry_supporting')}
			</button>
		{/if}

		<!-- Split Pane Layout -->
		<div class="editor-layout" class:with-preview={showPreview}>
			<!-- Editor Pane -->
			<div class="editor-pane">
				<form onsubmit={preventDefault(handleSubmit)} class="space-y-4 sm:space-y-6">
					<fieldset disabled={saving} class="contents">
						<!-- Basic Info -->
						<div class="card p-4 sm:p-6 space-y-4">
							<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Basic Information</h2>

							<div>
								<label for="name" class="label">Name *</label>
								<input
									type="text"
									id="name"
									bind:value={name}
									oninput={handleNameInput}
									class="input"
									placeholder="Recruiter View"
									required
								/>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
									Internal name for this view
								</p>
							</div>

							<div>
								<label for="slug" class="label">URL Slug *</label>
								<div class="flex items-center gap-2">
									<span class="text-gray-500 dark:text-gray-400 text-sm">/</span>
									<input
										type="text"
										id="slug"
										bind:value={slug}
										class="input flex-1"
										placeholder="recruiter"
										required
									/>
								</div>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
									Public URL will be: /{slug}
								</p>
							</div>

							<div>
								<label for="description" class="label">Description</label>
								<textarea
									id="description"
									bind:value={description}
									class="input min-h-[80px]"
									placeholder="Internal notes about this view..."></textarea>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
									Private notes (not shown publicly)
								</p>
							</div>

							<VisibilitySelector bind:value={visibility} includePassword />

							{#if visibility === 'password'}
								<div>
									<label for="password" class="label">Password *</label>
									<input
										type="password"
										id="password"
										bind:value={password}
										class="input"
										placeholder="Enter password for this view"
										required
										autocomplete="new-password"
									/>
									<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
										Visitors will need this password to access this view
									</p>
								</div>
							{/if}

							<!-- Inline Share Token Generation Panel -->
							{#if visibility === 'unlisted' && isActive}
								<div
									class="mt-2 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
								>
									<div class="flex items-center gap-2 mb-3">
										{@html icon('link')}
										<h3 class="font-medium text-gray-900 dark:text-white">Generate Share Link</h3>
									</div>
									<p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
										{$t('admin.wizard.visibility_unlisted')}
									</p>

									<!-- Token Generation Form -->
									<div
										class="space-y-3 p-3 bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700"
									>
										<label class="flex items-start gap-3">
											<input
												type="checkbox"
												bind:checked={generateTokenAfterCreate}
												class="mt-1 w-4 h-4 text-primary-600 rounded border-gray-300"
											/>
											<div>
												<span class="text-sm font-medium text-gray-700 dark:text-gray-300"
													>Generate share link when creating this view</span
												>
												<p class="text-xs text-gray-500 dark:text-gray-400 mt-0.5">
													You'll be redirected to the view editor to copy the link
												</p>
											</div>
										</label>

										{#if generateTokenAfterCreate}
											<div class="pt-2 space-y-3 border-t border-gray-200 dark:border-gray-700">
												<div>
													<label
														for="token_name"
														class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
													>
														Link Name (optional)
													</label>
													<input
														type="text"
														id="token_name"
														bind:value={newTokenName}
														placeholder="e.g., Sent to Company X"
														class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600 text-sm"
													/>
													<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
														A label to help you remember who this link was shared with.
													</p>
												</div>

												<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
													<div>
														<label
															for="token_expires"
															class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
														>
															Expiration
														</label>
														<input
															type="datetime-local"
															bind:this={tokenExpiresInput}
															id="token_expires"
															bind:value={newTokenExpires}
															class="w-full px-3 py-2 border rounded-lg dark:bg-gray-800 dark:border-gray-600 text-sm"
														/>
													</div>
													<div>
														<label
															for="token_max_uses"
															class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
														>
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
											</div>
										{/if}
									</div>

									<p class="text-xs text-gray-500 dark:text-gray-400 mt-3">
										{@html icon('info')} You can also generate additional share links from the view editor
										after creation.
									</p>
								</div>
							{/if}
						</div>

						<!-- Hero Overrides -->
						<div class="card p-4 sm:p-6 space-y-4">
							<h2 class="text-lg font-semibold text-gray-900 dark:text-white">Hero Overrides</h2>
							<p class="text-sm text-gray-500 dark:text-gray-400 -mt-2">
								Override your profile headline, summary, and location for this view
							</p>

							<div>
								<label for="hero_headline" class="label">Custom Headline</label>
								<input
									type="text"
									id="hero_headline"
									bind:value={heroHeadline}
									class="input"
									placeholder="Leave empty to use profile headline"
								/>
							</div>

							<div>
								<label for="hero_summary" class="label">Custom Summary</label>
								<textarea
									id="hero_summary"
									bind:value={heroSummary}
									class="input min-h-[120px]"
									placeholder="Leave empty to use profile summary (Markdown supported)"></textarea>
							</div>

							<div>
								<label for="hero_location" class="label">Custom Location</label>
								<input
									type="text"
									id="hero_location"
									bind:value={heroLocation}
									class="input"
									placeholder="e.g., Wellington, NZ | US Citizen | W-2 or EOR-ready"
								/>
							</div>
						</div>

						<!-- Call to Action -->
						<div class="card p-4 sm:p-6 space-y-4">
							<div class="flex items-start justify-between gap-4">
								<div>
									<h2
										id="view-cta-label"
										class="text-lg font-semibold text-gray-900 dark:text-white"
									>
										Call to Action
									</h2>
									<p class="text-sm text-gray-500 dark:text-gray-400">
										Add a prominent button to this view
									</p>
								</div>
								<label class="relative inline-flex items-center cursor-pointer">
									<input
										type="checkbox"
										class="sr-only peer"
										bind:checked={ctaEnabled}
										aria-labelledby="view-cta-label"
									/>
									<div
										class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary-300 dark:peer-focus:ring-primary-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-primary-600"
									></div>
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
									<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
										Text shown next to the button
									</p>
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
								<div
									class="flex flex-wrap items-center gap-3"
									role="group"
									aria-label={$t('admin.view_editor.aria.accent_color_label')}
								>
									<!-- Use Global Option -->
									<button
										type="button"
										class="flex items-center gap-2 px-3 py-2 rounded-lg border transition-all
								{accentColor === null
											? 'border-gray-900 dark:border-white bg-gray-100 dark:bg-gray-800'
											: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
										onclick={() => (accentColor = null)}
									>
										<div
											class="w-5 h-5 rounded-full bg-gradient-to-r from-primary-400 to-primary-600 border-2 border-white shadow-sm"
										></div>
										<span class="text-sm font-medium text-gray-700 dark:text-gray-300"
											>Use global</span
										>
										{#if accentColor === null}
											<svg
												class="w-4 h-4 text-gray-900 dark:text-white"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="3"
											>
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
											onclick={() => (accentColor = color)}
											title="{colorInfo.label} - {colorInfo.description}"
										>
											<div
												class="w-10 h-10 rounded-lg transition-all duration-200 ring-offset-2 ring-offset-white dark:ring-offset-gray-900
										{accentColor === color ? 'ring-2 ring-gray-900 dark:ring-white scale-110' : 'hover:scale-105'}"
												style="background-color: {colorInfo.scale[500]}"
											>
												{#if accentColor === color}
													<div class="absolute inset-0 flex items-center justify-center">
														<svg
															class="w-4 h-4 text-white"
															fill="none"
															viewBox="0 0 24 24"
															stroke="currentColor"
															stroke-width="3"
														>
															<path
																stroke-linecap="round"
																stroke-linejoin="round"
																d="M5 13l4 4L19 7"
															/>
														</svg>
													</div>
												{/if}
											</div>
										</button>
									{/each}
								</div>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
									{#if accentColor}
										Using <strong>{ACCENT_COLORS[accentColor].label}</strong> for this view
									{:else}
										Inherits from global profile setting
									{/if}
								</p>
							</div>

							<!-- Font Pack Override -->
							<div class="pt-4">
								<span class="label mb-3 block">Typography</span>
								<div
									class="flex flex-wrap items-center gap-2"
									role="group"
									aria-label="Font pack selection"
								>
									<button
										type="button"
										class="flex items-center gap-2 px-3 py-2 rounded-lg border transition-all text-sm
								{fontPack === null
											? 'border-gray-900 dark:border-white bg-gray-100 dark:bg-gray-800'
											: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
										onclick={() => (fontPack = null)}
									>
										<span class="font-medium text-gray-700 dark:text-gray-300">Global</span>
										{#if fontPack === null}
											<svg
												class="w-4 h-4 text-gray-900 dark:text-white"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="3"
											>
												<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
											</svg>
										{/if}
									</button>

									{#each FONT_PACK_LIST as pack}
										{@const packInfo = FONT_PACKS[pack]}
										<button
											type="button"
											class="px-3 py-2 rounded-lg border transition-all text-sm
									{fontPack === pack
												? 'border-gray-900 dark:border-white bg-gray-100 dark:bg-gray-800 font-medium'
												: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
											onclick={() => (fontPack = pack)}
											title={packInfo.description}
											aria-pressed={fontPack === pack}
										>
											{packInfo.label}
										</button>
									{/each}
								</div>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
									{#if fontPack}
										Using <strong>{FONT_PACKS[fontPack].label}</strong> for this view
									{:else}
										Inherits typography from global profile settings
									{/if}
								</p>
							</div>

							<!-- Hero Layout Override -->
							<div class="pt-4">
								<span class="label mb-3 block">Hero Layout</span>
								<div
									class="flex flex-wrap items-center gap-2"
									role="group"
									aria-label="Hero layout selection"
								>
									<button
										type="button"
										class="flex items-center gap-2 px-3 py-2 rounded-lg border transition-all text-sm
								{heroLayout === null
											? 'border-gray-900 dark:border-white bg-gray-100 dark:bg-gray-800'
											: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
										onclick={() => (heroLayout = null)}
									>
										<span class="font-medium text-gray-700 dark:text-gray-300">Global</span>
										{#if heroLayout === null}
											<svg
												class="w-4 h-4 text-gray-900 dark:text-white"
												fill="none"
												viewBox="0 0 24 24"
												stroke="currentColor"
												stroke-width="3"
											>
												<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
											</svg>
										{/if}
									</button>

									{#each [{ id: 'standard', label: 'Image & Gradient' }, { id: 'centered', label: 'Centered' }, { id: 'split', label: 'Split' }, { id: 'minimal', label: 'Minimal' }, { id: 'stacked', label: 'Stacked' }, { id: 'rail', label: 'Rail' }] as layoutOption}
										<button
											type="button"
											class="px-3 py-2 rounded-lg border transition-all text-sm
									{heroLayout === layoutOption.id
												? 'border-gray-900 dark:border-white bg-gray-100 dark:bg-gray-800 font-medium'
												: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'}"
											onclick={() => (heroLayout = layoutOption.id)}
											aria-pressed={heroLayout === layoutOption.id}
										>
											{layoutOption.label}
										</button>
									{/each}
								</div>
								<p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
									{#if heroLayout}
										Using <strong>{heroLayout}</strong> layout for this view
									{:else}
										Inherits hero layout from global profile settings
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
										<p class="text-xs text-gray-500 dark:text-gray-400">
											Inactive views are not accessible publicly
										</p>
									</div>
								</label>
							</div>
						</div>

						<!-- Sections -->
						<div class="card p-4 sm:p-6 space-y-4">
							<!-- Polite live region for keyboard + drag reorder announcements -->
							<ReorderAnnouncer bind:this={reorderAnnouncer} />
							<div class="flex items-center justify-between">
								<div>
									<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
										Content Sections
									</h2>
									<p class="text-sm text-gray-500 dark:text-gray-400">
										Choose which sections to show and drag to reorder.
									</p>
								</div>
							</div>

							<div
								class="space-y-3"
								use:dndzone={{ items: sectionOrder, flipDurationMs, type: 'sections' }}
								onconsider={handleSectionDndConsider}
								onfinalize={handleSectionDndFinalize}
							>
								{#each sectionOrder as sectionItem (sectionItem.id)}
									{@const sectionIdx = sectionOrder.findIndex((s) => s.id === sectionItem.id)}
									{@const sectionKey = sectionItem.key}
									{@const isCustom = isCustomSection(sectionKey)}
									{@const sectionDef = SECTION_DEFS[sectionKey]}
									{@const sectionLabel = isCustom
										? getCustomSectionTitle(sectionKey)
										: sectionDef?.label || sectionKey}
									{@const layoutKey = isCustom ? 'custom' : sectionKey}
									{@const sectionConfig = sections[sectionKey] || {
										enabled: false,
										items: [],
										expanded: false
									}}
									{@const items = sectionItems[sectionKey] || []}
									{@const publicItems = items.filter(
										(i) => i.visibility !== 'private' && !i.is_draft
									)}
									{@const publicItemIds = new Set(publicItems.map((i) => i.id))}
									{@const selectedPublicCount = sectionConfig.items.filter((id) =>
										publicItemIds.has(id)
									).length}
									{@const isAllPublic = sections[sectionKey]?.itemSelection === 'all_public'}
									{@const allPublicCount = items.filter((i) =>
										isAllPublicItem(sectionKey, i)
									).length}

									<div
										class="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden bg-white dark:bg-gray-900"
										animate:flip={{ duration: flipDurationMs }}
									>
										<!-- Section Header -->
										<div
											class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-800/50"
										>
											<div class="flex items-center gap-3">
												<!-- Drag Handle + Keyboard reorder -->
												<div class="flex items-center gap-1">
													<div
														class="cursor-grab active:cursor-grabbing p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
														title="Drag to reorder"
													>
														<svg
															class="w-5 h-5 text-gray-600 dark:text-gray-300"
															fill="none"
															viewBox="0 0 24 24"
															stroke="currentColor"
															stroke-width="2"
															aria-hidden="true"
														>
															<path
																stroke-linecap="round"
																stroke-linejoin="round"
																d="M4 8h16M4 16h16"
															/>
														</svg>
													</div>
													<button
														type="button"
														disabled={sectionIdx === 0}
														onclick={() => moveSection(sectionItem.id, 'up')}
														aria-label={$t('admin.view_editor.sections.move_up', {
															values: { section: sectionLabel }
														})}
														class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed"
													>
														<svg
															class="w-4 h-4 text-gray-600 dark:text-gray-300"
															fill="none"
															viewBox="0 0 24 24"
															stroke="currentColor"
															stroke-width="2"
															><path
																stroke-linecap="round"
																stroke-linejoin="round"
																d="M5 15l7-7 7 7"
															/></svg
														>
													</button>
													<button
														type="button"
														disabled={sectionIdx === sectionOrder.length - 1}
														onclick={() => moveSection(sectionItem.id, 'down')}
														aria-label={$t('admin.view_editor.sections.move_down', {
															values: { section: sectionLabel }
														})}
														class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 disabled:opacity-30 disabled:cursor-not-allowed"
													>
														<svg
															class="w-4 h-4 text-gray-600 dark:text-gray-300"
															fill="none"
															viewBox="0 0 24 24"
															stroke="currentColor"
															stroke-width="2"
															><path
																stroke-linecap="round"
																stroke-linejoin="round"
																d="M19 9l-7 7-7-7"
															/></svg
														>
													</button>
												</div>
												<button
													type="button"
													class="w-10 h-6 rounded-full transition-colors relative
										{sectionConfig.enabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-gray-600'}"
													onclick={() => toggleSection(sectionKey)}
													aria-label={$t('admin.homepage.toggle_section', {
														values: { section: sectionLabel }
													})}
												>
													<span
														class="absolute top-1 w-4 h-4 bg-white rounded-full transition-transform shadow-sm
											{sectionConfig.enabled ? 'left-5' : 'left-1'}"
													></span>
												</button>
												<span class="font-medium text-gray-900 dark:text-white">{sectionLabel}</span
												>
												<span class="text-xs text-gray-500 dark:text-gray-400">
													{#if isAllPublic}
														{$t('admin.view_editor.sections.all_items', {
															values: { count: allPublicCount }
														})}
													{:else if selectedPublicCount > 0}
														{selectedPublicCount} selected
													{:else if sectionConfig.enabled}
														{$t('admin.view_editor.sections.none_selected')}
													{:else}
														{publicItems.length} available
													{/if}
												</span>
											</div>

											<div class="flex items-center gap-2">
												<!-- Width Selector with visual indicator -->
												{#if sectionConfig.enabled}
													{@const validWidths = getValidWidthsForLayout(
														layoutKey,
														sectionConfig.layout
													)}
													{#if validWidths.length > 1}
														<div
															class="flex items-center gap-1"
															title="Section width - controls side-by-side layout"
														>
															<!-- Width icon indicator -->
															<div class="flex gap-0.5">
																{#if sectionConfig.width === 'half'}
																	<div class="w-2 h-4 bg-primary-500 rounded-sm"></div>
																	<div
																		class="w-2 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"
																	></div>
																{:else if sectionConfig.width === 'third'}
																	<div class="w-1.5 h-4 bg-primary-500 rounded-sm"></div>
																	<div
																		class="w-1.5 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"
																	></div>
																	<div
																		class="w-1.5 h-4 bg-gray-300 dark:bg-gray-600 rounded-sm"
																	></div>
																{:else}
																	<div class="w-5 h-4 bg-primary-500 rounded-sm"></div>
																{/if}
															</div>
															<select
																class="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
																value={sectionConfig.width}
																onchange={(e) =>
																	updateSectionWidth(sectionKey, e.currentTarget.value)}
																onclick={stopPropagation(bubble('click'))}
																aria-label={$t('admin.view_editor.sections.section_width_name', {
																	values: { section: sectionLabel }
																})}
															>
																{#each validWidths as widthOption}
																	<option value={widthOption.value}>{widthOption.label}</option>
																{/each}
															</select>
														</div>
													{/if}
												{/if}

												<!-- Layout Selector -->
												{#if sectionConfig.enabled && VALID_LAYOUTS[layoutKey]}
													{@const layoutConfig = VALID_LAYOUTS[layoutKey]}
													<select
														class="text-xs border border-gray-300 dark:border-gray-600 rounded px-2 py-1 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
														value={sectionConfig.layout}
														onchange={(e) => updateSectionLayout(sectionKey, e.currentTarget.value)}
														onclick={stopPropagation(bubble('click'))}
														title="Section layout"
														aria-label={$t('admin.view_editor.sections.section_layout_name', {
															values: { section: sectionLabel }
														})}
													>
														{#each layoutConfig.layouts as layoutOption}
															<option value={layoutOption}
																>{layoutConfig.labels[layoutOption] || layoutOption}</option
															>
														{/each}
													</select>
												{/if}

												{#if sectionConfig.enabled && items.length > 0 && !isAllPublic}
													<button
														type="button"
														class="p-1 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
														onclick={() => toggleSectionExpand(sectionKey)}
														aria-label={$t(
															sectionConfig.expanded
																? 'admin.homepage.collapse_section'
																: 'admin.homepage.expand_section',
															{ values: { section: sectionLabel } }
														)}
													>
														<svg
															class="w-5 h-5 transition-transform {sectionConfig.expanded
																? 'rotate-180'
																: ''}"
															fill="none"
															viewBox="0 0 24 24"
															stroke="currentColor"
															aria-hidden="true"
														>
															<path
																stroke-linecap="round"
																stroke-linejoin="round"
																stroke-width="2"
																d="M19 9l-7 7-7-7"
															/>
														</svg>
													</button>
												{/if}
											</div>
										</div>

										{#if sectionConfig.enabled && !isCustom}
											<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
											<fieldset
												class="px-3 py-2 border-t border-gray-200 dark:border-gray-700"
												aria-describedby="item-selection-hint-{sectionKey}"
												onpointerdown={(e) => e.stopPropagation()}
												onmousedown={(e) => e.stopPropagation()}
												ontouchstart={(e) => e.stopPropagation()}
											>
												<legend class="sr-only">
													{$t('admin.view_editor.sections.content_mode_legend', {
														values: { section: sectionLabel }
													})}
												</legend>
												<div class="flex flex-wrap gap-x-4 gap-y-1">
													<label
														class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"
													>
														<input
															type="radio"
															name="item-selection-{sectionKey}"
															checked={isAllPublic}
															onchange={() => setItemSelection(sectionKey, 'all_public')}
															class="w-4 h-4 text-primary-600 border-gray-300"
														/>
														{$t('admin.view_editor.sections.content_mode_all_public')}
													</label>
													<label
														class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300"
													>
														<input
															type="radio"
															name="item-selection-{sectionKey}"
															checked={!isAllPublic}
															onchange={() => setItemSelection(sectionKey, 'selected')}
															class="w-4 h-4 text-primary-600 border-gray-300"
														/>
														{$t('admin.view_editor.sections.content_mode_selected')}
													</label>
												</div>
												<p
													id="item-selection-hint-{sectionKey}"
													class="mt-1 text-xs text-gray-500 dark:text-gray-400"
												>
													{#if isAllPublic}
														{$t(allPublicHintKey(sectionKey))}
													{:else if sectionConfig.items.length === 0}
														{$t('admin.view_editor.sections.selected_empty_hint')}
													{:else}
														{$t('admin.view_editor.sections.choose_items_hint')}
													{/if}
												</p>
											</fieldset>
										{/if}

										<!-- Section Items -->
										{#if sectionConfig.enabled && sectionConfig.expanded && items.length > 0 && !isAllPublic}
											<div class="p-3 border-t border-gray-200 dark:border-gray-700">
												<div class="flex items-center justify-between mb-2">
													<p class="text-xs text-gray-500 dark:text-gray-400">
														{sectionConfig.items.length === 0
															? $t('admin.view_editor.sections.choose_items_hint')
															: `${selectedPublicCount} of ${publicItems.length} items selected. Drag to reorder.`}
													</p>
													<div class="flex gap-2">
														<button
															type="button"
															class="text-xs text-primary-600 hover:underline"
															onclick={() => selectAllItems(sectionKey)}
														>
															Select All
														</button>
														<button
															type="button"
															class="text-xs text-gray-500 hover:underline"
															onclick={() => clearAllItems(sectionKey)}
														>
															Clear
														</button>
													</div>
												</div>

												<div
													class="space-y-1 max-h-48 overflow-y-auto"
													use:dndzone={{
														items: sectionItems[sectionKey] || [],
														flipDurationMs,
														type: `items-${sectionKey}`
													}}
													onconsider={(e: any) => handleItemDndConsider(sectionKey, e)}
													onfinalize={(e: any) => handleItemDndFinalize(sectionKey, e)}
												>
													{#each items as item, itemIdx (item.id)}
														{@const isSelected = sectionConfig.items.includes(item.id)}
														<div
															class="flex items-center gap-2 p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-800 bg-white dark:bg-gray-900"
															animate:flip={{ duration: flipDurationMs }}
														>
															<!-- Drag Handle for Items -->
															<div
																class="cursor-grab active:cursor-grabbing p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
																title="Drag to reorder"
																role="presentation"
																onclick={(e) => e.stopPropagation()}
															>
																<svg
																	class="w-4 h-4 text-gray-500 dark:text-gray-400"
																	fill="none"
																	viewBox="0 0 24 24"
																	stroke="currentColor"
																	stroke-width="2"
																	aria-hidden="true"
																>
																	<path
																		stroke-linecap="round"
																		stroke-linejoin="round"
																		d="M4 8h16M4 16h16"
																	/>
																</svg>
															</div>
															<!-- Keyboard-accessible reorder controls (complement drag) -->
															<div class="flex flex-col flex-shrink-0">
																<button
																	type="button"
																	data-move="up"
																	class="p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed"
																	disabled={itemIdx === 0}
																	onclick={(e) =>
																		moveSectionItem(sectionKey, itemIdx, 'up', e.currentTarget)}
																	aria-label={$t('admin.view_editor.sections.item_move_up', {
																		values: { item: item.label }
																	})}
																>
																	<svg
																		class="w-3 h-3"
																		fill="none"
																		viewBox="0 0 24 24"
																		stroke="currentColor"
																		stroke-width="2.5"
																		aria-hidden="true"
																	>
																		<path
																			stroke-linecap="round"
																			stroke-linejoin="round"
																			d="M5 15l7-7 7 7"
																		/>
																	</svg>
																</button>
																<button
																	type="button"
																	data-move="down"
																	class="p-0.5 rounded hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 disabled:opacity-30 disabled:cursor-not-allowed"
																	disabled={itemIdx === items.length - 1}
																	onclick={(e) =>
																		moveSectionItem(sectionKey, itemIdx, 'down', e.currentTarget)}
																	aria-label={$t('admin.view_editor.sections.item_move_down', {
																		values: { item: item.label }
																	})}
																>
																	<svg
																		class="w-3 h-3"
																		fill="none"
																		viewBox="0 0 24 24"
																		stroke="currentColor"
																		stroke-width="2.5"
																		aria-hidden="true"
																	>
																		<path
																			stroke-linecap="round"
																			stroke-linejoin="round"
																			d="M19 9l-7 7-7-7"
																		/>
																	</svg>
																</button>
															</div>
															<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
															<label
																class="flex items-center gap-2 flex-1 cursor-pointer"
																onpointerdown={(e) => e.stopPropagation()}
																onmousedown={(e) => e.stopPropagation()}
															>
																<input
																	type="checkbox"
																	checked={isSelected}
																	onchange={() => toggleItem(sectionKey, item.id)}
																	class="w-4 h-4 text-primary-600 rounded border-gray-300"
																/>
																<span
																	class="flex-1 text-sm text-gray-700 dark:text-gray-300 truncate"
																>
																	{item.label}
																</span>
															</label>
															{#if item.visibility !== 'public'}
																<span
																	class="px-1.5 py-0.5 text-xs bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300 rounded"
																>
																	{item.visibility}
																</span>
															{/if}
															{#if item.is_draft}
																<span
																	class="px-1.5 py-0.5 text-xs bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded"
																>
																	draft
																</span>
															{/if}
														</div>
													{/each}
												</div>
											</div>
										{/if}
									</div>
								{/each}
							</div>
						</div>
					</fieldset>
				</form>
			</div>

			<!-- Preview Pane -->
			{#if showPreview && $editorLoad.loaded}
				<div class="preview-pane">
					<div class="sticky top-4">
						<div class="flex items-center justify-between mb-3 px-1">
							<h2
								class="text-sm font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wide"
							>
								Live Preview
							</h2>
							<!-- Phase 6.2.2: Desktop/Mobile toggle -->
							<div class="flex items-center gap-1 bg-gray-100 dark:bg-gray-800 rounded-lg p-0.5">
								<button
									type="button"
									class="px-2 py-1 text-xs rounded-md transition-colors {previewMode === 'desktop'
										? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
										: 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'}"
									onclick={() => (previewMode = 'desktop')}
									title="Desktop preview"
								>
									<svg
										class="w-4 h-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										aria-hidden="true"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
										/>
									</svg>
								</button>
								<button
									type="button"
									class="px-2 py-1 text-xs rounded-md transition-colors {previewMode === 'mobile'
										? 'bg-white dark:bg-gray-700 text-gray-900 dark:text-white shadow-sm'
										: 'text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300'}"
									onclick={() => (previewMode = 'mobile')}
									title="Mobile preview"
								>
									<svg
										class="w-4 h-4"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
										aria-hidden="true"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M12 18h.01M8 21h8a2 2 0 002-2V5a2 2 0 00-2-2H8a2 2 0 00-2 2v14a2 2 0 002 2z"
										/>
									</svg>
								</button>
							</div>
						</div>
						<ViewPreview
							{profile}
							{heroLayout}
							{heroHeadline}
							{heroSummary}
							{ctaText}
							{ctaUrl}
							{sections}
							{sectionOrder}
							{sectionItems}
							{previewMode}
							privateSelectionPublishes
						/>
					</div>
				</div>
			{/if}
		</div>
	{/if}
</div>

<!-- Resume Import Modal -->
{#if showImportModal}
	<ResumeImportModal
		onSuccess={(imported) => {
			showImportModal = false;
			applyImportedToSections(imported);
		}}
		onClose={() => (showImportModal = false)}
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
		max-width: 48rem;
	}

	.editor-layout.with-preview .editor-pane {
		flex: 3;
		max-width: none;
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
