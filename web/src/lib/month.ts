// Months are "YYYY-MM" strings with no time zone, like dates (see date.ts).

const ISO_MONTH = /^(\d{4})-(0[1-9]|1[0-2])$/;

/** This month in the local time zone as YYYY-MM. */
export function currentMonth(now = new Date()): string {
	return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

/** True for a real month in YYYY-MM form. */
export function isISOMonth(s: string): boolean {
	return ISO_MONTH.test(s);
}

/** "2026-12" + 1 → "2027-01". */
export function addMonths(month: string, n: number): string {
	const [y, m] = month.split('-').map(Number);
	const total = y * 12 + (m - 1) + n;
	const year = Math.floor(total / 12);
	return `${year}-${String((total % 12) + 1).padStart(2, '0')}`;
}

const long = new Intl.DateTimeFormat('en-US', {
	month: 'long',
	year: 'numeric',
	timeZone: 'UTC',
});
const short = new Intl.DateTimeFormat('en-US', {
	month: 'short',
	year: 'numeric',
	timeZone: 'UTC',
});

function toDate(month: string): Date {
	const [y, m] = month.split('-').map(Number);
	return new Date(Date.UTC(y, m - 1, 1));
}

/** "2026-10" → "October 2026". */
export function formatMonth(month: string): string {
	return isISOMonth(month) ? long.format(toDate(month)) : month;
}

/** "2026-10" → "Oct 2026". */
export function formatMonthShort(month: string): string {
	return isISOMonth(month) ? short.format(toDate(month)) : month;
}
