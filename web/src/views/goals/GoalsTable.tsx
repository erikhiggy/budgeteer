import { useState, type KeyboardEvent } from 'react';
import type { Goal } from '../../api/types';
import buttons from '../../components/Button.module.css';
import { cx } from '../../lib/cx';
import { formatCents, parseCap } from '../../lib/money';
import { formatMonthShort } from '../../lib/month';
import styles from './GoalsTable.module.css';

type Props = {
	/** "YYYY-MM" on screen. */
	month: string;
	goals: Goal[];
	uncategorizedCents: number;
	/** Sets the cap from month on, or removes it (null). Rejects with a shown Error. */
	onSetCap: (categoryId: number, cents: number | null) => Promise<unknown>;
	/** Turns rollover on from month, or off (null). Rejects with a shown Error. */
	onSetRollover: (categoryId: number, since: string | null) => Promise<unknown>;
};

/**
 * One row per expense category: cap (click to edit), spent, remaining, a
 * progress bar, and a rollover toggle. Rollover rows measure spending
 * against available (cap plus carryover) instead of the cap alone.
 */
export function GoalsTable({
	month,
	goals,
	uncategorizedCents,
	onSetCap,
	onSetRollover,
}: Props) {
	return (
		<table className={styles.table}>
			<thead>
				<tr>
					<th>Category</th>
					<th className={styles.money}>Cap</th>
					<th className={styles.money}>Spent</th>
					<th className={styles.money}>Remaining</th>
					<th className={styles.progressCell}>
						<span className={styles.visuallyHidden}>Progress</span>
					</th>
					<th className={styles.rollover}>Roll over</th>
				</tr>
			</thead>
			<tbody>
				{goals.length === 0 && (
					<tr>
						<td colSpan={6} className={styles.empty}>
							No spending categories yet. Add some under Categories.
						</td>
					</tr>
				)}
				{goals.map((g) => (
					<GoalRow
						key={g.category_id}
						month={month}
						goal={g}
						onSetCap={(cents) => onSetCap(g.category_id, cents)}
						onSetRollover={(since) => onSetRollover(g.category_id, since)}
					/>
				))}
				{uncategorizedCents !== 0 && (
					<tr className={styles.uncategorized}>
						<td>
							<div className={styles.name}>Uncategorized</div>
							<div className={styles.detail}>
								Give these transactions a category to count them toward a cap
							</div>
						</td>
						<td className={styles.money}>—</td>
						<td className={styles.money}>{formatCents(uncategorizedCents)}</td>
						<td className={styles.money}>—</td>
						<td />
						<td />
					</tr>
				)}
			</tbody>
		</table>
	);
}

