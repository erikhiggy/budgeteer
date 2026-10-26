import { useState, type KeyboardEvent } from 'react';
import type { Envelope } from '../../api/types';
import buttons from '../../components/Button.module.css';
import { cx } from '../../lib/cx';
import { formatCents, parseAssigned } from '../../lib/money';
import styles from './EnvelopesTable.module.css';

type Props = {
	envelopes: Envelope[];
	uncategorizedCents: number;
	/** Sets the month's assignment; 0 removes it. Rejects with a shown Error. */
	onAssign: (categoryId: number, cents: number) => Promise<unknown>;
};

/**
 * One row per expense category: assigned this month (click to edit), spent,
 * and the balance, which carries into the next month either way.
 */
export const EnvelopesTable = ({
	envelopes,
	uncategorizedCents,
	onAssign,
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
						envelope={e}
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
		</table>
	);
};

const EnvelopeRow = ({
	envelope: e,
	onAssign,
}: {
	envelope: Envelope;
	onAssign: (cents: number) => Promise<unknown>;
}) => {
	const [editing, setEditing] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const negative = e.balance_cents < 0;

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
					<div className={styles.name}>{e.name}</div>
					<Details envelope={e} />
				</td>
				<td className={styles.money}>
					{editing ? (
						<AssignedEditor
							envelope={e}
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
								e.assigned_cents === 0 && styles.zero,
							)}
							aria-label={`Assigned to ${e.name}`}
							onClick={() => setEditing(true)}
						>
							{formatCents(e.assigned_cents)}
						</button>
					)}
				</td>
				<td className={styles.money}>{formatCents(e.spent_cents)}</td>
				<td
					className={cx(
						styles.money,
						styles.balance,
						negative && styles.negative,
					)}
				>
					{formatCents(e.balance_cents)}
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

/** The small print under a category's name: carryover and the cap as a target. */
const Details = ({ envelope: e }: { envelope: Envelope }) => {
	const notes: string[] = [];
	if (e.carryover_cents > 0) {
		notes.push(`+${formatCents(e.carryover_cents)} from last month`);
	} else if (e.carryover_cents < 0) {
		notes.push(`${formatCents(e.carryover_cents)} overspent last month`);
	}
	if (e.cap_cents !== null) notes.push(`cap ${formatCents(e.cap_cents)}`);
	if (notes.length === 0) return null;
	return <div className={styles.detail}>{notes.join(' · ')}</div>;
};

/** Inline assignment input: Enter saves (empty or 0 removes it), Esc cancels. */
const AssignedEditor = ({
	envelope,
	onSave,
	onCancel,
	onInvalid,
}: {
	envelope: Envelope;
	onSave: (cents: number) => void;
	onCancel: () => void;
	onInvalid: (message: string) => void;
}) => {
	const [value, setValue] = useState(
		envelope.assigned_cents === 0 ? '' : formatCents(envelope.assigned_cents),
	);

	const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
		if (e.nativeEvent.isComposing) return;
		if (e.key === 'Enter') {
			e.preventDefault();
			const parsed = parseAssigned(value);
			if (!parsed.ok) return onInvalid(parsed.error);
			if (parsed.cents === envelope.assigned_cents) return onCancel();
			onSave(parsed.cents);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			onCancel();
		}
	};

	return (
		<input
			aria-label={`Assigned to ${envelope.name}`}
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
