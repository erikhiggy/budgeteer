import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { Category } from '../../api/types';
import { todayISO } from '../../lib/date';
import { EntryRow } from './EntryRow';

const categories: Category[] = [
	{ id: 1, name: 'Income', is_income: true },
	{ id: 3, name: 'Groceries', is_income: false },
	{ id: 4, name: 'Rent', is_income: false },
];

function setup(onSave = vi.fn().mockResolvedValue(undefined)) {
	const user = userEvent.setup();
	render(<EntryRow accountId={7} categories={categories} onSave={onSave} />);
	const field = (name: string) => screen.getByLabelText(name);
	return { user, onSave, field };
}

function deferred() {
	let resolve!: () => void;
	let reject!: (err: Error) => void;
	const promise = new Promise<void>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

describe('EntryRow', () => {
	it('rule 1: Enter on an incomplete row sends nothing and focuses the first missing field', async () => {
		const { user, onSave, field } = setup();

		await user.click(field('Payee'));
		await user.keyboard('Trader Joe’s{Enter}');

		expect(onSave).not.toHaveBeenCalled();
		// Date is prefilled, so category is the first thing missing.
		expect(field('Category')).toHaveFocus();
		expect(field('Category')).toHaveAttribute('aria-invalid', 'true');
		expect(field('Amount')).toHaveAttribute('aria-invalid', 'true');
		expect(screen.getByRole('alert')).toHaveTextContent('Pick a category');

		// Fixing a field clears its highlight and moves the message on.
		await user.selectOptions(field('Category'), 'Groceries');
		expect(field('Category')).not.toHaveAttribute('aria-invalid');
		await user.keyboard('{Enter}');
		expect(field('Amount')).toHaveFocus();
		expect(screen.getByRole('alert')).toHaveTextContent('Amount is required');
		expect(onSave).not.toHaveBeenCalled();
	});

	it('rule 2: Enter on a complete row saves from any field, then resets for the next entry', async () => {
		const { user, onSave, field } = setup();

		await user.type(field('Payee'), 'Trader Joe’s');
		await user.selectOptions(field('Category'), 'Groceries');
		await user.type(field('Amount'), '45.67');
		await user.type(field('Note'), ' weekly ');
		// Back in the payee field, not the last one.
		await user.click(field('Payee'));
		await user.keyboard('{Enter}');

		expect(onSave).toHaveBeenCalledTimes(1);
		expect(onSave).toHaveBeenCalledWith({
			account_id: 7,
			category_id: 3,
			date: todayISO(),
			payee: 'Trader Joe’s',
			amount_cents: -4567,
			note: 'weekly',
		});

		await waitFor(() => expect(field('Payee')).toHaveFocus());
		expect(field('Payee')).toHaveValue('');
		expect(field('Category')).toHaveValue('');
		expect(field('Amount')).toHaveValue('');
		expect(field('Date')).toHaveValue(todayISO());
	});

	it('saves a + amount as money in', async () => {
		const { user, onSave, field } = setup();
		await user.selectOptions(field('Category'), 'Income');
		await user.type(field('Amount'), '+2,500{Enter}');
		expect(onSave).toHaveBeenCalledWith(
			expect.objectContaining({ category_id: 1, amount_cents: 250000 }),
		);
	});

	it('Enter on the category select saves too', async () => {
		const { user, onSave, field } = setup();
		await user.type(field('Amount'), '12');
		await user.selectOptions(field('Category'), 'Rent');
		await user.keyboard('{Enter}');
		expect(onSave).toHaveBeenCalledTimes(1);
	});

	it('ignores repeated Enter while a save is in flight', async () => {
		const pending = deferred();
		const { user, onSave, field } = setup(vi.fn(() => pending.promise));

		await user.selectOptions(field('Category'), 'Rent');
		await user.type(field('Amount'), '1200');
		await user.keyboard('{Enter}{Enter}{Enter}');
		expect(onSave).toHaveBeenCalledTimes(1);

		pending.resolve();
		await waitFor(() => expect(field('Amount')).toHaveValue(''));
	});

	it('keeps the values and shows the server error when saving fails', async () => {
		const { user, field } = setup(
			vi.fn().mockRejectedValue(new Error('unknown category_id 3')),
		);

		await user.selectOptions(field('Category'), 'Groceries');
		await user.type(field('Amount'), '9.99{Enter}');

		expect(await screen.findByRole('alert')).toHaveTextContent(
			'unknown category_id 3',
		);
		expect(field('Amount')).toHaveValue('9.99');
		expect(field('Category')).toHaveValue('3');
	});

	it('Esc clears the row but keeps the date', async () => {
		const { user, field } = setup();
		await user.clear(field('Date'));
		await user.type(field('Date'), '2026-09-01');
		await user.type(field('Payee'), 'Landlord');
		await user.type(field('Amount'), '1500{Escape}');

		expect(field('Payee')).toHaveValue('');
		expect(field('Amount')).toHaveValue('');
		expect(field('Date')).toHaveValue('2026-09-01');
	});

	it('Clicking Add behaves like Enter', async () => {
		const { user, onSave, field } = setup();
		await user.click(screen.getByRole('button', { name: 'Add' }));
		expect(onSave).not.toHaveBeenCalled();
		expect(field('Category')).toHaveFocus();
	});
});
