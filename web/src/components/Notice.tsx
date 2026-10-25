import type { ReactNode } from 'react';
import { cx } from '../lib/cx';
import styles from './Notice.module.css';

type Props = {
	tone?: 'info' | 'error';
	children: ReactNode;
};

/** A one-line loading, empty, or error message. */
export const Notice = ({ tone = 'info', children }: Props) => {
	return (
		<p
			className={cx(styles.notice, tone === 'error' && styles.error)}
			role={tone === 'error' ? 'alert' : undefined}
		>
			{children}
		</p>
	);
};
