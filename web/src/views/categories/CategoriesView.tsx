import { useState } from 'react';
import {
	useCategories,
	useCreateCategory,
	useDeleteCategory,
	useUpdateCategory,
} from '../../api/hooks';
import { openingBalanceCategory, type Category } from '../../api/types';
import buttons from '../../components/Button.module.css';
import { ConfirmButton } from '../../components/ConfirmButton';
import { Notice } from '../../components/Notice';
import { cx } from '../../lib/cx';
import styles from './CategoriesView.module.css';

/** Route: /categories. Add, rename, move between spending and income, delete. */
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
						<CategoryItem key={c.id} category={c} />
					))}
				</ul>
			)}
		</div>
	);
}

function CategoryItem({ category }: { category: Category }) {
	const [renaming, setRenaming] = useState(false);
	const update = useUpdateCategory();
	const remove = useDeleteCategory();
	const error = update.error ?? remove.error;

	if (renaming) {
		return (
			<li>
				<RenameForm category={category} onDone={() => setRenaming(false)} />
			</li>
		);
	}

	const locked = category.name === openingBalanceCategory;
	return (
		<li>
			<div className={styles.item}>
				<span className={styles.name}>{category.name}</span>
				{locked ? (
					<span className={styles.muted}>Used for opening balances</span>
				) : (
					<span className={styles.actions}>
						<button
							type='button'
							className={cx(buttons.quiet, buttons.small)}
							aria-label={`Rename ${category.name}`}
							onClick={() => setRenaming(true)}
						>
							Rename
						</button>
						<button
							type='button'
							className={cx(buttons.quiet, buttons.small)}
							disabled={update.isPending}
							onClick={() =>
								update.mutate({
									id: category.id,
									input: { is_income: !category.is_income },
								})
							}
						>
							{category.is_income ? 'Move to spending' : 'Move to income'}
						</button>
						<ConfirmButton
							aria-label={`Delete ${category.name}`}
							disabled={remove.isPending}
							onConfirm={() => remove.mutate(category.id)}
						>
							Delete
						</ConfirmButton>
					</span>
				)}
			</div>
			{error && (
				<p className={styles.error} role='alert'>
					{error.message}
				</p>
			)}
		</li>
	);
}

function RenameForm({
	category,
	onDone,
}: {
	category: Category;
	onDone: () => void;
}) {
	const [name, setName] = useState(category.name);
	const update = useUpdateCategory();

	return (
		<form
			className={styles.rename}
			aria-label={`Rename ${category.name}`}
			onSubmit={(e) => {
				e.preventDefault();
				if (name.trim() === category.name) return onDone();
				update.mutate(
					{ id: category.id, input: { name: name.trim() } },
					{ onSuccess: onDone },
				);
			}}
			onKeyDown={(e) => {
				if (e.key === 'Escape') onDone();
			}}
		>
			<input
				aria-label='Category name'
				autoFocus
				required
				value={name}
				onChange={(e) => setName(e.target.value)}
			/>
			<button
				type='submit'
				className={buttons.small}
				disabled={update.isPending}
			>
				Save
			</button>
			<button
				type='button'
				className={cx(buttons.quiet, buttons.small)}
				onClick={onDone}
			>
				Cancel
			</button>
			{update.error && (
				<p className={styles.error} role='alert'>
					{update.error.message}
				</p>
			)}
		</form>
	);
}
