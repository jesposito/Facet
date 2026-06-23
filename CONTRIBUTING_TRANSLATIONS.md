# Contributing Translations to Facet

Thank you for helping make Facet accessible to more people around the world! This guide explains how anyone can contribute translations, regardless of technical experience.

## Need Help?

**Join our Discord community!** If you have questions about translating, need clarification on what a phrase means, or want to coordinate with other translators, come chat with us:

- **Discord**: [Join the Facet Discord](https://discord.gg/XD8eUudnmf) (ask in #translations channel)
- **GitHub Issues**: [Open an issue](https://github.com/jesposito/Facet/issues/new) with the `i18n` label

---

## For Non-Technical Contributors (No Coding Required!)

You can contribute translations directly through GitHub's website - no special software needed!

### Step 1: Create a GitHub Account

If you don't have one, [sign up for free at GitHub.com](https://github.com/signup).

### Step 2: Find the Translation Files

1. Go to the [Facet repository](https://github.com/jesposito/Facet)
2. Navigate to: `frontend` → `src` → `locales`
3. You'll see files like `en.json` (English), `de.json` (German), etc.

### Step 3: Choose Your Task

**Option A: Improve an Existing Translation**
1. Click on the file for your language (e.g., `de.json` for German)
2. Click the pencil icon (Edit) in the top right
3. Make your changes
4. Scroll down and click "Propose changes"
5. Click "Create pull request"
6. Done! We'll review your changes.

**Option B: Add a New Language**
1. Click on `en.json` (the English source file)
2. Click the pencil icon to edit
3. Select all the text (Ctrl+A or Cmd+A) and copy it (Ctrl+C or Cmd+C)
4. Go back to the `locales` folder
5. Click "Add file" → "Create new file"
6. Name it with your language code + `.json` (e.g., `es.json` for Spanish, `ja.json` for Japanese)
7. Paste the English content
8. Replace the English text with your translations (see "How to Translate" below)
9. Click "Propose new file"
10. Click "Create pull request"
11. **Important**: Mention in your PR that this is a new language so we can register it!

### How to Translate (The JSON Format)

Translation files look like this:

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

**The rules are simple:**

| Keep This (Don't Change!) | Translate This |
|---------------------------|----------------|
| `"dashboard":` | `"Dashboard"` → `"Tableau de bord"` |
| `"homepage":` | `"Homepage"` → `"Page d'accueil"` |
| All the `{` and `}` brackets | The text inside quotes after the `:` |

**Example - Translating to French:**

Before:
```json
"dashboard": "Dashboard",
"homepage": "Homepage"
```

After:
```json
"dashboard": "Tableau de bord",
"homepage": "Page d'accueil"
```

### Placeholders - Keep These!

Some text has special parts in `{curly braces}` that get filled in automatically. **Keep these exactly as they are!**

| You See | What It Means | Your Translation |
|---------|---------------|------------------|
| `"Hello, {name}!"` | `{name}` becomes the user's name | `"Bonjour, {name}!"` ✓ |
| `"You have {count} items"` | `{count}` becomes a number | `"Vous avez {count} articles"` ✓ |

**Wrong:** `"Bonjour, {nom}!"` ✗ (Don't translate the placeholder!)

### What NOT to Translate

- **Keys** (the part before the colon): `"dashboard"` stays `"dashboard"`
- **Brand names**: "Facet", "GitHub", "Google", etc.
- **Placeholders**: `{name}`, `{count}`, `{link}`, etc.

---

## For Developers (Technical Setup)

If you prefer working locally with git and want to test your translations:

### Adding a New Language

1. **Copy the English source file:**
   ```bash
   cp frontend/src/locales/en.json frontend/src/locales/XX.json
   ```
   Replace `XX` with your [ISO 639-1 language code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes).

2. **Translate the values** in your new file (keep keys in English).

3. **Register your locale** in `frontend/src/lib/i18n/index.ts`:
   ```typescript
   register('XX', () => import('../../locales/XX.json'));
   ```

4. **Add to supported locales** in the same file:
   ```typescript
   export const SUPPORTED_LOCALES = ['en', 'de', 'elvish', 'klingon', 'lolcat', 'XX'] as const;
   ```

5. **Add the display name**:
   ```typescript
   export const LOCALE_NAMES: Record<string, string> = {
     // ... existing locales
     XX: 'Language Name'
   };
   ```

6. **Submit a Pull Request**.

### Validation

Before submitting, validate your translations:

```bash
cd frontend
npm run i18n:validate
```

This checks for missing keys, extra keys, and shows coverage percentage.

### Testing Locally

```bash
cd frontend
npm install
npm run dev
```

Then change the language in Settings to see your translations.

---

## Translation Guidelines

### Do

- Use natural phrasing for your language (don't translate too literally)
- Preserve the same JSON structure as `en.json`
- Keep all placeholders like `{name}`, `{count}`, `{link}`
- Include context in your PR description for ambiguous translations
- Ask on Discord if you're unsure about context!

### Don't

- Change the keys (the part before the colon)
- Remove or modify placeholders
- Translate brand names
- Use machine translation without careful review

---

## Current Languages

| Code | Language | Status |
|------|----------|--------|
| `en` | English | Source (complete) |
| `de` | Deutsch (German) | Complete |
| `elvish` | Sindarin (Elvish) | Complete (fun!) |
| `klingon` | tlhIngan Hol (Klingon) | Complete (fun!) |
| `lolcat` | LOLcat | Complete (fun!) |

**Want to add a new language?** We'd love to have: Spanish, French, Japanese, Korean, Portuguese, Chinese, and more!

---

## Language Codes Reference

Use standard ISO 639-1 codes. For regional variants, add the country code:

| Code | Language |
|------|----------|
| `es` | Spanish |
| `fr` | French |
| `ja` | Japanese |
| `ko` | Korean |
| `pt-BR` | Portuguese (Brazil) |
| `zh-CN` | Chinese (Simplified) |
| `zh-TW` | Chinese (Traditional) |
| `it` | Italian |
| `ru` | Russian |
| `ar` | Arabic |

---

## Thank You!

All translation contributors are credited in release notes. You're helping make Facet accessible to more people around the world!

Questions? Ask on **Discord** or open a **GitHub issue** with the `i18n` label.
