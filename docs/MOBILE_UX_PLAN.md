# Mobile UX Improvement Plan

**Status:** Planned  
**Priority:** High  
**Created:** 2026-01-17  
**Approach:** Incremental improvements, no breaking changes

---

## Executive Summary

Facet's public-facing pages have **excellent** mobile UX. The admin interface, however, has significant mobile usability issues that make managing content on phones frustrating. This plan outlines a phased approach to fix these issues without breaking desktop functionality or requiring a full rewrite.

### Current State Assessment

| Area | Mobile Quality | Notes |
|------|---------------|-------|
| Public profile views | ✅ Excellent | Mobile-first, responsive grid, proper touch targets |
| Public posts/talks/projects | ✅ Excellent | Typography scales, content stacks properly |
| Admin sidebar | ⚠️ Poor | Takes 20% of screen even when collapsed |
| Admin forms | ⚠️ Poor | Cramped inputs, hardcoded widths |
| Admin lists | ⚠️ Moderate | Touch targets too small, buttons clustered |
| View editor | ❌ Bad | Unusable on phones, too many controls per row |
| Modals | ⚠️ Moderate | Can exceed viewport, no mobile optimization |

---

## Critical Issues Identified

### 1. Admin Layout Architecture (HIGH PRIORITY)

**Problem:** The admin sidebar is always visible, even on mobile.

```svelte
<!-- Current: AdminSidebar.svelte -->
class="fixed left-0 top-16 ... {$adminSidebarOpen ? 'w-64' : 'w-16'}"

<!-- Current: +layout.svelte -->
<main class="... {$adminSidebarOpen ? 'ml-64' : 'ml-16'} ...">
```

**Impact:**
- On 375px screen: Sidebar takes 64px (17%) even when "collapsed"
- Main content only gets 311px horizontal space
- Combined with `p-6` (24px x 2), usable width drops to ~263px

**Solution:** Convert to overlay drawer pattern on mobile:
- Below `lg` (1024px): Sidebar is hidden by default
- Hamburger menu reveals sidebar as full-screen overlay
- Main content gets full width on mobile

### 2. Form Input Cramping (HIGH PRIORITY)

**Problem:** Hardcoded widths break on mobile.

```svelte
<!-- projects/+page.svelte line 567 -->
<select class="input w-32">  <!-- Fixed 128px width! -->
```

**Impact:**
- Link type selector only 128px wide
- URL input gets remaining space (often < 100px)
- Users can't see what they're typing

**Solution:**
- Replace `w-32` with `w-full sm:w-32`
- Stack link type + URL vertically on mobile
- Apply same pattern across all forms

### 3. Action Button Density (MEDIUM PRIORITY)

**Problem:** Multiple small buttons clustered together.

```
Desktop: [Edit] [Delete] [Publish] [Featured] [Move]
Mobile:  Same row, ~32px buttons, ~4px gaps
```

**Impact:**
- Touch targets below 44px minimum
- Easy to accidentally hit Delete instead of Edit
- Requires precision tap with fingertip

**Solution:**
- Use dropdown/overflow menu for secondary actions
- Keep only 1-2 primary actions visible
- Increase touch targets to 44px minimum

### 4. View Editor Complexity (HIGH PRIORITY)

**Problem:** Section headers contain too many controls.

```
Current per section row:
[Drag Handle] [Toggle] [Label] [Width: Full ▾] [Layout: List ▾] [Expand]
```

**Impact:**
- 6 interactive elements in ~300px horizontal space
- Controls overlap or wrap into unusable mess
- Drag handles too small for touch

**Solution:**
- Hide Width/Layout selectors behind expand
- Use swipe actions for common operations
- Larger drag handles with explicit "grip" icon

### 5. Modal Overflow (MEDIUM PRIORITY)

**Problem:** Modals can exceed viewport height.

**Impact:**
- Top/bottom of modal inaccessible
- No scroll within modal wrapper
- Especially bad in landscape orientation

**Solution:**
- Set `max-h-[90vh]` on modal content
- Add `overflow-y-auto` to modal body
- Use full-screen sheets on mobile for complex modals

### 6. Bulk Action Bar (LOW PRIORITY)

**Problem:** Bar uses `flex justify-between` with multiple buttons.

**Impact:**
- Buttons can overflow on narrow screens
- Count + buttons compete for limited space

**Solution:**
- Stack buttons on mobile
- Use icon-only buttons with tooltips
- Or collapse into dropdown

---

## Implementation Phases

### Phase 1: Admin Layout Refactor (3-4 days)

**Goal:** Make sidebar overlay on mobile, give content full width.

