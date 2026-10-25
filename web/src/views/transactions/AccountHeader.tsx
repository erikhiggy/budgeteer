import { useState } from 'react';
import { useDeleteAccount, useUpdateAccount } from '../../api/hooks';
import { accountTypes, type Account, type AccountType } from '../../api/types';
import buttons from '../../components/Button.module.css';
import { ConfirmButton } from '../../components/ConfirmButton';
import { cx } from '../../lib/cx';
import { formatCents } from '../../lib/money';
import styles from './AccountHeader.module.css';

type Props = {
	account: Account;
	/** Called after the account is deleted. */
	onDeleted: () => void;
};

/** The selected account's name and balance, with rename and delete. */
export const AccountHeader = ({ account, onDeleted }: Props) => {
	const [editing, setEditing] = useState(false);
	const remove = useDeleteAccount();

	if (editing) {
		return (
			<EditAccountForm account={account} onDone={() => setEditing(false)} />
		);
	}

	return (
		<div className={styles.header}>
			<h2 className={styles.name}>{account.name}</h2>
			<span className={styles.type}>{account.type}</span>
			<span
				className={cx(
					styles.balance,
					account.balance_cents < 0 && styles.negative,
				)}
			>
				Balance {formatCents(account.balance_cents)}
			</span>
			<span className={styles.spacer} />
			<button
				type='button'
				className={cx(buttons.quiet, buttons.small)}
				onClick={() => setEditing(true)}
			>
				Edit account
			</button>
			<ConfirmButton
				aria-label='Delete account'
				onConfirm={() => remove.mutate(account.id, { onSuccess: onDeleted })}
				disabled={remove.isPending}
			>
				Delete account
			</ConfirmButton>
			{remove.error && (
				<p className={styles.error} role='alert'>
					{remove.error.message}
				</p>
			)}
		</div>
	);
};

const EditAccountForm = ({
	account,
	onDone,
}: {
	account: Account;
	onDone: () => void;
}) => {
	const [name, setName] = useState(account.name);
	const [type, setType] = useState<AccountType>(account.type);
	const update = useUpdateAccount();

	return (
		<form
			className={styles.header}
			aria-label='Edit account'
			onSubmit={(e) => {
				e.preventDefault();
				const input = {
					...(name.trim() !== account.name && { name: name.trim() }),
					...(type !== account.type && { type }),
				};
				if (Object.keys(input).length === 0) return onDone();
				update.mutate({ id: account.id, input }, { onSuccess: onDone });
			}}
			onKeyDown={(e) => {
				if (e.key === 'Escape') onDone();
			}}
		>
			<input
				aria-label='Account name'
				autoFocus
				required
				value={name}
				onChange={(e) => setName(e.target.value)}
			/>
			<select
				aria-label='Account type'
				value={type}
				onChange={(e) => setType(e.target.value as AccountType)}
			>
				{accountTypes.map((t) => (
					<option key={t} value={t}>
						{t[0].toUpperCase() + t.slice(1)}
					</option>
				))}
			</select>
			<button type='submit' disabled={update.isPending}>
				Save
			</button>
			<button type='button' className={buttons.quiet} onClick={onDone}>
				Cancel
			</button>
			{update.error && (
				<p className={styles.error} role='alert'>
					{update.error.message}
				</p>
			)}
		</form>
	);
};
