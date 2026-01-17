# Setup Wizard Plan

**Status:** Planned  
**Priority:** High  
**Created:** 2026-01-17  
**Related:** Phase 17.1 in ROADMAP.md

---

## Executive Summary

A **modal overlay wizard** that guides first-time users through initial setup without any routing changes. This approach was chosen over server-authoritative routing guards to minimize architectural risk while achieving 80% of the UX benefit.

### Why Modal Over Routing?

| Approach | Risk | Complexity | UX Benefit |
|----------|------|------------|------------|
| Server-authoritative routing | HIGH | Touches hooks.server.ts (global) | 100% |
| Modal overlay wizard | LOW | Isolated UI components | 80% |

Previous attempts at routing-based onboarding caused race conditions. The modal approach:
- Zero routing changes
- Zero risk of breaking existing flow
- Users can still navigate admin freely during setup
- Progressive disclosure (setup appears when relevant)

---

## User Experience Design

### Primary User Journey: "Progressive Disclosure Setup"

**Principle:** Start minimal, expand based on user engagement

```
Dashboard Load → Detect First-Time → Show "Setup Suggestion" → User Choice
    ↓
[Skip Forever] ← User Choice → [Quick Setup: 2-3 minutes]
    ↓                              ↓
Skip stored forever         Modal Wizard Opens
```

### Wizard Philosophy: "Facets First"

**Core insight:** Don't just collect data - teach users about why Facets (Views) matter.

---

## Trigger Logic & States

### User States Matrix

```typescript
interface SetupState {
  hasBasicProfile: boolean;    // name, title, summary exist
  hasViews: boolean;           // at least one view created  
  hasContent: boolean;         // any experience/projects/posts
  isFirstVisit: boolean;       // localStorage flag
  dismissedPermanently: boolean; // localStorage flag
  completedWizard: boolean;    // stored in user prefs
}
```

### Trigger Decision Tree

```
if (demoMode) → never show wizard
else if (dismissedPermanently) → never show 
else if (completedWizard) → never show
else if (!hasBasicProfile || !hasViews) → show wizard suggestion
else → show optional "finish setup" in dashboard
```

---

## Modal Design & Architecture

### Visual Design

```svelte
<!-- Wizard Container -->
<div class="fixed inset-0 bg-black/50 backdrop-blur-sm z-50">
  <div class="flex items-center justify-center min-h-screen p-4">
    <!-- Wizard Card -->
    <div class="card w-full max-w-2xl max-h-[90vh] overflow-y-auto">
      
      <!-- Header with Progress -->
      <header class="border-b px-6 py-4">
        <div class="flex items-center justify-between">
          <h1 class="text-lg font-semibold">Quick Setup</h1>
          <button class="btn-ghost btn-sm" on:click={handleSkip}>
            Skip setup
          </button>
        </div>
        <!-- Step Progress Bar (like import page) -->
        <ProgressSteps current={step} total={3} />
      </header>
      
      <!-- Dynamic Content -->
      <main class="p-6">
        {#if step === 1}<Step1BasicProfile />{/if}
        {#if step === 2}<Step2CreateView />{/if}
        {#if step === 3}<Step3ReviewAndLaunch />{/if}
      </main>
      
      <!-- Footer Actions -->
      <footer class="border-t px-6 py-4 flex justify-between">
        <button 
          class="btn-ghost" 
          disabled={step === 1}
          on:click={previousStep}
        >
          Back
        </button>
        <button class="btn-primary" on:click={nextStep}>
          {step === 3 ? 'Launch Profile' : 'Continue'}
        </button>
      </footer>
    </div>
  </div>
</div>
```

### Mobile-First Considerations

- Modal takes 95% screen width on mobile
- Steps stack vertically on small screens
- Touch-friendly button sizing (44px minimum)
- Swipe gestures for step navigation (optional)
- Proper keyboard navigation

---

## Wizard Steps Design

### Step 1: "Tell us about yourself" (60 seconds)

**Goal:** Capture the absolute minimum for a working profile

```svelte
<div class="space-y-6">
  <div class="text-center">
    <h2 class="text-xl font-semibold mb-2">Let's start with the basics</h2>
    <p class="text-gray-600">This creates your foundation profile</p>
  </div>
  
  <div class="space-y-4">
    <div>
      <label class="label" for="name">Full Name</label>
      <input class="input" id="name" bind:value={profile.name} placeholder="Jane Smith" />
    </div>
    
    <div>
      <label class="label" for="title">Professional Title</label>
      <input class="input" id="title" bind:value={profile.title} placeholder="Software Engineer" />
    </div>
    
    <div>
      <label class="label" for="summary">Brief Summary (2-3 sentences)</label>
      <textarea 
        class="input" 
        id="summary" 
        bind:value={profile.summary} 
        placeholder="I'm a passionate developer who loves building user-friendly applications..."
        rows="3"
      />
      <!-- AI Helper Integration -->
      {#if profile.title}
        <AIImproveButton 
          content={profile.summary} 
          context="Professional summary for {profile.title}"
          on:improved={(e) => profile.summary = e.detail}
        />
      {/if}
    </div>
  </div>
</div>
```

