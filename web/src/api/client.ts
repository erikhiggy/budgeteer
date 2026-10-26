import type {
	Account,
	AccountInput,
	AccountUpdate,
	Category,
	CategoryInput,
	CategoryUpdate,
	DateRange,
	Envelopes,
	Goal,
	Goals,
	MonthlySpending,
	Payee,
	SpendingByCategory,
	Transaction,
	TransactionInput,
	Transfer,
	TransferInput,
} from './types';

/** A non-2xx response. `message` is the server's user-facing error text. */
export class ApiError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

const request = async <T>(
	method: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE',
	path: string,
	body?: unknown,
): Promise<T> => {
	const init: RequestInit = { method };
	if (method !== 'GET') {
		// The server rejects mutations without this header (see guard in
		// internal/api/server.go), even when there's no body.
		init.headers = { 'Content-Type': 'application/json' };
		if (body !== undefined) init.body = JSON.stringify(body);
	}

	const res = await fetch(path, init);
	const text = await res.text();
	const data: unknown = text ? safeParse(text) : undefined;

	if (!res.ok) {
		const message = isErrorBody(data)
			? data.error
			: `${res.status} ${res.statusText}`.trim();
		throw new ApiError(res.status, message);
	}
	return data as T;
};

const safeParse = (text: string): unknown => {
	try {
		return JSON.parse(text);
	} catch {
		return undefined;
	}
};

const isErrorBody = (v: unknown): v is { error: string } => {
	return (
		typeof v === 'object' &&
		v !== null &&
		typeof (v as { error?: unknown }).error === 'string'
	);
};

const query = (params: Record<string, string | number | undefined>): string => {
	const q = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== '') q.set(k, String(v));
	}
	const s = q.toString();
	return s ? `?${s}` : '';
};

export const api = {
	listAccounts: () => request<Account[]>('GET', '/api/accounts'),
	createAccount: (input: AccountInput) =>
		request<Account>('POST', '/api/accounts', input),
	updateAccount: (id: number, input: AccountUpdate) =>
		request<Account>('PATCH', `/api/accounts/${id}`, input),
	deleteAccount: (id: number) => request<void>('DELETE', `/api/accounts/${id}`),

	listCategories: () => request<Category[]>('GET', '/api/categories'),
	createCategory: (input: CategoryInput) =>
		request<Category>('POST', '/api/categories', input),
	updateCategory: (id: number, input: CategoryUpdate) =>
		request<Category>('PATCH', `/api/categories/${id}`, input),
	deleteCategory: (id: number) =>
		request<void>('DELETE', `/api/categories/${id}`),

	listTransactions: (accountId: number, range: DateRange = {}) =>
		request<Transaction[]>(
			'GET',
			`/api/transactions${query({ account_id: accountId, ...range })}`,
		),
	createTransaction: (input: TransactionInput) =>
		request<Transaction>('POST', '/api/transactions', input),
	updateTransaction: (id: number, input: TransactionInput) =>
		request<Transaction>('PATCH', `/api/transactions/${id}`, input),
	deleteTransaction: (id: number) =>
		request<void>('DELETE', `/api/transactions/${id}`),
	createTransfer: (input: TransferInput) =>
		request<Transfer>('POST', '/api/transfers', input),

	listPayees: () => request<Payee[]>('GET', '/api/payees'),

	listGoals: (month: string) =>
		request<Goals>('GET', `/api/goals${query({ month })}`),
	/** Sets the cap from month on; null removes it. */
	setGoal: (categoryId: number, month: string, amountCents: number | null) =>
		request<Goal>('PUT', `/api/goals/${categoryId}${query({ month })}`, {
			amount_cents: amountCents,
		}),

	listEnvelopes: (month: string) =>
		request<Envelopes>('GET', `/api/envelopes${query({ month })}`),
	/** Starts envelopes from month, or stops them (null), deleting every assignment. */
	setEnvelopesStart: (month: string | null) =>
		request<{ start_month: string | null }>('PUT', '/api/envelopes/start', {
			month,
		}),
	/** Sets the month's assignment; 0 removes it. */
	setAssignment: (categoryId: number, month: string, amountCents: number) =>
		request<Envelopes>(
			'PUT',
			`/api/envelopes/${categoryId}${query({ month })}`,
			{
				amount_cents: amountCents,
			},
		),
	/** Sets the month's assignment to a credit card's envelope; 0 removes it. */
	setCardAssignment: (accountId: number, month: string, amountCents: number) =>
		request<Envelopes>(
			'PUT',
			`/api/envelopes/cards/${accountId}${query({ month })}`,
			{ amount_cents: amountCents },
		),
	/** Assigns each capped envelope with nothing assigned yet its cap. */
	fillFromCaps: (month: string) =>
		request<Envelopes>(
			'POST',
			`/api/envelopes/fill-from-caps${query({ month })}`,
		),

	spendingByCategory: (month: string) =>
		request<SpendingByCategory>(
			'GET',
			`/api/reports/spending-by-category${query({ month })}`,
		),
	/** from and to are "YYYY-MM", inclusive. */
	monthlySpending: (from: string, to: string) =>
		request<MonthlySpending>(
			'GET',
			`/api/reports/monthly-spending${query({ from, to })}`,
		),
};
