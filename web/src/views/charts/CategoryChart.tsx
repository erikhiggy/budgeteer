import {
	Bar,
	BarChart,
	CartesianGrid,
	LabelList,
	Tooltip,
	XAxis,
	YAxis,
} from 'recharts';
import type { CategorySpending } from '../../api/types';
import { chartColors } from '../../lib/chartColors';
import { formatCents } from '../../lib/money';
import { ChartTooltip } from './ChartTooltip';
import { compactDollars, niceTicks } from './chartData';

const ROW = 32;

/** Horizontal bars, biggest first, with the amount at each bar's tip. */
export function CategoryChart({
	categories,
}: {
	categories: CategorySpending[];
}) {
	const longest = Math.max(...categories.map((c) => c.name.length));
	const spent = categories.map((c) => c.spent_cents);
	const ticks = niceTicks(Math.min(...spent), Math.max(...spent));
	return (
		<BarChart
			responsive
			style={{ width: '100%', height: categories.length * ROW + 40 }}
			data={categories}
			layout='vertical'
			margin={{ top: 4, right: 72, bottom: 4, left: 4 }}
		>
			<CartesianGrid horizontal={false} stroke={chartColors.grid} />
			<XAxis
				type='number'
				ticks={ticks}
				domain={[ticks[0], ticks[ticks.length - 1]]}
				tickFormatter={compactDollars}
				stroke={chartColors.axis}
				tickLine={false}
				axisLine={false}
				fontSize={12}
			/>
			<YAxis
				type='category'
				dataKey='name'
				width={Math.min(180, Math.max(80, longest * 7.5))}
				stroke={chartColors.axis}
				tick={{ fill: chartColors.text }}
				tickLine={false}
				fontSize={13}
			/>
			<Tooltip
				cursor={{ fill: chartColors.grid, fillOpacity: 0.6 }}
				content={({ active, payload }) => {
					const c = payload?.[0]?.payload as CategorySpending | undefined;
					if (!active || !c) return null;
					return (
						<ChartTooltip
							title={c.name}
							lines={[{ label: 'Spent', value: formatCents(c.spent_cents) }]}
						/>
					);
				}}
			/>
			<Bar
				dataKey='spent_cents'
				fill={chartColors.spent}
				radius={[0, 4, 4, 0]}
				maxBarSize={24}
				isAnimationActive={false}
			>
				<LabelList
					dataKey='spent_cents'
					position='right'
					formatter={(v) => formatCents(Number(v))}
					fill={chartColors.text}
					fontSize={12}
				/>
			</Bar>
		</BarChart>
	);
}