function GoalRow({
	month,
	goal: g,
	onSetCap,
	onSetRollover,
}: {
	month: string;
	goal: Goal;
	onSetCap: (cents: number | null) => Promise<unknown>;
	onSetRollover: (since: string | null) => Promise<unknown>;
}) {
	const [editing, setEditing] = useState(false);
	const [error, setError] = useState<string | null>(null);

	const rolling = g.rollover_since !== null && g.rollover_since <= month;
	const available = g.available_cents;
	const remaining = available === null ? null : available - g.spent_cents;
	const over = remaining !== null && remaining < 0;
	// Share of what's available that's been spent, for the bar.
	const ratio =
		available === null
			? null
			: available > 0
				? g.spent_cents / available
				: g.spent_cents > 0
					? Infinity
					: 0;

	async function run(action: Promise<unknown>) {
		setError(null);
		try {
			await action;
			return true;
		} catch (err) {
			setError(err instanceof Error ? err.message : String(err));
			return false;
		}
	}

	return (
		<>
			<tr className={cx(over && styles.overRow)}>
				<td>
					<div className={styles.name}>{g.name}</div>
					<Details goal={g} month={month} rolling={rolling} />
				</td>
				<td className={styles.money}>
					{editing ? (
						<CapEditor
							goal={g}
							onSave={async (cents) => {
								if (await run(onSetCap(cents))) setEditing(false);
							}}
							onCancel={() => {
								setEditing(false);
								setError(null);
							}}
							onInvalid={setError}
						/>
					) : (
						<button
							type='button'
							className={cx(buttons.quiet, buttons.small, styles.capButton)}
							aria-label={`Cap for ${g.name}`}
							onClick={() => setEditing(true)}
						>
							{g.cap_cents === null ? 'Set cap' : formatCents(g.cap_cents)}
						</button>
					)}
				</td>
				<td className={styles.money}>{formatCents(g.spent_cents)}</td>
				<td className={cx(styles.money, over && styles.over)}>
					{remaining === null ? '—' : formatCents(remaining)}
				</td>
				<td className={styles.progressCell}>
					{ratio !== null && (
						<div
							className={styles.track}
							role='progressbar'
							aria-label={`${g.name} spending`}
							aria-valuemin={0}
							aria-valuemax={100}
							aria-valuenow={
								Number.isFinite(ratio) ? Math.round(ratio * 100) : 100
							}
							aria-valuetext={
								over
									? `Over by ${formatCents(-(remaining ?? 0))}`
									: `${formatCents(remaining ?? 0)} left`
							}
						>
							<div
								className={cx(styles.bar, over && styles.overBar)}
								style={{ width: `${Math.min(Math.max(ratio, 0), 1) * 100}%` }}
							/>
						</div>
					)}
				</td>
				<td className={styles.rollover}>
					<input
						type='checkbox'
						aria-label={`Roll over ${g.name}`}
						checked={g.rollover_since !== null}
						onChange={(e) =>
							void run(onSetRollover(e.target.checked ? month : null))
						}
					/>
				</td>
			</tr>
			{error && (
				<tr className={styles.errorRow}>
					<td colSpan={6} role='alert'>
						{error}
					</td>
				</tr>
			)}
		</>
	);
}

/** The small print under a category's name: where the cap and carryover come from. */
function Details({
	goal: g,
	month,
	rolling,
}: {
	goal: Goal;
	month: string;
	rolling: boolean;
}) {
	const notes: string[] = [];
	if (g.cap_since !== null && g.cap_since < month) {
		notes.push(`since ${formatMonthShort(g.cap_since)}`);
	}
	if (g.rollover_since !== null && !rolling) {
		notes.push(`rolls over from ${formatMonthShort(g.rollover_since)}`);
	} else if (rolling && g.carryover_cents > 0) {
		notes.push(`+${formatCents(g.carryover_cents)} rolled over`);
	} else if (rolling && g.carryover_cents < 0) {
		notes.push(`${formatCents(g.carryover_cents)} from last month`);
	} else if (rolling) {
		notes.push('rolls over');
	}
	if (notes.length === 0) return null;
	return <div className={styles.detail}>{notes.join(' · ')}</div>;
}

/** Inline cap input: Enter saves (empty removes the cap), Esc cancels. */
function CapEditor({
	goal,
	onSave,
	onCancel,
	onInvalid,
}: {
	goal: Goal;
	onSave: (cents: number | null) => void;
	onCancel: () => void;
	onInvalid: (message: string) => void;
}) {
	const [value, setValue] = useState(
		goal.cap_cents === null ? '' : formatCents(goal.cap_cents),
	);

	function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
		if (e.nativeEvent.isComposing) return;
		if (e.key === 'Enter') {
			e.preventDefault();
			const parsed = parseCap(value);
			if (!parsed.ok) return onInvalid(parsed.error);
			if (parsed.cents === goal.cap_cents) return onCancel();
			onSave(parsed.cents);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			onCancel();
		}
	}

	return (
		<input
			aria-label={`Cap for ${goal.name}`}
			className={styles.capInput}
			inputMode='decimal'
			autoComplete='off'
			placeholder='No cap'
			autoFocus
			value={value}
			onChange={(e) => setValue(e.target.value)}
			onKeyDown={onKeyDown}
			onBlur={onCancel}
		/>
	);
}
