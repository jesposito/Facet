import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
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
