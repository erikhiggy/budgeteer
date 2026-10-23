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
	/** "YYYY-MM" rollover starts from; null means off. */
	rollover_since: string | null;
};

export type CategoryInput = {
	name: string;
	is_income: boolean;
};

/**
 * PATCH body: only the fields sent change. rollover_since: null turns
 * rollover off; setting a month starts it fresh from there.
 */
export type CategoryUpdate = Partial<
	CategoryInput & { rollover_since: string | null }
>;

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
	/** Set on both legs of a transfer; null on ordinary transactions. */
	transfer_id: number | null;
	/** The other leg's account. */
	transfer_account_id: number | null;
};

/**
 * Used for both POST and PATCH; a transaction PATCH takes the full body.
 * PATCH on a transfer leg updates both legs.
 */
export type TransactionInput = Omit<
	Transaction,
	'id' | 'transfer_id' | 'transfer_account_id'
>;

/**
 * POST /api/transfers, written from account_id's side: negative amount_cents
 * is money out of account_id into other_account_id, positive is money in.
 */
export type TransferInput = {
	account_id: number;
	other_account_id: number;
	date: string;
	payee: string;
	amount_cents: number;
	note: string;
};

/** legs[0] is in the request's account_id. */
export type Transfer = { id: number; legs: [Transaction, Transaction] };

/** A payee and the category of its most recent transaction. */
export type Payee = {
	payee: string;
	category_id: number | null;
};

/** Inclusive YYYY-MM-DD bounds; either may be omitted. */
export type DateRange = { from?: string; to?: string };

/** One expense category's cap and spending for a month. See docs/api.md. */
export type Goal = {
	category_id: number;
	name: string;
	/** The cap in effect for the month; null = no cap. */
	cap_cents: number | null;
	/** "YYYY-MM" the cap was set, which may be before the month shown. */
	cap_since: string | null;
	rollover_since: string | null;
	/** Leftover from earlier months (negative if overspent); 0 without rollover. */
	carryover_cents: number;
	/** Cap plus carryover; null with no cap and no rollover. */
	available_cents: number | null;
	spent_cents: number;
};

export type Goals = {
	/** "YYYY-MM" */
	month: string;
	/** Every expense category, sorted by name. */
	goals: Goal[];
	/** Uncategorized spending in the month, not counting transfers. */
	uncategorized_cents: number;
};
