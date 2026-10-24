import { describe, expect, it } from 'vitest';
import type { Goal } from '../../api/types';
import {
	compactDollars,
	goalRows,
	monthTick,
	niceTicks,
	trendRange,
} from './chartData';

function goal(over: Partial<Goal>): Goal {
	return {
		category_id: 1,
		name: 'Food',
		cap_cents: null,
		cap_since: null,
		rollover_since: null,
		carryover_cents: 0,
		available_cents: null,
		spent_cents: 0,
		...over,
	};
}

describe('trendRange', () => {
	it('covers the 12 months ending at the month', () => {
		expect(trendRange('2026-10')).toEqual({ from: '2025-11', to: '2026-10' });
		expect(trendRange('2027-01')).toEqual({ from: '2026-02', to: '2027-01' });
	});
});

describe('goalRows', () => {
	it('measures against the cap, or available with rollover', () => {
		const rows = goalRows(
			[
				goal({ category_id: 1, name: 'Coffee', spent_cents: 500 }),
				goal({
					category_id: 2,
					name: 'Food',
					cap_cents: 40000,
					available_cents: 40000,
					spent_cents: 41000,
				}),
				goal({
					category_id: 3,
					name: 'Insurance',
					cap_cents: 10000,
					rollover_since: '2026-08',
					carryover_cents: 20000,
					available_cents: 30000,
				}),
				// Rollover from a later month doesn't apply yet.
				goal({
					category_id: 4,
					name: 'Travel',
					cap_cents: 5000,
					rollover_since: '2026-12',
					available_cents: 5000,
				}),
			],
			'2026-10',
		);
		expect(rows).toEqual([
			{
				categoryId: 2,
				name: 'Food',
				goalCents: 40000,
				spentCents: 41000,
				rollover: false,
			},
			{
				categoryId: 3,
				name: 'Insurance',
				goalCents: 30000,
				spentCents: 0,
				rollover: true,
			},
			{
				categoryId: 4,
				name: 'Travel',
				goalCents: 5000,
				spentCents: 0,
				rollover: false,
			},
		]);
	});

	it('keeps a rollover category with no cap', () => {
		const rows = goalRows(
			[
				goal({
					rollover_since: '2026-09',
					carryover_cents: -3500,
					available_cents: -3500,
				}),
			],
			'2026-10',
		);
		expect(rows).toEqual([
			{
				categoryId: 1,
				name: 'Food',
				goalCents: -3500,
				spentCents: 0,
				rollover: true,
			},
		]);
	});
});

describe('monthTick', () => {
	it('shows the year only on January', () => {
		expect(monthTick('2026-11')).toBe('Nov');
		expect(monthTick('2027-01')).toBe('Jan ’27');
	});
});

describe('compactDollars', () => {
	it.each([
		[0, '0'],
		[25049, '250'],
		[100000, '1K'],
		[150000, '1.5K'],
		[1234500, '12K'],
		[-50000, '−500'],
	])('%i → %s', (cents, want) => {
		expect(compactDollars(cents)).toBe(want);
	});
});

describe('niceTicks', () => {
	it('steps by round amounts and covers the data', () => {
		expect(niceTicks(1250, 180000)).toEqual([0, 50000, 100000, 150000, 200000]);
		expect(niceTicks(0, 270000)).toEqual([0, 100000, 200000, 300000]);
		expect(niceTicks(0, 50000)).toEqual([0, 20000, 40000, 60000]);
		expect(niceTicks(-400, 2000)).toEqual([-1000, 0, 1000, 2000]);
		expect(niceTicks(0, 0)).toEqual([0]);
	});
});
