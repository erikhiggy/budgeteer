import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { Account, Category, Payee, Transaction } from '../../api/types';
import { todayISO } from '../../lib/date';
import { EntryRow } from './EntryRow';

const categories: Category[] = [
	{ id: 1, name: 'Income', is_income: true, rollover_since: null },
	{ id: 3, name: 'Groceries', is_income: false, rollover_since: null },
	{ id: 4, name: 'Rent', is_income: false, rollover_since: null },
];

const payees: Payee[] = [
	{ payee: 'Trader Joe’s', category_id: 3 },
	{ payee: 'Target', category_id: null },
	{ payee: 'Landlord', category_id: 4 },
];

function setup(onSave = vi.fn().mockResolvedValue(undefined)) {
	const user = userEvent.setup();
	render(
		<EntryRow
			accountId={7}
			categories={categories}
			payees={payees}
			onSave={onSave}
		/>,
	);
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
		await user.keyboard('Whole Foods{Enter}');

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

	it('rule 3: Enter on a highlighted payee suggestion picks it instead of saving', async () => {
		const { user, onSave, field } = setup();
		// A complete row, so a plain Enter would save.
		await user.type(field('Amount'), '20');
		await user.click(field('Payee'));
		await user.keyboard('t');

		const list = screen.getByRole('listbox', { name: 'Payee suggestions' });
		expect(list).toBeVisible();
		expect(field('Payee')).toHaveAttribute('aria-expanded', 'true');

		await user.keyboard('{ArrowDown}{ArrowDown}');
		expect(screen.getByRole('option', { name: /Target/ })).toHaveAttribute(
			'aria-selected',
			'true',
		);
		await user.keyboard('{Enter}');

		expect(onSave).not.toHaveBeenCalled();
		expect(field('Payee')).toHaveValue('Target');
		expect(field('Payee')).toHaveAttribute('aria-expanded', 'false');

		// With the list closed, Enter saves as usual (category still missing).
		await user.keyboard('{Enter}');
		expect(field('Category')).toHaveFocus();
	});

	it('Enter while suggestions are open but none is highlighted saves the row', async () => {
		const { user, onSave, field } = setup();
		await user.selectOptions(field('Category'), 'Groceries');
		await user.type(field('Amount'), '5');
		await user.click(field('Payee'));
		await user.keyboard('Tra');
		expect(field('Payee')).toHaveAttribute('aria-expanded', 'true');

		await user.keyboard('{Enter}');
		expect(onSave).toHaveBeenCalledWith(
			expect.objectContaining({ payee: 'Tra', amount_cents: -500 }),
		);
	});

	it('Esc order: the first Esc closes the suggestions, the next clears the row', async () => {
		const { user, field } = setup();
		await user.type(field('Amount'), '12');
		await user.click(field('Payee'));
		await user.keyboard('La');
		expect(field('Payee')).toHaveAttribute('aria-expanded', 'true');

		await user.keyboard('{Escape}');
		expect(field('Payee')).toHaveAttribute('aria-expanded', 'false');
		expect(field('Payee')).toHaveValue('La');
		expect(field('Amount')).toHaveValue('12');

		await user.keyboard('{Escape}');
		expect(field('Payee')).toHaveValue('');
		expect(field('Amount')).toHaveValue('');
	});

	it('picking a payee fills in its latest category', async () => {
		const { user, field } = setup();
		await user.click(field('Payee'));
		await user.keyboard('trader{ArrowDown}{Enter}');
		expect(field('Payee')).toHaveValue('Trader Joe’s');
		expect(field('Category')).toHaveValue('3');
	});

	it('Enter right after a known payee counts its category', async () => {
		const { user, onSave, field } = setup();
		await user.click(field('Payee'));
		await user.keyboard('trader joe’s{Enter}');
		// The category came from the payee, so the amount is what's missing.
		expect(onSave).not.toHaveBeenCalled();
		expect(field('Category')).toHaveValue('3');
		expect(field('Amount')).toHaveFocus();
		expect(screen.getByRole('alert')).toHaveTextContent('Amount is required');

		await user.keyboard('3{Enter}');
		expect(onSave).toHaveBeenCalledWith(
			expect.objectContaining({ category_id: 3, payee: 'trader joe’s' }),
		);
	});

	it('typing a known payee in full fills in its category on Tab', async () => {
		const { user, field } = setup();
		await user.click(field('Payee'));
		await user.keyboard('landlord{Tab}');
		expect(field('Category')).toHaveFocus();
		expect(field('Category')).toHaveValue('4');
	});

	it('autofill leaves a chosen category alone, and skips uncategorized payees', async () => {
		const { user, field } = setup();
		await user.selectOptions(field('Category'), 'Rent');
		await user.click(field('Payee'));
		await user.keyboard('trader{ArrowDown}{Enter}');
		expect(field('Category')).toHaveValue('4');

		await user.keyboard('{Escape}');
		await user.click(field('Payee'));
		await user.keyboard('target{Tab}');
		expect(field('Category')).toHaveValue('');
	});

	it('suggestions can be picked with the mouse', async () => {
		const { user, field } = setup();
		await user.click(field('Payee'));
		await user.keyboard('land');
		await user.click(screen.getByRole('option', { name: /Landlord/ }));
		expect(field('Payee')).toHaveValue('Landlord');
		expect(field('Category')).toHaveValue('4');
	});

	it('enters ten rows without the mouse', async () => {
		const onSave = vi.fn().mockResolvedValue(undefined);
		const { user, field } = setup(onSave);
		await user.click(field('Payee'));

		for (let i = 1; i <= 10; i++) {
			// Payee (from a suggestion, which fills the category), Tab past
			// the category, then the amount and Enter.
			await user.keyboard('trad{ArrowDown}{Enter}{Tab}{Tab}');
			expect(field('Amount')).toHaveFocus();
			await user.keyboard(`${i}.25{Enter}`);
			await waitFor(() => expect(onSave).toHaveBeenCalledTimes(i));
			await waitFor(() => expect(field('Payee')).toHaveFocus());
		}

		expect(onSave).toHaveBeenLastCalledWith({
			account_id: 7,
			category_id: 3,
			date: todayISO(),
			payee: 'Trader Joe’s',
			amount_cents: -1025,
			note: '',
		});
	});
});

