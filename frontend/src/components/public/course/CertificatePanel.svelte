<script lang="ts">
	/**
	 * The learner's course certificate, as the server reports it. Nothing
	 * here decides whether a certificate exists; it shows the state loaded
	 * from GET /api/courses/{slug}/certificate.
	 */
	import { invalidateAll } from '$app/navigation';
	import { t } from 'svelte-i18n';
	import { formatDate } from '$lib/utils';
	import { certificatePath, type CertificateState } from '$lib/commerce';

	interface Props {
		certificate: CertificateState;
		/** True once the server has recorded the course as finished. */
		courseCompleted: boolean;
	}

	let { certificate, courseCompleted }: Props = $props();

	const uid = $props.id();
	let checking = $state(false);

	async function checkAgain() {
		checking = true;
		await invalidateAll();
		checking = false;
	}
</script>

{#if certificate.status === 'issued'}
	{@const cert = certificate.certificate}
	<section
		class="card p-6"
		aria-labelledby="{uid}-heading"
		data-testid="certificate-panel"
		data-status="issued"
	>
		<h2 id="{uid}-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
			{$t('public.commerce.certificate.your')}
		</h2>
		<p class="mt-2 text-sm text-gray-600 dark:text-gray-300">
			{#if cert.issued_at}{$t('public.commerce.certificate.issued', {
					values: {
						date: formatDate(cert.issued_at, { month: 'long', day: 'numeric', year: 'numeric' })
					}
				})}{#if cert.student_name}
					{$t('public.commerce.certificate.issued_to', {
						values: { name: cert.student_name }
					})}{/if}.
			{/if}
			{$t('public.commerce.certificate.verify')}
		</p>
		<a href={certificatePath(cert.certificate_id)} class="btn btn-primary mt-4"
			>{$t('public.commerce.certificate.view')}</a
		>
	</section>
{:else if courseCompleted && certificate.status === 'missing'}
	<section
		class="card p-6"
		aria-labelledby="{uid}-heading"
		data-testid="certificate-panel"
		data-status="pending"
	>
		<h2 id="{uid}-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
			{$t('public.commerce.certificate.pending')}
		</h2>
		<p role="status" class="mt-2 text-sm text-gray-600 dark:text-gray-300">
			{$t('public.commerce.certificate.pending_help')}
		</p>
		<button
			type="button"
			class="btn btn-secondary mt-4"
			onclick={checkAgain}
			disabled={checking}
			aria-busy={checking}
		>
			{checking ? $t('public.commerce.common.checking') : $t('public.commerce.common.check_again')}
		</button>
	</section>
{:else if certificate.status === 'unavailable'}
	<section
		class="card p-6"
		aria-labelledby="{uid}-heading"
		data-testid="certificate-panel"
		data-status="unavailable"
	>
		<h2 id="{uid}-heading" class="text-lg font-semibold text-gray-900 dark:text-white">
			{$t('public.commerce.certificate.title')}
		</h2>
		<p role="alert" class="mt-2 text-sm text-gray-600 dark:text-gray-300">
			{$t('public.commerce.certificate.error')}
		</p>
		<button
			type="button"
			class="btn btn-secondary mt-4"
			onclick={checkAgain}
			disabled={checking}
			aria-busy={checking}
		>
			{checking ? $t('public.commerce.common.checking') : $t('public.commerce.common.try_again')}
		</button>
	</section>
{/if}
