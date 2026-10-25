import { NavLink } from 'react-router';
import type { Account, AccountType } from '../../api/types';
import { cx } from '../../lib/cx';
import { formatCents } from '../../lib/money';
import styles from './AccountSidebar.module.css';

type Props = {
	accounts: Account[];
	onNewAccount: () => void;
};

/** Sidebar group order: money you have first, then debt and long-term money. */
const groups: { type: AccountType; label: string }[] = [
	{ type: 'checking', label: 'Checking' },
	{ type: 'savings', label: 'Savings' },
	{ type: 'cash', label: 'Cash' },
	{ type: 'credit', label: 'Credit cards' },
	{ type: 'investment', label: 'Investments' },
];

const sum = (accounts: Account[]) =>
	accounts.reduce((total, a) => total + a.balance_cents, 0);

const Balance = ({ cents }: { cents: number }) => {
	return (
		<span className={cx(styles.balance, cents < 0 && styles.negative)}>
			{formatCents(cents)}
		</span>
	);
};

/**
 * Every account with its balance, grouped by type with a subtotal per group;
 * the selected one is the current route.
 */
export const AccountSidebar = ({ accounts, onNewAccount }: Props) => {
	return (
		<nav className={styles.sidebar} aria-label='Accounts'>
			{groups.map(({ type, label }) => {
				// The API sorts by name, so each group keeps that order.
				const members = accounts.filter((a) => a.type === type);
				if (members.length === 0) return null;
				const headingId = `account-group-${type}`;
				return (
					<section key={type} aria-labelledby={headingId}>
						<div className={styles.groupHeading}>
							<h2 id={headingId} className={styles.heading}>
								{label}
							</h2>
							<Balance cents={sum(members)} />
						</div>
						<ul className={styles.list}>
							{members.map((a) => (
								<li key={a.id}>
									<NavLink
										to={`/transactions/${a.id}`}
										className={({ isActive }) =>
											cx(styles.account, isActive && styles.active)
										}
									>
										<span className={styles.name}>{a.name}</span>
										<Balance cents={a.balance_cents} />
									</NavLink>
								</li>
							))}
						</ul>
					</section>
				);
			})}
			<div className={styles.total}>
				<span>Total</span>
				<Balance cents={sum(accounts)} />
			</div>
			<button type='button' className={styles.add} onClick={onNewAccount}>
				New account
			</button>
		</nav>
	);
};
