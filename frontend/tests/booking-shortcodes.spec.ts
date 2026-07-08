import { expect, test } from '@playwright/test';
import { parseMarkdown, sanitizeRichHtml } from '../src/lib/utils';

test.describe('booking markdown shortcodes', () => {
	test('renders Calendly as a sandboxed inline embed', () => {
		const html = parseMarkdown('{{calendly:https://calendly.com/jed/30min}}');

		expect(html).toContain('calendly.com/jed/30min');
		expect(html).toContain('embed_type=Inline');
		expect(html).toContain('title="Calendly booking calendar"');
		expect(html).toContain('aria-label="Book a meeting via Calendly"');
		expect(html).toContain('sandbox=');
	});

	test('renders Cal.com as a sandboxed embed without double appending', () => {
		const html = parseMarkdown('{{calcom:https://cal.com/jed/30min/embed}}');

		expect(html).toContain('cal.com/jed/30min/embed');
		expect(html).not.toContain('/embed/embed');
		expect(html).toContain('title="Cal.com booking calendar"');
		expect(html).toContain('sandbox=');
	});

	test('renders Google Calendar and generic booking as safe outbound links', () => {
		const html = parseMarkdown('{{googlecal:https://calendar.app.google/abc123}}\n{{booking:https://example.com/book}}');

		expect(html).toContain('calendar.app.google/abc123');
		expect(html).toContain('example.com/book');
		expect(html).toContain('rel="noopener noreferrer"');
		expect(html).toContain('(opens in new tab)');
		expect(html).not.toContain('<iframe');
	});

	test('keeps booking iframe allowlist narrow', () => {
		expect(
			sanitizeRichHtml('<iframe src="https://calendly.com/jed/30min?embed_type=Inline" title="Calendly"></iframe>')
		).toContain('<iframe');
		expect(sanitizeRichHtml('<iframe src="https://evil.example/jed/30min" title="Bad"></iframe>')).not.toContain(
			'<iframe'
		);
		expect(parseMarkdown('{{calendly:https://evil.example/jed/30min}}')).not.toContain('<iframe');
	});
});
