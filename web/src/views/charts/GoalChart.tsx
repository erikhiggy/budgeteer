import {
	Bar,
	BarChart,
	CartesianGrid,
	Legend,
	Tooltip,
	XAxis,
	YAxis,
} from 'recharts';
import { chartColors } from '../../lib/chartColors';
import { formatCents } from '../../lib/money';
import { ChartTooltip } from './ChartTooltip';
import { compactDollars, niceTicks, type GoalRow } from './chartData';

const ROW = 52;

/** A pair of bars per category: the goal, then what was spent. */
export const GoalChart = ({ rows }: { rows: GoalRow[] }) => {
	const longest = Math.max(...rows.map((r) => r.name.length));
	const values = rows.flatMap((r) => [r.goalCents, r.spentCents]);
	const ticks = niceTicks(Math.min(...values), Math.max(...values));
	return (
		<BarChart
			responsive
			style={{ width: '100%', height: rows.length * ROW + 64 }}
			data={rows}
			layout='vertical'
			barGap={2}
			margin={{ top: 4, right: 24, bottom: 4, left: 4 }}
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
			<Legend
				verticalAlign='top'
				align='left'
				iconType='square'
				iconSize={10}
				wrapperStyle={{ fontSize: 13, paddingBottom: 8 }}
				// Legend text stays in the text color; the swatch carries identity.
				formatter={(value) => (
					<span style={{ color: chartColors.text }}>{value}</span>
				)}
			/>
			<Tooltip
				cursor={{ fill: chartColors.grid, fillOpacity: 0.6 }}
				content={({ active, payload }) => {
					const r = payload?.[0]?.payload as GoalRow | undefined;
					if (!active || !r) return null;
					const left = r.goalCents - r.spentCents;
					return (
						<ChartTooltip
							title={r.name}
							lines={[
								{
									label: r.rollover ? 'Available' : 'Cap',
									value: formatCents(r.goalCents),
									color: chartColors.goal,
								},
								{
									label: 'Spent',
									value: formatCents(r.spentCents),
									color: chartColors.spent,
								},
							]}
							footer={
								left < 0
									? `${formatCents(-left)} over`
									: `${formatCents(left)} left`
							}
						/>
					);
				}}
			/>
			<Bar
				name='Goal (cap, or available with rollover)'
				dataKey='goalCents'
				fill={chartColors.goal}
				radius={[0, 4, 4, 0]}
				maxBarSize={18}
				isAnimationActive={false}
			/>
			<Bar
				name='Spent'
				dataKey='spentCents'
				fill={chartColors.spent}
				radius={[0, 4, 4, 0]}
				maxBarSize={18}
				isAnimationActive={false}
			/>
		</BarChart>
	);
};
