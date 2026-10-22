import { useState } from 'react';
import { useCategories, useCreateCategory } from '../../api/hooks';
import type { Category } from '../../api/types';
import { Notice } from '../../components/Notice';
import styles from './CategoriesView.module.css';

/** Route: /categories. Add and list categories; edit and delete arrive in M2. */
export function CategoriesView() {
	const categories = useCategories();

	return (
		<section className={styles.view}>
			<AddCategoryForm />
			{categories.isPending && <Notice>Loading categories…</Notice>}
			{categories.isError && (
				<Notice tone='error'>
					Couldn’t load categories: {categories.error.message}
				</Notice>
			)}
			{categories.isSuccess && (
				<div className={styles.columns}>
					<CategoryList
						title='Spending'
						categories={categories.data.filter((c) => !c.is_income)}
						empty='No spending categories yet.'
					/>
					<CategoryList
						title='Income'
						categories={categories.data.filter((c) => c.is_income)}
						empty='No income categories.'
					/>
				</div>
			)}
		</section>
	);
}

function AddCategoryForm() {
	const [name, setName] = useState('');
	const [isIncome, setIsIncome] = useState(false);
	const create = useCreateCategory();

	return (
		<form
			className={styles.form}
			aria-label='New category'
			onSubmit={(e) => {
				e.preventDefault();
				create.mutate(
					{ name: name.trim(), is_income: isIncome },
					{
						onSuccess: () => {
							setName('');
							setIsIncome(false);
						},
					},
				);
			}}
		>
			<input
				aria-label='Category name'
				placeholder='New category'
				required
				value={name}
				onChange={(e) => setName(e.target.value)}
			/>
			<label className={styles.checkbox}>
				<input
					type='checkbox'
					checked={isIncome}
					onChange={(e) => setIsIncome(e.target.checked)}
				/>
				Income
			</label>
			<button type='submit' disabled={create.isPending}>
				Add category
			</button>
			{create.error && (
				<p className={styles.error} role='alert'>
					{create.error.message}
				</p>
			)}
		</form>
	);
}

function CategoryList({
	title,
	categories,
	empty,
}: {
	title: string;
	categories: Category[];
	empty: string;
}) {
	return (
		<div className={styles.list}>
			<h2 className={styles.heading}>{title}</h2>
			{categories.length === 0 ? (
				<p className={styles.muted}>{empty}</p>
			) : (
				<ul>
					{categories.map((c) => (
						<li key={c.id}>{c.name}</li>
					))}
				</ul>
			)}
		</div>
	);
}
