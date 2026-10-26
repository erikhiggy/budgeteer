import {
	QueryClient,
	useMutation,
	useQuery,
	useQueryClient,
} from '@tanstack/react-query';
import { api, ApiError } from './client';
import type {
	AccountUpdate,
	CategoryUpdate,
	Transaction,
	TransactionInput,
} from './types';

export const queryKeys = {
	accounts: ['accounts'] as const,
	categories: ['categories'] as const,
	transactions: (accountId: number) => ['transactions', accountId] as const,
	payees: ['payees'] as const,
	/** Every month's goals, for invalidating all of them at once. */
	allGoals: ['goals'] as const,
	goals: (month: string) => ['goals', month] as const,
	/** Every month's envelopes, for invalidating all of them at once. */
	allEnvelopes: ['envelopes'] as const,
	envelopes: (month: string) => ['envelopes', month] as const,
	/** Every report, for invalidating all of them at once. */
	allReports: ['reports'] as const,
	spendingByCategory: (month: string) =>
		['reports', 'by-category', month] as const,
	monthlySpending: (from: string, to: string) =>
		['reports', 'monthly', from, to] as const,
};

export const createQueryClient = () => {
	return new QueryClient({
		defaultOptions: {
			queries: {
				// The data only changes when this app changes it, and every
				// mutation invalidates what it touched.
				refetchOnWindowFocus: false,
				staleTime: Infinity,
				// A 4xx won't fix itself; retry only network/server errors, once.
				retry: (failures, err) =>
					!(err instanceof ApiError && err.status < 500) && failures < 1,
			},
		},
	});
};

export const useAccounts = () => {
	return useQuery({ queryKey: queryKeys.accounts, queryFn: api.listAccounts });
};

export const useCreateAccount = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.createAccount,
		// An opening balance also adds a transaction, and with it a payee, and
		// is money envelopes can be filled from.
		onSuccess: () =>
			Promise.all([
				qc.invalidateQueries({ queryKey: queryKeys.accounts }),
				qc.invalidateQueries({ queryKey: queryKeys.payees }),
				qc.invalidateQueries({ queryKey: queryKeys.allEnvelopes }),
			]),
	});
};

export const useUpdateAccount = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: ({ id, input }: { id: number; input: AccountUpdate }) =>
			api.updateAccount(id, input),
		// Investment accounts are off budget, so the type feeds envelopes.
		onSuccess: () =>
			Promise.all([
				qc.invalidateQueries({ queryKey: queryKeys.accounts }),
				qc.invalidateQueries({ queryKey: queryKeys.allEnvelopes }),
			]),
	});
};

export const useDeleteAccount = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.deleteAccount,
		onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.accounts }),
	});
};

export const useCategories = () => {
	return useQuery({
		queryKey: queryKeys.categories,
		queryFn: api.listCategories,
	});
};

export const useCreateCategory = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.createCategory,
		onSuccess: () => invalidateCategories(qc),
	});
};

export const useUpdateCategory = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: ({ id, input }: { id: number; input: CategoryUpdate }) =>
			api.updateCategory(id, input),
		onSuccess: () => invalidateCategories(qc),
	});
};

export const useDeleteCategory = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.deleteCategory,
		onSuccess: () => invalidateCategories(qc),
	});
};

export const useTransactions = (accountId: number) => {
	return useQuery({
		queryKey: queryKeys.transactions(accountId),
		queryFn: () => api.listTransactions(accountId),
	});
};

export const usePayees = () => {
	return useQuery({ queryKey: queryKeys.payees, queryFn: api.listPayees });
};

/**
 * Invalidates what a change to transactions in these accounts touches: their
 * lists, the balances, and the payees. Only queries currently on screen
 * refetch (TanStack's default refetchType: 'active').
 */
