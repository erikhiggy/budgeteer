import { useState } from 'react';
import { useCreateAccount } from '../../api/hooks';
import { accountTypes, type Account, type AccountType } from '../../api/types';
import styles from './AddAccountForm.module.css';

type Props = {
	onCreated: (account: Account) => void;
	onCancel?: () => void;
};

export function AddAccountForm({ onCreated, onCancel }: Props) {
	const [name, setName] = useState('');
	const [type, setType] = useState<AccountType>('checking');
	const create = useCreateAccount();

	return (
		<form
			className={styles.form}
			aria-label='New account'
			onSubmit={(e) => {
				e.preventDefault();
				create.mutate({ name: name.trim(), type }, { onSuccess: onCreated });
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
			<button type='submit' disabled={create.isPending}>
				Add account
			</button>
			{onCancel && (
				<button type='button' className={styles.secondary} onClick={onCancel}>
					Cancel
				</button>
			)}
			{create.error && (
				<p className={styles.error} role='alert'>
					{create.error.message}
				</p>
			)}
		</form>
	);
}
