import { useId, useRef, useState, type KeyboardEvent } from 'react';
import type {
	Category,
	Payee,
	Transaction,
	TransactionInput,
} from '../../api/types';
import buttons from '../../components/Button.module.css';
import { CategorySelect } from '../../components/CategorySelect';
import { PayeeCombobox } from '../../components/PayeeCombobox';
import { cx } from '../../lib/cx';
import { isISODate, todayISO } from '../../lib/date';
import { formatAmount, parseAmount } from '../../lib/money';
import styles from './EntryRow.module.css';

type Props = {
	accountId: number;
	categories: Category[];
	/** Suggestions for the payee field, with the category to autofill. */
	payees: Payee[];
	/** Resolves when saved; rejects with an Error whose message is shown. */
	onSave: (input: TransactionInput) => Promise<unknown>;
	/**
	 * Edit mode: start from this transaction. Esc calls onCancel instead of
	 * clearing, and after a save the parent is expected to close the row.
	 */
	initial?: Transaction;
	onCancel?: () => void;
	className?: string;
};

type Field = 'date' | 'category' | 'amount';

/**
 * Keyboard-first entry row, also used to edit a row in the table. Tab moves
 * between fields; Enter anywhere saves a complete row, or jumps to the first
 * missing field without sending anything; Esc clears (or cancels an edit).
 * Enter and Esc go to the payee suggestions first while they're open.
 */
