import { Navigate, useNavigate, useParams } from 'react-router';
import { useGoals, useSetGoal, useUpdateCategory } from '../../api/hooks';
import buttons from '../../components/Button.module.css';
import { Notice } from '../../components/Notice';
import { cx } from '../../lib/cx';
import {
	addMonths,
	currentMonth,
	formatMonth,
	isISOMonth,
} from '../../lib/month';
import { GoalsTable } from './GoalsTable';
import styles from './GoalsView.module.css';

/** Route: /goals/:month. Goes to the current month when none (or a bad one) is given. */
export function GoalsView() {
	const { month } = useParams();
	if (!month || !isISOMonth(month)) {
		return <Navigate to={`/goals/${currentMonth()}`} replace />;
	}
	// Keyed so a cap being edited doesn't follow you to another month.
	return <MonthGoals key={month} month={month} />;
}

function MonthGoals({ month }: { month: string }) {
	const navigate = useNavigate();
	const goals = useGoals(month);
	const setGoal = useSetGoal();
	const updateCategory = useUpdateCategory();
	const today = currentMonth();
	const go = (m: string) => navigate(`/goals/${m}`);

	return (
		<section className={styles.view}>
			<div className={styles.picker}>
				<button
					type='button'
					className={cx(buttons.quiet, styles.step)}
					aria-label='Previous month'
					onClick={() => go(addMonths(month, -1))}
				>
					‹
				</button>
				<h2 className={styles.month}>{formatMonth(month)}</h2>
				<button
					type='button'
					className={cx(buttons.quiet, styles.step)}
					aria-label='Next month'
					onClick={() => go(addMonths(month, 1))}
				>
					›
				</button>
				{month !== today && (
					<button
						type='button'
						className={cx(buttons.quiet, buttons.small)}
						onClick={() => go(today)}
					>
						This month
					</button>
				)}
			</div>

			{goals.isPending && <Notice>Loading goals…</Notice>}
			{goals.isError && (
				<Notice tone='error'>Couldn’t load goals: {goals.error.message}</Notice>
			)}
			{goals.isSuccess && (
				<GoalsTable
					month={month}
					goals={goals.data.goals}
					uncategorizedCents={goals.data.uncategorized_cents}
					onSetCap={(categoryId, cents) =>
						setGoal.mutateAsync({ categoryId, month, amountCents: cents })
					}
					onSetRollover={(id, since) =>
						updateCategory.mutateAsync({ id, input: { rollover_since: since } })
					}
				/>
			)}
			<p className={styles.hint}>
				A cap applies from this month on. With roll over on, what’s left (or
				overspent) carries into next month.
			</p>
		</section>
	);
}
