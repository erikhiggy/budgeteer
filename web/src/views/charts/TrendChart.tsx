import {
	Bar,
	BarChart,
	CartesianGrid,
	Cell,
	Tooltip,
	XAxis,
	YAxis,
} from 'recharts';
import type { MonthSpending } from '../../api/types';
import { chartColors } from '../../lib/chartColors';
import { formatCents } from '../../lib/money';
import { formatMonth } from '../../lib/month';
import { ChartTooltip } from './ChartTooltip';
import { compactDollars, monthTick, niceTicks } from './chartData';

type Props = {
	months: MonthSpending[];
	/** The month on screen, drawn solid; the others are lighter. */
	selected: string;
	onSelect: (month: string) => void;
};

/** A column per month. Clicking one moves the page to that month. */
export const TrendChart = ({ months, selected, onSelect }: Props) => {
	const spent = months.map((m) => m.spent_cents);
	const ticks = niceTicks(Math.min(...spent), Math.max(...spent));
	return (
		<BarChart
			responsive
			style={{ width: '100%', height: 240 }}
			data={months}
			margin={{ top: 8, right: 4, bottom: 4, left: 4 }}
		>
			<CartesianGrid vertical={false} stroke={chartColors.grid} />
			<XAxis
				dataKey='month'
				tickFormatter={monthTick}
				stroke={chartColors.axis}
				tickLine={false}
				fontSize={12}
			/>
			<YAxis
				ticks={ticks}
				domain={[ticks[0], ticks[ticks.length - 1]]}
				tickFormatter={compactDollars}
				stroke={chartColors.axis}
				tickLine={false}
				axisLine={false}
				width={48}
				fontSize={12}
			/>
			<Tooltip
				cursor={{ fill: chartColors.grid, fillOpacity: 0.6 }}
				content={({ active, payload }) => {
					const m = payload?.[0]?.payload as MonthSpending | undefined;
					if (!active || !m) return null;
					return (
						<ChartTooltip
							title={formatMonth(m.month)}
							lines={[{ label: 'Spent', value: formatCents(m.spent_cents) }]}
						/>
					);
				}}
			/>
			<Bar
				dataKey='spent_cents'
				fill={chartColors.spent}
				radius={[4, 4, 0, 0]}
				maxBarSize={24}
				isAnimationActive={false}
				cursor='pointer'
				onClick={(bar) => {
					const m = (bar.payload as MonthSpending | undefined)?.month;
					if (m) onSelect(m);
				}}
			>
				{months.map((m) => (
					<Cell key={m.month} fillOpacity={m.month === selected ? 1 : 0.45} />
				))}
			</Bar>
		</BarChart>
	);
};
