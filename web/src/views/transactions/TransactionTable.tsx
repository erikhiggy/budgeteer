import { useMemo, useState, type ReactNode } from 'react';
import type {
	Account,
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
	/** Every account, to name the other side of transfers. */
	accounts: Account[];
	payees: Payee[];
	/** Resolves when saved; rejects with an Error whose message is shown. */
	onUpdate: (before: Transaction, input: TransactionInput) => Promise<unknown>;
	onDelete: (tx: Transaction) => void;
	/** Sets the transaction's cleared flag to the value given. */
	onClear: (tx: Transaction, cleared: boolean) => void;
};

const COLUMNS = 7;

type SortKey = 'cleared' | 'date' | 'payee' | 'category' | 'note' | 'amount';
type Sort = { key: SortKey; dir: 'asc' | 'desc' };

export const TransactionTable = ({
	transactions,
	categories,
	accounts,
	payees,
	onUpdate,
	onDelete,
	onClear,
}: Props) => {
	const categoryName = new Map(categories.map((c) => [c.id, c.name]));
	const accountName = new Map(accounts.map((a) => [a.id, a.name]));

	const categoryCell = (tx: Transaction): string => {
		if (tx.transfer_account_id !== null) {
			const other = accountName.get(tx.transfer_account_id) ?? '—';
			return `Transfer ${tx.amount_cents < 0 ? '→' : '←'} ${other}`;
		}
		if (tx.category_id === null) return 'Uncategorized';
		return categoryName.get(tx.category_id) ?? '—';
	};
	// One row at a time is open for editing.
	const [editingId, setEditingId] = useState<number | null>(null);
	// null keeps the order the transactions arrived in.
	const [sort, setSort] = useState<Sort | null>(null);

	// Clicking a header cycles ascending, descending, then back to unsorted.
	const toggleSort = (key: SortKey) => {
		setSort((prev) => {
			if (prev?.key !== key) return { key, dir: 'asc' };
			return prev.dir === 'asc' ? { key, dir: 'desc' } : null;
		});
	};

	const sorted = useMemo(() => {
		if (!sort) return transactions;
		const value = (tx: Transaction): string | number => {
			switch (sort.key) {
				case 'cleared':
					return tx.cleared ? 1 : 0;
				case 'date':
					return tx.date;
				case 'payee':
					return tx.payee;
				case 'category':
					return categoryCell(tx);
				case 'note':
					return tx.note;
				case 'amount':
					return tx.amount_cents;
			}
		};
		const sign = sort.dir === 'asc' ? 1 : -1;
		// Array.prototype.sort is stable, so ties keep their incoming order.
		return [...transactions].sort((a, b) => {
			const x = value(a);
			const y = value(b);
			if (typeof x === 'number' && typeof y === 'number') {
				return (x - y) * sign;
			}
			return (
				String(x).localeCompare(String(y), undefined, { numeric: true }) * sign
			);
		});
		// categoryCell closes over the name maps, which derive from these props.
	}, [transactions, categories, accounts, sort]);

	const header = (key: SortKey, label: ReactNode, className?: string) => (
		<th
			className={className}
			aria-sort={
				sort?.key !== key
					? undefined
					: sort.dir === 'asc'
						? 'ascending'
						: 'descending'
			}
		>
			<button
				type='button'
				className={styles.sortButton}
				onClick={() => toggleSort(key)}
			>
				{label}
				<span className={styles.sortArrow} aria-hidden='true'>
					{sort?.key === key ? (sort.dir === 'asc' ? '▲' : '▼') : ''}
				</span>
			</button>
		</th>
	);

	return (
		<table className={styles.table}>
			<thead>
				<tr>
					{header(
						'cleared',
						<>
							<Check className={styles.headerIcon} />
							<span className={styles.visuallyHidden}>Cleared</span>
						</>,
						styles.cleared,
					)}
					{header('date', 'Date', styles.date)}
					{header('payee', 'Payee')}
					{header('category', 'Category')}
					{header('note', 'Note')}
					{header('amount', 'Amount', styles.amount)}
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
				{sorted.map((tx) =>
					tx.id === editingId ? (
						<tr key={tx.id} className={styles.editing}>
							<td colSpan={COLUMNS}>
								<EntryRow
									accountId={tx.account_id}
									categories={categories}
									payees={payees}
									accounts={accounts.filter((a) => a.id !== tx.account_id)}
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
							<td className={styles.cleared}>
								<button
									type='button'
									className={styles.clearToggle}
									aria-pressed={tx.cleared}
									aria-label={`Cleared ${describe(tx)}`}
									title={tx.cleared ? 'Mark uncleared' : 'Mark cleared'}
									onClick={() => onClear(tx, !tx.cleared)}
								>
									<span className={styles.ring}>
										<Check />
									</span>
								</button>
							</td>
							<td className={styles.date}>{formatDate(tx.date)}</td>
							<td>{tx.payee}</td>
							<td className={styles.muted}>{categoryCell(tx)}</td>
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
};

/** A check mark that takes its color from the surrounding text. */
const Check = ({ className }: { className?: string }) => {
	return (
		<svg
			viewBox='0 0 16 16'
			width='12'
			height='12'
			aria-hidden='true'
			className={className}
		>
			<path
				d='M3.5 8.5l3 3 6-7'
				fill='none'
				stroke='currentColor'
				strokeWidth='2'
				strokeLinecap='round'
				strokeLinejoin='round'
			/>
		</svg>
	);
};

/**
 * "Trader Joe's, Oct 5, 2026, 45.67", to tell rows' buttons apart for screen
 * readers.
 */
const describe = (tx: Transaction): string => {
	return [tx.payee, formatDate(tx.date), formatAmount(tx.amount_cents)]
		.filter(Boolean)
		.join(', ');
};
