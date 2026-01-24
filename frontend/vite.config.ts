import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { execSync } from 'child_process';

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

export default defineConfig({
	plugins: [sveltekit()],
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
