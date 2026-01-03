# AI Writing Assistant

**Status:** Implemented
**Last Updated:** 2026-01-03

---

## Overview

The AI Writing Assistant provides intelligent content rewriting and feedback across all text fields in Facet's admin interface. It helps users craft professional, impactful portfolio content with multiple tone options and inline critique functionality.

## Features

### 1. **Multi-Tone Rewriting**

Transform your content with five distinct professional tones:

#### 👔 Executive
- **Best for:** C-suite resumes, leadership roles, strategic positions
- **Style:** Formal, authoritative, business-outcome focused
- **Example:**
  - Before: "Worked on improving team processes"
  - After: "Directed cross-functional initiative to streamline operations, resulting in 30% reduction in cycle time and $2M annual savings"

#### 💼 Professional (Default)
- **Best for:** Standard resumes, LinkedIn profiles, general portfolios
- **Style:** Balanced, achievement-focused, industry-appropriate
- **Example:**
  - Before: "Built some features for the app"
  - After: "Developed core product features including real-time notifications and advanced search, improving user engagement by 45%"

#### ⚙️ Technical
- **Best for:** Developer portfolios, engineering roles, technical blogs
- **Style:** Technology-focused, precise, methodology-driven
- **Example:**
  - Before: "Made the app faster"
  - After: "Implemented distributed caching using Redis and optimized database queries with indexed column strategies, reducing API latency from 800ms to 120ms"

#### 💬 Conversational
- **Best for:** Personal websites, creative portfolios, about pages
- **Style:** Approachable, human, first-person friendly
- **Example:**
  - Before: "Responsible for UI development"
  - After: "I designed and built the entire user interface using React and Tailwind, focusing on accessibility and mobile-first design"

#### 🎨 Creative
- **Best for:** Design portfolios, agencies, storytelling projects
- **Style:** Engaging, descriptive, impact-focused
- **Example:**
  - Before: "Created a new dashboard"
  - After: "Transformed a complex data landscape into an intuitive visual story, helping users discover insights 3x faster through thoughtful information architecture"

### 2. **AI Critique Mode** 💭

Get constructive inline feedback without rewriting:

```
Original Input:
"Worked on improving the application's performance using various techniques"

AI Feedback:
"Worked on [Weak verb - try 'optimized' or 'redesigned'] improving the application's performance [By how much? 2x? 50% faster?] using various techniques [Too vague - name specific techniques like caching, query optimization, etc.]"
```

**Critique focuses on:**
- Identifying vague language
- Requesting quantification
- Flagging buzzwords and AI-sounding phrases
- Suggesting stronger verbs
- Noting missing context

### 3. **Anti-AI Writing Rules**

All tones enforce strict "don't sound like AI" guidelines:

**Banned Words/Phrases:**
- ❌ delve, leverage, utilize, spearheaded
- ❌ synergy, cutting-edge, comprehensive, robust
- ❌ streamline, optimize, revolutionize, game-changing
- ❌ state-of-the-art, paradigm, holistic, seamless

**Banned Punctuation:**
- ❌ Em dashes (—) → Use commas, periods, or "and"

**Enforced Best Practices:**
- ✅ Active voice over passive
- ✅ Specific verbs over generic ones
- ✅ Quantified achievements
- ✅ Varied sentence structure
- ✅ Concrete details over abstractions

---

## Where to Use

The AI Writing Assistant appears on these admin pages:

### High-Value Fields
1. **Experience** (`/admin/experience`)
   - Description field
   - Key achievements/bullets

2. **Projects** (`/admin/projects`)
   - Summary field
   - Full description

3. **Profile** (`/admin/profile`)
   - Headline
   - Professional summary

4. **Education** (`/admin/education`)
   - Description field

5. **View Editors** (`/admin/views/[id]`)
   - Item override descriptions

### Medium-Value Fields
6. **Posts** (`/admin/posts`)
   - Excerpt
   - Content (for blog posts)

7. **Talks** (`/admin/talks`)
   - Description

---

## How to Use

### Basic Workflow

1. **Enter or paste your content** into any text field
2. **Click the "AI Assistant" button** (appears to the right of field labels)
3. **Choose your action:**
   - **Rewrite tab:** Select a tone (Executive, Professional, Technical, etc.)
   - **Get Feedback tab:** Click to receive inline critique
4. **Review the preview:**
   - For rewrites: See side-by-side comparison
   - For critiques: See original with [bracketed feedback]