#### Files to Modify:

1. **`frontend/src/routes/admin/+layout.svelte`**
   - Remove `ml-64` / `ml-16` for screens < `lg`
   - Add overlay backdrop when sidebar open on mobile

2. **`frontend/src/components/admin/AdminSidebar.svelte`**
   - Change positioning to overlay on mobile
   - Add transition animation
   - Close on navigation (mobile only)
   - Add swipe-to-close gesture

3. **`frontend/src/components/admin/AdminHeader.svelte`**
   - Hamburger always visible on mobile
   - Update toggle behavior for overlay mode

4. **`frontend/src/lib/stores.ts`**
   - Add `isDesktop` derived store from window width
   - Sidebar behavior varies based on viewport

#### Implementation Details:

```svelte
<!-- New +layout.svelte pattern -->
<script>
  // Detect mobile
  let isMobile = $state(false);
  onMount(() => {
    const mq = window.matchMedia('(max-width: 1023px)');
    isMobile = mq.matches;
    mq.addEventListener('change', (e) => isMobile = e.matches);
  });
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
  <AdminHeader />
  
  <!-- Overlay backdrop (mobile only) -->
  {#if isMobile && $adminSidebarOpen}
    <button 
      class="fixed inset-0 bg-black/50 z-20"
      onclick={() => adminSidebarOpen.set(false)}
      aria-label="Close menu"
    />
  {/if}
  
  <AdminSidebar {isMobile} />
  
  <!-- No margin-left on mobile -->
  <main class="flex-1 p-4 lg:p-6 {isMobile ? '' : ($adminSidebarOpen ? 'lg:ml-64' : 'lg:ml-16')} transition-all mt-16">
    {@render children?.()}
  </main>
</div>
```

```svelte
<!-- New AdminSidebar.svelte pattern -->
<aside
  class="fixed top-16 h-[calc(100vh-4rem)] bg-white dark:bg-gray-800 border-r transition-all duration-200 z-30
    {isMobile 
      ? ($adminSidebarOpen ? 'left-0 w-64' : '-left-64 w-64') 
      : ($adminSidebarOpen ? 'left-0 w-64' : 'left-0 w-16')
    }"
>
```

### Phase 2: Touch Target & Button Improvements (2-3 days)

**Goal:** Make list actions touch-friendly.

#### Pattern: Overflow Menu for Secondary Actions

```svelte
<!-- Before: All buttons in a row -->
<div class="flex gap-1">
  <button class="p-2">Edit</button>
  <button class="p-2">Delete</button>
  <button class="p-2">Publish</button>
  <button class="p-2">Featured</button>
</div>

<!-- After: Primary action + overflow menu -->
<div class="flex items-center gap-2">
  <a href="/admin/projects/{item.id}" class="btn btn-sm btn-ghost">
    Edit
  </a>
  <OverflowMenu>
    <MenuItem onclick={togglePublish}>
      {item.is_draft ? 'Publish' : 'Unpublish'}
    </MenuItem>
    <MenuItem onclick={toggleFeatured}>
      {item.is_featured ? 'Unfeature' : 'Feature'}
    </MenuItem>
    <MenuDivider />
    <MenuItem onclick={confirmDelete} danger>
      Delete
    </MenuItem>
  </OverflowMenu>
</div>
```

#### New Component: OverflowMenu.svelte

```svelte
<script>
  let open = $state(false);
  let { children } = $props();
</script>

<div class="relative">
  <button 
    class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 min-w-[44px] min-h-[44px] flex items-center justify-center"
    onclick={() => open = !open}
    aria-label="More actions"
  >
    <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
      <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
    </svg>
  </button>
  
  {#if open}
    <div 
      class="absolute right-0 mt-1 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border z-50"
      use:clickOutside={() => open = false}
    >
      {@render children?.()}
    </div>
  {/if}
</div>
```

#### Files to Update:
- `frontend/src/routes/admin/projects/+page.svelte`
- `frontend/src/routes/admin/experience/+page.svelte`
- `frontend/src/routes/admin/posts/+page.svelte`
- `frontend/src/routes/admin/talks/+page.svelte`
- `frontend/src/routes/admin/skills/+page.svelte`
- `frontend/src/routes/admin/education/+page.svelte`
- `frontend/src/routes/admin/certifications/+page.svelte`
- `frontend/src/routes/admin/awards/+page.svelte`

### Phase 3: Form Layout Fixes (2 days)

**Goal:** Forms work properly on mobile.

#### Pattern: Responsive Flex Stacking

