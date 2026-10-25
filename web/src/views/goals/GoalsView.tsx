import { Navigate, useNavigate, useParams } from 'react-router';
import { useGoals, useSetGoal, useUpdateCategory } from '../../api/hooks';
import { MonthPicker } from '../../components/MonthPicker';
import { Notice } from '../../components/Notice';
import { currentMonth, isISOMonth } from '../../lib/month';
import { GoalsTable } from './GoalsTable';
import styles from './GoalsView.module.css';

/** Route: /goals/:month. Goes to the current month when none (or a bad one) is given. */
export const GoalsView = () => {
	const { month } = useParams();
	if (!month || !isISOMonth(month)) {
		return <Navigate to={`/goals/${currentMonth()}`} replace />;
	}
	// Keyed so a cap being edited doesn't follow you to another month.
	return <MonthGoals key={month} month={month} />;
};

const MonthGoals = ({ month }: { month: string }) => {
	const navigate = useNavigate();
	const goals = useGoals(month);
	const setGoal = useSetGoal();
	const updateCategory = useUpdateCategory();

	return (
		<section className={styles.view}>
			<MonthPicker month={month} onChange={(m) => navigate(`/goals/${m}`)} />

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
};
