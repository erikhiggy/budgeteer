import { lazy, Suspense } from 'react';
import { BrowserRouter, Navigate, NavLink, Route, Routes } from 'react-router';
import { cx } from './lib/cx';
import { CategoriesView } from './views/categories/CategoriesView';
import { GoalsView } from './views/goals/GoalsView';
import { TransactionsView } from './views/transactions/TransactionsView';
import styles from './App.module.css';

// Recharts is most of the bundle, so the Charts view loads on first visit.
const ChartsView = lazy(() =>
	import('./views/charts/ChartsView').then((m) => ({ default: m.ChartsView })),
);

export const App = () => {
	return (
		<BrowserRouter>
			<div className={styles.shell}>
				<header className={styles.header}>
					<h1 className={styles.title}>Budgeteer</h1>
					<nav className={styles.nav}>
						<Tab to='/transactions'>Transactions</Tab>
						<Tab to='/goals'>Goals</Tab>
						<Tab to='/charts'>Charts</Tab>
						<Tab to='/categories'>Categories</Tab>
					</nav>
				</header>
				<main className={styles.main}>
					<Suspense>
						<Routes>
							<Route path='/transactions' element={<TransactionsView />} />
							<Route
								path='/transactions/:accountId'
								element={<TransactionsView />}
							/>
							<Route path='/goals' element={<GoalsView />} />
							<Route path='/goals/:month' element={<GoalsView />} />
							<Route path='/charts' element={<ChartsView />} />
							<Route path='/charts/:month' element={<ChartsView />} />
							<Route path='/categories' element={<CategoriesView />} />
							<Route
								path='*'
								element={<Navigate to='/transactions' replace />}
							/>
						</Routes>
					</Suspense>
				</main>
			</div>
		</BrowserRouter>
	);
};

const Tab = ({ to, children }: { to: string; children: string }) => {
	return (
		<NavLink
			to={to}
			className={({ isActive }) => cx(styles.tab, isActive && styles.active)}
		>
			{children}
		</NavLink>
	);
};
