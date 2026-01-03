# AI Writing Assistant - Integration Guide

Quick reference for adding AI writing assistance to admin forms.

---

## Quick Start

### 1. Import the Component

```svelte
<script lang="ts">
  import AIContentHelper from '$components/admin/AIContentHelper.svelte';

  let description = '';
  let title = '';
  let company = '';
</script>
```

### 2. Add to Your Form

```svelte
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="description">Description</label>
    <AIContentHelper
      fieldType="description"
      content={description}
      context={{ title, company }}
      on:apply={(e) => (description = e.detail.content)}
    />
  </div>
  <textarea id="description" bind:value={description} class="input" />
</div>
```

That's it! The AI Assistant button will appear when an AI provider is configured.

---

## Field Types

Choose the appropriate `fieldType` for optimal prompts:

| Field Type | Use For | Output Format |
|------------|---------|---------------|
| `headline` | Profile headlines, taglines | Single line, <100 chars |
| `summary` | Profile summaries, bios | 2-4 sentences |
| `description` | Experience, projects, education | 1-3 paragraphs |
| `bullets` | Achievements, key points | Bullet list with • |
| `content` | Blog posts, long-form | Multiple paragraphs |

---

## Context Best Practices

Provide relevant context to improve AI quality:

### Experience Forms
```svelte
context={{
  role: title,
  company,
  location,
  start_date: startDate,
  end_date: endDate
}}
```

### Project Forms
```svelte
context={{
  title,
  tech_stack: techStack.join(', '),
  categories: categories.join(', ')
}}
```

### Education Forms
```svelte
context={{
  degree,
  field,
  institution,
  year: graduationYear
}}
```

### Posts/Blog Forms
```svelte
context={{
  title,
  tags: tags.join(', '),
  audience: 'technical' // or 'general', 'executive'
}}
```

---

## Integration Examples

### Experience Page (DONE ✅)

Location: `/frontend/src/routes/admin/experience/+page.svelte`

```svelte
<!-- Description field -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="description" class="label mb-0">Description</label>
    <AIContentHelper
      fieldType="description"
      content={description}
      context={{ role: title, company, location }}
      on:apply={(e) => (description = e.detail.content)}
    />
  </div>
  <textarea
    id="description"
    bind:value={description}
    class="input min-h-[100px]"
    placeholder="Brief overview of your role..."
  />
</div>

<!-- Bullets field -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="bullets" class="label mb-0">Key Achievements</label>
    <AIContentHelper
      fieldType="bullets"
      content={bulletsText}
      context={{ role: title, company, description }}
      on:apply={(e) => (bulletsText = e.detail.content)}
    />
  </div>
  <textarea
    id="bullets"
    bind:value={bulletsText}
    class="input min-h-[120px]"
    placeholder="One achievement per line..."
  />
</div>
```

### Projects Page (TODO)

Location: `/frontend/src/routes/admin/projects/+page.svelte`

```svelte
<!-- Summary field -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="summary">Summary</label>
    <AIContentHelper
      fieldType="summary"
      content={summary}
      context={{
        title,
        tech_stack: techStack.join(', '),
        role: userRole
      }}
      on:apply={(e) => (summary = e.detail.content)}
    />
  </div>
  <textarea id="summary" bind:value={summary} class="input" />
</div>

<!-- Description field -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="description">Full Description</label>
    <AIContentHelper
      fieldType="description"
      content={description}
      context={{
        title,
        summary,
        tech_stack: techStack.join(', ')
      }}
      on:apply={(e) => (description = e.detail.content)}
    />
  </div>
  <textarea id="description" bind:value={description} class="input min-h-[200px]" />
</div>
```

### Profile Page (TODO)

Location: `/frontend/src/routes/admin/profile/+page.svelte`

```svelte
<!-- Headline -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="headline">Professional Headline</label>
    <AIContentHelper
      fieldType="headline"
      content={headline}
      context={{
        name,
        current_role: title,
        years_experience: yearsExp
      }}
      on:apply={(e) => (headline = e.detail.content)}
    />
  </div>
  <input id="headline" bind:value={headline} class="input" />
</div>

<!-- Summary -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="summary">Professional Summary</label>
    <AIContentHelper
      fieldType="summary"
      content={summary}
      context={{
        name,
        headline,
        specialties: skills.slice(0, 5).join(', ')
      }}
      on:apply={(e) => (summary = e.detail.content)}
    />
  </div>
  <textarea id="summary" bind:value={summary} class="input min-h-[120px]" />
</div>
```

### Posts/Blog Page (TODO)

Location: `/frontend/src/routes/admin/posts/+page.svelte`

```svelte
<!-- Excerpt -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="excerpt">Excerpt</label>
    <AIContentHelper
      fieldType="summary"
      content={excerpt}
      context={{
        title,
        tags: tags.join(', ')
      }}
      on:apply={(e) => (excerpt = e.detail.content)}
    />
  </div>
  <textarea id="excerpt" bind:value={excerpt} class="input" />
</div>

<!-- Content -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="content">Content</label>
    <AIContentHelper
      fieldType="content"
      content={content}
      context={{
        title,
        excerpt,
        tags: tags.join(', ')
      }}
      on:apply={(e) => (content = e.detail.content)}
    />
  </div>
  <textarea id="content" bind:value={content} class="input min-h-[400px]" />
</div>
```

