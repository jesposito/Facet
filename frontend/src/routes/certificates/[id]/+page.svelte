<script lang="ts">
	import type { PageData } from './$types';
	import { t } from 'svelte-i18n';
	import { formatDate } from '$lib/utils';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();
	let cert = $derived(data.certificate);
	let issued = $derived(
		cert.issued_at
			? formatDate(cert.issued_at, { month: 'long', day: 'numeric', year: 'numeric' })
			: ''
	);
</script>

<svelte:head>
	<title
		>{$t('public.commerce.certificate_page.title', {
			values: { course: cert.course_title }
		})}</title
	>
	<meta name="robots" content="noindex" />
</svelte:head>

<main
	id="main-content"
	class="min-h-screen bg-gray-50 dark:bg-gray-900 print:bg-white flex items-center justify-center px-4 py-16 print:py-0"
>
	<div class="max-w-2xl w-full">
		<article
			class="card p-10 text-center border-4 border-double border-primary-300 dark:border-primary-700 print:border-gray-400 print:shadow-none"
			aria-labelledby="certificate-heading"
			data-testid="certificate"
		>
			<p class="text-sm uppercase tracking-widest text-gray-500 dark:text-gray-400">
				{$t('public.commerce.certificate_page.heading')}
			</p>
			<p class="mt-6 text-gray-600 dark:text-gray-300">
				{$t('public.commerce.certificate_page.certifies')}
			</p>
			<p class="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">
				{cert.student_name || $t('public.commerce.certificate_page.learner')}
			</p>
			<p class="mt-4 text-gray-600 dark:text-gray-300">
				{$t('public.commerce.certificate_page.completed')}
			</p>
			<h1 id="certificate-heading" class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">
				{cert.course_title}
			</h1>
			{#if cert.instructor_name}
				<p class="mt-4 text-gray-600 dark:text-gray-300">
					{$t('public.commerce.certificate_page.taught_by', {
						values: { name: cert.instructor_name }
					})}
				</p>
			{/if}
			{#if issued}
				<p class="mt-6 text-sm text-gray-500 dark:text-gray-400">
					{$t('public.commerce.certificate_page.issued', { values: { date: issued } })}
				</p>
			{/if}
			<p class="mt-2 text-xs text-gray-500 dark:text-gray-400 break-all">
				{$t('public.commerce.certificate_page.id', { values: { id: cert.certificate_id } })}
			</p>
		</article>

		<div class="mt-6 flex flex-wrap items-center justify-center gap-3 print:hidden">
			<p role="status" class="w-full text-center text-sm text-emerald-700 dark:text-emerald-400">
				{$t('public.commerce.certificate_page.verified')}
			</p>
			<button type="button" class="btn btn-primary" onclick={() => window.print()}
				>{$t('public.commerce.certificate_page.print')}</button
			>
			<a href="/" class="btn btn-secondary">{$t('public.commerce.common.homepage')}</a>
		</div>
	</div>
</main>
