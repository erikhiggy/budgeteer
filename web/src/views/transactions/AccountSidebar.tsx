import { NavLink } from 'react-router';
import type { Account } from '../../api/types';
import { cx } from '../../lib/cx';
import { formatCents } from '../../lib/money';
import styles from './AccountSidebar.module.css';

type Props = {
	accounts: Account[];
	onNewAccount: () => void;
};

/** Every account with its balance; the selected one is the current route. */
export function AccountSidebar({ accounts, onNewAccount }: Props) {
	const total = accounts.reduce((sum, a) => sum + a.balance_cents, 0);

	return (
		<nav className={styles.sidebar} aria-label='Accounts'>
			<h2 className={styles.heading}>Accounts</h2>
			<ul className={styles.list}>
				{accounts.map((a) => (
					<li key={a.id}>
						<NavLink
							to={`/transactions/${a.id}`}
							className={({ isActive }) =>
								cx(styles.account, isActive && styles.active)
							}
						>
							<span className={styles.name}>{a.name}</span>
							<span
								className={cx(
									styles.balance,
									a.balance_cents < 0 && styles.negative,
								)}
							>
								{formatCents(a.balance_cents)}
							</span>
						</NavLink>
					</li>
				))}
			</ul>
			<div className={styles.total}>
				<span>Total</span>
				<span className={cx(styles.balance, total < 0 && styles.negative)}>
					{formatCents(total)}
				</span>
			</div>
			<button type='button' className={styles.add} onClick={onNewAccount}>
				New account
			</button>
		</nav>
	);
}
