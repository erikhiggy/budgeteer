import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import type { Account } from '../../api/types';
import { AccountSidebar } from './AccountSidebar';

const account = (overrides: Partial<Account>): Account => {
	return {
		id: 1,
		name: 'Checking',
		type: 'checking',
		balance_cents: 0,
		...overrides,
	};
};

const setup = (accounts: Account[]) => {
	render(
		<MemoryRouter>
			<AccountSidebar accounts={accounts} onNewAccount={vi.fn()} />
		</MemoryRouter>,
	);
};

const group = (label: string) => screen.getByRole('region', { name: label });

describe('AccountSidebar', () => {
	it('groups accounts by type in a fixed order', () => {
		setup([
			account({ id: 1, name: 'Amex', type: 'credit' }),
			account({ id: 2, name: 'BofA', type: 'checking' }),
			account({ id: 3, name: 'Roth IRA', type: 'investment' }),
			account({ id: 4, name: 'Marcus', type: 'savings' }),
			account({ id: 5, name: 'Wallet', type: 'cash' }),
		]);

		const headings = screen.getAllByRole('heading').map((h) => h.textContent);
		expect(headings).toEqual([
			'Checking',
			'Savings',
			'Cash',
			'Credit cards',
			'Investments',
		]);
		expect(within(group('Credit cards')).getByText('Amex')).toBeTruthy();
		expect(within(group('Investments')).getByText('Roth IRA')).toBeTruthy();
	});

	it('keeps API order within a group', () => {
		setup([
			account({ id: 1, name: 'Amex', type: 'credit' }),
			account({ id: 2, name: 'Citi', type: 'credit' }),
		]);

		const names = within(group('Credit cards'))
			.getAllByRole('link')
			.map((l) => l.textContent);
		expect(names[0]).toContain('Amex');
		expect(names[1]).toContain('Citi');
	});

	it('shows a subtotal per group and a total of every account', () => {
		setup([
			account({ id: 1, name: 'Amex', type: 'credit', balance_cents: -5000 }),
			account({ id: 2, name: 'Citi', type: 'credit', balance_cents: -2500 }),
			account({ id: 3, name: 'BofA', type: 'checking', balance_cents: 100000 }),
		]);

		const cards = screen.getByRole('heading', { name: 'Credit cards' })
			.parentElement as HTMLElement;
		expect(within(cards).getByText('−75.00')).toBeTruthy();
		const checking = screen.getByRole('heading', { name: 'Checking' })
			.parentElement as HTMLElement;
		expect(within(checking).getByText('1,000.00')).toBeTruthy();
		const total = screen.getByText('Total').parentElement as HTMLElement;
		expect(within(total).getByText('925.00')).toBeTruthy();
	});

	it('leaves out types with no accounts', () => {
		setup([account({ id: 1, name: 'BofA', type: 'checking' })]);

		expect(screen.getAllByRole('heading')).toHaveLength(1);
		expect(screen.queryByRole('heading', { name: 'Cash' })).toBeNull();
	});
});
