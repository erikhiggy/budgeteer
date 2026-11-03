import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { Account } from '../../api/types';
import { AccountHeader } from './AccountHeader';

const account: Account = {
	id: 1,
	name: 'Checking',
	type: 'checking',
	balance_cents: 96451,
	cleared_balance_cents: 100000,
	uncleared_balance_cents: -3549,
};

const setup = (overrides: Partial<Account> = {}) => {
	render(
		<QueryClientProvider client={new QueryClient()}>
			<AccountHeader
				account={{ ...account, ...overrides }}
				onDeleted={vi.fn()}
			/>
		</QueryClientProvider>,
	);
};

describe('AccountHeader', () => {
	it('shows the cleared, uncleared and working balances', () => {
		setup();
		expect(screen.getByText('Cleared 1,000.00')).toBeInTheDocument();
		expect(screen.getByText('Uncleared −35.49')).toBeInTheDocument();
		expect(screen.getByText('Working 964.51')).toBeInTheDocument();
	});

	it('shows zeros for an account with no transactions', () => {
		setup({
			balance_cents: 0,
			cleared_balance_cents: 0,
			uncleared_balance_cents: 0,
		});
		expect(screen.getByText('Cleared 0.00')).toBeInTheDocument();
		expect(screen.getByText('Uncleared 0.00')).toBeInTheDocument();
		expect(screen.getByText('Working 0.00')).toBeInTheDocument();
	});
});
