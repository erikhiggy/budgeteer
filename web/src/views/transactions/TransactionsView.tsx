import { useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router';
import {
	useAccounts,
	useCategories,
	useCreateTransaction,
	useTransactions,
} from '../../api/hooks';
import type { Account, Category } from '../../api/types';
import { Notice } from '../../components/Notice';
import { cx } from '../../lib/cx';
import { formatCents } from '../../lib/money';
import { AddAccountForm } from './AddAccountForm';
import { EntryRow } from './EntryRow';
import { TransactionTable } from './TransactionTable';
import styles from './TransactionsView.module.css';

/** Route: /transactions/:accountId. Picks the first account when none is given. */
export function TransactionsView() {
	const { accountId } = useParams();
	const navigate = useNavigate();
	const accounts = useAccounts();
	const [adding, setAdding] = useState(false);

	if (accounts.isPending) return <Notice>Loading accounts…</Notice>;
	if (accounts.isError) {
		return (
			<Notice tone='error'>
				Couldn’t load accounts: {accounts.error.message}
			</Notice>
		);
	}

	const goTo = (a: Account) => navigate(`/transactions/${a.id}`);

	if (accounts.data.length === 0) {
		return (
			<section className={styles.firstRun}>
				<h2 className={styles.heading}>Add your first account</h2>
				<p className={styles.muted}>
					An account is anywhere money lives: checking, a credit card, cash.
				</p>
				<AddAccountForm onCreated={goTo} />
			</section>
		);
	}

	const account = accounts.data.find((a) => String(a.id) === accountId);
	if (!account) {
		return <Navigate to={`/transactions/${accounts.data[0].id}`} replace />;
	}

	return (
		<section className={styles.view}>
			<div className={styles.toolbar}>
				<select
					aria-label='Account'
					className={styles.accountSelect}
					value={account.id}
					onChange={(e) => navigate(`/transactions/${e.target.value}`)}
				>
					{accounts.data.map((a) => (
						<option key={a.id} value={a.id}>
							{a.name}
						</option>
					))}
				</select>
				<span
					className={cx(
						styles.balance,
						account.balance_cents < 0 && styles.negative,
					)}
				>
					Balance {formatCents(account.balance_cents)}
				</span>
				<span className={styles.spacer} />
				{!adding && (
					<button type='button' onClick={() => setAdding(true)}>
						New account
					</button>
				)}
			</div>

			{adding && (
				<AddAccountForm
					onCreated={(a) => {
						setAdding(false);
						goTo(a);
					}}
					onCancel={() => setAdding(false)}
				/>
			)}

			<AccountLedger key={account.id} account={account} />
		</section>
	);
}

function AccountLedger({ account }: { account: Account }) {
	const categories = useCategories();

	if (categories.isPending) return <Notice>Loading categories…</Notice>;
	if (categories.isError) {
		return (
			<Notice tone='error'>
				Couldn’t load categories: {categories.error.message}
			</Notice>
		);
	}

	return (
		<>
			<Entry account={account} categories={categories.data} />
			<Ledger account={account} categories={categories.data} />
		</>
	);
}

function Entry({
	account,
	categories,
}: {
	account: Account;
	categories: Category[];
}) {
	const create = useCreateTransaction();
	return (
		<EntryRow
			accountId={account.id}
			categories={categories}
			onSave={create.mutateAsync}
		/>
	);
}

function Ledger({
	account,
	categories,
}: {
	account: Account;
	categories: Category[];
}) {
	const transactions = useTransactions(account.id);

	if (transactions.isPending) return <Notice>Loading transactions…</Notice>;
	if (transactions.isError) {
		return (
			<Notice tone='error'>
				Couldn’t load transactions: {transactions.error.message}
			</Notice>
		);
	}
	return (
		<TransactionTable
			transactions={transactions.data}
			categories={categories}
		/>
	);
}
