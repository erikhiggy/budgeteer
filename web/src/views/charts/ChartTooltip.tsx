import type { ReactNode } from 'react';
import styles from './ChartsView.module.css';

type Line = { label: string; value: string; color?: string };

/** The hover box for every chart: a title and label/value lines. */
export const ChartTooltip = ({
	title,
	lines,
	footer,
}: {
	title: string;
	lines: Line[];
	footer?: ReactNode;
}) => {
	return (
		<div className={styles.tooltip}>
			<div className={styles.tooltipTitle}>{title}</div>
			{lines.map((l) => (
				<div key={l.label} className={styles.tooltipLine}>
					{l.color && (
						<span className={styles.swatch} style={{ background: l.color }} />
					)}
					<span>{l.label}</span>
					<span className={styles.tooltipValue}>{l.value}</span>
				</div>
			))}
			{footer && <div className={styles.tooltipFooter}>{footer}</div>}
		</div>
	);
};