### Step 2: "Create your first Facet" (90 seconds)

**Goal:** Teach the core concept while creating something useful

```svelte
<div class="space-y-6">
  <div class="text-center">
    <h2 class="text-xl font-semibold mb-2">Create your first view</h2>
    <p class="text-gray-600">Different audiences see different sides of you</p>
  </div>
  
  <!-- View Type Selection -->
  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    {#each viewTemplates as template}
      <button 
        class="p-4 border-2 rounded-lg text-left transition-colors"
        class:border-primary-500={selectedTemplate === template.id}
        on:click={() => selectTemplate(template.id)}
      >
        <div class="flex items-center space-x-3">
          <div class="text-2xl">{template.icon}</div>
          <div>
            <h3 class="font-medium">{template.name}</h3>
            <p class="text-sm text-gray-600">{template.description}</p>
          </div>
        </div>
      </button>
    {/each}
  </div>
  
  <!-- View Name Input -->
  <div>
    <label class="label" for="viewName">View Name</label>
    <input 
      class="input" 
      id="viewName" 
      bind:value={newView.name}
      placeholder={selectedTemplate?.suggestedName}
    />
  </div>
  
  <!-- Visibility Preview -->
  <div class="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
    <h4 class="font-medium text-blue-900 dark:text-blue-100 mb-2">Who can see this?</h4>
    <div class="text-sm text-blue-700 dark:text-blue-300">
      <VisibilityBadge visibility={newView.visibility} />
      {getVisibilityDescription(newView.visibility)}
    </div>
  </div>
</div>
```

**View Templates:**

```typescript
const viewTemplates = [
  {
    id: 'recruiter',
    name: 'For Recruiters',
    icon: '👔',
    description: 'Professional focus, skills, experience',
    suggestedName: 'Recruiter View',
    visibility: 'public',
    sections: ['experience', 'skills', 'education', 'contact']
  },
  {
    id: 'portfolio',
    name: 'Portfolio',
    icon: '🎨',
    description: 'Creative work, projects, visual focus',
    suggestedName: 'Portfolio',
    visibility: 'public', 
    sections: ['projects', 'skills', 'about']
  },
  {
    id: 'consulting', 
    name: 'For Clients',
    icon: '🤝',
    description: 'Expertise, case studies, testimonials',
    suggestedName: 'Consulting',
    visibility: 'unlisted',
    sections: ['projects', 'experience', 'testimonials']
  },
  {
    id: 'personal',
    name: 'Personal',
    icon: '🙋',
    description: 'Broader interests, full story',
    suggestedName: 'About Me', 
    visibility: 'public',
    sections: ['about', 'projects', 'posts', 'talks']
  }
];
```

### Step 3: "Review & Launch" (30 seconds)

**Goal:** Show the immediate value and next steps

```svelte
<div class="space-y-6">
  <div class="text-center">
    <h2 class="text-xl font-semibold mb-2">🎉 You're ready to go!</h2>
    <p class="text-gray-600">Here's what you've created</p>
  </div>
  
  <!-- Preview Card -->
  <div class="border-2 border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20 rounded-lg p-6">
    <div class="flex items-start space-x-4">
      <div class="bg-gradient-to-br from-primary-500 to-primary-600 w-12 h-12 rounded-full flex items-center justify-center text-white font-semibold">
        {profile.name?.[0] || 'J'}
      </div>
      <div class="flex-1">
        <h3 class="font-semibold text-lg">{profile.name}</h3>
        <p class="text-gray-600 dark:text-gray-400">{profile.title}</p>
        <p class="text-sm text-gray-500 mt-1">{profile.summary}</p>
      </div>
    </div>
    
    <div class="mt-4 pt-4 border-t border-green-200 dark:border-green-800">
      <div class="flex items-center justify-between">
        <div>
          <p class="font-medium">{newView.name}</p>
          <VisibilityBadge visibility={newView.visibility} />
        </div>
        <a href="/api/view/{newView.slug}" class="btn-ghost btn-sm">
          Preview →
        </a>
      </div>
    </div>
  </div>
  
  <!-- Next Steps -->
  <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
    <h4 class="font-medium mb-3">Suggested next steps:</h4>
    <div class="space-y-2 text-sm">
      <div class="flex items-center space-x-2">
        <span class="text-blue-500">📁</span>
        <span>Add your first project or work experience</span>
      </div>
      <div class="flex items-center space-x-2">
        <span class="text-purple-500">🔗</span>
        <span>Share your {newView.name.toLowerCase()} with the world</span>
      </div>
      <div class="flex items-center space-x-2">
        <span class="text-orange-500">⚡</span>
        <span>Try importing from GitHub or uploading your resume</span>
      </div>
    </div>
  </div>
</div>
```

