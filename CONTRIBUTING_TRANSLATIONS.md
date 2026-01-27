# Contributing Translations to Facet

Thank you for helping make Facet accessible to more people around the world!

## Quick Start

### Adding a New Language

1. **Copy the English source file:**
   ```bash
   cp frontend/src/locales/en.json frontend/src/locales/XX.json
   ```
   Replace `XX` with your [ISO 639-1 language code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) (e.g., `es` for Spanish, `ja` for Japanese).

2. **Edit your new file** and translate each value (keep the keys in English):
   ```json
   {
     "admin": {
       "sidebar": {
         "dashboard": "Tableau de bord",
         "homepage": "Page d'accueil"
       }
     }
   }
   ```

3. **Register your locale** in `frontend/src/lib/i18n/index.ts`:
   ```typescript
   register('XX', () => import('../../locales/XX.json'));
   ```

4. **Add to supported locales** in the same file:
   ```typescript
   export const SUPPORTED_LOCALES = ['en', 'XX'] as const;
   ```

5. **Submit a Pull Request** with your changes.

### Updating Existing Translations

1. Find the key in `frontend/src/locales/en.json`
2. Update the corresponding key in your locale file
3. Submit a Pull Request

## Translation Guidelines

### Do

- Keep the same JSON structure as `en.json`
- Preserve placeholders like `{name}`, `{count}`, `{link}`
- Use natural phrasing for your language (don't translate too literally)
- Test your translations by running the app locally
- Include context in PR description for ambiguous translations

### Don't

- Change the keys (left side of the colon)
- Remove or modify placeholders from strings
- Translate brand names ("Facet", "GitHub", "Google", etc.)
- Include HTML tags in translations
- Use machine translation without review

## Placeholders

Some strings contain placeholders that get replaced with dynamic values:

| Placeholder | Description | Example |
|-------------|-------------|---------|
| `{count}` | A number | "You have {count} items" |
| `{name}` | A name or title | "Welcome, {name}" |
| `{link}` | A clickable link | "Powered by {link}" |
| `{min}`, `{max}` | Validation limits | "Must be at least {min} characters" |

Always keep these placeholders in your translations!

## Validation

Before submitting, validate your translations:

```bash
cd frontend
npm run i18n:validate -- XX
```

This will show:
- Missing keys (in English but not in your translation)
- Extra keys (in your translation but not in English)
- Coverage percentage

## Example

**English (`en.json`):**
```json
{
  "admin": {
    "dashboard": {
      "welcome_title": "This is your space.",
      "pending_alert": "You have {count} pending proposal(s) to review"
    }
  }
}
```

**Spanish (`es.json`):**
```json
{
  "admin": {
    "dashboard": {
      "welcome_title": "Este es tu espacio.",
      "pending_alert": "Tienes {count} propuesta(s) pendiente(s) de revisar"
    }
  }
}
```

## Language Codes

Use standard ISO 639-1 codes. For regional variants, use ISO 639-1 + ISO 3166-1 alpha-2:

| Code | Language |
|------|----------|
| `en` | English (source) |
| `es` | Spanish |
| `fr` | French |
| `de` | German |
| `ja` | Japanese |
| `ko` | Korean |
| `pt-BR` | Portuguese (Brazil) |
| `zh-CN` | Chinese (Simplified) |
| `zh-TW` | Chinese (Traditional) |

## Testing Locally

1. Clone the repository
2. Make your translation changes
3. Run the development server:
   ```bash
   cd frontend
   npm install
   npm run dev
   ```
4. Change language in the app to test your translations

## Questions?

- Open an issue with the `i18n` label
- Tag `@jesposito` for translation questions
- Check existing translations for style guidance

## Credits

All translation contributors will be credited in the release notes. Thank you for making Facet accessible to more people!
