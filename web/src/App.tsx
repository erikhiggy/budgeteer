import { useEffect, useState } from 'react';
import { cx } from './lib/cx';
import styles from './App.module.css';

type Health = 'checking' | 'ok' | 'down';

// M0 placeholder: proves the frontend, proxy/embed, and API are connected.
// M1 replaces this with the router and the Transactions view.
export function App() {
	const [health, setHealth] = useState<Health>('checking');

	useEffect(() => {
		const ctrl = new AbortController();
		fetch('/api/health', { signal: ctrl.signal })
			.then((res) => setHealth(res.ok ? 'ok' : 'down'))
			.catch((err: unknown) => {
				if (!ctrl.signal.aborted) {
					console.error('health check failed', err);
					setHealth('down');
				}
			});
		return () => ctrl.abort();
	}, []);

	return (
		<div className={styles.shell}>
			<header className={styles.header}>
				<h1 className={styles.title}>Budgeteer</h1>
				<span className={cx(styles.status, styles[health])} role='status'>
					{health === 'checking' && 'Connecting…'}
					{health === 'ok' && 'API connected'}
					{health === 'down' && 'API unreachable'}
				</span>
			</header>
			<main className={styles.main}>
				<p className={styles.muted}>
					Transactions, Goals and Charts arrive in the next milestones.
				</p>
			</main>
		</div>
	);
}
