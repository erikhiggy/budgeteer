import { useState } from 'react';
import { useCreateAccount } from '../../api/hooks';
import { accountTypes, type Account, type AccountType } from '../../api/types';
import buttons from '../../components/Button.module.css';
import { parseBalance } from '../../lib/money';
import styles from './AddAccountForm.module.css';

type Props = {
	onCreated: (account: Account) => void;
	onCancel?: () => void;
};

export function AddAccountForm({ onCreated, onCancel }: Props) {
	const [name, setName] = useState('');
	const [type, setType] = useState<AccountType>('checking');
	const [balance, setBalance] = useState('');
	const [balanceError, setBalanceError] = useState<string | null>(null);
	const create = useCreateAccount();

	const error = balanceError ?? create.error?.message;

	return (
		<form
			className={styles.form}
			aria-label='New account'
			onSubmit={(e) => {
				e.preventDefault();
				const parsed = parseBalance(balance);
				if (!parsed.ok) {
					setBalanceError(parsed.error);
					return;
				}
				setBalanceError(null);
				create.mutate(
					{
						name: name.trim(),
						type,
						opening_balance_cents: parsed.cents,
					},
					{ onSuccess: onCreated },
				);
			}}
			onKeyDown={(e) => {
				if (e.key === 'Escape' && onCancel) onCancel();
			}}
		>
			<input
				aria-label='Account name'
				placeholder='Account name'
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
			<input
				aria-label='Opening balance'
				placeholder='Opening balance'
				inputMode='decimal'
				autoComplete='off'
				aria-invalid={balanceError ? true : undefined}
				className={styles.balance}
				value={balance}
				onChange={(e) => setBalance(e.target.value)}
			/>
			<button type='submit' disabled={create.isPending}>
				Add account
			</button>
			{onCancel && (
				<button type='button' className={buttons.quiet} onClick={onCancel}>
					Cancel
				</button>
			)}
			{error ? (
				<p className={styles.error} role='alert'>
					{error}
				</p>
			) : (
				<p className={styles.hint}>
					Opening balance is optional. Use a minus for money owed, like a credit
					card balance: <kbd>-450.00</kbd>
				</p>
			)}
		</form>
	);
}
