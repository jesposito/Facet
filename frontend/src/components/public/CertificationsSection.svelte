<script lang="ts">
	import type { Certification } from '$lib/pocketbase';
	import { formatDate } from '$lib/utils';

	export let items: Certification[];
	export let layout: string = 'grouped';

	// Group certifications by issuer
	function groupByIssuer(certs: Certification[]): Map<string, Certification[]> {
		const groups = new Map<string, Certification[]>();
		for (const cert of certs) {
			const issuerKey = cert.issuer || 'Other';
			if (!groups.has(issuerKey)) {
				groups.set(issuerKey, []);
			}
			groups.get(issuerKey)!.push(cert);
		}
		return groups;
	}

	// Check if a certification is expired
	function isExpired(cert: Certification): boolean {
		if (!cert.expiry_date) return false;
		return new Date(cert.expiry_date) < new Date();
	}

	// Check if a certification expires soon (within 30 days)
	function expiresSoon(cert: Certification): boolean {
		if (!cert.expiry_date) return false;
		const expiry = new Date(cert.expiry_date);
		const now = new Date();
		const thirtyDaysFromNow = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000);
		return expiry > now && expiry <= thirtyDaysFromNow;
	}

	$: groupedCertifications = groupByIssuer(items);
</script>

<section id="certifications" class="mb-16" data-layout={layout}>
	<h2 class="section-title">Certifications & Credentials</h2>

	<div class="space-y-8">
		{#each [...groupedCertifications] as [issuerName, certs] (issuerName)}
			<div class="animate-fade-in">
				<h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-4">
					{issuerName}
				</h3>
				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					{#each certs as cert (cert.id)}
						<article class="card p-5 flex gap-4">
							<!-- Badge icon -->
							<div class="flex-shrink-0">
								<div class="w-12 h-12 rounded-full bg-gradient-to-br from-primary-500 to-primary-700 flex items-center justify-center">
									<svg class="w-6 h-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
									</svg>
								</div>
							</div>

							<!-- Content -->
							<div class="flex-1 min-w-0">
								<div class="flex items-start justify-between gap-2">
									<h4 class="font-semibold text-gray-900 dark:text-white">
										{cert.name}
									</h4>
									{#if isExpired(cert)}
										<span class="px-2 py-0.5 text-xs bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300 rounded flex-shrink-0">
											Expired
										</span>
									{:else if expiresSoon(cert)}
										<span class="px-2 py-0.5 text-xs bg-amber-100 dark:bg-amber-900/50 text-amber-700 dark:text-amber-300 rounded flex-shrink-0">
											Expiring Soon
										</span>
									{/if}
								</div>

								<!-- Dates -->
								<div class="mt-1 text-sm text-gray-600 dark:text-gray-400">
									{#if cert.issue_date}
										<span>Issued {formatDate(cert.issue_date, { month: 'short', year: 'numeric' })}</span>
									{/if}
									{#if cert.expiry_date}
										<span class="text-gray-400 dark:text-gray-500"> - </span>
										<span class:text-red-600={isExpired(cert)} class:dark:text-red-400={isExpired(cert)}>
											{isExpired(cert) ? 'Expired' : 'Expires'} {formatDate(cert.expiry_date, { month: 'short', year: 'numeric' })}
										</span>
									{:else if cert.issue_date}
										<span class="text-gray-400 dark:text-gray-500"> - </span>
										<span class="text-green-600 dark:text-green-400">No expiration</span>
									{/if}
								</div>

								<!-- Credential ID -->
								{#if cert.credential_id}
									<div class="mt-1 text-xs text-gray-500 dark:text-gray-500">
										Credential ID: {cert.credential_id}
									</div>
								{/if}

								<!-- Verify link -->
								{#if cert.credential_url}
									<a
										href={cert.credential_url}
										target="_blank"
										rel="noopener noreferrer"
										class="mt-2 inline-flex items-center gap-1.5 text-sm font-medium text-primary-600 dark:text-primary-400 hover:text-primary-700 dark:hover:text-primary-300"
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
										</svg>
										Verify Credential
									</a>
								{/if}
							</div>
						</article>
					{/each}
				</div>
			</div>
		{/each}
	</div>
</section>
