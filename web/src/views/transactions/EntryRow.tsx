import { useRef, useState, type KeyboardEvent } from 'react';
import type { Category, TransactionInput } from '../../api/types';
import { CategorySelect } from '../../components/CategorySelect';
import { cx } from '../../lib/cx';
import { isISODate, todayISO } from '../../lib/date';
import { parseAmount } from '../../lib/money';
import styles from './EntryRow.module.css';

type Props = {
	accountId: number;
	categories: Category[];
	/** Resolves when saved; rejects with an Error whose message is shown. */
	onSave: (input: TransactionInput) => Promise<unknown>;
};

type Field = 'date' | 'category' | 'amount';

/**
 * Keyboard-first entry row. Tab moves between fields; Enter anywhere saves
 * a complete row, or jumps to the first missing field without sending
 * anything; Esc clears.
 */
export function EntryRow({ accountId, categories, onSave }: Props) {
	const [date, setDate] = useState(todayISO);
	const [payee, setPayee] = useState('');
	const [categoryId, setCategoryId] = useState('');
	const [amount, setAmount] = useState('');
	const [note, setNote] = useState('');
	const [attempted, setAttempted] = useState(false);
	const [saving, setSaving] = useState(false);
	const [serverError, setServerError] = useState<string | null>(null);

	// Guards against a second Enter (or key repeat) arriving before React
	// re-renders with saving=true.
	const savingRef = useRef(false);

	const dateRef = useRef<HTMLInputElement>(null);
	const payeeRef = useRef<HTMLInputElement>(null);
	const categoryRef = useRef<HTMLSelectElement>(null);
	const amountRef = useRef<HTMLInputElement>(null);

	const parsed = parseAmount(amount);
	// Tab order, so focus goes to the first problem the user would reach.
	const problems: [Field, string][] = [];
	if (!isISODate(date)) problems.push(['date', 'Date is required']);
	if (categoryId === '') problems.push(['category', 'Pick a category']);
	if (!parsed.ok) problems.push(['amount', parsed.error]);

	const invalid = (field: Field) =>
		attempted && problems.some(([f]) => f === field);

	function focusField(field: Field) {
		const refs = { date: dateRef, category: categoryRef, amount: amountRef };
		refs[field].current?.focus();
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
		if (problems.length > 0 || !parsed.ok) {
			setAttempted(true);
			focusField(problems[0][0]);
			return;
		}

		savingRef.current = true;
		setSaving(true);
		setServerError(null);
		try {
			await onSave({
				account_id: accountId,
				category_id: Number(categoryId),
				date,
				payee: payee.trim(),
				amount_cents: parsed.cents,
				note: note.trim(),
			});
			clear(); // keeps the date, for entering several from one receipt
			payeeRef.current?.focus();
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
			clear();
		}
	}

	const message = serverError ?? (attempted ? problems[0]?.[1] : undefined);

	return (
		<form
			className={styles.row}
			aria-label='New transaction'
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
				className={cx(styles.field, invalid('date') && styles.invalid)}
				value={date}
				onChange={(e) => setDate(e.target.value)}
			/>
			<input
				ref={payeeRef}
				aria-label='Payee'
				placeholder='Payee'
				autoComplete='off'
				className={styles.field}
				value={payee}
				onChange={(e) => setPayee(e.target.value)}
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
				aria-describedby='entry-row-hint'
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
			<button type='submit' className={styles.save} disabled={saving}>
				Add
			</button>

			<p id='entry-row-hint' className={styles.hint}>
				{message ? (
					<span className={styles.error} role='alert'>
						{message}
					</span>
				) : (
					<>
						<kbd>Enter</kbd> saves · <kbd>Esc</kbd> clears · <kbd>+</kbd> before
						an amount means money in
					</>
				)}
			</p>
		</form>
	);
}