```svelte
<!-- Before -->
<div class="flex gap-2">
  <select class="input w-32">...</select>
  <input class="input flex-1" />
</div>

<!-- After -->
<div class="flex flex-col sm:flex-row gap-2">
  <select class="input w-full sm:w-32">...</select>
  <input class="input w-full sm:flex-1" />
</div>
```

#### Changes Required:

1. **Link inputs** (projects, talks, etc.)
   - Stack type selector + URL on mobile
   
2. **Date range inputs**
   - Stack start + end on mobile
   
3. **Search + filter bars**
   - Stack search + visibility filter
   
4. **Bullet point editors**
   - Full-width input + smaller delete button

5. **Reduce padding on mobile**
   - Change `p-6` to `p-4 lg:p-6` in cards

### Phase 4: View Editor Mobile Redesign (3-4 days)

**Goal:** Make view editor usable on mobile.

#### Current Problems:
1. Preview pane takes half the screen (useless on mobile)
2. Section headers have 6 controls in a row
3. Drag handles too small for touch
4. Item override editor is cramped

#### Solution: Mobile-First Editor Layout

```svelte
<!-- Desktop: Side-by-side -->
<div class="flex gap-6">
  <div class="flex-1"><!-- Editor --></div>
  <div class="w-96"><!-- Preview --></div>
</div>

<!-- Mobile: Stacked with collapsible preview -->
<div class="space-y-4">
  <div><!-- Editor --></div>
  <details class="lg:hidden">
    <summary class="btn btn-secondary w-full">Preview</summary>
    <div class="mt-4"><!-- Preview --></div>
  </details>
</div>
```

#### Section Header Simplification:

```svelte
<!-- Desktop: All controls visible -->
<div class="hidden lg:flex items-center gap-2">
  <DragHandle />
  <Toggle bind:checked={section.enabled} />
  <span>{section.label}</span>
  <select bind:value={section.width}>...</select>
  <select bind:value={section.layout}>...</select>
  <ExpandButton />
</div>

<!-- Mobile: Minimal + expandable -->
<div class="lg:hidden">
  <button class="flex items-center justify-between w-full p-3 touch-manipulation">
    <div class="flex items-center gap-3">
      <DragHandle class="w-8 h-8" /> <!-- Larger touch target -->
      <Toggle />
      <span>{section.label}</span>
    </div>
    <ChevronIcon class="w-5 h-5" />
  </button>
  
  {#if section.expanded}
    <div class="p-4 bg-gray-50 dark:bg-gray-900 space-y-3">
      <div>
        <label class="label">Width</label>
        <select class="input">{widthOptions}</select>
      </div>
      <div>
        <label class="label">Layout</label>
        <select class="input">{layoutOptions}</select>
      </div>
    </div>
  {/if}
</div>
```

### Phase 5: Modal Optimization (1-2 days)

**Goal:** Modals work on all screen sizes.

#### Pattern: Mobile Sheet

```svelte
<script>
  let isMobile = /* from media query */;
</script>

{#if isMobile}
  <!-- Full-screen sheet from bottom -->
  <div class="fixed inset-0 z-50 flex flex-col">
    <button class="flex-1 bg-black/50" onclick={close} />
    <div class="bg-white dark:bg-gray-800 rounded-t-2xl max-h-[90vh] overflow-y-auto">
      <div class="sticky top-0 flex items-center justify-between p-4 border-b bg-inherit">
        <h2>{title}</h2>
        <button onclick={close}>✕</button>
      </div>
      <div class="p-4">
        {@render children?.()}
      </div>
    </div>
  </div>
{:else}
  <!-- Centered modal for desktop -->
  <div class="fixed inset-0 z-50 flex items-center justify-center">
    <button class="absolute inset-0 bg-black/50" onclick={close} />
    <div class="relative bg-white dark:bg-gray-800 rounded-xl max-w-lg w-full max-h-[90vh] overflow-y-auto m-4">
      {@render children?.()}
    </div>
  </div>
{/if}
```

#### Files to Update:
- `frontend/src/components/shared/ConfirmDialog.svelte`
- `frontend/src/components/admin/PasswordChangeModal.svelte`
- All inline modals in admin pages

---

## CSS Utilities to Add

Add these to `frontend/src/app.css`:

```css
@layer utilities {
  /* Prevent text selection on touch (for drag handles, buttons) */
  .touch-manipulation {
    touch-action: manipulation;
    -webkit-tap-highlight-color: transparent;
    user-select: none;
  }
  
  /* Minimum touch target size */
  .touch-target {
    min-width: 44px;
    min-height: 44px;
  }
  
  /* Safe area padding for notched phones */
  .safe-bottom {
    padding-bottom: env(safe-area-inset-bottom);
  }
  
  .safe-top {
    padding-top: env(safe-area-inset-top);
  }
}
```

