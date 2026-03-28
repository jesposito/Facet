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
				serif: ['Lora', 'Georgia', 'Times New Roman', 'serif'],
				mono: ['JetBrains Mono', 'monospace']
			},
			fontSize: {
				'display-sm': ['2.25rem', { lineHeight: '1.1', letterSpacing: '-0.02em' }],
				'display': ['3rem', { lineHeight: '1.1', letterSpacing: '-0.02em' }],
				'display-lg': ['3.75rem', { lineHeight: '1.05', letterSpacing: '-0.025em' }],
				'display-xl': ['4.5rem', { lineHeight: '1', letterSpacing: '-0.03em' }]
			},
			boxShadow: {
				'editorial': '0 1px 3px rgba(0,0,0,0.04), 0 4px 12px rgba(0,0,0,0.06)',
				'editorial-md': '0 2px 6px rgba(0,0,0,0.05), 0 8px 24px rgba(0,0,0,0.08)',
				'editorial-lg': '0 4px 12px rgba(0,0,0,0.06), 0 16px 40px rgba(0,0,0,0.1)',
				'editorial-xl': '0 8px 24px rgba(0,0,0,0.08), 0 32px 64px rgba(0,0,0,0.12)',
				'glow': '0 0 20px rgba(var(--glow-rgb, 14,165,233), 0.35), 0 0 60px rgba(var(--glow-rgb, 14,165,233), 0.15)'
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
