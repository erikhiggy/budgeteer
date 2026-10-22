import { describe, expect, it } from 'vitest';
import { formatDate, isISODate, todayISO } from './date';

describe('todayISO', () => {
	it('uses local date parts, zero-padded', () => {
		expect(todayISO(new Date(2026, 0, 5, 23, 59))).toBe('2026-01-05');
	});
});

describe('isISODate', () => {
	it.each(['2026-10-22', '2024-02-29', '2026-12-31'])('%s is valid', (s) => {
		expect(isISODate(s)).toBe(true);
	});

	it.each(['', '2026-02-30', '2025-02-29', '2026-13-01', '2026-1-5', 'today'])(
		'%j is invalid',
		(s) => {
			expect(isISODate(s)).toBe(false);
		},
	);
});

describe('formatDate', () => {
	it('formats a calendar day without timezone drift', () => {
		expect(formatDate('2026-01-01')).toBe('Jan 1, 2026');
	});

	it('passes through anything that is not a date', () => {
		expect(formatDate('garbage')).toBe('garbage');
	});
});
