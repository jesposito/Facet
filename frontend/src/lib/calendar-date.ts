const CALENDAR_DATE = /^(\d{4})(?:-(\d{2})(?:-(\d{2})(?:[ T]00:00:00(?:\.\d{1,3})?Z)?)?)?$/;

function isLeapYear(year: number): boolean {
	return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
}

function daysInMonth(year: number, month: number): number {
	const days = [31, isLeapYear(year) ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
	return days[month - 1];
}

export function formatCalendarDate(
	dateString: string,
	locales?: string | string[],
	options: Intl.DateTimeFormatOptions = { month: 'short', year: 'numeric' }
): string {
	const match = CALENDAR_DATE.exec(dateString);
	if (!match) throw new RangeError(`Invalid calendar date: ${dateString}`);
	if (!match[2]) return dateString;

	const year = Number(match[1]);
	const month = Number(match[2]);
	const day = match[3] ? Number(match[3]) : 1;
	if (month < 1 || month > 12 || day < 1 || day > daysInMonth(year, month)) {
		throw new RangeError(`Invalid calendar date: ${dateString}`);
	}

	const date = new Date(0);
	date.setUTCFullYear(year, month - 1, day);
	return new Intl.DateTimeFormat(locales, { ...options, timeZone: 'UTC' }).format(date);
}