describe('EntryRow in edit mode', () => {
	const tx: Transaction = {
		id: 42,
		account_id: 7,
		category_id: 4,
		date: '2026-10-01',
		payee: 'Landlord',
		amount_cents: 150000,
		note: 'October',
		transfer_id: null,
		transfer_account_id: null,
	};

	function setupEdit(onSave = vi.fn().mockResolvedValue(undefined)) {
		const user = userEvent.setup();
		const onCancel = vi.fn();
		render(
			<EntryRow
				accountId={7}
				categories={categories}
				payees={payees}
				onSave={onSave}
				initial={tx}
				onCancel={onCancel}
			/>,
		);
		return { user, onSave, onCancel, field: screen.getByLabelText };
	}

	it('starts from the transaction and saves every field with Enter', async () => {
		const { user, onSave, field } = setupEdit();
		expect(field('Date')).toHaveFocus();
		expect(field('Payee')).toHaveValue('Landlord');
		expect(field('Category')).toHaveValue('4');
		expect(field('Amount')).toHaveValue('+1,500.00');
		expect(field('Note')).toHaveValue('October');

		await user.clear(field('Amount'));
		await user.type(field('Amount'), '1450{Enter}');
		expect(onSave).toHaveBeenCalledWith({
			account_id: 7,
			category_id: 4,
			date: '2026-10-01',
			payee: 'Landlord',
			amount_cents: -145000,
			note: 'October',
		});
		// The parent closes the row; the editor doesn't clear itself.
		expect(field('Payee')).toHaveValue('Landlord');
	});

	it('uses the same validation as a new row', async () => {
		const { user, onSave, field } = setupEdit();
		await user.clear(field('Amount'));
		await user.keyboard('{Enter}');
		expect(onSave).not.toHaveBeenCalled();
		expect(field('Amount')).toHaveFocus();
		expect(screen.getByRole('alert')).toHaveTextContent('Amount is required');
	});

	it('Esc cancels instead of clearing', async () => {
		const { user, onCancel, field } = setupEdit();
		await user.type(field('Note'), '!{Escape}');
		expect(onCancel).toHaveBeenCalledTimes(1);
		expect(field('Note')).toHaveValue('October!');
	});
});

