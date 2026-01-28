/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	darkMode: 'class',
	theme: {
		extend: {
			colors: {
				// Primary colors now use CSS custom properties for dynamic theming
				// Default values (sky) are defined in app.css :root
				primary: {
					50: 'var(--color-primary-50)',
					100: 'var(--color-primary-100)',
					200: 'var(--color-primary-200)',
					300: 'var(--color-primary-300)',
					400: 'var(--color-primary-400)',
					500: 'var(--color-primary-500)',
					600: 'var(--color-primary-600)',
					700: 'var(--color-primary-700)',
					800: 'var(--color-primary-800)',
					900: 'var(--color-primary-900)',
					950: 'var(--color-primary-950)'
				}
			},
			fontFamily: {
				sans: ['Inter', 'system-ui', 'sans-serif'],
				mono: ['JetBrains Mono', 'monospace']
			},
			// UX Delight: Standardized transition timing
			transitionDuration: {
				fast: '150ms',
				normal: '200ms',
				slow: '300ms'
			},
			// UX Delight: Custom easing for snappy interactions
			transitionTimingFunction: {
				'out-expo': 'cubic-bezier(0.16, 1, 0.3, 1)',
				'in-out-expo': 'cubic-bezier(0.87, 0, 0.13, 1)'
			}
		}
	},
	plugins: [require('@tailwindcss/typography')]
};
