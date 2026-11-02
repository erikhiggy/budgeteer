import { useState, type KeyboardEvent } from 'react';
import type { CardEnvelope, Envelope, EnvelopeGroup } from '../../api/types';
import buttons from '../../components/Button.module.css';
import { cx } from '../../lib/cx';
import { formatCents, parseAssigned } from '../../lib/money';
import styles from './EnvelopesTable.module.css';

type Props = {
	/** In display order. With none, the rows are one flat list. */
	groups: EnvelopeGroup[];
	envelopes: Envelope[];
	cards: CardEnvelope[];
	uncategorizedCents: number;
	/** Sets the month's assignment; 0 removes it. Rejects with a shown Error. */
	onAssign: (categoryId: number, cents: number) => Promise<unknown>;
	/** onAssign for a credit card's envelope. */
	onAssignCard: (accountId: number, cents: number) => Promise<unknown>;
};

/** The key for the "Ungrouped" section in the collapsed set; real ids start at 1. */
const UNGROUPED = 0;

const COLLAPSED_KEY = 'budgeteer.collapsedEnvelopeGroups';

const loadCollapsed = (): Set<number> => {
	try {
		const parsed: unknown = JSON.parse(
			localStorage.getItem(COLLAPSED_KEY) ?? '[]',
		);
		return new Set(
			Array.isArray(parsed)
				? parsed.filter((v): v is number => typeof v === 'number')
				: [],
		);
	} catch {
		return new Set();
	}
};

/** Which group sections are folded, remembered per browser. */
const useCollapsed = () => {
	const [collapsed, setCollapsed] = useState(loadCollapsed);
	const toggle = (key: number) => {
		setCollapsed((prev) => {
			const next = new Set(prev);
			if (!next.delete(key)) next.add(key);
			try {
				localStorage.setItem(COLLAPSED_KEY, JSON.stringify([...next]));
			} catch {
				// Private mode or blocked storage: it just won't be remembered.
			}
			return next;
		});
	};
	return { collapsed, toggle };
};

type Section = {
	key: number;
	name: string;
	assignedCents: number;
	spentCents: number;
	balanceCents: number;
	envelopes: Envelope[];
};

/**
 * Sections for the groups that have envelopes, in order, then the ungrouped
 * ones. The ungrouped totals are summed here; the server only totals groups.
 */
const buildSections = (
	groups: EnvelopeGroup[],
	envelopes: Envelope[],
): Section[] => {
	const sections: Section[] = [];
	for (const g of groups) {
		const members = envelopes.filter((e) => e.group_id === g.id);
		if (members.length === 0) continue;
		sections.push({
			key: g.id,
			name: g.name,
			assignedCents: g.assigned_cents,
			spentCents: g.spent_cents,
			balanceCents: g.balance_cents,
			envelopes: members,
		});
	}
	const known = new Set(groups.map((g) => g.id));
	const loose = envelopes.filter(
		(e) => e.group_id === null || !known.has(e.group_id),
	);
	if (loose.length > 0) {
		sections.push({
			key: UNGROUPED,
			name: 'Ungrouped',
			assignedCents: sum(loose, (e) => e.assigned_cents),
			spentCents: sum(loose, (e) => e.spent_cents),
			balanceCents: sum(loose, (e) => e.balance_cents),
			envelopes: loose,
		});
	}
	return sections;
};

const sum = (envelopes: Envelope[], pick: (e: Envelope) => number): number => {
	return envelopes.reduce((total, e) => total + pick(e), 0);
};

/**
 * One row per expense category: assigned this month (click to edit), spent,
 * and the balance, which carries into the next month either way. With
 * groups, the rows sit under collapsible group headers that total them.
 * Then one row per credit card, whose envelope card purchases fill and
 * payments draw down.
 */
export const EnvelopesTable = ({
	groups,
	envelopes,
	cards,
	uncategorizedCents,
	onAssign,
	onAssignCard,
}: Props) => {
	const { collapsed, toggle } = useCollapsed();
	const grouped = groups.length > 0;

	const rows = (list: Envelope[]) =>
		list.map((e) => (
			<EnvelopeRow
				key={e.category_id}
				name={e.name}
				notes={categoryNotes(e)}
				assignedCents={e.assigned_cents}
				spentCents={e.spent_cents}
				balanceCents={e.balance_cents}
				onAssign={(cents) => onAssign(e.category_id, cents)}
			/>
		));

	return (
		<table className={styles.table}>
			<thead>
				<tr>
					<th>Category</th>
					<th className={styles.money}>Assigned</th>
					<th className={styles.money}>Spent</th>
					<th className={styles.money}>Balance</th>
				</tr>
			</thead>
			{grouped
				? buildSections(groups, envelopes).map((section) => {
						const folded = collapsed.has(section.key);
						return (
							<tbody key={section.key}>
								<tr className={styles.group}>
									<th scope='rowgroup'>
										<button
											type='button'
											className={styles.groupToggle}
											aria-expanded={!folded}
											onClick={() => toggle(section.key)}
										>
											<span className={styles.chevron} aria-hidden='true'>
												{folded ? '▸' : '▾'}
											</span>
											{section.name}
										</button>
									</th>
									<td className={styles.money}>
										{formatCents(section.assignedCents)}
									</td>
									<td className={styles.money}>
										{formatCents(section.spentCents)}
									</td>
									<td
										className={cx(
											styles.money,
											styles.balance,
											section.balanceCents < 0 && styles.negative,
										)}
									>
										{formatCents(section.balanceCents)}
									</td>
								</tr>
								{!folded && rows(section.envelopes)}
							</tbody>
						);
					})
				: null}
			<tbody>
				{!grouped && envelopes.length === 0 && (
					<tr>
						<td colSpan={4} className={styles.empty}>
							No spending categories yet. Add some under Categories.
						</td>
					</tr>
				)}
				{!grouped && rows(envelopes)}
				{uncategorizedCents !== 0 && (
					<tr className={styles.uncategorized}>
						<td>
							<div className={styles.name}>Uncategorized</div>
							<div className={styles.detail}>
								Give these transactions a category to take them out of an
								envelope
							</div>
						</td>
						<td className={styles.money}>—</td>
						<td className={styles.money}>{formatCents(uncategorizedCents)}</td>
						<td className={styles.money}>—</td>
					</tr>
				)}
			</tbody>
			{cards.length > 0 && (
				<tbody>
					<tr className={styles.section}>
						<th scope='rowgroup'>Credit cards</th>
						<th className={styles.money} />
						<th className={styles.money}>Paid</th>
						<th className={styles.money} />
					</tr>
					{cards.map((c) => (
						<EnvelopeRow
							key={c.account_id}
							name={c.name}
							notes={cardNotes(c)}
							assignedCents={c.assigned_cents}
							spentCents={c.paid_cents}
							balanceCents={c.balance_cents}
							onAssign={(cents) => onAssignCard(c.account_id, cents)}
						/>
					))}
				</tbody>
			)}
		</table>
	);
};

