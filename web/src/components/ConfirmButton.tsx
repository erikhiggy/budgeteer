import { useEffect, useState } from 'react';
import { cx } from '../lib/cx';
import styles from './Button.module.css';

type Props = {
	/** The first click arms the button; the second runs this. */
	onConfirm: () => void;
	children: string;
	/** Shown while armed. */
	confirmLabel?: string;
	/** The accessible name, when children alone is ambiguous (e.g. in a table). */
	'aria-label'?: string;
	disabled?: boolean;
	className?: string;
};

/** How long the armed state lasts before it quietly resets. */
const ARMED_MS = 4000;

/**
 * A two-click button for deletes: the first click turns it into
 * "Confirm", the second confirms. Leaving it (blur) or waiting
 * a few seconds disarms it.
 */
export function ConfirmButton({
	onConfirm,
	children,
	confirmLabel = 'Confirm',
	'aria-label': ariaLabel,
	disabled,
	className,
}: Props) {
	const [armed, setArmed] = useState(false);

	useEffect(() => {
		if (!armed) return;
		const timer = setTimeout(() => setArmed(false), ARMED_MS);
		return () => clearTimeout(timer);
	}, [armed]);

	return (
		<button
			type='button'
			aria-label={armed ? confirmLabel : ariaLabel}
			data-armed={armed || undefined}
			className={cx(
				styles.small,
				armed ? styles.danger : styles.quiet,
				className,
			)}
			disabled={disabled}
			onClick={() => {
				if (armed) {
					setArmed(false);
					onConfirm();
				} else {
					setArmed(true);
				}
			}}
			onBlur={() => setArmed(false)}
		>
			{armed ? confirmLabel : children}
		</button>
	);
}
