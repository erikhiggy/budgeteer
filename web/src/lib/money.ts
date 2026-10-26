const ZERO_AMOUNT = 'Amount cannot be zero';

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
export const parseAmount = (input: string): ParsedAmount => {
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
	if (cents === 0) return { ok: false, error: ZERO_AMOUNT };
	return { ok: true, cents: m[1] === '+' ? cents : -cents };
};

/**
 * Parses an opening balance. Unlike a transaction amount, a plain number is
 * money you have (positive), and a leading - is money owed, e.g. on a credit
 * card. Empty means no opening balance (0).
 */
export const parseBalance = (input: string): ParsedAmount => {
	const s = input.trim();
	if (s === '') return { ok: true, cents: 0 };
	const parsed = parseAmount('+' + s.replace(/^[+-]/, ''));
	if (parsed.ok) {
		return {
			ok: true,
			cents: s.startsWith('-') ? -parsed.cents : parsed.cents,
		};
	}
	// A zero balance is fine; it just means none.
	if (parsed.error === ZERO_AMOUNT) return { ok: true, cents: 0 };
	return {
		ok: false,
		error: parsed.error.replace(/^Amount/, 'Opening balance'),
	};
};

/** A cap as typed: a positive amount, or empty for no cap (null). */
export type ParsedCap =
	{ ok: true; cents: number | null } | { ok: false; error: string };

/** Parses a goal cap. Empty means no cap; zero and negatives are rejected. */
export const parseCap = (input: string): ParsedCap => {
	const s = input.trim();
	if (s === '') return { ok: true, cents: null };
	if (s.startsWith('-')) return { ok: false, error: 'A cap must be positive' };
	const parsed = parseAmount('+' + s.replace(/^\+/, ''));
	if (parsed.ok) return parsed;
	if (parsed.error === ZERO_AMOUNT) {
		return {
			ok: false,
			error: 'A cap must be more than zero; leave it empty for no cap',
		};
	}
	return { ok: false, error: parsed.error.replace(/^Amount/, 'Cap') };
};

const MINUS = '−';

const digits = (absCents: number): string => {
	const whole = Math.floor(absCents / 100).toLocaleString('en-US');
	const frac = String(absCents % 100).padStart(2, '0');
	return `${whole}.${frac}`;
};

/** A balance or total with its natural sign: "1,234.56", "−12.00". */
export const formatCents = (cents: number): string => {
	return (cents < 0 ? MINUS : '') + digits(Math.abs(cents));
};

/**
 * A transaction amount the way it's entered: money out is plain ("12.34"),
 * money in gets a + ("+12.34").
 */
export const formatAmount = (cents: number): string => {
	return (cents > 0 ? '+' : '') + digits(Math.abs(cents));
};

/**
 * Parses an envelope assignment. A plain number puts money in, a leading -
 * (or the − formatCents writes) takes it out, and empty or zero means
 * nothing assigned (0).
 */
export const parseAssigned = (input: string): ParsedAmount => {
	const parsed = parseBalance(input.trim().replace(/^−/, '-'));
	if (parsed.ok) return parsed;
	return {
		ok: false,
		error: parsed.error.replace(/^Opening balance/, 'Assigned'),
	};
};