### Education Page (TODO)

Location: `/frontend/src/routes/admin/education/+page.svelte`

```svelte
<!-- Description -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="description">Description</label>
    <AIContentHelper
      fieldType="description"
      content={description}
      context={{
        degree,
        field,
        institution,
        graduation_year: gradYear
      }}
      on:apply={(e) => (description = e.detail.content)}
    />
  </div>
  <textarea id="description" bind:value={description} class="input" />
</div>
```

### Talks Page (TODO)

Location: `/frontend/src/routes/admin/talks/+page.svelte`

```svelte
<!-- Description -->
<div>
  <div class="flex items-center justify-between mb-2">
    <label for="description">Description</label>
    <AIContentHelper
      fieldType="description"
      content={description}
      context={{
        title,
        event,
        date: talkDate,
        location
      }}
      on:apply={(e) => (description = e.detail.content)}
    />
  </div>
  <textarea id="description" bind:value={description} class="input" />
</div>
```

---

## View Editor Integration (Advanced)

For view-specific overrides in `/admin/views/[id]/`:

```svelte
<!-- When editing item overrides -->
{#if editingItem}
  <div>
    <div class="flex items-center justify-between mb-2">
      <label>Override Description</label>
      <AIContentHelper
        fieldType="description"
        content={overrideDescription}
        context={{
          original_title: editingItem.title,
          view_name: viewName,
          view_purpose: viewPurpose // e.g., 'resume', 'portfolio', 'personal'
        }}
        on:apply={(e) => (overrideDescription = e.detail.content)}
      />
    </div>
    <textarea bind:value={overrideDescription} class="input" />
  </div>
{/if}
```

---

## Styling Customization

The component uses Tailwind classes. Customize size and placement:

### Small Size (for tight spaces)
```svelte
<AIContentHelper
  size="sm"
  fieldType="description"
  content={description}
  on:apply={(e) => (description = e.detail.content)}
/>
```

### Disabled State
```svelte
<AIContentHelper
  disabled={!description || loading}
  fieldType="description"
  content={description}
  on:apply={(e) => (description = e.detail.content)}
/>
```

---

## Testing Checklist

When integrating AI Assistant into a new form:

- [ ] Import `AIContentHelper` component
- [ ] Add button next to field label
- [ ] Set correct `fieldType`
- [ ] Provide relevant `context` fields
- [ ] Wire up `on:apply` event handler
- [ ] Test with empty content (should work)
- [ ] Test with existing content (should improve)
- [ ] Try all 5 tones (executive, professional, technical, conversational, creative)
- [ ] Test critique mode
- [ ] Verify preview modal shows correctly
- [ ] Confirm "Apply Changes" updates the field
- [ ] Confirm "Cancel" keeps original content

---

## Troubleshooting

### Button Doesn't Appear

**Check:**
1. Is an AI provider configured and active? (Settings > AI Providers)
2. Is the import correct? `import AIContentHelper from '$components/admin/AIContentHelper.svelte'`
3. Browser console for errors?

### AI Returns Poor Quality

**Improve by:**
1. Adding more context fields
2. Using a better model (Claude Sonnet > GPT-4o-mini)
3. Providing better initial content
4. Trying different tones

### Preview Modal Issues

**Common fixes:**
1. Check z-index conflicts (modal is z-50)
2. Ensure parent doesn't have `overflow: hidden`
3. Check dark mode compatibility

---

## Component API Reference

### Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `content` | `string` | `''` | Current field content |
| `fieldType` | `'headline' \| 'summary' \| 'description' \| 'bullets' \| 'content'` | `'description'` | Type of content |
| `context` | `Record<string, string>` | `{}` | Additional context for AI |
| `disabled` | `boolean` | `false` | Disable the button |
| `size` | `'sm' \| 'md'` | `'md'` | Button size |

### Events

| Event | Payload | Description |
|-------|---------|-------------|
| `apply` | `{ content: string }` | User clicked "Apply Changes" |

---

## Rollout Plan

### Phase 1: Core Content (✅ Done)
- [x] Experience: description, bullets

### Phase 2: Projects & Profile
- [ ] Projects: summary, description
- [ ] Profile: headline, summary

### Phase 3: Secondary Content
- [ ] Education: description
- [ ] Talks: description
- [ ] Posts: excerpt, content

### Phase 4: Advanced
- [ ] View overrides
- [ ] Custom sections
- [ ] Bulk operations

---

## Performance Considerations

**AI API call timings:**
- Typical response: 2-5 seconds
- Max timeout: 60 seconds
- Retry strategy: None (user must retry manually)

**Best practices:**
- Don't make AI calls on every keystroke
- Show loading states clearly
- Allow users to cancel requests (future enhancement)
- Cache results per session (future enhancement)

---

**Last Updated:** 2026-01-03
**Status:** In Progress - Phase 1 Complete
