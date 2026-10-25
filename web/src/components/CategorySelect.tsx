import type { Ref } from 'react';
import type { Account, Category } from '../api/types';

type Props = {
	categories: Category[];
	/**
	 * A category id as a string, a transfer target as transferValue(id), or
	 * '' for none selected.
	 */
	value: string;
	onChange: (value: string) => void;
	/** Accounts to offer under "Transfer", e.g. every account but this one. */
	transferAccounts?: Account[];
	invalid?: boolean;
	disabled?: boolean;
	className?: string;
	ref?: Ref<HTMLSelectElement>;
};

const TRANSFER_PREFIX = 'transfer:';

/** The select value for a transfer to or from account id. */
export const transferValue = (accountId: number): string => {
	return TRANSFER_PREFIX + accountId;
};

/** The account id in a transfer value, or null for a category or ''. */
export const transferAccountId = (value: string): number | null => {
	return value.startsWith(TRANSFER_PREFIX)
		? Number(value.slice(TRANSFER_PREFIX.length))
		: null;
};

/**
 * Native select (keeps type-to-jump and Tab) grouped into expense and
 * income categories, plus transfer targets when given.
 */
export const CategorySelect = ({
	categories,
	value,
	onChange,
	transferAccounts = [],
	invalid,
	disabled,
	className,
	ref,
}: Props) => {
	const expense = categories.filter((c) => !c.is_income);
	const income = categories.filter((c) => c.is_income);

	return (
		<select
			ref={ref}
			aria-label='Category'
			aria-invalid={invalid || undefined}
			disabled={disabled}
			className={className}
			value={value}
			onChange={(e) => onChange(e.target.value)}
		>
			<option value=''>Category…</option>
			{expense.length > 0 && (
				<optgroup label='Spending'>
					{expense.map((c) => (
						<option key={c.id} value={c.id}>
							{c.name}
						</option>
					))}
				</optgroup>
			)}
			{income.length > 0 && (
				<optgroup label='Income'>
					{income.map((c) => (
						<option key={c.id} value={c.id}>
							{c.name}
						</option>
					))}
				</optgroup>
			)}
			{transferAccounts.length > 0 && (
				<optgroup label='Transfer'>
					{transferAccounts.map((a) => (
						<option key={a.id} value={transferValue(a.id)}>
							⇄ {a.name}
						</option>
					))}
				</optgroup>
			)}
		</select>
	);
};
