# View Editor Component Integration Guide

## Status: Phase 2 Complete ✅

### What's Available

**Phase 1 Components** (Fully Integrated):
- `ViewOverrideEditor` - Modal for customizing item overrides ✅ 
- `ViewResumeGenerator` - Modal for AI resume generation ✅

**Phase 2 Components** (Ready for Integration):
- `ViewSettingsPanel` - Complete form covering all sections
- `ViewBasicInfo` - Basic information subset

### Integration Approach

The main challenge is that the original form has complex nested structures (e.g., share token generation embedded within basic info). Here are recommended integration strategies:

#### Option 1: Gradual Section-by-Section
```svelte
<!-- Replace one section at a time -->
<ViewBasicInfo bind:name bind:slug bind:description bind:visibility bind:password {view} />

<!-- Keep share token logic separate for now -->
{#if visibility === 'unlisted' || visibility === 'private'}
  <div class="card p-4 sm:p-6 space-y-4">
    <!-- Share token generation logic stays here -->
  </div>
{/if}
```

#### Option 2: Complete Replacement
```svelte
<!-- Use the comprehensive component -->
<ViewSettingsPanel
  bind:name bind:slug bind:description bind:visibility bind:password
  bind:heroHeadline bind:heroSummary bind:heroLocation bind:heroImageUrl
  bind:ctaText bind:ctaUrl bind:accentColor bind:isActive
  {profile} onHeroImageChange={handleHeroImageChange} onRemoveHeroImage={removeHeroImage}
/>

<!-- Move share token logic to separate component or keep in main page -->
```

### Key Integration Points

1. **Share Token Logic** - Currently embedded in basic info, should be extracted separately
2. **Hero Image Handlers** - Need to maintain `handleHeroImageChange` and `removeHeroImage` functions
3. **Form Binding** - All form fields use `bind:` for two-way binding

### Files to Replace

When ready to integrate:

1. **Basic Info**: Lines ~1018-1092 in `+page.svelte`
2. **Hero Overrides**: Lines ~1193-1405 
3. **Call to Action**: Lines ~1408-1435
4. **Settings**: Lines ~1437-1512

### Next Phase Components

**Phase 3** (DnD Components):
- `ViewSectionManager` - Section ordering and configuration
- `ViewItemPicker` - Item selection within sections  

**Phase 4** (Context Integration):
- `createViewEditor()` store when prop drilling becomes problematic

### Testing Strategy

1. Test individual components in isolation
2. Create integration branch for each section
3. Verify all functionality works before merging
4. Keep modals working throughout (they're already integrated)

---

**Current State**: All Phase 2 components are complete and ready. The main page remains untouched to avoid breaking existing functionality while components are being tested.