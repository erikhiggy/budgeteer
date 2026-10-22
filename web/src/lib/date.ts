const ISO_DATE = /^(\d{4})-(\d{2})-(\d{2})$/;

/** Today in the local timezone as YYYY-MM-DD. */
export function todayISO(now = new Date()): string {
	const y = now.getFullYear();
	const m = String(now.getMonth() + 1).padStart(2, '0');
	const d = String(now.getDate()).padStart(2, '0');
	return `${y}-${m}-${d}`;
}

/** True for a real calendar date in YYYY-MM-DD form (rejects 2026-02-30). */
export function isISODate(s: string): boolean {
	const m = ISO_DATE.exec(s);
	if (!m) return false;
	const [y, mo, d] = [Number(m[1]), Number(m[2]), Number(m[3])];
	const date = new Date(Date.UTC(y, mo - 1, d));
	return (
		date.getUTCFullYear() === y &&
		date.getUTCMonth() === mo - 1 &&
		date.getUTCDate() === d
	);
}

const display = new Intl.DateTimeFormat('en-US', {
	month: 'short',
	day: 'numeric',
	year: 'numeric',
	timeZone: 'UTC',
});

/** "2026-10-22" → "Oct 22, 2026". Treats the date as a calendar day, not an instant. */
export function formatDate(iso: string): string {
	if (!isISODate(iso)) return iso;
	const [y, m, d] = iso.split('-').map(Number);
	return display.format(new Date(Date.UTC(y, m - 1, d)));
}
