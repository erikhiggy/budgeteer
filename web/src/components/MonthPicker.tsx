import buttons from './Button.module.css';
import { cx } from '../lib/cx';
import { addMonths, currentMonth, formatMonth } from '../lib/month';
import styles from './MonthPicker.module.css';

type Props = {
	/** "YYYY-MM" on screen. */
	month: string;
	onChange: (month: string) => void;
};

/** ‹ October 2026 ›, plus "This month" when another month is showing. */
export const MonthPicker = ({ month, onChange }: Props) => {
	const today = currentMonth();
	return (
		<div className={styles.picker}>
			<button
				type='button'
				className={cx(buttons.quiet, styles.step)}
				aria-label='Previous month'
				onClick={() => onChange(addMonths(month, -1))}
			>
				‹
			</button>
			<h2 className={styles.month}>{formatMonth(month)}</h2>
			<button
				type='button'
				className={cx(buttons.quiet, styles.step)}
				aria-label='Next month'
				onClick={() => onChange(addMonths(month, 1))}
			>
				›
			</button>
			{month !== today && (
				<button
					type='button'
					className={cx(buttons.quiet, buttons.small)}
					onClick={() => onChange(today)}
				>
					This month
				</button>
			)}
		</div>
	);
};