export function EntryRow({
	accountId,
	categories,
	payees,
	onSave,
	initial,
	onCancel,
	className,
}: Props) {
	const editing = initial !== undefined;
	const [date, setDate] = useState(initial?.date ?? todayISO);
	const [payee, setPayee] = useState(initial?.payee ?? '');
	const [categoryId, setCategoryId] = useState(
		initial?.category_id != null ? String(initial.category_id) : '',
	);
	const [amount, setAmount] = useState(
		initial ? formatAmount(initial.amount_cents) : '',
	);
	const [note, setNote] = useState(initial?.note ?? '');
	const [attempted, setAttempted] = useState(false);
	const [saving, setSaving] = useState(false);
	const [serverError, setServerError] = useState<string | null>(null);
	const hintId = useId();

	// Guards against a second Enter (or key repeat) arriving before React
	// re-renders with saving=true.
	const savingRef = useRef(false);

	const dateRef = useRef<HTMLInputElement>(null);
	const payeeRef = useRef<HTMLInputElement>(null);
	const categoryRef = useRef<HTMLSelectElement>(null);
	const amountRef = useRef<HTMLInputElement>(null);

	const parsed = parseAmount(amount);

	/** Tab order, so focus goes to the first problem the user would reach. */
	function findProblems(category: string): [Field, string][] {
		const found: [Field, string][] = [];
		if (!isISODate(date)) found.push(['date', 'Date is required']);
		if (category === '') found.push(['category', 'Pick a category']);
		if (!parsed.ok) found.push(['amount', parsed.error]);
		return found;
	}
	const problems = findProblems(categoryId);

	const invalid = (field: Field) =>
		attempted && problems.some(([f]) => f === field);

	const categoryName = new Map(categories.map((c) => [c.id, c.name]));

	function focusField(field: Field) {
		const refs = { date: dateRef, category: categoryRef, amount: amountRef };
		refs[field].current?.focus();
	}

	/** The known payee typed out in full, ignoring case, if any. */
	function knownPayee(): Payee | undefined {
		const typed = payee.trim().toLowerCase();
		return payees.find((p) => p.payee.toLowerCase() === typed);
	}

	/**
	 * Category autofill: a known payee brings its latest category, unless a
	 * category is already chosen. Returns the category to use.
	 */
	function autofill(p: Payee | undefined): string {
		if (
			p?.category_id != null &&
			categoryId === '' &&
			categoryName.has(p.category_id)
		) {
			const id = String(p.category_id);
			setCategoryId(id);
			return id;
		}
		return categoryId;
	}

	function clear() {
		setPayee('');
		setCategoryId('');
		setAmount('');
		setNote('');
		setAttempted(false);
		setServerError(null);
	}

	async function save() {
		if (savingRef.current) return;
		// Enter straight after typing a known payee should count its category,
		// even though the payee field hasn't lost focus yet.
		const category = autofill(knownPayee());
		const found = findProblems(category);
		if (found.length > 0 || !parsed.ok) {
			setAttempted(true);
			focusField(found[0][0]);
			return;
		}

		savingRef.current = true;
		setSaving(true);
		setServerError(null);
		try {
			await onSave({
				account_id: accountId,
				category_id: Number(category),
				date,
				payee: payee.trim(),
				amount_cents: parsed.cents,
				note: note.trim(),
			});
			if (!editing) {
				clear(); // keeps the date, for entering several from one receipt
				payeeRef.current?.focus();
			}
		} catch (err) {
			setServerError(err instanceof Error ? err.message : String(err));
		} finally {
			savingRef.current = false;
			setSaving(false);
		}
	}

	function onKeyDown(e: KeyboardEvent<HTMLFormElement>) {
		if (e.nativeEvent.isComposing) return;
		if (e.key === 'Enter') {
			// Also stops the browser's implicit form submit and button click,
			// so a save only ever starts here.
			e.preventDefault();
			void save();
		} else if (e.key === 'Escape') {
			e.preventDefault();
			if (editing) onCancel?.();
			else clear();
		}
	}

	const message = serverError ?? (attempted ? problems[0]?.[1] : undefined);

	return (
		<form
			className={cx(styles.row, className)}
			aria-label={editing ? 'Edit transaction' : 'New transaction'}
			aria-busy={saving || undefined}
			noValidate
			onKeyDown={onKeyDown}
			onSubmit={(e) => {
				e.preventDefault();
				void save();
			}}
		>
			<input
				ref={dateRef}
				type='date'
				aria-label='Date'
				aria-invalid={invalid('date') || undefined}
				autoFocus={editing}
				className={cx(styles.field, invalid('date') && styles.invalid)}
				value={date}
				onChange={(e) => setDate(e.target.value)}
			/>
			<PayeeCombobox
				ref={payeeRef}
				value={payee}
				onChange={setPayee}
				payees={payees}
				onPick={autofill}
				onBlur={() => autofill(knownPayee())}
				describe={(p) =>
					p.category_id === null ? undefined : categoryName.get(p.category_id)
				}
				className={styles.field}
			/>
			<CategorySelect
				ref={categoryRef}
				categories={categories}
				value={categoryId}
				onChange={setCategoryId}
				invalid={invalid('category')}
				className={cx(styles.field, invalid('category') && styles.invalid)}
			/>
			<input
				ref={amountRef}
				aria-label='Amount'
				placeholder='0.00'
				inputMode='decimal'
				autoComplete='off'
				aria-invalid={invalid('amount') || undefined}
				aria-describedby={hintId}
				className={cx(
					styles.field,
					styles.amount,
					invalid('amount') && styles.invalid,
				)}
				value={amount}
				onChange={(e) => setAmount(e.target.value)}
			/>
			<input
				aria-label='Note'
				placeholder='Note'
				autoComplete='off'
				className={styles.field}
				value={note}
				onChange={(e) => setNote(e.target.value)}
			/>
			<div className={styles.actions}>
				<button type='submit' className={styles.save} disabled={saving}>
					{editing ? 'Save' : 'Add'}
				</button>
				{editing && (
					<button type='button' className={buttons.quiet} onClick={onCancel}>
						Cancel
					</button>
				)}
			</div>

			<p id={hintId} className={styles.hint}>
				{message ? (
					<span className={styles.error} role='alert'>
						{message}
					</span>
				) : editing ? (
					<>
						<kbd>Enter</kbd> saves · <kbd>Esc</kbd> cancels
					</>
				) : (
					<>
						<kbd>Enter</kbd> saves · <kbd>Esc</kbd> clears · <kbd>↓</kbd> picks
						a past payee · <kbd>+</kbd> before an amount means money in
					</>
				)}
			</p>
		</form>
	);
}
