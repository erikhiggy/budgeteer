import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { Account, Category, Transaction } from '../../api/types';
import { TransactionTable } from './TransactionTable';

const categories: Category[] = [{ id: 3, name: 'Groceries', is_income: false }];
const accounts: Account[] = [
	{ id: 7, name: 'Checking', type: 'checking', balance_cents: 0 },
	{ id: 9, name: 'Savings', type: 'savings', balance_cents: 0 },
];
const transactions: Transaction[] = [
	{
		id: 1,
		account_id: 7,
		category_id: 3,
		date: '2026-10-05',
		payee: 'Trader Joe’s',
		amount_cents: -4567,
		note: '',
		transfer_id: null,
		transfer_account_id: null,
	},
	{
		id: 2,
		account_id: 7,
		category_id: null,
		date: '2026-10-01',
		payee: 'Paycheck',
		amount_cents: 250000,
		note: '',
		transfer_id: null,
		transfer_account_id: null,
	},
];

function setup() {
	const user = userEvent.setup();
	const onUpdate = vi.fn().mockResolvedValue(undefined);
	const onDelete = vi.fn();
	render(
		<TransactionTable
			transactions={transactions}
			categories={categories}
			accounts={accounts}
			payees={[]}
			onUpdate={onUpdate}
			onDelete={onDelete}
		/>,
	);
	return { user, onUpdate, onDelete };
}

describe('TransactionTable', () => {
	it('edits a row inline and closes the editor after saving', async () => {
		const { user, onUpdate } = setup();
		await user.click(
			screen.getByRole('button', {
				name: 'Edit Trader Joe’s, Oct 5, 2026, 45.67',
			}),
		);
		const editor = screen.getByRole('form', { name: 'Edit transaction' });
		await user.type(within(editor).getByLabelText('Note'), 'weekly{Enter}');

		expect(onUpdate).toHaveBeenCalledWith(
			transactions[0],
			expect.objectContaining({ note: 'weekly', amount_cents: -4567 }),
		);
		expect(
			await screen.findByRole('button', { name: /^Edit Trader Joe’s/ }),
		).toBeInTheDocument();
		expect(screen.queryByRole('form', { name: 'Edit transaction' })).toBeNull();
	});

	it('keeps the editor open with the error when saving fails', async () => {
		const { user, onUpdate } = setup();
		onUpdate.mockRejectedValueOnce(new Error('unknown category_id 3'));
		await user.click(
			screen.getByRole('button', { name: /^Edit Trader Joe’s/ }),
		);
		await user.keyboard('{Enter}');
		expect(await screen.findByRole('alert')).toHaveTextContent(
			'unknown category_id 3',
		);
		expect(
			screen.getByRole('form', { name: 'Edit transaction' }),
		).toBeInTheDocument();
	});

	it('Esc cancels the edit without saving', async () => {
		const { user, onUpdate } = setup();
		await user.click(
			screen.getByRole('button', { name: /^Edit Trader Joe’s/ }),
		);
		await user.keyboard('{Escape}');
		expect(onUpdate).not.toHaveBeenCalled();
		expect(screen.queryByRole('form', { name: 'Edit transaction' })).toBeNull();
	});

	it('deletes only on the second click', async () => {
		const { user, onDelete } = setup();
		const del = screen.getByRole('button', {
			name: 'Delete Paycheck, Oct 1, 2026, +2,500.00',
		});
		await user.click(del);
		expect(onDelete).not.toHaveBeenCalled();
		expect(del).toHaveTextContent('Confirm');

		await user.click(del);
		expect(onDelete).toHaveBeenCalledWith(transactions[1]);
	});

	it('disarms delete when focus moves away', async () => {
		const { user, onDelete } = setup();
		const del = screen.getByRole('button', { name: /^Delete Paycheck/ });
		await user.click(del);
		await user.tab();
		expect(del).toHaveTextContent('Delete');
		await user.click(del);
		expect(onDelete).not.toHaveBeenCalled();
	});

	it('shows transfer legs with their direction and other account', () => {
		const leg = (id: number, cents: number): Transaction => ({
			id,
			account_id: 7,
			category_id: null,
			date: '2026-10-02',
			payee: 'Rainy day',
			amount_cents: cents,
			note: '',
			transfer_id: id,
			transfer_account_id: 9,
		});
		render(
			<TransactionTable
				transactions={[leg(10, -20000), leg(11, 5000)]}
				categories={categories}
				accounts={accounts}
				payees={[]}
				onUpdate={vi.fn()}
				onDelete={vi.fn()}
			/>,
		);
		expect(screen.getByText('Transfer → Savings')).toBeInTheDocument();
		expect(screen.getByText('Transfer ← Savings')).toBeInTheDocument();
	});
});
