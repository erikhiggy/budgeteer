import { describe, expect, it } from 'vitest';
import {
	addMonths,
	currentMonth,
	formatMonth,
	formatMonthShort,
	isISOMonth,
} from './month';

describe('month', () => {
	it('currentMonth uses local time', () => {
		expect(currentMonth(new Date(2026, 0, 31, 23, 59))).toBe('2026-01');
		expect(currentMonth(new Date(2026, 11, 1))).toBe('2026-12');
	});

	it.each([
		['2026-10', true],
		['2026-01', true],
		['2026-13', false],
		['2026-00', false],
		['2026-1', false],
		['2026-10-01', false],
		['', false],
	])('isISOMonth(%j) is %s', (s, ok) => {
		expect(isISOMonth(s)).toBe(ok);
	});

	it.each([
		['2026-10', 1, '2026-11'],
		['2026-12', 1, '2027-01'],
		['2027-01', -1, '2026-12'],
		['2026-10', -13, '2025-09'],
		['2026-10', 0, '2026-10'],
	])('addMonths(%j, %i) is %j', (m, n, want) => {
		expect(addMonths(m, n)).toBe(want);
	});

	it('formats months', () => {
		expect(formatMonth('2026-10')).toBe('October 2026');
		expect(formatMonthShort('2026-08')).toBe('Aug 2026');
		expect(formatMonth('nope')).toBe('nope');
	});
});