5. **Apply or cancel:**
   - Click "Apply Changes" to replace your content
   - Click "Cancel" to keep the original

### Context-Aware Prompts

The AI automatically uses context from other fields:

**Example: Experience Description**
- Uses: Job title, company name, location
- Prompt includes: "Context: role: Senior Engineer, company: Acme Inc, location: SF"

**Example: Project Description**
- Uses: Project title, tech stack, categories
- Helps AI understand domain and technical level

### Tips for Best Results

1. **Provide some starting content** - AI rewrites work better with a base to improve
2. **Fill in context fields first** - Job title, company, etc. help AI understand the domain
3. **Try different tones** - See which voice fits your target audience
4. **Use critique for learning** - Understand what makes content stronger
5. **Iterate** - You can rewrite multiple times with different tones
6. **Edit AI output** - Always review and personalize AI suggestions

---

## Configuration

### Setup

AI Writing Assistant requires an active AI provider configured in Settings:

1. Go to **Settings > AI Providers**
2. Add a provider:
   - **OpenAI:** Requires API key, uses `gpt-4o` or `gpt-4o-mini`
   - **Anthropic:** Requires API key, uses `claude-sonnet-4-20250514`
   - **Ollama:** Local installation, no API key needed
3. Mark one provider as **default**
4. Test the connection

**Environment Auto-Configuration:**

Set environment variables to auto-configure providers:

```bash
# Anthropic (recommended)
ANTHROPIC_API_KEY=sk-ant-...

# OpenAI
OPENAI_API_KEY=sk-...

# Ollama (local)
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.2
```

### Cost Considerations

**Typical Usage Costs (per rewrite):**
- OpenAI GPT-4o-mini: ~$0.001-0.003
- Anthropic Claude Sonnet: ~$0.003-0.008
- Ollama (local): Free (uses local compute)

**Recommendations:**
- **Budget-conscious:** Use GPT-4o-mini or local Ollama
- **Best quality:** Claude Sonnet 4 or GPT-4o
- **Privacy-focused:** Run Ollama locally

---

## API Reference

### POST `/api/ai/rewrite`

Rewrite content with a specific tone.

**Request:**
```json
{
  "content": "Your original text here",
  "field_type": "description",
  "context": {
    "role": "Software Engineer",
    "company": "Acme Inc",
    "location": "San Francisco"
  },
  "tone": "professional"
}
```

**Response:**
```json
{
  "content": "Rewritten professional content...",
  "tone": "professional",
  "provider": "Claude (Auto)"
}
```

**Tones:** `executive`, `professional`, `technical`, `conversational`, `creative`

---

### POST `/api/ai/critique`

Get inline feedback on content.

**Request:**
```json
{
  "content": "Your text to critique",
  "field_type": "description",
  "context": {
    "role": "Product Manager",
    "company": "TechCorp"
  }
}
```

**Response:**
```json
{
  "content": "Your text [with inline feedback in brackets]",
  "provider": "Claude (Auto)"
}
```

---

## Component Usage

### AIContentHelper Component

```svelte
<script>
  import AIContentHelper from '$components/admin/AIContentHelper.svelte';

  let description = '';
</script>

<div>
  <div class="flex items-center justify-between mb-2">
    <label for="description">Description</label>
    <AIContentHelper
      fieldType="description"
      content={description}
      context={{ role: 'Engineer', company: 'Acme' }}
      on:apply={(e) => (description = e.detail.content)}
    />
  </div>
  <textarea id="description" bind:value={description} />
</div>
```

**Props:**
- `content` (string) - Current field content
- `fieldType` (string) - One of: `headline`, `summary`, `description`, `bullets`, `content`
- `context` (object) - Additional context for AI (role, company, etc.)
- `disabled` (boolean) - Disable the button
- `size` (`'sm' | 'md'`) - Button size

**Events:**
- `on:apply` - Fired when user applies AI changes, receives `{ content: string }`

---

## Prompt Engineering

### Rewrite Prompt Structure

1. **Style Rules** - Anti-AI guidelines (no buzzwords, no em-dashes, etc.)
2. **Tone Instructions** - Specific guidance for selected tone
3. **Context** - Job title, company, etc. from form fields
4. **Field Type** - Special instructions for headlines, bullets, etc.
5. **Content** - The actual text to rewrite

### Critique Prompt Structure

1. **Task Definition** - Return original text with [bracketed feedback]
2. **Feedback Guidelines** - Be specific, actionable, brief
3. **Example** - Show desired format
4. **Context** - Related fields for domain understanding
5. **Content** - The text to critique

