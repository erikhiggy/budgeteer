import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import type { Goal } from '../../api/types';
import { GoalsTable } from './GoalsTable';

const goal = (overrides: Partial<Goal>): Goal => {
	return {
		category_id: 1,
		name: 'Groceries',
		cap_cents: null,
		cap_since: null,
		rollover_since: null,
		carryover_cents: 0,
		available_cents: null,
		spent_cents: 0,
		...overrides,
	};
};

const setup = (goals: Goal[], uncategorizedCents = 0) => {
	const user = userEvent.setup();
	const onSetCap = vi.fn().mockResolvedValue(undefined);
	const onSetRollover = vi.fn().mockResolvedValue(undefined);
	render(
		<GoalsTable
			month='2026-11'
			goals={goals}
			uncategorizedCents={uncategorizedCents}
			onSetCap={onSetCap}
			onSetRollover={onSetRollover}
		/>,
	);
	return { user, onSetCap, onSetRollover };
};

const row = (name: string) =>
	screen.getByText(name, { selector: 'div' }).closest('tr') as HTMLElement;

describe('GoalsTable', () => {
	it('shows cap, spent, remaining and progress, with "since" for an earlier cap', () => {
		setup([
			goal({
				cap_cents: 40000,
				cap_since: '2026-08',
				available_cents: 40000,
				spent_cents: 10000,
			}),
		]);
		const r = row('Groceries');
		expect(r).toHaveTextContent('400.00');
		expect(r).toHaveTextContent('100.00');
		expect(r).toHaveTextContent('300.00');
		expect(r).toHaveTextContent('since Aug 2026');
		const bar = screen.getByRole('progressbar', { name: 'Groceries spending' });
		expect(bar).toHaveAttribute('aria-valuenow', '25');
		expect(bar).toHaveAttribute('aria-valuetext', '300.00 left');
	});

	it('leaves out "since" when the cap was set this month', () => {
		setup([
			goal({ cap_cents: 100, cap_since: '2026-11', available_cents: 100 }),
		]);
		expect(row('Groceries')).not.toHaveTextContent('since');
	});

	it('marks a category over its cap', () => {
		setup([
			goal({
				cap_cents: 10000,
				cap_since: '2026-11',
				available_cents: 10000,
				spent_cents: 12500,
			}),
		]);
		expect(row('Groceries')).toHaveTextContent('−25.00');
		const bar = screen.getByRole('progressbar');
		expect(bar).toHaveAttribute('aria-valuenow', '125');
		expect(bar).toHaveAttribute('aria-valuetext', 'Over by 25.00');
	});

	it('shows no remaining or progress without a cap', () => {
		setup([goal({ spent_cents: 500 })]);
		expect(
			screen.getByRole('button', { name: 'Cap for Groceries' }),
		).toHaveTextContent('Set cap');
		expect(screen.queryByRole('progressbar')).toBeNull();
	});

	it('edits a cap inline with Enter', async () => {
		const { user, onSetCap } = setup([
			goal({ cap_cents: 40000, cap_since: '2026-11', available_cents: 40000 }),
		]);
		await user.click(screen.getByRole('button', { name: 'Cap for Groceries' }));
		const input = screen.getByRole('textbox', { name: 'Cap for Groceries' });
		expect(input).toHaveFocus();
		expect(input).toHaveValue('400.00');

		await user.clear(input);
		await user.type(input, '450{Enter}');
		expect(onSetCap).toHaveBeenCalledWith(1, 45000);
		await waitFor(() =>
			expect(
				screen.queryByRole('textbox', { name: 'Cap for Groceries' }),
			).toBeNull(),
		);
	});

	it('an empty cap removes it', async () => {
		const { user, onSetCap } = setup([
			goal({ cap_cents: 40000, cap_since: '2026-11', available_cents: 40000 }),
		]);
		await user.click(screen.getByRole('button', { name: 'Cap for Groceries' }));
		await user.clear(
			screen.getByRole('textbox', { name: 'Cap for Groceries' }),
		);
		await user.keyboard('{Enter}');
		expect(onSetCap).toHaveBeenCalledWith(1, null);
	});

	it('rejects a bad cap without saving, and Esc cancels', async () => {
		const { user, onSetCap } = setup([goal({})]);
		await user.click(screen.getByRole('button', { name: 'Cap for Groceries' }));
		await user.keyboard('-5{Enter}');
		expect(onSetCap).not.toHaveBeenCalled();
		expect(screen.getByRole('alert')).toHaveTextContent(
			'A cap must be positive',
		);

		await user.keyboard('{Escape}');
		expect(
			screen.queryByRole('textbox', { name: 'Cap for Groceries' }),
		).toBeNull();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('keeps the editor open and shows a server error', async () => {
		const { user, onSetCap } = setup([goal({})]);
		onSetCap.mockRejectedValueOnce(new Error('no category with id 1'));
		await user.click(screen.getByRole('button', { name: 'Cap for Groceries' }));
		await user.keyboard('20{Enter}');
		expect(await screen.findByRole('alert')).toHaveTextContent(
			'no category with id 1',
		);
		expect(
			screen.getByRole('textbox', { name: 'Cap for Groceries' }),
		).toBeInTheDocument();
	});

	it('turns rollover on from the month on screen, and off', async () => {
		const { user, onSetRollover } = setup([
			goal({ category_id: 1, name: 'Insurance' }),
			goal({ category_id: 2, name: 'Gifts', rollover_since: '2026-10' }),
		]);
		await user.click(
			screen.getByRole('checkbox', { name: 'Roll over Insurance' }),
		);
		expect(onSetRollover).toHaveBeenCalledWith(1, '2026-11');

		const gifts = screen.getByRole('checkbox', { name: 'Roll over Gifts' });
		expect(gifts).toBeChecked();
		await user.click(gifts);
		expect(onSetRollover).toHaveBeenCalledWith(2, null);
	});

	it('shows where rollover stands', () => {
		setup([
			goal({
				category_id: 1,
				name: 'Insurance',
				rollover_since: '2026-10',
				cap_cents: 10000,
				cap_since: '2026-10',
				carryover_cents: 10000,
				available_cents: 20000,
			}),
			goal({
				category_id: 2,
				name: 'Car',
				rollover_since: '2026-10',
				cap_cents: 5000,
				cap_since: '2026-10',
				carryover_cents: -3500,
				available_cents: 1500,
			}),
			goal({
				category_id: 3,
				name: 'Gifts',
				rollover_since: '2026-11',
				cap_cents: 2000,
				cap_since: '2026-11',
				available_cents: 2000,
			}),
			goal({
				category_id: 4,
				name: 'Travel',
				rollover_since: '2026-12',
				cap_cents: 2000,
				cap_since: '2026-11',
				available_cents: 2000,
			}),
		]);
		expect(row('Insurance')).toHaveTextContent('+100.00 rolled over');
		// Measured against available, not the cap.
		expect(row('Insurance')).toHaveTextContent('200.00');
		expect(row('Car')).toHaveTextContent('−35.00 from last month');
		expect(row('Gifts')).toHaveTextContent('rolls over');
		expect(row('Travel')).toHaveTextContent('rolls over from Dec 2026');
	});

	it('shows uncategorized spending only when there is some', () => {
		const { unmount } = render(
			<GoalsTable
				month='2026-11'
				goals={[]}
				uncategorizedCents={0}
				onSetCap={vi.fn()}
				onSetRollover={vi.fn()}
			/>,
		);
		expect(screen.queryByText('Uncategorized')).toBeNull();
		unmount();

		setup([], 4500);
		expect(row('Uncategorized')).toHaveTextContent('45.00');
	});
});
