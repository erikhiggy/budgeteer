import { useState, type KeyboardEvent } from 'react';
import type { CardEnvelope, Envelope } from '../../api/types';
import buttons from '../../components/Button.module.css';
import { cx } from '../../lib/cx';
import { formatCents, parseAssigned } from '../../lib/money';
import styles from './EnvelopesTable.module.css';

type Props = {
	envelopes: Envelope[];
	cards: CardEnvelope[];
	uncategorizedCents: number;
	/** Sets the month's assignment; 0 removes it. Rejects with a shown Error. */
	onAssign: (categoryId: number, cents: number) => Promise<unknown>;
	/** onAssign for a credit card's envelope. */
	onAssignCard: (accountId: number, cents: number) => Promise<unknown>;
};

/**
 * One row per expense category: assigned this month (click to edit), spent,
 * and the balance, which carries into the next month either way. Then one
 * row per credit card, whose envelope card purchases fill and payments
 * draw down.
 */
export const EnvelopesTable = ({
	envelopes,
	cards,
	uncategorizedCents,
	onAssign,
	onAssignCard,
}: Props) => {
	return (
		<table className={styles.table}>
			<thead>
				<tr>
					<th>Category</th>
					<th className={styles.money}>Assigned</th>
					<th className={styles.money}>Spent</th>
					<th className={styles.money}>Balance</th>
				</tr>
			</thead>
			<tbody>
				{envelopes.length === 0 && (
					<tr>
						<td colSpan={4} className={styles.empty}>
							No spending categories yet. Add some under Categories.
						</td>
					</tr>
				)}
				{envelopes.map((e) => (
					<EnvelopeRow
						key={e.category_id}
						name={e.name}
						notes={categoryNotes(e)}
						assignedCents={e.assigned_cents}
						spentCents={e.spent_cents}
						balanceCents={e.balance_cents}
						onAssign={(cents) => onAssign(e.category_id, cents)}
					/>
				))}
				{uncategorizedCents !== 0 && (
					<tr className={styles.uncategorized}>
						<td>
							<div className={styles.name}>Uncategorized</div>
							<div className={styles.detail}>
								Give these transactions a category to take them out of an
								envelope
							</div>
						</td>
						<td className={styles.money}>—</td>
						<td className={styles.money}>{formatCents(uncategorizedCents)}</td>
						<td className={styles.money}>—</td>
					</tr>
				)}
			</tbody>
			{cards.length > 0 && (
				<tbody>
					<tr className={styles.section}>
						<th scope='rowgroup'>Credit cards</th>
						<th className={styles.money} />
						<th className={styles.money}>Paid</th>
						<th className={styles.money} />
					</tr>
					{cards.map((c) => (
						<EnvelopeRow
							key={c.account_id}
							name={c.name}
							notes={cardNotes(c)}
							assignedCents={c.assigned_cents}
							spentCents={c.paid_cents}
							balanceCents={c.balance_cents}
							onAssign={(cents) => onAssignCard(c.account_id, cents)}
						/>
					))}
				</tbody>
			)}
		</table>
	);
};

/** The small print under a category: carryover and the cap as a target. */
const categoryNotes = (e: Envelope): string[] => {
	const notes = carryoverNotes(e.carryover_cents);
	if (e.cap_cents !== null) notes.push(`cap ${formatCents(e.cap_cents)}`);
	return notes;
};

/** The small print under a card: carryover, purchases, and what's owed. */
const cardNotes = (c: CardEnvelope): string[] => {
	const notes = carryoverNotes(c.carryover_cents);
	if (c.purchases_cents !== 0) {
		notes.push(`${formatCents(c.purchases_cents)} in purchases`);
	}
	if (c.owed_cents > 0) {
		const uncovered = c.owed_cents - Math.max(c.balance_cents, 0);
		notes.push(
			uncovered > 0
				? `owes ${formatCents(c.owed_cents)} (${formatCents(uncovered)} not covered)`
				: `owes ${formatCents(c.owed_cents)}`,
		);
	}
	return notes;
};

const carryoverNotes = (carryoverCents: number): string[] => {
	if (carryoverCents > 0) {
		return [`+${formatCents(carryoverCents)} from last month`];
	}
	if (carryoverCents < 0) {
		return [`${formatCents(carryoverCents)} overspent last month`];
	}
	return [];
};

const EnvelopeRow = ({
	name,
	notes,
	assignedCents,
	spentCents,
	balanceCents,
	onAssign,
}: {
	name: string;
	notes: string[];
	assignedCents: number;
	spentCents: number;
	balanceCents: number;
	onAssign: (cents: number) => Promise<unknown>;
}) => {
	const [editing, setEditing] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const negative = balanceCents < 0;

	const save = async (cents: number) => {
		setError(null);
		try {
			await onAssign(cents);
			setEditing(false);
		} catch (err) {
			setError(err instanceof Error ? err.message : String(err));
		}
	};

	return (
		<>
			<tr>
				<td>
					<div className={styles.name}>{name}</div>
					{notes.length > 0 && (
						<div className={styles.detail}>{notes.join(' · ')}</div>
					)}
				</td>
				<td className={styles.money}>
					{editing ? (
						<AssignedEditor
							name={name}
							assignedCents={assignedCents}
							onSave={(cents) => void save(cents)}
							onCancel={() => {
								setEditing(false);
								setError(null);
							}}
							onInvalid={setError}
						/>
					) : (
						<button
							type='button'
							className={cx(
								buttons.quiet,
								buttons.small,
								styles.assignButton,
								assignedCents === 0 && styles.zero,
							)}
							aria-label={`Assigned to ${name}`}
							onClick={() => setEditing(true)}
						>
							{formatCents(assignedCents)}
						</button>
					)}
				</td>
				<td className={styles.money}>{formatCents(spentCents)}</td>
				<td
					className={cx(
						styles.money,
						styles.balance,
						negative && styles.negative,
					)}
				>
					{formatCents(balanceCents)}
				</td>
			</tr>
			{error && (
				<tr className={styles.errorRow}>
					<td colSpan={4} role='alert'>
						{error}
					</td>
				</tr>
			)}
		</>
	);
};

/** Inline assignment input: Enter saves (empty or 0 removes it), Esc cancels. */
const AssignedEditor = ({
	name,
	assignedCents,
	onSave,
	onCancel,
	onInvalid,
}: {
	name: string;
	assignedCents: number;
	onSave: (cents: number) => void;
	onCancel: () => void;
	onInvalid: (message: string) => void;
}) => {
	const [value, setValue] = useState(
		assignedCents === 0 ? '' : formatCents(assignedCents),
	);

	const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
		if (e.nativeEvent.isComposing) return;
		if (e.key === 'Enter') {
			e.preventDefault();
			const parsed = parseAssigned(value);
			if (!parsed.ok) return onInvalid(parsed.error);
			if (parsed.cents === assignedCents) return onCancel();
			onSave(parsed.cents);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			onCancel();
		}
	};

	return (
		<input
			aria-label={`Assigned to ${name}`}
			className={styles.assignInput}
			inputMode='decimal'
			autoComplete='off'
			placeholder='0.00'
			autoFocus
			value={value}
			onChange={(e) => setValue(e.target.value)}
			onKeyDown={onKeyDown}
			onBlur={onCancel}
		/>
	);
};
