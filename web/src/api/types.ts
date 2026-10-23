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
	/** Optional; non-zero adds an "Opening balance" transaction dated today. */
	opening_balance_cents?: number;
};

/** PATCH body: only the fields sent change. */
export type AccountUpdate = Partial<Pick<Account, 'name' | 'type'>>;

export type Category = {
	id: number;
	name: string;
	is_income: boolean;
};

export type CategoryInput = {
	name: string;
	is_income: boolean;
};

/** PATCH body: only the fields sent change. */
export type CategoryUpdate = Partial<CategoryInput>;

/** Renaming or deleting this seeded category is a 409. */
export const openingBalanceCategory = 'Opening balance';

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

/** Used for both POST and PATCH; a transaction PATCH takes the full body. */
export type TransactionInput = Omit<Transaction, 'id'>;

/** A payee and the category of its most recent transaction. */
export type Payee = {
	payee: string;
	category_id: number | null;
};

/** Inclusive YYYY-MM-DD bounds; either may be omitted. */
export type DateRange = { from?: string; to?: string };