/** The small print under a category: carryover and the cap as a target. */
const categoryNotes = (e: Envelope): string[] => {
	const notes = carryoverNotes(e.carryover_cents);
	if (e.cap_cents !== null) notes.push(`cap ${formatCents(e.cap_cents)}`);
	return notes;
};

/** The small print under a card: carryover, purchases, and what's owed. */
const cardNotes = (c: CardEnvelope): string[] => {
	const notes = carryoverNotes(c.carryover_cents);
	if (c.purchases_cents !== 0) {
		notes.push(`${formatCents(c.purchases_cents)} in purchases`);
	}
	if (c.owed_cents > 0) {
		const uncovered = c.owed_cents - Math.max(c.balance_cents, 0);
		notes.push(
			uncovered > 0
				? `owes ${formatCents(c.owed_cents)} (${formatCents(uncovered)} not covered)`
				: `owes ${formatCents(c.owed_cents)}`,
		);
	}
	return notes;
};

const carryoverNotes = (carryoverCents: number): string[] => {
	if (carryoverCents > 0) {
		return [`+${formatCents(carryoverCents)} from last month`];
	}
	if (carryoverCents < 0) {
		return [`${formatCents(carryoverCents)} overspent last month`];
	}
	return [];
};

const EnvelopeRow = ({
	name,
	notes,
	assignedCents,
	spentCents,
	balanceCents,
	onAssign,
}: {
	name: string;
	notes: string[];
	assignedCents: number;
	spentCents: number;
	balanceCents: number;
	onAssign: (cents: number) => Promise<unknown>;
}) => {
	const [editing, setEditing] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const negative = balanceCents < 0;

	const save = async (cents: number) => {
		setError(null);
		try {
			await onAssign(cents);
			setEditing(false);
		} catch (err) {
			setError(err instanceof Error ? err.message : String(err));
		}
	};

	return (
		<>
			<tr>
				<td>
					<div className={styles.name}>{name}</div>
					{notes.length > 0 && (
						<div className={styles.detail}>{notes.join(' · ')}</div>
					)}
				</td>
				<td className={styles.money}>
					{editing ? (
						<AssignedEditor
							name={name}
							assignedCents={assignedCents}
							onSave={(cents) => void save(cents)}
							onCancel={() => {
								setEditing(false);
								setError(null);
							}}
							onInvalid={setError}
						/>
					) : (
						<button
							type='button'
							className={cx(
								buttons.quiet,
								buttons.small,
								styles.assignButton,
								assignedCents === 0 && styles.zero,
							)}
							aria-label={`Assigned to ${name}`}
							onClick={() => setEditing(true)}
						>
							{formatCents(assignedCents)}
						</button>
					)}
				</td>
				<td className={styles.money}>{formatCents(spentCents)}</td>
				<td
					className={cx(
						styles.money,
						styles.balance,
						negative && styles.negative,
					)}
				>
					{formatCents(balanceCents)}
				</td>
			</tr>
			{error && (
				<tr className={styles.errorRow}>
					<td colSpan={4} role='alert'>
						{error}
					</td>
				</tr>
			)}
		</>
	);
};

/** Inline assignment input: Enter saves (empty or 0 removes it), Esc cancels. */
const AssignedEditor = ({
	name,
	assignedCents,
	onSave,
	onCancel,
	onInvalid,
}: {
	name: string;
	assignedCents: number;
	onSave: (cents: number) => void;
	onCancel: () => void;
	onInvalid: (message: string) => void;
}) => {
	const [value, setValue] = useState(
		assignedCents === 0 ? '' : formatCents(assignedCents),
	);

	const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
		if (e.nativeEvent.isComposing) return;
		if (e.key === 'Enter') {
			e.preventDefault();
			const parsed = parseAssigned(value);
			if (!parsed.ok) return onInvalid(parsed.error);
			if (parsed.cents === assignedCents) return onCancel();
			onSave(parsed.cents);
		} else if (e.key === 'Escape') {
			e.preventDefault();
			onCancel();
		}
	};

	return (
		<input
			aria-label={`Assigned to ${name}`}
			className={styles.assignInput}
			inputMode='decimal'
			autoComplete='off'
			placeholder='0.00'
			autoFocus
			value={value}
			onChange={(e) => setValue(e.target.value)}
			onKeyDown={onKeyDown}
			onBlur={onCancel}
		/>
	);
};
