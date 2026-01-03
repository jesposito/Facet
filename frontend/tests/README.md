# Facet E2E Testing

End-to-end tests for Facet using Playwright.

## Test Structure

```
tests/
├── config.ts              # Test configuration and environment variables
├── helpers.ts             # Shared test utilities (auth, API helpers)
├── admin-flows.spec.ts    # Admin-only tests (auth, views, share tokens)
├── media-management.spec.ts # Media upload, orphan detection, bulk delete
├── public-api.spec.ts     # Public APIs, feeds (RSS, iCal)
├── seo-and-errors.spec.ts # SEO meta tags, error pages
└── README.md              # This file
```

## Running Tests

### Prerequisites

1. **Start the development servers:**
   ```bash
   # From project root
   make dev
   # OR separately:
   make backend   # Backend on :8090
   make frontend  # Frontend on :5173
   ```

2. **Set environment variables for admin tests:**
   ```bash
   export ADMIN_EMAIL="admin@example.com"
   export ADMIN_PASSWORD="changeme123"
   ```

### Run All Tests

```bash
npm test
# or
npm run test:e2e
```

### Run Specific Test Suites

```bash
# Public tests only (no auth required)
npm run test:public

# Admin tests only (requires ADMIN_EMAIL/ADMIN_PASSWORD)
npm run test:admin
```

### Interactive Testing

```bash
# UI mode (visual test runner)
npm run test:ui

# Headed mode (see browser)
npm run test:headed

# Debug mode (step through tests)
npm run test:debug
```

### Run Individual Test Files

```bash
npx playwright test tests/seo-and-errors.spec.ts
npx playwright test tests/media-management.spec.ts --headed
```

## Test Coverage

### Public API Tests (`public-api.spec.ts`)
- ✅ Default view/homepage reachability
- ✅ AI and AI Print capability endpoints
- ✅ Posts and talks listings
- ✅ RSS feed (posts)
- ✅ iCal export (talks)

### Admin Flow Tests (`admin-flows.spec.ts`)
- ✅ Admin authentication
- ✅ View listing and CRUD
- ✅ Share token lifecycle (generate, validate, revoke)

### Media Management Tests (`media-management.spec.ts`)
- ✅ Media file listing
- ✅ Orphan detection
- ✅ Bulk delete validation (empty array, limit enforcement)
- ✅ Bulk delete security (path traversal protection)
- ✅ Bulk delete authentication requirement
- ✅ External media collection access
- ✅ Media rendering on project/post pages

### SEO & Error Page Tests (`seo-and-errors.spec.ts`)
- ✅ Sitemap.xml validity
- ✅ Robots.txt format
- ✅ Canonical URLs on all pages
- ✅ Open Graph tags (homepage, projects, posts)
- ✅ Twitter Card meta tags
- ✅ JSON-LD structured data
- ✅ Article metadata (published/modified times)
- ✅ Custom 404 error page
- ✅ Custom 500 error page (via /test-500)
- ✅ Error page SVG illustrations
- ✅ Error page navigation

## Configuration

Tests use environment variables for configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `PLAYWRIGHT_BASE_URL` | `http://localhost:5173` | Frontend URL |
| `API_BASE_URL` | `http://localhost:8090` | Backend API URL |
| `ADMIN_EMAIL` | _(none)_ | Admin email for authenticated tests |
| `ADMIN_PASSWORD` | _(none)_ | Admin password for authenticated tests |

**Note:** Admin tests are automatically skipped if `ADMIN_EMAIL` or `ADMIN_PASSWORD` are not set.

## Writing New Tests

### Example: Public API Test

```typescript
import { test, expect } from '@playwright/test';
import { apiBaseURL } from './config';

test('my new public endpoint works', async ({ request }) => {
  const response = await request.get(`${apiBaseURL}/api/my-endpoint`);
  expect(response.ok()).toBeTruthy();
  const data = await response.json();
  expect(data).toHaveProperty('expectedField');
});
```

### Example: Admin-Only Test

```typescript
import { test, expect } from '@playwright/test';
import { loginAsAdmin } from './helpers';
import { apiBaseURL, adminEmail, adminPassword } from './config';

const shouldRunAdmin = Boolean(adminEmail && adminPassword);

test.describe('My admin feature', () => {
  test.skip(!shouldRunAdmin, 'Admin credentials required');

  test('can perform admin action', async ({ request }) => {
    const { token } = await loginAsAdmin(request);

    const response = await request.post(`${apiBaseURL}/api/admin/action`, {
      headers: { Authorization: token },
      data: { foo: 'bar' }
    });

    expect(response.ok()).toBeTruthy();
  });
});
```

### Example: UI Test

```typescript
import { test, expect } from '@playwright/test';

test('homepage displays correctly', async ({ page }) => {
  await page.goto('/');

  await expect(page.locator('h1')).toBeVisible();
  await expect(page.locator('h1')).toContainText('Expected Text');

  // Interact with the page
  await page.click('button#my-button');
  await expect(page.locator('.result')).toHaveText('Success');
});
```

## Best Practices

1. **Use descriptive test names** - Test names should clearly describe what's being tested
2. **Skip gracefully** - Use `test.skip()` when prerequisites aren't met
3. **Clean up after tests** - Delete created resources when possible
4. **Test both success and failure** - Validate error handling
5. **Use helpers** - Extract common patterns to `helpers.ts`
6. **Avoid hardcoding** - Use config variables for URLs and credentials

## CI/CD Integration

Tests are configured to retry 2 times in CI environments:

```typescript
retries: process.env.CI ? 2 : 0
```

Test artifacts (traces, screenshots, videos) are captured on failure:

```typescript
use: {
  trace: 'retain-on-failure',
  screenshot: 'only-on-failure',
  video: 'retain-on-failure'
}
```

## Debugging Failed Tests

1. **Run in headed mode** to see what's happening:
   ```bash
   npm run test:headed
   ```

2. **Use debug mode** to step through:
   ```bash
   npm run test:debug
   ```

3. **Check test artifacts** in `test-results/`:
   - Screenshots
   - Videos
   - Trace files (view with `npx playwright show-trace trace.zip`)

4. **Run specific test** to isolate issues:
   ```bash
   npx playwright test -g "specific test name"
   ```

## Troubleshooting

### "ADMIN_EMAIL and ADMIN_PASSWORD are required"

This is expected if you haven't set credentials. Admin tests will be skipped. To run them:

```bash
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="changeme123"
npm run test:admin
```

### "No views/projects/posts available"

Some tests require existing data. Run seed data:

```bash
make seed-dev
```

### Connection refused errors

Make sure both frontend and backend are running:

```bash
# Check if services are up
curl http://localhost:5173
curl http://localhost:8090/api/health
```

### Browser not found

Install Playwright browsers:

```bash
npx playwright install
```

## Future Test Ideas

- [ ] GitHub import flow
- [ ] AI enrichment (with mocked AI provider)
- [ ] View editor drag-and-drop
- [ ] Password-protected views
- [ ] Media upload flow
- [ ] Custom CSS application
- [ ] Theme switching
- [ ] Print stylesheet rendering
- [ ] Export (JSON/YAML)
- [ ] AI resume generation
