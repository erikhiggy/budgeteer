import { Fragment, useState } from 'react';
import { Navigate, useNavigate, useParams } from 'react-router';
import {
	useAccounts,
	useCategories,
	useCreateTransaction,
	useDeleteTransaction,
	usePayees,
	useTransactions,
	useUpdateTransaction,
} from '../../api/hooks';
import type { Account, Category, Payee } from '../../api/types';
import { Notice } from '../../components/Notice';
import { AccountHeader } from './AccountHeader';
import { AccountSidebar } from './AccountSidebar';
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
		<div className={styles.layout}>
			<AccountSidebar
				accounts={accounts.data}
				onNewAccount={() => setAdding(true)}
			/>
			<section className={styles.view}>
				{adding && (
					<AddAccountForm
						onCreated={(a) => {
							setAdding(false);
							goTo(a);
						}}
						onCancel={() => setAdding(false)}
					/>
				)}
				{/* Keyed so switching accounts starts both fresh (closes editors, clears the entry row). */}
				<Fragment key={account.id}>
					<AccountHeader
						account={account}
						onDeleted={() => navigate('/transactions', { replace: true })}
					/>
					<AccountLedger account={account} />
				</Fragment>
			</section>
		</div>
	);
}

function AccountLedger({ account }: { account: Account }) {
	const categories = useCategories();
	// Suggestions are a nicety: the row still works while they load or if
	// they fail.
	const payees = usePayees();

	if (categories.isPending) return <Notice>Loading categories…</Notice>;
	if (categories.isError) {
		return (
			<Notice tone='error'>
				Couldn’t load categories: {categories.error.message}
			</Notice>
		);
	}

	const props = {
		account,
		categories: categories.data,
		payees: payees.data ?? [],
	};
	return (
		<>
			<Entry {...props} />
			<Ledger {...props} />
		</>
	);
}

type LedgerProps = {
	account: Account;
	categories: Category[];
	payees: Payee[];
};

function Entry({ account, categories, payees }: LedgerProps) {
	const create = useCreateTransaction();
	return (
		<EntryRow
			accountId={account.id}
			categories={categories}
			payees={payees}
			onSave={create.mutateAsync}
		/>
	);
}

function Ledger({ account, categories, payees }: LedgerProps) {
	const transactions = useTransactions(account.id);
	const update = useUpdateTransaction();
	const remove = useDeleteTransaction();

	if (transactions.isPending) return <Notice>Loading transactions…</Notice>;
	if (transactions.isError) {
		return (
			<Notice tone='error'>
				Couldn’t load transactions: {transactions.error.message}
			</Notice>
		);
	}
	return (
		<>
			{remove.error && (
				<Notice tone='error'>
					Couldn’t delete the transaction: {remove.error.message}
				</Notice>
			)}
			<TransactionTable
				transactions={transactions.data}
				categories={categories}
				payees={payees}
				onUpdate={(before, input) => update.mutateAsync({ before, input })}
				onDelete={(tx) => remove.mutate(tx)}
			/>
		</>
	);
}
