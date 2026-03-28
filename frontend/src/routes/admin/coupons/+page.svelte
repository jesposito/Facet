<script lang="ts">
	import { onMount } from 'svelte';
	import { t } from 'svelte-i18n';
	import { pb } from '$lib/pocketbase';
	import { toasts, confirm } from '$lib/stores';
	import { brandName } from '$lib/stores/plan';

	type Coupon = {
		id: string;
		code: string;
		discount_type: 'percentage' | 'fixed';
		discount_value: number;
		applicable_type: 'all' | 'courses' | 'posts' | 'projects';
		applicable_ids: string[];
		max_uses: number;
		current_uses: number;
		expires_at: string;
		active: boolean;
		created: string;
		updated: string;
	};

	let coupons: Coupon[] = $state([]);
	let loading = $state(true);
	let showForm = $state(false);
	let editingCoupon: Coupon | null = $state(null);
	let saving = $state(false);

	// Form state
	let code = $state('');
	let discountType: 'percentage' | 'fixed' = $state('percentage');
	let discountValue = $state(10);
	let appliesTo: 'all' | 'courses' | 'posts' | 'projects' = $state('all');
	let contentIds = $state('');
	let maxUses = $state(0);
	let expiresAt = $state('');
	let isActive = $state(true);

	onMount(async () => {
		await loadCoupons();
	});

	async function loadCoupons() {
		loading = true;
		try {
			const response = await fetch('/api/admin/coupons', {
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (!response.ok) throw new Error('Failed to load coupons');
			const data = await response.json();
			coupons = data.coupons || [];
		} catch (err) {
			console.error('Failed to load coupons:', err);
			toasts.add('error', $t('admin.coupons.error_load'));
		} finally {
			loading = false;
		}
	}

	function resetForm() {
		code = '';
		discountType = 'percentage';
		discountValue = 10;
		appliesTo = 'all';
		contentIds = '';
		maxUses = 0;
		expiresAt = '';
		isActive = true;
		editingCoupon = null;
	}

	function openCreateForm() {
		resetForm();
		showForm = true;
	}

	function openEditForm(coupon: Coupon) {
		editingCoupon = coupon;
		code = coupon.code;
		discountType = coupon.discount_type;
		discountValue = coupon.discount_value;
		appliesTo = coupon.applicable_type;
		contentIds = (coupon.applicable_ids || []).join(', ');
		maxUses = coupon.max_uses;
		expiresAt = coupon.expires_at ? coupon.expires_at.split('T')[0] : '';
		isActive = coupon.active;
		showForm = true;
	}

	function closeForm() {
		showForm = false;
		resetForm();
	}

	function generateCode() {
		const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
		const suffix = Array.from({ length: 4 }, () => chars[Math.floor(Math.random() * chars.length)]).join('');
		code = `SAVE${suffix}`;
	}

	async function saveCoupon() {
		if (!code.trim()) {
			toasts.add('error', $t('admin.coupons.error_code_required'));
			return;
		}

		saving = true;
		try {
			const payload = {
				code: code.trim().toUpperCase(),
				discount_type: discountType,
				discount_value: discountValue,
				applicable_type: appliesTo,
				applicable_ids: contentIds.trim() ? contentIds.split(',').map(s => s.trim()).filter(Boolean) : [],
				max_uses: maxUses,
				expires_at: expiresAt || null,
				active: isActive
			};

			const url = editingCoupon
				? `/api/admin/coupons/${editingCoupon.id}`
				: '/api/admin/coupons';
			const method = editingCoupon ? 'PATCH' : 'POST';

			const response = await fetch(url, {
				method,
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify(payload)
			});

			if (!response.ok) {
				const data = await response.json();
				throw new Error(data.error || 'Failed to save coupon');
			}

			toasts.add('success', editingCoupon ? $t('admin.coupons.updated') : $t('admin.coupons.created'));
			closeForm();
			await loadCoupons();
		} catch (err) {
			console.error('Failed to save coupon:', err);
			toasts.add('error', err instanceof Error ? err.message : $t('admin.coupons.error_save'));
		} finally {
			saving = false;
		}
	}

	async function deleteCoupon(coupon: Coupon) {
		const confirmed = await confirm({
			title: $t('admin.coupons.delete_title'),
			message: $t('admin.coupons.delete_message', { values: { uses: coupon.current_uses } }),
			confirmText: $t('admin.coupons.delete_confirm'),
			danger: true
		});
		if (!confirmed) return;

		try {
			const response = await fetch(`/api/admin/coupons/${coupon.id}`, {
				method: 'DELETE',
				headers: pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {}
			});
			if (!response.ok) throw new Error('Failed to delete coupon');
			toasts.add('success', $t('admin.coupons.deleted'));
			await loadCoupons();
		} catch (err) {
			console.error('Failed to delete coupon:', err);
			toasts.add('error', $t('admin.coupons.error_delete'));
		}
	}

	async function toggleCouponActive(coupon: Coupon) {
		try {
			const response = await fetch(`/api/admin/coupons/${coupon.id}`, {
				method: 'PATCH',
				headers: {
					'Content-Type': 'application/json',
					...(pb.authStore.isValid ? { Authorization: `Bearer ${pb.authStore.token}` } : {})
				},
				body: JSON.stringify({ active: !coupon.active })
			});
			if (!response.ok) throw new Error('Failed to update coupon');
			await loadCoupons();
		} catch (err) {
			console.error('Failed to toggle coupon:', err);
			toasts.add('error', $t('admin.coupons.error_save'));
		}
	}

	function getCouponStatus(coupon: Coupon): { label: string; class: string } {
		if (coupon.expires_at && new Date(coupon.expires_at) < new Date()) {
			return { label: $t('admin.coupons.status_expired'), class: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300' };
		}
		if (coupon.max_uses > 0 && coupon.current_uses >= coupon.max_uses) {
			return { label: $t('admin.coupons.status_maxed'), class: 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300' };
		}
		if (!coupon.active) {
			return { label: $t('admin.coupons.status_inactive'), class: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400' };
		}
		return { label: $t('admin.coupons.status_active'), class: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300' };
	}
</script>

<svelte:head>
	<title>{$t('admin.coupons.page_title')} | {$brandName}</title>
</svelte:head>

<div class="max-w-5xl mx-auto space-y-6">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold text-gray-900 dark:text-white">{$t('admin.coupons.page_title')}</h1>
			<p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{$t('admin.coupons.description')}</p>
		</div>
		<button type="button" class="btn btn-primary" onclick={openCreateForm}>
			+ {$t('admin.coupons.create_coupon')}
		</button>
	</div>

	<!-- Create/Edit Form -->
	{#if showForm}
		<div class="card p-6 space-y-4">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white">
				{editingCoupon ? $t('admin.coupons.edit_coupon') : $t('admin.coupons.create_coupon')}
			</h2>

			<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
				<div>
					<label for="coupon_code" class="label text-xs">{$t('admin.coupons.code_label')} *</label>
					<div class="flex gap-2">
						<input type="text" id="coupon_code" bind:value={code} class="input text-sm flex-1 uppercase" placeholder="SAVE20" />
						<button type="button" class="btn btn-secondary btn-sm whitespace-nowrap" onclick={generateCode}>
							{$t('admin.coupons.generate_code')}
						</button>
					</div>
				</div>

				<div>
					<label for="coupon_discount_type" class="label text-xs">{$t('admin.coupons.discount_type')}</label>
					<select id="coupon_discount_type" bind:value={discountType} class="input text-sm">
						<option value="percentage">{$t('admin.coupons.type_percentage')}</option>
						<option value="fixed">{$t('admin.coupons.type_fixed')}</option>
					</select>
				</div>

				<div>
					<label for="coupon_value" class="label text-xs">{$t('admin.coupons.discount_value')}</label>
					<div class="flex items-center gap-1">
						{#if discountType === 'fixed'}
							<span class="text-sm text-gray-500">$</span>
						{/if}
						<input type="number" id="coupon_value" bind:value={discountValue} class="input text-sm" min="1" step={discountType === 'percentage' ? 1 : 0.01} max={discountType === 'percentage' ? 100 : undefined} />
						{#if discountType === 'percentage'}
							<span class="text-sm text-gray-500">%</span>
						{/if}
					</div>
				</div>

				<div>
					<label for="coupon_applies_to" class="label text-xs">{$t('admin.coupons.applies_to')}</label>
					<select id="coupon_applies_to" bind:value={appliesTo} class="input text-sm">
						<option value="all">{$t('admin.coupons.applies_all')}</option>
						<option value="courses">{$t('admin.coupons.applies_courses')}</option>
						<option value="posts">{$t('admin.coupons.applies_posts')}</option>
						<option value="projects">{$t('admin.coupons.applies_projects')}</option>
					</select>
				</div>

				{#if appliesTo !== 'all'}
					<div class="md:col-span-2">
						<label for="coupon_content_ids" class="label text-xs">{$t('admin.coupons.content_ids_label')}</label>
						<input type="text" id="coupon_content_ids" bind:value={contentIds} class="input text-sm" placeholder={$t('admin.coupons.content_ids_placeholder')} />
						<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{$t('admin.coupons.content_ids_help')}</p>
					</div>
				{/if}

				<div>
					<label for="coupon_max_uses" class="label text-xs">{$t('admin.coupons.max_uses')}</label>
					<input type="number" id="coupon_max_uses" bind:value={maxUses} class="input text-sm" min="0" />
					<p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{$t('admin.coupons.max_uses_help')}</p>
				</div>

				<div>
					<label for="coupon_expires" class="label text-xs">{$t('admin.coupons.expires_at')}</label>
					<input type="date" id="coupon_expires" bind:value={expiresAt} class="input text-sm" />
				</div>
			</div>

			<div class="flex items-center gap-3">
				<label class="flex items-center gap-2 text-sm">
					<input type="checkbox" bind:checked={isActive} class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
					<span class="text-gray-700 dark:text-gray-300">{$t('admin.coupons.active_label')}</span>
				</label>
			</div>

			<div class="flex gap-2 justify-end border-t border-gray-200 dark:border-gray-700 pt-4">
				<button type="button" class="btn btn-ghost text-sm" onclick={closeForm}>{$t('admin.coupons.cancel')}</button>
				<button type="button" class="btn btn-primary text-sm" onclick={saveCoupon} disabled={saving || !code.trim()}>
					{saving ? $t('admin.coupons.saving') : (editingCoupon ? $t('admin.coupons.save') : $t('admin.coupons.create_coupon'))}
				</button>
			</div>
		</div>
	{/if}

	<!-- Coupon List -->
	<div class="card">
		{#if loading}
			<div class="flex items-center justify-center py-12">
				<svg class="animate-spin h-6 w-6 text-primary-600" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
				</svg>
			</div>
		{:else if coupons.length === 0}
			<div class="text-center py-12 px-6">
				<svg class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
				</svg>
				<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">{$t('admin.coupons.empty_title')}</h3>
				<p class="text-sm text-gray-500 dark:text-gray-400 mb-4">{$t('admin.coupons.empty_description')}</p>
				<button type="button" class="btn btn-primary" onclick={openCreateForm}>
					+ {$t('admin.coupons.create_coupon')}
				</button>
			</div>
		{:else}
			<div class="overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-gray-200 dark:border-gray-700">
							<th class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400">{$t('admin.coupons.col_code')}</th>
							<th class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400">{$t('admin.coupons.col_type')}</th>
							<th class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400">{$t('admin.coupons.col_discount')}</th>
							<th class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400">{$t('admin.coupons.col_applies_to')}</th>
							<th class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400">{$t('admin.coupons.col_uses')}</th>
							<th class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400">{$t('admin.coupons.col_expires')}</th>
							<th class="text-left py-3 px-4 font-medium text-gray-500 dark:text-gray-400">{$t('admin.coupons.col_status')}</th>
							<th class="text-right py-3 px-4 font-medium text-gray-500 dark:text-gray-400">{$t('admin.coupons.col_actions')}</th>
						</tr>
					</thead>
					<tbody>
						{#each coupons as coupon (coupon.id)}
							{@const status = getCouponStatus(coupon)}
							<tr class="border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-800/50">
								<td class="py-3 px-4">
									<code class="text-sm font-mono font-medium text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">{coupon.code}</code>
								</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400">
									{coupon.discount_type === 'percentage' ? '%' : '$'}
								</td>
								<td class="py-3 px-4 text-gray-900 dark:text-white font-medium">
									{coupon.discount_type === 'percentage' ? `${coupon.discount_value}%` : `$${coupon.discount_value.toFixed(2)}`}
								</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400 capitalize">
									{coupon.applicable_type === 'all' ? $t('admin.coupons.applies_all') : coupon.applicable_type}
								</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400">
									{coupon.current_uses}{coupon.max_uses > 0 ? `/${coupon.max_uses}` : ''}
								</td>
								<td class="py-3 px-4 text-gray-600 dark:text-gray-400">
									{#if coupon.expires_at}
										{new Date(coupon.expires_at).toLocaleDateString()}
									{:else}
										<span class="text-gray-400">-</span>
									{/if}
								</td>
								<td class="py-3 px-4">
									<span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium {status.class}">
										{status.label}
									</span>
								</td>
								<td class="py-3 px-4 text-right">
									<div class="flex items-center justify-end gap-1">
										<button
											type="button"
											class="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
											onclick={() => toggleCouponActive(coupon)}
											title={coupon.active ? $t('admin.coupons.deactivate') : $t('admin.coupons.activate')}
										>
											{#if coupon.active}
												<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
												</svg>
											{:else}
												<svg class="w-4 h-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
												</svg>
											{/if}
										</button>
										<button
											type="button"
											class="p-1.5 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
											onclick={() => openEditForm(coupon)}
											title={$t('admin.coupons.edit')}
										>
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
											</svg>
										</button>
										<button
											type="button"
											class="p-1.5 text-gray-400 hover:text-red-500 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
											onclick={() => deleteCoupon(coupon)}
											title={$t('admin.coupons.delete')}
										>
											<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
											</svg>
										</button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>