---

## Testing Strategy

### Device Matrix

| Device | Screen | Priority |
|--------|--------|----------|
| iPhone SE | 375x667 | Critical (smallest common) |
| iPhone 14 | 390x844 | High |
| iPhone 14 Pro Max | 430x932 | Medium |
| iPad Mini | 768x1024 | High (tablet breakpoint) |
| Android (Pixel 7) | 412x915 | Medium |

### Test Scenarios

1. **Sidebar Navigation**
   - Open/close sidebar on mobile
   - Navigate between pages
   - Sidebar closes after navigation

2. **Form Editing**
   - Create new project on phone
   - Add multiple links
   - Edit experience with many bullet points

3. **List Management**
   - Scroll long lists
   - Use action buttons
   - Bulk select items

4. **View Editor**
   - Reorder sections via drag
   - Toggle section visibility
   - Edit section settings
   - Use preview

### Playwright Mobile Tests

```typescript
// New test file: frontend/tests/mobile-admin.spec.ts
import { test, expect, devices } from '@playwright/test';

test.use(devices['iPhone 14']);

test.describe('Mobile Admin', () => {
  test.beforeEach(async ({ page }) => {
    // Login flow
  });

  test('sidebar opens and closes', async ({ page }) => {
    await page.goto('/admin');
    
    // Sidebar should be hidden by default
    await expect(page.locator('[data-testid="admin-sidebar"]')).not.toBeVisible();
    
    // Click hamburger
    await page.click('[data-testid="menu-toggle"]');
    await expect(page.locator('[data-testid="admin-sidebar"]')).toBeVisible();
    
    // Click backdrop to close
    await page.click('[data-testid="sidebar-backdrop"]');
    await expect(page.locator('[data-testid="admin-sidebar"]')).not.toBeVisible();
  });

  test('can create project on mobile', async ({ page }) => {
    await page.goto('/admin/projects');
    await page.click('text=New Project');
    
    // Form should be usable
    await page.fill('[data-testid="project-title"]', 'Mobile Test Project');
    await page.fill('[data-testid="project-summary"]', 'Created on mobile');
    
    // Save should work
    await page.click('text=Save');
    await expect(page.locator('text=Mobile Test Project')).toBeVisible();
  });
});
```

---

## Success Metrics

| Metric | Current | Target |
|--------|---------|--------|
| Sidebar usable width on 375px | 311px | 351px (+40px) |
| Touch target minimum | ~32px | 44px |
| Form completion on mobile | Unknown | Test passes |
| View editor usable on mobile | No | Yes |
| Modal scroll on small screens | Broken | Works |

---

## Risk Assessment

| Change | Risk | Mitigation |
|--------|------|------------|
| Sidebar refactor | Medium | Feature flag, A/B test |
| Overflow menu pattern | Low | Gradual rollout |
| Form layout fixes | Very Low | Pure CSS, backward compatible |
| View editor redesign | Medium | Preserve desktop UX exactly |
| Modal sheets | Low | Isolated component |

---

## Implementation Priority

1. **Phase 1: Admin Layout** - Biggest impact, unblocks everything else
2. **Phase 3: Form Fixes** - Quick wins, CSS-only
3. **Phase 2: Touch Targets** - Important for usability
4. **Phase 5: Modal Fixes** - Moderate impact
5. **Phase 4: View Editor** - Most complex, do last

---

## Appendix: Best Practices Reference (from Industry Research)

### Mobile Navigation Patterns

**Evidence from NNGroup, UXPin, Material Design:**

| Pattern | Best For | Discoverability |
|---------|----------|----------------|
| Bottom Navigation | 3-5 primary sections | 76% higher than hamburger |
| Hamburger Menu | 20+ pages, deep hierarchy | Lower engagement (-25%) |
| Hybrid (Bottom + Drawer) | Complex apps | Best of both worlds |

**Recommendation for Facet**: 
- Hamburger menu with overlay drawer (not persistent sidebar)
- Consider: Bottom nav for top 4 sections (Dashboard, Profile, Facets, More)
- "More" opens full drawer with remaining 15+ pages

### Touch Target Guidelines

| Platform | Minimum | Recommended |
|----------|---------|-------------|
| iOS (Apple HIG) | 44×44px | 48×48px |
| Android (Material) | 48×48px | 56×56px for FABs |
| Spacing | 8px min | 16-20px between actions |

