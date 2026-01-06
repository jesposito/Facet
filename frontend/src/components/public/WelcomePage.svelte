<script lang="ts">
	import ThemeToggle from '$components/shared/ThemeToggle.svelte';
</script>

<div class="min-h-screen bg-gradient-to-br from-primary-50 via-white to-accent-50 dark:from-gray-900 dark:via-gray-900 dark:to-gray-800 flex items-center justify-center p-4">
	<!-- Theme toggle -->
	<div class="fixed top-4 right-4 z-40">
		<ThemeToggle />
	</div>

	<div class="max-w-4xl w-full bg-white dark:bg-gray-800 rounded-2xl shadow-xl border border-gray-200 dark:border-gray-700 p-8 md:p-12 space-y-10">
		<!-- Header -->
		<div class="text-center space-y-4">
			<div class="inline-block p-4 bg-primary-100 dark:bg-primary-900/30 rounded-full">
				<svg class="w-16 h-16 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
					<path stroke-linecap="round" stroke-linejoin="round" d="M17.982 18.725A7.488 7.488 0 0012 15.75a7.488 7.488 0 00-5.982 2.975m11.963 0a9 9 0 10-11.963 0m11.963 0A8.966 8.966 0 0112 21a8.966 8.966 0 01-5.982-2.275M15 9.75a3 3 0 11-6 0 3 3 0 016 0z" />
				</svg>
			</div>
			<h1 class="text-4xl md:text-5xl font-bold text-gray-900 dark:text-white">
				Welcome to Facet
			</h1>
			<p class="text-xl text-gray-600 dark:text-gray-300 max-w-2xl mx-auto">
				A self-hosted personal profile platform. Own your professional identity, control who sees what, and skip the tracking.
			</p>
			<div class="pt-6 space-y-3">
				<a
					href="/admin/login"
					class="inline-flex items-center gap-2 px-8 py-4 text-lg font-semibold text-white bg-primary-600 hover:bg-primary-700 rounded-lg shadow-sm transition-colors"
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
					</svg>
					Sign In to Get Started
				</a>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					Try Demo Mode to explore all features
				</p>
			</div>
		</div>

		<!-- Features Grid -->
		<div class="space-y-6">
			<h2 class="text-2xl font-semibold text-gray-900 dark:text-white text-center">
				What can you do with Facet?
			</h2>

			<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				<!-- Feature 1: Multiple Views -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
							<path stroke-linecap="round" stroke-linejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">Multiple Views</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">Different profiles for recruiters, conferences, clients</p>
					</div>
				</div>

				<!-- Feature 2: Privacy Controls -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">Privacy Controls</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">Public, unlisted, password-protected, or private</p>
					</div>
				</div>

				<!-- Feature 3: Own Your Data -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">Own Your Data</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">SQLite database you control, easy to backup</p>
					</div>
				</div>

				<!-- Feature 4: GitHub Import -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.166 6.839 9.489.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.603-3.369-1.341-3.369-1.341-.454-1.155-1.11-1.462-1.11-1.462-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.268 2.75 1.026A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.026 2.747-1.026.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">GitHub Import</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">Pull in your repos with optional AI summaries</p>
					</div>
				</div>

				<!-- Feature 5: Resume Parsing -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">Resume Parsing</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">Upload PDF/DOCX, AI extracts everything</p>
					</div>
				</div>

				<!-- Feature 6: AI Writing -->
				<div class="flex gap-4 p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
					<div class="flex-shrink-0">
						<svg class="w-6 h-6 text-primary-600 dark:text-primary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
						</svg>
					</div>
					<div>
						<h3 class="font-semibold text-gray-900 dark:text-white">AI Writing Assistant</h3>
						<p class="text-sm text-gray-600 dark:text-gray-300">Polish your content with 5 tones + critique mode</p>
					</div>
				</div>
			</div>
		</div>

		<!-- Why Self-Host -->
		<div class="space-y-4 pt-4">
			<h2 class="text-xl font-semibold text-gray-900 dark:text-white text-center">
				Why self-host your profile?
			</h2>
			<div class="flex flex-wrap justify-center gap-6 text-sm text-gray-600 dark:text-gray-300">
				<div class="flex items-center gap-2">
					<svg class="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
					</svg>
					<span>No tracking or analytics</span>
				</div>
				<div class="flex items-center gap-2">
					<svg class="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
					</svg>
					<span>Your data never leaves your server</span>
				</div>
				<div class="flex items-center gap-2">
					<svg class="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
					</svg>
					<span>No algorithms deciding who sees you</span>
				</div>
			</div>
		</div>

		<!-- Get Involved -->
		<div class="pt-6 border-t border-gray-200 dark:border-gray-700 space-y-4">
			<h2 class="text-lg font-semibold text-gray-900 dark:text-white text-center">
				Open Source & Community Driven
			</h2>
			<p class="text-sm text-gray-600 dark:text-gray-300 text-center max-w-xl mx-auto">
				Facet is MIT licensed and built in the open. Contributions, bug reports, and feature ideas are always welcome.
			</p>
			<div class="flex flex-wrap justify-center gap-4">
				<a
					href="https://github.com/jesposito/Facet"
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors"
				>
					<svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
						<path d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.166 6.839 9.489.5.092.682-.217.682-.482 0-.237-.009-.866-.013-1.7-2.782.603-3.369-1.341-3.369-1.341-.454-1.155-1.11-1.462-1.11-1.462-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.268 2.75 1.026A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.026 2.747-1.026.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.163 22 16.418 22 12c0-5.523-4.477-10-10-10z" />
					</svg>
					View on GitHub
				</a>
				<a
					href="https://github.com/jesposito/Facet/issues"
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors"
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
					</svg>
					Report an Issue
				</a>
				<a
					href="https://buymeacoffee.com/jesposito"
					target="_blank"
					rel="noopener noreferrer"
					class="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors"
				>
					<svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
						<path stroke-linecap="round" stroke-linejoin="round" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
					</svg>
					Support the Project
				</a>
			</div>
		</div>

		<!-- Footer -->
		<div class="pt-4 text-center text-sm text-gray-500 dark:text-gray-400 space-y-2">
			<p>No tracking. No ads. Just your profile, your way.</p>
			<p class="text-xs">
				<a href="https://github.com/jesposito/Facet#readme" target="_blank" rel="noopener noreferrer" class="hover:text-primary-600 dark:hover:text-primary-400">
					Documentation
				</a>
				<span class="mx-2">·</span>
				<a href="https://github.com/jesposito/Facet/blob/main/docs/SELF-HOSTING-GUIDE.md" target="_blank" rel="noopener noreferrer" class="hover:text-primary-600 dark:hover:text-primary-400">
					Self-Hosting Guide
				</a>
			</p>
		</div>
	</div>
</div>
