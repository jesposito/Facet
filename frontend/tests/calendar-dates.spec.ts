import { expect, test } from '@playwright/test';
import { formatCalendarDate } from '../src/lib/calendar-date';
import {
	formatCalendarDate as formatDisplayCalendarDate,
	formatCalendarDateRange,
	formatDate
} from '../src/lib/utils';

const fullDate = { month: 'long', day: 'numeric', year: 'numeric' } as const;
const zones = ['America/New_York', 'America/Los_Angeles', 'Pacific/Auckland', 'UTC'];

function inTimeZone<T>(zone: string, run: () => T): T {
	const previous = process.env.TZ;
	process.env.TZ = zone;
	try {
		return run();
	} finally {
		if (previous === undefined) delete process.env.TZ;
		else process.env.TZ = previous;
	}
}

test('calendar dates retain their selected day and month in every visitor time zone', () => {
	for (const zone of zones) {
		inTimeZone(zone, () => {
			expect(formatCalendarDate('2026-06-01 00:00:00.000Z', 'en-US', fullDate), zone).toBe(
				'June 1, 2026'
			);
			expect(formatCalendarDate('2026-01-01T00:00:00.000Z', 'en-US', fullDate), zone).toBe(
				'January 1, 2026'
			);
			expect(formatCalendarDate('2026-06', 'en-US', { month: 'long', year: 'numeric' }), zone).toBe(
				'June 2026'
			);
			expect(formatCalendarDate('2026', 'en-US', fullDate), zone).toBe('2026');
		});
	}
});

test('calendar formatting uses the requested locale', () => {
	expect(formatCalendarDate('2026-06', 'fr-FR', { month: 'long', year: 'numeric' })).toBe(
		'juin 2026'
	);
});

test('calendar formatting accepts leap days and rejects invalid or non-calendar values', () => {
	expect(formatCalendarDate('2024-02-29', 'en-US', fullDate)).toBe('February 29, 2024');
	for (const value of [
		'2023-02-29',
		'2024-02-30 00:00:00.000Z',
		'2024-00',
		'2024-13',
		'2024-01-00',
		'2024-01-32',
		'2024-1-01',
		'not-a-date',
		'2024-01-01T12:00:00.000Z'
	]) {
		expect(() => formatCalendarDate(value, 'en-US', fullDate), value).toThrow(RangeError);
	}
});

test('early calendar years are not shifted into the twentieth century', () => {
	for (const [value, expected] of [
		['0001-01', 'January 1'],
		['0099-01', 'January 99']
	] as const) {
		expect(formatCalendarDate(value, 'en-US', { month: 'long', year: 'numeric' })).toBe(expected);
	}
});

test('malformed imported calendar dates remain visible without breaking the page', () => {
	expect(formatDisplayCalendarDate('2026-02-30')).toBe('2026-02-30');
	expect(formatDisplayCalendarDate(undefined)).toBe('');
});

test('calendar ranges format both endpoints as calendar dates', () => {
	expect(
		formatCalendarDateRange('2026-06-01 00:00:00.000Z', '2027-01-01 00:00:00.000Z', 'Present')
	).toBe('Jun 2026 - Jan 2027');
	expect(formatCalendarDateRange('2026-06', undefined, 'Present')).toBe('Jun 2026 - Present');
});

test('the instant formatter retains visitor-local timestamp behavior', () => {
	for (const zone of zones) {
		inTimeZone(zone, () => {
			const expected = ['America/New_York', 'America/Los_Angeles'].includes(zone)
				? 'May 31, 2026'
				: 'June 1, 2026';
			expect(formatDate('2026-06-01 00:00:00.000Z', fullDate), zone).toBe(expected);
		});
	}
});
