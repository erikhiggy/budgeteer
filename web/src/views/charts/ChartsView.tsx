import type { ReactNode } from 'react';
import { Link, Navigate, useNavigate, useParams } from 'react-router';
import {
	useGoals,
	useMonthlySpending,
	useSpendingByCategory,
} from '../../api/hooks';
import { MonthPicker } from '../../components/MonthPicker';
import { Notice } from '../../components/Notice';
import { formatCents } from '../../lib/money';
import { currentMonth, isISOMonth } from '../../lib/month';
import { CategoryChart } from './CategoryChart';
import { goalRows, trendRange } from './chartData';
import { GoalChart } from './GoalChart';
import { TrendChart } from './TrendChart';
import styles from './ChartsView.module.css';

/** Route: /charts/:month. Goes to the current month when none (or a bad one) is given. */
export const ChartsView = () => {
	const { month } = useParams();
	if (!month || !isISOMonth(month)) {
		return <Navigate to={`/charts/${currentMonth()}`} replace />;
	}
	return <MonthCharts month={month} />;
};

const MonthCharts = ({ month }: { month: string }) => {
	const navigate = useNavigate();
	const go = (m: string) => navigate(`/charts/${m}`);
	const { from, to } = trendRange(month);
	const byCategory = useSpendingByCategory(month);
	const trend = useMonthlySpending(from, to);
	const goals = useGoals(month);

	return (
		<section className={styles.view}>
			<MonthPicker month={month} onChange={go} />

			<Card
				title='Spending by category'
				aside={
					byCategory.isSuccess &&
					`${formatCents(byCategory.data.total_cents)} total`
				}
			>
				{byCategory.isPending && <Notice>Loading…</Notice>}
				{byCategory.isError && (
					<Notice tone='error'>
						Couldn’t load spending: {byCategory.error.message}
					</Notice>
				)}
				{byCategory.isSuccess &&
					(byCategory.data.categories.length === 0 ? (
						<Notice>No spending this month.</Notice>
					) : (
						<CategoryChart categories={byCategory.data.categories} />
					))}
			</Card>

			<Card title='Last 12 months' aside='Click a month to see it'>
				{trend.isPending && <Notice>Loading…</Notice>}
				{trend.isError && (
					<Notice tone='error'>
						Couldn’t load the trend: {trend.error.message}
					</Notice>
				)}
				{trend.isSuccess && (
					<TrendChart
						months={trend.data.months}
						selected={month}
						onSelect={go}
					/>
				)}
			</Card>

			<Card title='Goal vs. actual'>
				{goals.isPending && <Notice>Loading…</Notice>}
				{goals.isError && (
					<Notice tone='error'>
						Couldn’t load goals: {goals.error.message}
					</Notice>
				)}
				{goals.isSuccess &&
					(() => {
						const rows = goalRows(goals.data.goals, month);
						return rows.length === 0 ? (
							<Notice>
								No caps this month. Set some on{' '}
								<Link to={`/goals/${month}`}>Goals</Link>.
							</Notice>
						) : (
							<GoalChart rows={rows} />
						);
					})()}
			</Card>
		</section>
	);
};

const Card = ({
	title,
	aside,
	children,
}: {
	title: string;
	aside?: ReactNode;
	children: ReactNode;
}) => {
	return (
		<div className={styles.card}>
			<div className={styles.cardHeader}>
				<h3 className={styles.cardTitle}>{title}</h3>
				{aside && <span className={styles.aside}>{aside}</span>}
			</div>
			{children}
		</div>
	);
};
