# Contributing translations

Thank you for helping Facet speak more languages. You don't need to be a programmer to help: if you can edit a text file on GitHub's website, you can translate.

## Getting help

- **Discord:** [join Facet HQ](https://discord.gg/XD8eUudnmf) and ask in the translations channel, especially when a phrase is ambiguous out of context.
- **GitHub:** [open an issue](https://github.com/jesposito/Facet/issues/new) and mention that it's about translations.

## Current languages

| Code | Language | Notes |
|------|----------|-------|
| `en` | English | The source. Every other file follows its keys. |
| `de` | Deutsch (German) | |
| `elvish` | Sindarin (Elvish) | For fun |
| `klingon` | tlhIngan Hol (Klingon) | For fun |
| `lolcat` | LOLcat | For fun |

Spanish, French, Japanese, Korean, Portuguese, Chinese and others would be very welcome.

## Translating on GitHub's website

### Improve an existing language

1. Open [`frontend/src/locales`](frontend/src/locales) in the repository.
2. Click your language's file, for example `de.json`.
3. Click the pencil icon to edit. GitHub offers to make a copy (a "fork") for you.
4. Make your changes, then click **Propose changes** and **Create pull request**.

### Start a new language

1. Open `en.json` in [`frontend/src/locales`](frontend/src/locales) and copy its entire contents.
2. Go back to the `locales` folder and choose **Add file → Create new file**.
3. Name it with your language code and `.json`, for example `es.json` for Spanish or `pt-BR.json` for Brazilian Portuguese.
4. Paste the English text and translate the values (see below).
5. Propose the file and open a pull request. Say in the description that it's a new language, so a maintainer can register it.

You don't have to finish in one go. A partial translation is a fine start, and others can help complete it.

## How the files work

Each file is JSON: keys on the left, text on the right.

```json
{
  "admin": {
    "sidebar": {
      "dashboard": "Dashboard",
      "homepage": "Homepage"
    }
  }
}
```

| Leave as it is | Translate |
|----------------|-----------|
| Keys, the part before the colon: `"dashboard":` | The text in quotes after the colon: `"Dashboard"` becomes `"Tableau de bord"` |
| Braces, brackets, commas and quote marks | |
| Placeholders in curly braces: `{name}`, `{count}` | The words around them |
| Brand names: Facet, GitHub, Google | |

Placeholders are filled in by the app. `"Hello, {name}!"` becomes `"Bonjour, {name}!"`, never `"Bonjour, {nom}!"`.

Some values contain small bits of HTML, such as `<strong>` or `<code>`. Keep the tags and translate the text between them.

## Guidelines

- Write what a native speaker would say, not a word-for-word copy of the English.
- Keep the tone friendly and plain, like the English.
- If you use machine translation as a starting point, read and correct every line.
- When a phrase is ambiguous, ask, or explain your choice in the pull request.

## For developers

### Register a new language

A new language file needs registering in a few places before it appears in the app:

1. **Frontend** ([`frontend/src/lib/i18n/index.ts`](frontend/src/lib/i18n/index.ts)):
   ```ts
   register('es', () => import('../../locales/es.json'));
   ```
   Add the code to `SUPPORTED_LOCALES`, and its display name to `LOCALE_NAMES` if it isn't there already.
2. **Default site language.** The `default_locale` setting only accepts the codes listed in its migration ([`1738000001_add_site_default_locale.go`](backend/migrations/1738000001_add_site_default_locale.go)). Add a new migration that extends the list, so owners can choose the language for visitors.
3. **Emails.** Testimonial emails are translated in [`backend/services/email_i18n.go`](backend/services/email_i18n.go). Languages without an entry fall back to English.

### Check your work

```bash
cd frontend
npm run i18n:validate
```

This runs two checks: [`validate-translations.js`](scripts/validate-translations.js) compares every language with `en.json` and reports missing and extra keys with a coverage percentage, and [`check-i18n-values-params.mjs`](scripts/check-i18n-values-params.mjs) makes sure code that uses a placeholder actually supplies a value for it. The [i18n workflow](.github/workflows/i18n-check.yml) runs both on pull requests that touch translations or frontend code, with the first in strict `--ci` mode.

To see translations in the running app, start the [development environment](docs/DEV.md) and switch the language in the admin, or set the site's default language in **Site Settings**.

## Thank you

Translators make Facet usable by people who would otherwise be left out. When a new contributor's pull request is merged, the project's Discord gives them a welcome in its hall of fame.
