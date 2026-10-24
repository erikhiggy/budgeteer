// Chart colors by role. The values are tokens in styles/tokens.css, so dark
// mode (later) only has to redefine the tokens. SVG fill and stroke accept
// var() like any other CSS color.

export const chartColors = {
	/** Spending: the one series in most charts. */
	spent: 'var(--color-chart-1)',
	/** What spending is measured against: the cap, or available with rollover. */
	goal: 'var(--color-chart-2)',
	grid: 'var(--color-chart-grid)',
	axis: 'var(--color-text-muted)',
	text: 'var(--color-text)',
} as const;