---

## Implementation Architecture

### Component Structure

```
SetupWizard.svelte (main modal)
├── WizardHeader.svelte (progress + skip)
├── WizardStep.svelte (generic step wrapper)
│   ├── Step1BasicProfile.svelte  
│   ├── Step2CreateView.svelte
│   └── Step3ReviewLaunch.svelte
├── WizardFooter.svelte (navigation)
└── TemplateSelector.svelte (reusable)
```

### State Management

```typescript
// lib/stores/setupWizard.ts
interface WizardState {
  isOpen: boolean;
  currentStep: number;
  profile: Partial<Profile>;
  newView: Partial<View>;
  selectedTemplate: string | null;
  canProceed: boolean[];
}

export const setupWizard = writable<WizardState>({
  isOpen: false,
  currentStep: 1,
  profile: {},
  newView: {},
  selectedTemplate: null,
  canProceed: [false, false, false]
});

// Validation logic
export const canProceedToStep = derived(
  setupWizard, 
  ($wizard) => ({
    step1: !!$wizard.profile.name && !!$wizard.profile.title,
    step2: !!$wizard.selectedTemplate && !!$wizard.newView.name,
    step3: true
  })
);
```

### Integration Points

```svelte
<!-- admin/+layout.svelte integration -->
{#if shouldShowSetupWizard}
  <SetupWizard />
{/if}

<script>
  import { setupWizard, shouldShowWizard } from '$lib/stores/setupWizard';
  
  let shouldShowSetupWizard = $state(false);
  
  // Check on mount and reactive updates
  $effect(() => {
    if (user && profile && views) {
      shouldShowSetupWizard = shouldShowWizard(user, profile, views);
    }
  });
</script>
```

---

## Edge Cases & Solutions

### Edge Case Matrix

| Scenario | Behavior | Implementation |
|----------|----------|----------------|
| User dismisses mid-step | Save progress, offer to resume | localStorage + step state |
| Multiple tabs open | Sync wizard state across tabs | BroadcastChannel API |
| Demo mode activated during wizard | Auto-close wizard, show success | Reactive demo state |
| Partial existing data | Pre-populate forms, skip completed sections | Data detection logic |
| Mobile landscape/portrait | Responsive modal sizing | CSS container queries |
| Slow network | Progressive enhancement, local state | Optimistic updates |
| JavaScript disabled | Graceful degradation to dashboard | Server-side detection |
| Back button pressed | Maintain wizard state | History API integration |
| Import started during wizard | Close wizard, redirect to import | Route change detection |

### Critical Edge Case: The "Demo Dance"

**Problem:** User starts wizard → enables demo → disables demo → expects wizard state

**Solution:**

```typescript
const handleDemoModeChange = (isDemoMode: boolean) => {
  if (isDemoMode) {
    // Store wizard state before closing
    sessionStorage.setItem('wizardState', JSON.stringify($setupWizard));
    setupWizard.update(state => ({ ...state, isOpen: false }));
  } else {
    // Restore wizard if it was interrupted by demo
    const savedState = sessionStorage.getItem('wizardState');
    if (savedState) {
      setupWizard.set(JSON.parse(savedState));
      sessionStorage.removeItem('wizardState');
    }
  }
};
```

---

## Accessibility Considerations

### WCAG 2.1 AA Compliance

- Focus trap within modal
- ESC key closes wizard (with confirmation)
- Screen reader announcements for step changes
- High contrast support
- Keyboard navigation for all interactive elements
- Alternative text for icons
- Form validation with clear error messaging

```svelte
<div 
  role="dialog" 
  aria-labelledby="wizard-title"
  aria-describedby="wizard-description"
  use:focusTrap
  use:escapeKey={handleEscape}
>
  <!-- Wizard content -->
</div>
```

---

## Performance Considerations

### Lazy Loading Strategy

```typescript
// Only load wizard components when needed
const SetupWizard = lazy(() => import('$components/admin/SetupWizard.svelte'));

// Preload on idle for faster UX  
let wizardPreloaded = false;
onIdle(() => {
  if (!wizardPreloaded) {
    import('$components/admin/SetupWizard.svelte');
    wizardPreloaded = true;
  }
});
```

