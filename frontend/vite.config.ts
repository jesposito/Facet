import adapter from '@sveltejs/adapter-node';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { execSync } from 'child_process';
import { copyFileSync, existsSync, mkdirSync } from 'fs';
import { resolve, dirname } from 'path';

function getVersion(): string {
	if (process.env.FACET_VERSION && process.env.FACET_VERSION !== 'dev') {
		return process.env.FACET_VERSION;
	}
	try {
		return execSync('git describe --tags --always', { encoding: 'utf8' }).trim();
	} catch {
		return 'dev';
	}
}

// Plugin to copy CHANGELOG.md to static folder for frontend access
function copyChangelog() {
	return {
		name: 'copy-changelog',
		buildStart() {
			const src = resolve(import.meta.dirname, '../CHANGELOG.md');
			const dest = resolve(import.meta.dirname, 'static/CHANGELOG.md');

			if (existsSync(src)) {
				const destDir = dirname(dest);
				if (!existsSync(destDir)) {
					mkdirSync(destDir, { recursive: true });
				}
				copyFileSync(src, dest);
				console.log('[vite] Copied CHANGELOG.md to static folder');
			}
		}
	};
}

// Dev proxy upstream. POCKETBASE_URL is server-side config (no VITE_ prefix),
// so it never reaches the browser bundle. Native dev uses loopback; the
// Docker dev frontend sets http://backend:8090.
function pocketbaseProxyTarget(): string {
	const raw = process.env.POCKETBASE_URL || 'http://127.0.0.1:8090';
	let url: URL;
	try {
		url = new URL(raw);
	} catch {
		throw new Error('POCKETBASE_URL must be an absolute http(s) URL');
	}
	if (
		(url.protocol !== 'http:' && url.protocol !== 'https:') ||
		url.username ||
		url.password ||
		url.pathname !== '/' ||
		url.search ||
		url.hash
	) {
		throw new Error(
			'POCKETBASE_URL must be an http(s) origin without credentials, path, query or fragment'
		);
	}
	return url.origin;
}

const pocketbaseTarget = pocketbaseProxyTarget();

export default defineConfig({
	plugins: [
		sveltekit({
			preprocess: vitePreprocess(),
			adapter: adapter({ out: 'build', precompress: true, envPrefix: '' }),
			alias: { $components: 'src/components', $lib: 'src/lib' }
		}),
		copyChangelog()
	],
	define: { __APP_VERSION__: JSON.stringify(getVersion()) },
	server: {
		proxy: {
			'/api': {
				target: pocketbaseTarget,
				changeOrigin: true
			},
			'/_': {
				target: pocketbaseTarget,
				changeOrigin: true
			}
		}
	}
});
