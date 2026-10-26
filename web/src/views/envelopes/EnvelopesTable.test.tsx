import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { CardEnvelope, Envelope } from '../../api/types';
import { EnvelopesTable } from './EnvelopesTable';

const envelope = (overrides: Partial<Envelope>): Envelope => {
	return {
		category_id: 1,
		name: 'Groceries',
		cap_cents: null,
		carryover_cents: 0,
		assigned_cents: 0,
		spent_cents: 0,
		balance_cents: 0,
		...overrides,
	};
};

const card = (overrides: Partial<CardEnvelope>): CardEnvelope => {
	return {
		account_id: 7,
		name: 'Visa',
		carryover_cents: 0,
		assigned_cents: 0,
		purchases_cents: 0,
		paid_cents: 0,
		balance_cents: 0,
		owed_cents: 0,
		...overrides,
	};
};

const setup = (
	envelopes: Envelope[],
	uncategorizedCents = 0,
	cards: CardEnvelope[] = [],
) => {
	const user = userEvent.setup();
	const onAssign = vi.fn().mockResolvedValue(undefined);
	const onAssignCard = vi.fn().mockResolvedValue(undefined);
	render(
		<EnvelopesTable
			envelopes={envelopes}
			cards={cards}
			uncategorizedCents={uncategorizedCents}
			onAssign={onAssign}
			onAssignCard={onAssignCard}
		/>,
	);
	return { user, onAssign, onAssignCard };
};

const row = (name: string) =>
	screen.getByText(name, { selector: 'div' }).closest('tr') as HTMLElement;

describe('EnvelopesTable', () => {
	it('shows assigned, spent, balance, carryover and the cap', () => {
		setup([
			envelope({
				cap_cents: 50000,
				carryover_cents: 2500,
				assigned_cents: 40000,
				spent_cents: 10000,
				balance_cents: 32500,
			}),
		]);
		const r = row('Groceries');
		expect(r).toHaveTextContent('400.00');
		expect(r).toHaveTextContent('100.00');
		expect(r).toHaveTextContent('325.00');
		expect(r).toHaveTextContent('+25.00 from last month · cap 500.00');
	});

	it('notes overspending carried in', () => {
		setup([envelope({ carryover_cents: -1500, balance_cents: -1500 })]);
		expect(row('Groceries')).toHaveTextContent('−15.00 overspent last month');
	});

	it('edits an assignment inline with Enter', async () => {
		const { user, onAssign } = setup([envelope({ assigned_cents: 40000 })]);
		await user.click(
			screen.getByRole('button', { name: 'Assigned to Groceries' }),
		);
		const input = screen.getByRole('textbox', {
			name: 'Assigned to Groceries',
		});
		expect(input).toHaveFocus();
		expect(input).toHaveValue('400.00');

		await user.clear(input);
		await user.type(input, '-20{Enter}');
		expect(onAssign).toHaveBeenCalledWith(1, -2000);
		await waitFor(() =>
			expect(
				screen.queryByRole('textbox', { name: 'Assigned to Groceries' }),
			).toBeNull(),
		);
	});

	it('an empty assignment removes it', async () => {
		const { user, onAssign } = setup([envelope({ assigned_cents: 40000 })]);
		await user.click(
			screen.getByRole('button', { name: 'Assigned to Groceries' }),
		);
		await user.clear(
			screen.getByRole('textbox', { name: 'Assigned to Groceries' }),
		);
		await user.keyboard('{Enter}');
		expect(onAssign).toHaveBeenCalledWith(1, 0);
	});

	it('rejects a bad amount, and shows a server error', async () => {
		const { user, onAssign } = setup([envelope({})]);
		await user.click(
			screen.getByRole('button', { name: 'Assigned to Groceries' }),
		);
		await user.keyboard('abc{Enter}');
		expect(onAssign).not.toHaveBeenCalled();
		expect(screen.getByRole('alert')).toHaveTextContent(
			'Assigned must be a number like 12.34',
		);

		onAssign.mockRejectedValueOnce(
			new Error("envelope budgeting hasn't been started"),
		);
		const input = screen.getByRole('textbox', {
			name: 'Assigned to Groceries',
		});
		await user.clear(input);
		await user.type(input, '20{Enter}');
		expect(await screen.findByRole('alert')).toHaveTextContent(
			"envelope budgeting hasn't been started",
		);
		expect(input).toBeInTheDocument();
	});

	it('shows uncategorized spending only when there is some', () => {
		setup([envelope({})], 1234);
		expect(row('Uncategorized')).toHaveTextContent('12.34');
	});

	it('shows no credit card section without cards', () => {
		setup([envelope({})]);
		expect(screen.queryByText('Credit cards')).toBeNull();
	});

	it('shows a card’s payments, purchases, and how much of what’s owed is covered', () => {
		setup([], 0, [
			card({
				assigned_cents: 50000,
				purchases_cents: 10000,
				paid_cents: 60000,
				balance_cents: 0,
				owed_cents: 150000,
			}),
			card({
				account_id: 8,
				name: 'Amex',
				purchases_cents: 4000,
				balance_cents: 4000,
				owed_cents: 4000,
			}),
		]);
		expect(screen.getByText('Credit cards')).toBeInTheDocument();
		const visa = row('Visa');
		expect(visa).toHaveTextContent('500.00');
		expect(visa).toHaveTextContent('600.00');
		expect(visa).toHaveTextContent(
			'100.00 in purchases · owes 1,500.00 (1,500.00 not covered)',
		);
		expect(row('Amex')).toHaveTextContent('40.00 in purchases · owes 40.00');
		expect(row('Amex')).not.toHaveTextContent('not covered');
	});

	it('assigns to a card', async () => {
		const { user, onAssignCard } = setup([], 0, [card({ owed_cents: 2000 })]);
		await user.click(screen.getByRole('button', { name: 'Assigned to Visa' }));
		await user.keyboard('20{Enter}');
		expect(onAssignCard).toHaveBeenCalledWith(7, 2000);
	});
});