describe('EntryRow transfers', () => {
	const others: Account[] = [
		{ id: 9, name: 'Savings', type: 'savings', balance_cents: 0 },
		{ id: 10, name: 'Visa', type: 'credit', balance_cents: 0 },
	];

	function setupTransfer() {
		const user = userEvent.setup();
		const onSave = vi.fn().mockResolvedValue(undefined);
		const onSaveTransfer = vi.fn().mockResolvedValue(undefined);
		render(
			<EntryRow
				accountId={7}
				categories={categories}
				payees={payees}
				accounts={others}
				onSave={onSave}
				onSaveTransfer={onSaveTransfer}
			/>,
		);
		return { user, onSave, onSaveTransfer, field: screen.getByLabelText };
	}

	it('offers the other accounts under Transfer and saves a transfer', async () => {
		const { user, onSave, onSaveTransfer, field } = setupTransfer();
		const group = screen.getByRole('group', { name: 'Transfer' });
		expect(
			within(group)
				.getAllByRole('option')
				.map((o) => o.textContent),
		).toEqual(['⇄ Savings', '⇄ Visa']);

		await user.type(field('Payee'), 'Rainy day');
		await user.selectOptions(field('Category'), '⇄ Savings');
		await user.type(field('Amount'), '200{Enter}');

		expect(onSave).not.toHaveBeenCalled();
		// A plain number is money out of this account, into Savings.
		expect(onSaveTransfer).toHaveBeenCalledWith({
			account_id: 7,
			other_account_id: 9,
			date: todayISO(),
			payee: 'Rainy day',
			amount_cents: -20000,
			note: '',
		});
		await waitFor(() => expect(field('Category')).toHaveValue(''));
	});

	it('a + amount is money in from the other account', async () => {
		const { user, onSaveTransfer, field } = setupTransfer();
		await user.selectOptions(field('Category'), '⇄ Visa');
		await user.type(field('Amount'), '+50{Enter}');
		expect(onSaveTransfer).toHaveBeenCalledWith(
			expect.objectContaining({ other_account_id: 10, amount_cents: 5000 }),
		);
	});

	it('mentions transfers when the category is missing', async () => {
		const { user, field } = setupTransfer();
		await user.type(field('Amount'), '5{Enter}');
		expect(screen.getByRole('alert')).toHaveTextContent(
			'Pick a category or a transfer',
		);
	});

	it('editing a leg keeps the other account fixed and saves both legs through onSave', async () => {
		const user = userEvent.setup();
		const onSave = vi.fn().mockResolvedValue(undefined);
		const leg: Transaction = {
			id: 50,
			account_id: 7,
			category_id: null,
			date: '2026-10-01',
			payee: 'Rainy day',
			amount_cents: -20000,
			note: '',
			transfer_id: 3,
			transfer_account_id: 9,
		};
		render(
			<EntryRow
				accountId={7}
				categories={categories}
				payees={payees}
				accounts={others}
				onSave={onSave}
				initial={leg}
				onCancel={() => {}}
			/>,
		);
		const select = screen.getByLabelText('Category');
		expect(select).toBeDisabled();
		expect(select).toHaveValue('transfer:9');
		expect(screen.getAllByRole('option', { name: /⇄/ })).toHaveLength(1);

		await user.clear(screen.getByLabelText('Amount'));
		await user.type(screen.getByLabelText('Amount'), '250{Enter}');
		expect(onSave).toHaveBeenCalledWith({
			account_id: 7,
			category_id: null,
			date: '2026-10-01',
			payee: 'Rainy day',
			amount_cents: -25000,
			note: '',
		});
	});

	it('editing an ordinary transaction offers no transfers', () => {
		render(
			<EntryRow
				accountId={7}
				categories={categories}
				payees={payees}
				accounts={others}
				onSave={vi.fn()}
				onSaveTransfer={vi.fn()}
				initial={{
					id: 51,
					account_id: 7,
					category_id: 3,
					date: '2026-10-01',
					payee: '',
					amount_cents: -100,
					note: '',
					transfer_id: null,
					transfer_account_id: null,
				}}
			/>,
		);
		expect(screen.queryByRole('group', { name: 'Transfer' })).toBeNull();
	});
});