**Facet-specific fixes needed:**
- List action buttons: Currently ~32px → increase to 44px
- Drag handles in view editor: Add explicit grip area
- Form inputs: Already good (using `.input` class with proper padding)

### Modal Best Practices (from NNGroup 2023)

**Key Insight:** "Bottom sheets preserve substantial visibility of underlying content... especially useful when users are likely to need to refer to main, background information."

| Use Case | Desktop | Mobile |
|----------|---------|--------|
| Confirmations | Centered dialog | Centered dialog |
| Quick actions | Dropdown/popover | Bottom sheet |
| Complex forms | Side panel | Full-screen or bottom sheet |
| Settings | Side panel | Full-screen |

**Critical Rules:**
1. Always include visible close button (not just swipe)
2. Support back button navigation
3. Never stack bottom sheets
4. Use responsive pattern: Desktop dialog → Mobile sheet

### Swipe Gesture Guidelines

**Use Swipe Actions When:**
- ✅ Quick actions on list items (delete, archive, flag)
- ✅ Decisions based on limited visible information
- ✅ Frequently used functions

**Avoid Swipe When:**
- ❌ Users need more context to make decisions
- ❌ Action consequences are significant/irreversible
- ❌ Multiple complex actions

**Implementation Pattern:**
```svelte
<!-- Swipe left for destructive, right for positive -->
<SwipeableListItem
  onSwipeLeft={showDeleteAction}
  onSwipeRight={showArchiveAction}
  threshold={50}  <!-- Prevent accidental triggers -->
>
  <ListItemContent />
</SwipeableListItem>
```

### Form Best Practices (from IvyForms/Smashing Magazine 2025)

1. **Single-column layout** on mobile (no side-by-side)
2. **Real-time inline validation** (validate as user moves through fields)
3. **Error messages directly below fields** (vertical reading flow)
4. **Auto-focus first field** for keyboard flow
5. **Field width reflects expected content** (75px, 150px, 250px, 350px, 500px)
6. **16px minimum font size** to prevent iOS zoom on focus

### Component Patterns (from Shadcn/Tremor)

**Responsive Dialog/Sheet Pattern:**
```svelte
<script>
  const isMobile = useMediaQuery('(max-width: 768px)');
</script>

{#if isMobile}
  <Sheet side="bottom">
    <SheetContent class="max-h-[90vh] overflow-y-auto">
      {@render content?.()}
    </SheetContent>
  </Sheet>
{:else}
  <Dialog>
    <DialogContent class="max-w-lg">
      {@render content?.()}
    </DialogContent>
  </Dialog>
{/if}
```

**Drawer Navigation Pattern:**
```svelte
<!-- Mobile: Full overlay drawer -->
<Drawer class="fixed inset-y-0 left-0 z-50 w-64 bg-white">
  <DrawerOverlay class="fixed inset-0 bg-black/50" />
  <DrawerContent>
    <nav class="p-4 space-y-2">
      <!-- Navigation items -->
    </nav>
  </DrawerContent>
</Drawer>
```

---

## Implementation Checklist

### Layout
- [ ] Sidebar hidden on mobile (< 1024px)
- [ ] Overlay drawer pattern for navigation
- [ ] Main content gets full width on mobile
- [ ] Reduce padding: `p-6` → `p-4 lg:p-6`

### Navigation  
- [ ] Hamburger always visible on mobile
- [ ] Drawer closes after navigation
- [ ] Backdrop click closes drawer
- [ ] Consider bottom nav for top 4 sections (future)

### Forms
- [ ] Stack all inputs vertically on mobile
- [ ] Remove hardcoded widths (`w-32` → `w-full sm:w-32`)
- [ ] Touch targets ≥ 44px
- [ ] 16px font size on inputs

### Lists
- [ ] Overflow menu for secondary actions
- [ ] Increased spacing between action buttons
- [ ] Swipe actions for quick operations (optional)

### Modals
- [ ] Bottom sheets on mobile
- [ ] max-h-[90vh] with overflow-y-auto
- [ ] Always visible close button
- [ ] Backdrop click to close

### View Editor
- [ ] Hide preview on mobile (collapsible)
- [ ] Simplify section headers
- [ ] Larger drag handles
- [ ] Settings in expandable panel

### Accessibility
- [ ] Focus trap in modals/drawers
- [ ] ESC key closes overlays
- [ ] ARIA labels on all controls
- [ ] Screen reader announcements
- [ ] Reduced motion support (already in app.css)

---

*Document created: 2026-01-17*  
*Research sources: NNGroup, UXPin, Shadcn UI, Tremor, Material Design, IvyForms 2025*  
*Ready for implementation when prioritized*
