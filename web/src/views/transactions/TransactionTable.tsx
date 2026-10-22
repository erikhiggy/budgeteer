import type { Category, Transaction } from '../../api/types';
import { cx } from '../../lib/cx';
import { formatDate } from '../../lib/date';
import { formatAmount } from '../../lib/money';
import styles from './TransactionTable.module.css';

type Props = {
	transactions: Transaction[];
	categories: Category[];
};

export function TransactionTable({ transactions, categories }: Props) {
	const categoryName = new Map(categories.map((c) => [c.id, c.name]));

	return (
		<table className={styles.table}>
			<thead>
				<tr>
					<th className={styles.date}>Date</th>
					<th>Payee</th>
					<th>Category</th>
					<th>Note</th>
					<th className={styles.amount}>Amount</th>
				</tr>
			</thead>
			<tbody>
				{transactions.length === 0 && (
					<tr>
						<td colSpan={5} className={styles.empty}>
							No transactions yet. Add one above.
						</td>
					</tr>
				)}
				{transactions.map((tx) => (
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
					</tr>
				))}
			</tbody>
		</table>
	);
}