---

## Troubleshooting

### AI Assistant Button Not Appearing

**Causes:**
1. No AI provider configured
2. AI provider is inactive
3. JavaScript error in component

**Solutions:**
1. Check Settings > AI Providers
2. Ensure at least one provider is active and marked as default
3. Check browser console for errors

### "AI request failed" Error

**Common Issues:**
1. **Invalid API Key** - Verify key in Settings > AI Providers
2. **Rate Limit** - Wait a few minutes, then retry
3. **Network Error** - Check internet connection
4. **Model Not Available** - Update model name in provider settings

### Poor Quality Rewrites

**Improvements:**
1. **Add more context** - Fill in all relevant fields (title, company, etc.)
2. **Provide better input** - Give AI more to work with initially
3. **Try different tones** - Some tones work better for different content
4. **Switch providers** - Claude Sonnet often produces better results than GPT-4o-mini
5. **Use critique mode** - Understand what's weak, then rewrite manually

### Critique Not Showing Inline Feedback

**Possible Issues:**
1. Content too short (AI may not have much to say)
2. Content already well-written (AI agrees it's good!)
3. AI misunderstood the task

**Solutions:**
- Try the rewrite feature instead
- Manually edit based on general writing best practices
- Try a different AI provider

---

## Privacy & Security

### Data Handling

**What gets sent to AI providers:**
- ✅ Text content from the field
- ✅ Context fields (title, company, location)
- ❌ No user identity information
- ❌ No other portfolio content

**Data retention:**
- OpenAI/Anthropic: Follow their data retention policies
- Ollama (local): No data leaves your server

**Recommendations:**
- Use Ollama for maximum privacy
- Review AI provider's terms of service
- Don't include sensitive information in content

### API Key Security

**Storage:**
- API keys are encrypted at rest using AES-256
- Keys never sent to frontend
- Decrypted only during AI API calls

**Best Practices:**
- Use read-only or restricted API keys when possible
- Rotate keys periodically
- Monitor usage in your AI provider dashboard
- Revoke keys immediately if compromised

---

## Future Enhancements

**Planned Features:**
1. Custom tone creation (user-defined style guides)
2. Batch rewriting (apply tone to all descriptions at once)
3. Before/after history (see all AI rewrites)
4. Tone detection (analyze existing content's tone)
5. Length targeting ("make this 50 words")
6. Audience targeting ("optimize for recruiters" vs "optimize for clients")
7. A/B testing (which tone performs better on views)

**Community Requests:**
- [ ] Save favorite tone per field type
- [ ] Undo/redo for AI changes
- [ ] Compare multiple tone outputs side-by-side
- [ ] Export prompts for external use

---

## Examples

### Example 1: Experience Description

**Original:**
```
Worked on various projects related to the company's web platform.
Used JavaScript and other technologies.
```

**Professional Tone:**
```
Developed and maintained core features for the company's web platform,
serving 50K+ daily users. Built responsive interfaces using React and
TypeScript, improving page load times by 35% through code splitting
and lazy loading techniques.
```

**Technical Tone:**
```
Architected and implemented features for the company's production web
platform using React 18, TypeScript, and Next.js 13. Optimized bundle
size from 2.3MB to 890KB through tree shaking, code splitting, and
migration to ES modules. Integrated with RESTful APIs and implemented
client-side caching using React Query.
```

### Example 2: Project Summary

**Original:**
```
A mobile app I built for tracking fitness activities
```

**Creative Tone:**
```
Designed and built a fitness companion that transforms raw workout data
into meaningful insights. Users can track runs, cycles, and gym sessions
while visualizing progress through beautiful charts and personalized goals.
```

**Executive Tone:**
```
Delivered a mobile fitness tracking application achieving 10K downloads
in first month. Product drove 40% improvement in user workout consistency
through gamification and social features, validated by 4.7 App Store rating.
```

---

## References

- [OpenAI API Documentation](https://platform.openai.com/docs)
- [Anthropic API Documentation](https://docs.anthropic.com/)
- [Ollama Documentation](https://ollama.ai/docs)
- [Writing Without Buzzwords Guide](https://www.plainlanguage.gov/)
- [Resume Action Verbs](https://hls.harvard.edu/dept/opia/job-search-toolkit/cover-letters-and-resumes/action-verbs/)

---

**Last Updated:** 2026-01-03
**Feature Status:** ✅ Implemented and Active
