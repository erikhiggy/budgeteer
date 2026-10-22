import {
	QueryClient,
	useMutation,
	useQuery,
	useQueryClient,
} from '@tanstack/react-query';
import { api, ApiError } from './client';

export const queryKeys = {
	accounts: ['accounts'] as const,
	categories: ['categories'] as const,
	transactions: (accountId: number) => ['transactions', accountId] as const,
};

export function createQueryClient() {
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
}

export function useAccounts() {
	return useQuery({ queryKey: queryKeys.accounts, queryFn: api.listAccounts });
}

export function useCreateAccount() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.createAccount,
		onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.accounts }),
	});
}

export function useCategories() {
	return useQuery({
		queryKey: queryKeys.categories,
		queryFn: api.listCategories,
	});
}

export function useCreateCategory() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.createCategory,
		onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.categories }),
	});
}

export function useTransactions(accountId: number) {
	return useQuery({
		queryKey: queryKeys.transactions(accountId),
		queryFn: () => api.listTransactions(accountId),
	});
}

export function useCreateTransaction() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: api.createTransaction,
		// Only queries currently on screen refetch (TanStack's default
		// refetchType: 'active'): this account's list and the balances.
		onSuccess: (tx) =>
			Promise.all([
				qc.invalidateQueries({
					queryKey: queryKeys.transactions(tx.account_id),
				}),
				qc.invalidateQueries({ queryKey: queryKeys.accounts }),
			]),
	});
}