### Memory Management

```typescript
// Clean up wizard state when completed
const completeWizard = () => {
  setupWizard.reset();
  localStorage.setItem('setup_completed', 'true');
  // Remove event listeners
  window.removeEventListener('beforeunload', saveProgress);
};
```

---

## Testing Strategy

### Unit Tests

- Wizard state transitions
- Form validation logic
- Template selection
- Data persistence

### Integration Tests

- Modal open/close behavior
- Step navigation
- Progress saving/restoration
- Demo mode interaction

### E2E Tests

```typescript
// Playwright test
test('Complete setup wizard flow', async ({ page }) => {
  await page.goto('/admin');
  
  // Should show wizard for new user
  await expect(page.locator('[data-testid="setup-wizard"]')).toBeVisible();
  
  // Step 1: Profile
  await page.fill('[data-testid="profile-name"]', 'John Doe');
  await page.fill('[data-testid="profile-title"]', 'Software Engineer');
  await page.click('[data-testid="wizard-next"]');
  
  // Step 2: View creation
  await page.click('[data-testid="template-recruiter"]');
  await page.click('[data-testid="wizard-next"]');
  
  // Step 3: Launch
  await page.click('[data-testid="wizard-launch"]');
  
  // Should close and show dashboard
  await expect(page.locator('[data-testid="setup-wizard"]')).not.toBeVisible();
  await expect(page.locator('[data-testid="dashboard"]')).toBeVisible();
});

test('Skip wizard permanently', async ({ page }) => {
  await page.goto('/admin');
  await page.click('[data-testid="wizard-skip"]');
  await page.click('[data-testid="confirm-skip"]');
  
  // Reload and verify wizard doesn't appear
  await page.reload();
  await expect(page.locator('[data-testid="setup-wizard"]')).not.toBeVisible();
});
```

### Accessibility Testing

```typescript
test('Wizard is accessible', async ({ page }) => {
  await page.goto('/admin');
  
  // Focus management
  await expect(page.locator('[data-testid="setup-wizard"]')).toBeFocused();
  
  // Keyboard navigation
  await page.keyboard.press('Tab');
  await page.keyboard.press('Enter'); // Should proceed to next step
  
  // ESC closes with confirmation
  await page.keyboard.press('Escape');
  await expect(page.locator('[data-testid="exit-confirmation"]')).toBeVisible();
});
```

---

## Rollout Plan

### Phase 1: Foundation (1-2 days)

- Create wizard component structure
- Implement basic state management
- Add trigger logic to admin layout

### Phase 2: Core Steps (2-3 days)

- Build and test each wizard step
- Implement form validation
- Add progress indication

### Phase 3: Polish & Integration (1-2 days)

- Accessibility improvements
- Mobile optimization
- Demo mode integration
- Edge case handling

### Phase 4: Testing & Refinement (1-2 days)

- E2E test coverage
- User testing feedback
- Performance optimization
- Analytics implementation

---

## Success Definition

The wizard succeeds if:

1. **95%+** of new users who see it can complete it in <3 minutes
2. **80%+** completion rate (don't abandon mid-flow)
3. **Zero** accessibility violations in automated testing
4. **No** performance regression in admin dashboard load time
5. Users who complete it add more content than those who skip

---

## Files to Create/Modify

### New Files

```
frontend/src/components/admin/SetupWizard.svelte
frontend/src/components/admin/wizard/
├── WizardHeader.svelte
├── WizardFooter.svelte
├── WizardStep.svelte
├── Step1BasicProfile.svelte
├── Step2CreateView.svelte
├── Step3ReviewLaunch.svelte
└── TemplateSelector.svelte
frontend/src/lib/stores/setupWizard.ts
frontend/tests/setup-wizard.spec.ts
```

### Modified Files

```
frontend/src/routes/admin/+layout.svelte  # Add wizard trigger
backend/migrations/                        # Add setup_completed field (optional)
```

---

## Summary

This modal wizard design provides:

✅ **Zero routing complexity** (main concern addressed)  
✅ **Professional, accessible UX** that teaches Facet concepts  
✅ **Comprehensive edge case handling** for robust operation  
✅ **Integration with existing patterns** (no architectural disruption)  
✅ **Progressive enhancement** (works without JS)  
✅ **Extensive testing strategy** to prevent regressions  

The approach turns the initial overwhelming "22 admin pages" into a gentle, educational introduction to Facet's core concept: **different audiences see different sides of you**.

---

*Document created: 2026-01-17*  
*Ready for implementation when prioritized*
