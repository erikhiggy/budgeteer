import { useState } from 'react';
import type {
	Category,
	Payee,
	Transaction,
	TransactionInput,
} from '../../api/types';
import buttons from '../../components/Button.module.css';
import { ConfirmButton } from '../../components/ConfirmButton';
import { cx } from '../../lib/cx';
import { formatDate } from '../../lib/date';
import { formatAmount } from '../../lib/money';
import { EntryRow } from './EntryRow';
import styles from './TransactionTable.module.css';

type Props = {
	transactions: Transaction[];
	categories: Category[];
	payees: Payee[];
	/** Resolves when saved; rejects with an Error whose message is shown. */
	onUpdate: (before: Transaction, input: TransactionInput) => Promise<unknown>;
	onDelete: (tx: Transaction) => void;
};

const COLUMNS = 6;

export function TransactionTable({
	transactions,
	categories,
	payees,
	onUpdate,
	onDelete,
}: Props) {
	const categoryName = new Map(categories.map((c) => [c.id, c.name]));
	// One row at a time is open for editing.
	const [editingId, setEditingId] = useState<number | null>(null);

	return (
		<table className={styles.table}>
			<thead>
				<tr>
					<th className={styles.date}>Date</th>
					<th>Payee</th>
					<th>Category</th>
					<th>Note</th>
					<th className={styles.amount}>Amount</th>
					<th className={styles.actions}>
						<span className={styles.visuallyHidden}>Actions</span>
					</th>
				</tr>
			</thead>
			<tbody>
				{transactions.length === 0 && (
					<tr>
						<td colSpan={COLUMNS} className={styles.empty}>
							No transactions yet. Add one above.
						</td>
					</tr>
				)}
				{transactions.map((tx) =>
					tx.id === editingId ? (
						<tr key={tx.id} className={styles.editing}>
							<td colSpan={COLUMNS}>
								<EntryRow
									accountId={tx.account_id}
									categories={categories}
									payees={payees}
									initial={tx}
									onSave={async (input) => {
										await onUpdate(tx, input);
										setEditingId(null);
									}}
									onCancel={() => setEditingId(null)}
									className={styles.editor}
								/>
							</td>
						</tr>
					) : (
						<tr key={tx.id}>
							<td className={styles.date}>{formatDate(tx.date)}</td>
							<td>{tx.payee}</td>
							<td className={styles.muted}>
								{tx.category_id === null
									? 'Uncategorized'
									: (categoryName.get(tx.category_id) ?? '—')}
							</td>
							<td className={styles.muted}>{tx.note}</td>
							<td
								className={cx(
									styles.amount,
									tx.amount_cents > 0 && styles.inflow,
								)}
							>
								{formatAmount(tx.amount_cents)}
							</td>
							<td className={styles.actions}>
								<button
									type='button'
									className={cx(buttons.quiet, buttons.small)}
									aria-label={`Edit ${describe(tx)}`}
									onClick={() => setEditingId(tx.id)}
								>
									Edit
								</button>
								<ConfirmButton
									aria-label={`Delete ${describe(tx)}`}
									onConfirm={() => onDelete(tx)}
								>
									Delete
								</ConfirmButton>
							</td>
						</tr>
					),
				)}
			</tbody>
		</table>
	);
}

/**
 * "Trader Joe's, Oct 5, 2026, 45.67", to tell rows' buttons apart for screen
 * readers.
 */
function describe(tx: Transaction): string {
	return [tx.payee, formatDate(tx.date), formatAmount(tx.amount_cents)]
		.filter(Boolean)
		.join(', ');
}
