import { describe, expect, it } from 'vitest';
import { formatAmount, formatCents, parseAmount } from './money';

describe('parseAmount', () => {
	it.each([
		['12', -1200],
		['12.34', -1234],
		['12.3', -1230],
		['12.', -1200],
		['.5', -50],
		['0.01', -1],
		['+12.34', 1234],
		['+.5', 50],
		['-12.34', -1234],
		['$12', -1200],
		['+$1,234.56', 123456],
		['1,234,567', -123456700],
		['  42  ', -4200],
		['0.10', -10],
	])('%j → %i cents', (input, cents) => {
		expect(parseAmount(input)).toEqual({ ok: true, cents });
	});

	it.each([
		['', 'Amount is required'],
		['   ', 'Amount is required'],
		['0', 'Amount cannot be zero'],
		['0.00', 'Amount cannot be zero'],
		['+', 'Amount must be a number like 12.34'],
		['.', 'Amount must be a number like 12.34'],
		['12.345', 'Amount must be a number like 12.34'],
		['1,23', 'Amount must be a number like 12.34'],
		['12,34.00', 'Amount must be a number like 12.34'],
		['abc', 'Amount must be a number like 12.34'],
		['1e5', 'Amount must be a number like 12.34'],
		['--5', 'Amount must be a number like 12.34'],
		['12 34', 'Amount must be a number like 12.34'],
		['99999999999999', 'Amount is too large'],
	])('%j is rejected', (input, error) => {
		expect(parseAmount(input)).toEqual({ ok: false, error });
	});

	it('does not suffer float rounding', () => {
		// 0.29 * 100 === 28.999999999999996 in floating point.
		expect(parseAmount('0.29')).toEqual({ ok: true, cents: -29 });
		expect(parseAmount('+1.15')).toEqual({ ok: true, cents: 115 });
	});
});

describe('formatCents', () => {
	it.each([
		[0, '0.00'],
		[5, '0.05'],
		[123456, '1,234.56'],
		[-1200, '−12.00'],
	])('%i → %j', (cents, text) => {
		expect(formatCents(cents)).toBe(text);
	});
});

describe('formatAmount', () => {
	it.each([
		[-1234, '12.34'],
		[1234, '+12.34'],
		[-123456700, '1,234,567.00'],
	])('%i → %j', (cents, text) => {
		expect(formatAmount(cents)).toBe(text);
	});
});
