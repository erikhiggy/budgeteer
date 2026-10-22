// Mirrors docs/api.md. Keep the two in sync.

export type AccountType = 'checking' | 'savings' | 'credit' | 'cash';

export const accountTypes: AccountType[] = [
	'checking',
	'savings',
	'credit',
	'cash',
];

export type Account = {
	id: number;
	name: string;
	type: AccountType;
	balance_cents: number;
};

export type AccountInput = {
	name: string;
	type: AccountType;
};

export type Category = {
	id: number;
	name: string;
	is_income: boolean;
};

export type CategoryInput = {
	name: string;
	is_income: boolean;
};

export type Transaction = {
	id: number;
	account_id: number;
	category_id: number | null;
	/** YYYY-MM-DD */
	date: string;
	payee: string;
	/** Negative = money out, positive = money in. */
	amount_cents: number;
	note: string;
};

export type TransactionInput = Omit<Transaction, 'id'>;

/** Inclusive YYYY-MM-DD bounds; either may be omitted. */
export type DateRange = { from?: string; to?: string };
