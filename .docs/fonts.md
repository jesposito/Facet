# Bundled fonts

Checked **2026-11-11**.

Facet self-hosts its font packages through Fontsource and imports the selected weights from `frontend/src/lib/fonts.css`. Every direct Fontsource package matches its latest npm registry release on this date; none of the inspected latest records was marked deprecated.

| Locked version | Packages                                                                                                                                                                                                                                                    |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 5.3.0          | `@fontsource/dm-mono`, `dm-sans`, `fira-code`, `hanken-grotesk`, `ibm-plex-mono`, `ibm-plex-sans`, `inter`, `jetbrains-mono`, `lato`, `lora`, `merriweather`, `newsreader`, `nunito-sans`, `playfair-display`, `plus-jakarta-sans`, `sora`, `space-grotesk` |
| 5.3.1          | `@fontsource/source-code-pro`, `@fontsource/source-sans-3`                                                                                                                                                                                                  |

The production build runs `frontend/scripts/build-font-notices.mjs` before Vite. It creates `frontend/static/fonts/THIRD-PARTY-NOTICES.txt` from installed font licences. Preserve that step and the upstream licence notices when changing font packages.

Sources: [`frontend/package.json`](../frontend/package.json), [`frontend/package-lock.json`](../frontend/package-lock.json), [`frontend/src/lib/fonts.css`](../frontend/src/lib/fonts.css), [`frontend/scripts/build-font-notices.mjs`](../frontend/scripts/build-font-notices.mjs), and each package's `https://registry.npmjs.org/@fontsource%2f<name>/latest` registry record, all checked 2026-11-11. Example: [Fontsource Inter latest](https://registry.npmjs.org/%40fontsource%2Finter/latest).
