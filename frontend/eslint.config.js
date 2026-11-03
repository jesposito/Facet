import js from '@eslint/js';
import { defineConfig, globalIgnores } from 'eslint/config';
import prettier from 'eslint-config-prettier';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import ts from 'typescript-eslint';

export default defineConfig(
	globalIgnores([
		'build/**',
		'.svelte-kit/**',
		'static/**',
		'vendor/**',
		'test-results/**',
		'playwright-report/**'
	]),
	js.configs.recommended,
	ts.configs.recommended,
	svelte.configs.recommended,
	prettier,
	{
		languageOptions: {
			globals: { ...globals.browser, ...globals.node, __APP_VERSION__: 'readonly' }
		},
		rules: {
			// Content is sanitized at rendering boundaries and covered by XSS tests.
			'svelte/no-at-html-tags': 'off',
			// Facet is served at the origin root; links never target a configured base path.
			'svelte/no-navigation-without-resolve': 'off',
			// Static/value lists can contain duplicates and do not require identity keys.
			'svelte/require-each-key': 'off',
			// Plain Map/Set values are also used outside reactive rendering.
			'svelte/prefer-svelte-reactivity': 'off',
			'@typescript-eslint/no-explicit-any': 'warn',
			'@typescript-eslint/no-unused-vars': [
				'warn',
				{ argsIgnorePattern: '^_', caughtErrors: 'none' }
			]
		}
	},
	{
		files: ['**/*.ts'],
		languageOptions: { parser: ts.parser },
		rules: {
			// TypeScript, rather than core ESLint, resolves TS-only types and value globals.
			'no-undef': 'off'
		}
	},
	{
		files: ['**/*.svelte'],
		languageOptions: { parserOptions: { parser: ts.parser } },
		rules: {
			// Svelte templates and TS-only types are checked by svelte-check.
			'no-undef': 'off'
		}
	},
	{
		files: ['**/*.svelte.ts', '**/*.svelte.js'],
		languageOptions: { parser: ts.parser }
	},
	{
		files: ['src/lib/discovery.ts', 'src/lib/experiences/types.ts'],
		rules: {
			// These validators deliberately reject control characters in external URLs.
			'no-control-regex': 'off'
		}
	},
	{
		files: [
			'src/routes/+page.svelte',
			'src/routes/[[]slug=slug]/+page.svelte',
			'src/routes/posts/[[]slug]/+page.svelte'
		],
		rules: {
			// The Svelte parser requires an escaped closing script tag inside {@html} strings.
			'no-useless-escape': 'off'
		}
	},
	{
		files: ['src/lib/components/admin/SingleMediaPicker.svelte'],
		rules: {
			// The bindable prop's null default is part of the generated component API.
			'no-useless-assignment': 'off'
		}
	},
	{
		files: ['tailwind.config.js'],
		rules: {
			// Tailwind loads its typography plugin through this CommonJS configuration.
			'@typescript-eslint/no-require-imports': 'off'
		}
	}
);
