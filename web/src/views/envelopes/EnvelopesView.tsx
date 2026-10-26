import { useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router';
import {
	useEnvelopes,
	useFillFromCaps,
	useSetAssignment,
	useSetCardAssignment,
	useSetEnvelopesStart,
} from '../../api/hooks';
import type { Envelopes } from '../../api/types';
import { ConfirmButton } from '../../components/ConfirmButton';
import { MonthPicker } from '../../components/MonthPicker';
import { Notice } from '../../components/Notice';
import { cx } from '../../lib/cx';
import { formatCents } from '../../lib/money';
import { currentMonth, formatMonth, isISOMonth } from '../../lib/month';
import { EnvelopesTable } from './EnvelopesTable';
import styles from './EnvelopesView.module.css';

/** Route: /envelopes/:month. Goes to the current month when none (or a bad one) is given. */
export const EnvelopesView = () => {
	const { month } = useParams();
	if (!month || !isISOMonth(month)) {
		return <Navigate to={`/envelopes/${currentMonth()}`} replace />;
	}
	// Keyed so an assignment being edited doesn't follow you to another month.
	return <MonthEnvelopes key={month} month={month} />;
};

const MonthEnvelopes = ({ month }: { month: string }) => {
	const navigate = useNavigate();
	const envelopes = useEnvelopes(month);
	const setStart = useSetEnvelopesStart();
	const setAssignment = useSetAssignment();
	const setCardAssignment = useSetCardAssignment();
	const fillFromCaps = useFillFromCaps();
	const [error, setError] = useState<string | null>(null);

	const run = async (action: Promise<unknown>) => {
		setError(null);
		try {
			await action;
		} catch (err) {
			setError(err instanceof Error ? err.message : String(err));
		}
	};

	const start = envelopes.data?.start_month ?? null;
	const startButton = (
		<button
			type='button'
			disabled={setStart.isPending}
			onClick={() => void run(setStart.mutateAsync(month))}
		>
			{start === null
				? `Start envelopes in ${formatMonth(month)}`
				: `Move the start to ${formatMonth(month)}`}
		</button>
	);

	return (
		<section className={styles.view}>
			<MonthPicker
				month={month}
				onChange={(m) => navigate(`/envelopes/${m}`)}
			/>

			{envelopes.isPending && <Notice>Loading envelopes…</Notice>}
			{envelopes.isError && (
				<Notice tone='error'>
					Couldn’t load envelopes: {envelopes.error.message}
				</Notice>
			)}
			{error && <Notice tone='error'>{error}</Notice>}

			{envelopes.isSuccess && start === null && (
				<div className={styles.intro}>
					<p>
						Envelope budgeting is an experiment next to caps. You give every
						dollar you have a job by assigning it to a category. What’s left in
						an envelope, or overspent, carries into the next month.
					</p>
					<p>
						Your account balances at the start (except investments and credit
						cards) and any income after that are what you can assign. Each
						credit card gets an envelope: purchases on it move money there to
						pay the card, and debt you already owe waits there until you assign
						toward it. Caps and Goals don’t change.
					</p>
					{startButton}
				</div>
			)}

			{envelopes.isSuccess && start !== null && month < start && (
				<div className={styles.intro}>
					<p>Envelopes start in {formatMonth(start)}.</p>
					<div className={styles.actions}>
						<button
							type='button'
							onClick={() => navigate(`/envelopes/${start}`)}
						>
							Go to {formatMonth(start)}
						</button>
						{startButton}
					</div>
				</div>
			)}

			{envelopes.isSuccess && start !== null && month >= start && (
				<>
					<Summary
						envelopes={envelopes.data}
						filling={fillFromCaps.isPending}
						onFillFromCaps={() => void run(fillFromCaps.mutateAsync(month))}
					/>
					<EnvelopesTable
						envelopes={envelopes.data.envelopes}
						cards={envelopes.data.cards}
						uncategorizedCents={envelopes.data.uncategorized_cents}
						onAssign={(categoryId, cents) =>
							setAssignment.mutateAsync({
								categoryId,
								month,
								amountCents: cents,
							})
						}
						onAssignCard={(accountId, cents) =>
							setCardAssignment.mutateAsync({
								accountId,
								month,
								amountCents: cents,
							})
						}
					/>
					<div className={styles.footer}>
						<p className={styles.hint}>
							Click an amount to assign; a leading - takes money out. Envelopes
							count from {formatMonth(start)}.
						</p>
						<ConfirmButton
							confirmLabel='Delete every assignment'
							disabled={setStart.isPending}
							onConfirm={() => void run(setStart.mutateAsync(null))}
						>
							Stop envelopes
						</ConfirmButton>
					</div>
				</>
			)}
		</section>
	);
};

/** Ready to assign, the month's income and assigned totals, and "Assign caps". */
const Summary = ({
	envelopes: e,
	filling,
	onFillFromCaps,
}: {
	envelopes: Envelopes;
	filling: boolean;
	onFillFromCaps: () => void;
}) => {
	const over = e.ready_to_assign_cents < 0;
	return (
		<div className={styles.summary}>
			<div className={cx(styles.ready, over && styles.over)}>
				<div className={styles.readyAmount}>
					{formatCents(e.ready_to_assign_cents)}
				</div>
				<div className={styles.label}>
					{over ? 'Over-assigned' : 'Ready to assign'}
				</div>
			</div>
			<dl className={styles.totals}>
				<div>
					<dt>Income this month</dt>
					<dd>{formatCents(e.income_cents)}</dd>
				</div>
				<div>
					<dt>Assigned this month</dt>
					<dd>{formatCents(e.assigned_cents)}</dd>
				</div>
			</dl>
			<button
				type='button'
				className={styles.fill}
				disabled={filling}
				title='Assigns each capped category its cap, where nothing is assigned yet this month'
				onClick={onFillFromCaps}
			>
				Assign caps
			</button>
		</div>
	);
};
