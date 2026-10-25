import type { Goal } from '../../api/types';
import { addMonths } from '../../lib/month';

/** The trend's 12 months, ending at month (inclusive). */
export const trendRange = (month: string): { from: string; to: string } => {
	return { from: addMonths(month, -11), to: month };
};

export type GoalRow = {
	categoryId: number;
	name: string;
	/** Available when rollover applies this month, otherwise the cap. */
	goalCents: number;
	spentCents: number;
	rollover: boolean;
};

/**
 * Goal vs. actual rows: every category with something to measure against,
 * the same number the Goals progress bar uses. Categories with no cap and no
 * rollover are left out. Keeps the goals' order (by name).
 */
export const goalRows = (goals: Goal[], month: string): GoalRow[] => {
	return goals.flatMap((g) =>
		g.available_cents === null
			? []
			: [
					{
						categoryId: g.category_id,
						name: g.name,
						goalCents: g.available_cents,
						spentCents: g.spent_cents,
						rollover: g.rollover_since !== null && g.rollover_since <= month,
					},
				],
	);
};

const shortMonth = new Intl.DateTimeFormat('en-US', {
	month: 'short',
	timeZone: 'UTC',
});

/** Axis label for a trend bar: "Nov", with the year on January ("Jan ’27"). */
export const monthTick = (month: string): string => {
	const [y, m] = month.split('-').map(Number);
	const name = shortMonth.format(new Date(Date.UTC(y, m - 1, 1)));
	return m === 1 ? `${name} ’${String(y).slice(2)}` : name;
};

/** Axis tick in whole dollars, compact from 1,000: 0, 250, 1K, 1.5K, 12K. */
export const compactDollars = (cents: number): string => {
	const dollars = cents / 100;
	const abs = Math.abs(dollars);
	const sign = dollars < 0 ? '−' : '';
	if (abs < 1000) return sign + String(Math.round(abs));
	const k = abs / 1000;
	return (
		sign +
		(k < 10 ? String(Math.round(k * 10) / 10) : String(Math.round(k))) +
		'K'
	);
};

/**
 * Round axis ticks in cents covering min..max (and 0), at most about five
 * steps of 1, 2, 2.5 or 5 × a power of ten dollars.
 */
export const niceTicks = (min: number, max: number): number[] => {
	const lo = Math.min(0, min);
	const hi = Math.max(0, max);
	if (lo === hi) return [0];
	const rough = (hi - lo) / 4;
	const pow = 10 ** Math.floor(Math.log10(rough));
	const step = [1, 2, 2.5, 5, 10].map((m) => m * pow).find((s) => s >= rough)!;
	const ticks: number[] = [];
	for (let t = Math.floor(lo / step) * step; t < hi + step; t += step) {
		ticks.push(t);
	}
	return ticks;
};