const invalidateTransactions = (
	qc: QueryClient,
	...accountIds: (number | null)[]
) => {
	const ids = new Set(accountIds.filter((id) => id !== null));
	return Promise.all([
		...[...ids].map((id) =>
			qc.invalidateQueries({ queryKey: queryKeys.transactions(id) }),
		),
		qc.invalidateQueries({ queryKey: queryKeys.accounts }),
		qc.invalidateQueries({ queryKey: queryKeys.payees }),
		// Spending feeds every month's goals from then on, through rollover,
		// and the reports.
		qc.invalidateQueries({ queryKey: queryKeys.allGoals }),
		qc.invalidateQueries({ queryKey: queryKeys.allEnvelopes }),
		qc.invalidateQueries({ queryKey: queryKeys.allReports }),
	]);
};

/** Categories are also the rows of every month's goals, envelopes and reports. */
const invalidateCategories = (qc: QueryClient) => {
	return Promise.all([
		qc.invalidateQueries({ queryKey: queryKeys.categories }),
		qc.invalidateQueries({ queryKey: queryKeys.allGoals }),
		qc.invalidateQueries({ queryKey: queryKeys.allEnvelopes }),
		qc.invalidateQueries({ queryKey: queryKeys.allReports }),
	]);
};

export const useCreateTransaction = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.createTransaction,
		onSuccess: (tx) => invalidateTransactions(qc, tx.account_id),
	});
};

export const useUpdateTransaction = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: ({
			before,
			input,
		}: {
			before: Transaction;
			input: TransactionInput;
		}) => api.updateTransaction(before.id, input),
		// Moving a transaction changes both accounts' lists, and editing a
		// transfer leg changes the other leg too.
		onSuccess: (tx, { before }) =>
			invalidateTransactions(
				qc,
				before.account_id,
				tx.account_id,
				before.transfer_account_id,
			),
	});
};

export const useDeleteTransaction = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (tx: Transaction) => api.deleteTransaction(tx.id),
		// Deleting a transfer leg deletes the other leg too.
		onSuccess: (_, tx) =>
			invalidateTransactions(qc, tx.account_id, tx.transfer_account_id),
	});
};

export const useCreateTransfer = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.createTransfer,
		onSuccess: (transfer) =>
			invalidateTransactions(qc, ...transfer.legs.map((leg) => leg.account_id)),
	});
};

export const useGoals = (month: string) => {
	return useQuery({
		queryKey: queryKeys.goals(month),
		queryFn: () => api.listGoals(month),
	});
};

export const useSetGoal = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: ({
			categoryId,
			month,
			amountCents,
		}: {
			categoryId: number;
			month: string;
			amountCents: number | null;
		}) => api.setGoal(categoryId, month, amountCents),
		// A cap applies from its month on, and rolls into later months. It's
		// also each envelope's target.
		onSuccess: () =>
			Promise.all([
				qc.invalidateQueries({ queryKey: queryKeys.allGoals }),
				qc.invalidateQueries({ queryKey: queryKeys.allEnvelopes }),
			]),
	});
};

export const useEnvelopes = (month: string) => {
	return useQuery({
		queryKey: queryKeys.envelopes(month),
		queryFn: () => api.listEnvelopes(month),
	});
};

/** Every month's envelopes, since balances roll forward and assignments in
 * any month come out of Ready to assign. */
const invalidateEnvelopes = (qc: QueryClient) => {
	return qc.invalidateQueries({ queryKey: queryKeys.allEnvelopes });
};

export const useSetEnvelopesStart = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.setEnvelopesStart,
		onSuccess: () => invalidateEnvelopes(qc),
	});
};

export const useSetAssignment = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: ({
			categoryId,
			month,
			amountCents,
		}: {
			categoryId: number;
			month: string;
			amountCents: number;
		}) => api.setAssignment(categoryId, month, amountCents),
		onSuccess: () => invalidateEnvelopes(qc),
	});
};

export const useFillFromCaps = () => {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.fillFromCaps,
		onSuccess: () => invalidateEnvelopes(qc),
	});
};

export const useSpendingByCategory = (month: string) => {
	return useQuery({
		queryKey: queryKeys.spendingByCategory(month),
		queryFn: () => api.spendingByCategory(month),
	});
};

export const useMonthlySpending = (from: string, to: string) => {
	return useQuery({
		queryKey: queryKeys.monthlySpending(from, to),
		queryFn: () => api.monthlySpending(from, to),
	});
};
