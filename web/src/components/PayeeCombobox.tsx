import { useId, useState, type KeyboardEvent, type Ref } from 'react';
import type { Payee } from '../api/types';
import { cx } from '../lib/cx';
import styles from './PayeeCombobox.module.css';

type Props = {
	value: string;
	onChange: (value: string) => void;
	payees: Payee[];
	/** Called when a suggestion is picked, after onChange. */
	onPick: (payee: Payee) => void;
	/** Shown next to each suggestion, e.g. its category's name. */
	describe?: (payee: Payee) => string | undefined;
	onBlur?: () => void;
	className?: string;
	ref?: Ref<HTMLInputElement>;
};

const MAX_SUGGESTIONS = 8;

/** Payees containing the typed text, those that start with it first. */
export const matchPayees = (payees: Payee[], text: string): Payee[] => {
	const q = text.trim().toLowerCase();
	if (q === '') return [];
	const starts: Payee[] = [];
	const contains: Payee[] = [];
	for (const p of payees) {
		const name = p.payee.toLowerCase();
		// Nothing to suggest when it's already typed out in full.
		if (p.payee === text.trim()) continue;
		if (name.startsWith(q)) starts.push(p);
		else if (name.includes(q)) contains.push(p);
	}
	return [...starts, ...contains].slice(0, MAX_SUGGESTIONS);
};

/**
 * The payee field with suggestions from past transactions (ARIA combobox
 * pattern). A native <datalist> can't be used, because the entry row has to
 * know whether a suggestion is highlighted:
 *   - Arrow keys open the list and move the highlight. Nothing is
 *     highlighted until then, so Enter after typing still saves the row.
 *   - Enter on a highlighted suggestion picks it instead of saving (rule 3).
 *   - Esc closes the list if it's open; otherwise it reaches the row, which
 *     clears.
 */
export const PayeeCombobox = ({
	value,
	onChange,
	payees,
	onPick,
	describe,
	onBlur,
	className,
	ref,
}: Props) => {
	const listId = useId();
	const [open, setOpen] = useState(false);
	const [highlight, setHighlight] = useState(-1);

	const matches = open ? matchPayees(payees, value) : [];
	const shown = matches.length > 0;
	const optionId = (i: number) => `${listId}-${i}`;

	const close = () => {
		setOpen(false);
		setHighlight(-1);
	};

	const pick = (p: Payee) => {
		onChange(p.payee);
		onPick(p);
		close();
	};

	const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
		if (e.nativeEvent.isComposing) return;
		switch (e.key) {
			case 'ArrowDown':
			case 'ArrowUp': {
				const all = matchPayees(payees, value);
				if (all.length === 0) return;
				e.preventDefault();
				setOpen(true);
				const n = all.length;
				setHighlight((h) =>
					e.key === 'ArrowDown' ? (h + 1) % n : h <= 0 ? n - 1 : h - 1,
				);
				break;
			}
			case 'Enter':
				if (shown && highlight >= 0 && highlight < matches.length) {
					// Keep the row from saving.
					e.preventDefault();
					e.stopPropagation();
					pick(matches[highlight]);
				}
				break;
			case 'Escape':
				if (shown) {
					// Keep the row from clearing.
					e.preventDefault();
					e.stopPropagation();
					close();
				}
				break;
			case 'Tab':
				close();
				break;
		}
	};

	return (
		<div className={cx(styles.combobox, className)}>
			<input
				ref={ref}
				role='combobox'
				aria-label='Payee'
				aria-autocomplete='list'
				aria-expanded={shown}
				aria-controls={listId}
				aria-activedescendant={
					shown && highlight >= 0 ? optionId(highlight) : undefined
				}
				placeholder='Payee'
				autoComplete='off'
				className={styles.input}
				value={value}
				onChange={(e) => {
					onChange(e.target.value);
					setOpen(true);
					setHighlight(-1);
				}}
				onKeyDown={onKeyDown}
				onBlur={() => {
					close();
					onBlur?.();
				}}
			/>
			<ul
				id={listId}
				role='listbox'
				aria-label='Payee suggestions'
				className={styles.list}
				hidden={!shown}
			>
				{matches.map((p, i) => (
					<li
						key={p.payee}
						id={optionId(i)}
						role='option'
						aria-selected={i === highlight}
						className={cx(styles.option, i === highlight && styles.highlighted)}
						// mousedown, not click: picking must happen before the input blurs.
						onMouseDown={(e) => {
							e.preventDefault();
							pick(p);
						}}
						onMouseEnter={() => setHighlight(i)}
					>
						<span>{p.payee}</span>
						<span className={styles.detail}>{describe?.(p)}</span>
					</li>
				))}
			</ul>
		</div>
	);
};
