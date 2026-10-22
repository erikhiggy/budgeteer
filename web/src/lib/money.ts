export type ParsedAmount =
	{ ok: true; cents: number } | { ok: false; error: string };

// Optional sign, optional $, digits (optionally grouped with commas), and up
// to two decimal places. Whole or fraction may be empty, but not both.
const AMOUNT = /^([+-]?)\$?(\d{1,3}(?:,\d{3})+|\d*)(?:\.(\d{0,2}))?$/;

/**
 * Parses what the user typed in an amount field into signed cents, without
 * going through floating point. A plain number is money out (negative);
 * a leading + is money in. A leading - is accepted and also means out.
 */
export function parseAmount(input: string): ParsedAmount {
	const s = input.trim();
	if (s === '') return { ok: false, error: 'Amount is required' };

	const m = AMOUNT.exec(s);
	const whole = m?.[2].replaceAll(',', '') ?? '';
	const frac = m?.[3] ?? '';
	if (!m || (whole === '' && frac === '')) {
		return { ok: false, error: 'Amount must be a number like 12.34' };
	}
	if (whole.length > 13) return { ok: false, error: 'Amount is too large' };

	const cents = Number(whole || '0') * 100 + Number(frac.padEnd(2, '0'));
	if (cents === 0) return { ok: false, error: 'Amount cannot be zero' };
	return { ok: true, cents: m[1] === '+' ? cents : -cents };
}

const MINUS = '−';

function digits(absCents: number): string {
	const whole = Math.floor(absCents / 100).toLocaleString('en-US');
	const frac = String(absCents % 100).padStart(2, '0');
	return `${whole}.${frac}`;
}

/** A balance or total with its natural sign: "1,234.56", "−12.00". */
export function formatCents(cents: number): string {
	return (cents < 0 ? MINUS : '') + digits(Math.abs(cents));
}

/**
 * A transaction amount the way it's entered: money out is plain ("12.34"),
 * money in gets a + ("+12.34").
 */
export function formatAmount(cents: number): string {
	return (cents > 0 ? '+' : '') + digits(Math.abs(cents));
}
