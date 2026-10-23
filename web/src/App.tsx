import { BrowserRouter, Navigate, NavLink, Route, Routes } from 'react-router';
import { cx } from './lib/cx';
import { CategoriesView } from './views/categories/CategoriesView';
import { GoalsView } from './views/goals/GoalsView';
import { TransactionsView } from './views/transactions/TransactionsView';
import styles from './App.module.css';

export function App() {
	return (
		<BrowserRouter>
			<div className={styles.shell}>
				<header className={styles.header}>
					<h1 className={styles.title}>Budgeteer</h1>
					<nav className={styles.nav}>
						<Tab to='/transactions'>Transactions</Tab>
						<Tab to='/goals'>Goals</Tab>
						<Tab to='/categories'>Categories</Tab>
					</nav>
				</header>
				<main className={styles.main}>
					<Routes>
						<Route path='/transactions' element={<TransactionsView />} />
						<Route
							path='/transactions/:accountId'
							element={<TransactionsView />}
						/>
						<Route path='/goals' element={<GoalsView />} />
						<Route path='/goals/:month' element={<GoalsView />} />
						<Route path='/categories' element={<CategoriesView />} />
						<Route path='*' element={<Navigate to='/transactions' replace />} />
					</Routes>
				</main>
			</div>
		</BrowserRouter>
	);
}

function Tab({ to, children }: { to: string; children: string }) {
	return (
		<NavLink
			to={to}
			className={({ isActive }) => cx(styles.tab, isActive && styles.active)}
		>
			{children}
		</NavLink>
	);
}
