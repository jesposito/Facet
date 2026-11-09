# Tailwind CSS and PostCSS

Checked **2026-11-10**.

| Package                            | Locked                                  | Latest                                  | Status                                   |
| ---------------------------------- | --------------------------------------- | --------------------------------------- | ---------------------------------------- |
| `tailwindcss`                      | 3.4.19                                  | 4.3.5                                   | planned major migration                  |
| `@tailwindcss/typography`          | 0.5.20                                  | 0.5.20                                  | current                                  |
| `postcss`                          | 8.5.32                                  | 8.5.32                                  | current                                  |
| `autoprefixer`                     | 10.6.2                                  | 10.6.2                                  | current                                  |
| `postcss-selector-parser` override | 7.x range; locked graph in package-lock | override selected for patched behaviour | retain until the consuming chain is safe |

Facet uses a Tailwind 3 content configuration for `src/**/*.{html,js,svelte,ts}`, class-based dark mode, CSS-variable-backed colour ramps, typography and custom Soft Premium variants. PostCSS loads Tailwind and Autoprefixer. These are compatibility-sensitive paths: typography, arbitrary variants and dynamic colour classes need visual checks in a Tailwind 4 migration.

Dependabot ignores routine major updates. `Facet-ehg` tracks an intentional Tailwind 4 migration alongside compiler/runtime compatibility work. Run the production build and full browser suite, including contrast and design-system specs, before accepting that major.

Sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/tailwind.config.js`](../frontend/tailwind.config.js), [`frontend/postcss.config.js`](../frontend/postcss.config.js), [`.github/dependabot.yml`](../.github/dependabot.yml), [`CONTRIBUTING.md`](../CONTRIBUTING.md), npm registry latest endpoints for [Tailwind CSS](https://registry.npmjs.org/tailwindcss/latest), [typography](https://registry.npmjs.org/%40tailwindcss%2Ftypography/latest), [PostCSS](https://registry.npmjs.org/postcss/latest) and [Autoprefixer](https://registry.npmjs.org/autoprefixer/latest), all checked 2026-11-10.
