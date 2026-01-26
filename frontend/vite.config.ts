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
			const src = resolve(__dirname, '../CHANGELOG.md');
			const dest = resolve(__dirname, 'static/CHANGELOG.md');

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

export default defineConfig({
	plugins: [sveltekit(), copyChangelog()],
	define: {
		__APP_VERSION__: JSON.stringify(getVersion())
	},
	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:8090',
				changeOrigin: true
			},
			'/_': {
				target: 'http://localhost:8090',
				changeOrigin: true
			}
		}
	}
});
