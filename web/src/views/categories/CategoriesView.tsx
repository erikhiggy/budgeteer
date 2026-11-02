import { useState } from 'react';
import {
	useCategories,
	useCategoryGroups,
	useCreateCategory,
	useCreateCategoryGroup,
	useDeleteCategory,
	useDeleteCategoryGroup,
	useReorderCategoryGroups,
	useUpdateCategory,
	useUpdateCategoryGroup,
} from '../../api/hooks';
import {
	openingBalanceCategory,
	type Category,
	type CategoryGroup,
} from '../../api/types';
import buttons from '../../components/Button.module.css';
import { ConfirmButton } from '../../components/ConfirmButton';
import { Notice } from '../../components/Notice';
import { cx } from '../../lib/cx';
import styles from './CategoriesView.module.css';

/**
 * Route: /categories. Add, rename, move between spending and income, delete,
 * and organize spending categories under groups.
 */
export const CategoriesView = () => {
	const categories = useCategories();
	const groupsQuery = useCategoryGroups();
	const groups = groupsQuery.data ?? [];

	return (
		<section className={styles.view}>
			<AddCategoryForm groups={groups} />
			{categories.isPending && <Notice>Loading categories…</Notice>}
			{categories.isError && (
				<Notice tone='error'>
					Couldn’t load categories: {categories.error.message}
				</Notice>
			)}
			{categories.isSuccess && (
				<div className={styles.columns}>
					<SpendingList
						categories={categories.data.filter((c) => !c.is_income)}
						groups={groups}
					/>
					<CategoryList
						title='Income'
						categories={categories.data.filter((c) => c.is_income)}
						groups={[]}
						empty='No income categories.'
					/>
					<GroupsPanel groups={groups} error={groupsQuery.error?.message} />
				</div>
			)}
		</section>
	);
};

const AddCategoryForm = ({ groups }: { groups: CategoryGroup[] }) => {
	const [name, setName] = useState('');
	const [isIncome, setIsIncome] = useState(false);
	const [groupId, setGroupId] = useState('');
	const create = useCreateCategory();

	return (
		<form
			className={styles.form}
			aria-label='New category'
			onSubmit={(e) => {
				e.preventDefault();
				create.mutate(
					{
						name: name.trim(),
						is_income: isIncome,
						// Income categories can't be grouped.
						group_id: isIncome || groupId === '' ? null : Number(groupId),
					},
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
			{!isIncome && groups.length > 0 && (
				<GroupSelect
					label='Group for new category'
					groups={groups}
					value={groupId === '' ? null : Number(groupId)}
					onChange={(id) => setGroupId(id === null ? '' : String(id))}
				/>
			)}
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
};

/** The spending categories, under a subheading per group when there are any. */
const SpendingList = ({
	categories,
	groups,
}: {
	categories: Category[];
	groups: CategoryGroup[];
}) => {
	if (groups.length === 0) {
		return (
			<CategoryList
				title='Spending'
				categories={categories}
				groups={groups}
				empty='No spending categories yet.'
			/>
		);
	}
	const known = new Set(groups.map((g) => g.id));
	const sections = [
		...groups.map((g) => ({
			key: g.id,
			title: g.name,
			members: categories.filter((c) => c.group_id === g.id),
		})),
		{
			key: 0,
			title: 'Ungrouped',
			members: categories.filter(
				(c) => c.group_id === null || !known.has(c.group_id),
			),
		},
	];
	return (
		<div className={styles.list}>
			<h2 className={styles.heading}>Spending</h2>
			{categories.length === 0 && (
				<p className={styles.muted}>No spending categories yet.</p>
			)}
			{sections.map(
				(section) =>
					// An empty group still gets its heading, as a place to move things to.
					(section.members.length > 0 || section.key !== 0) && (
						<div key={section.key} className={styles.section}>
							<h3 className={styles.subheading}>{section.title}</h3>
							{section.members.length === 0 ? (
								<p className={styles.muted}>Nothing in this group yet.</p>
							) : (
								<ul>
									{section.members.map((c) => (
										<CategoryItem key={c.id} category={c} groups={groups} />
									))}
								</ul>
							)}
						</div>
					),
			)}
		</div>
	);
};

const CategoryList = ({
	title,
	categories,
	groups,
	empty,
}: {
	title: string;
	categories: Category[];
	groups: CategoryGroup[];
	empty: string;
}) => {
	return (
		<div className={styles.list}>
			<h2 className={styles.heading}>{title}</h2>
			{categories.length === 0 ? (
				<p className={styles.muted}>{empty}</p>
			) : (
				<ul>
					{categories.map((c) => (
						<CategoryItem key={c.id} category={c} groups={groups} />
					))}
				</ul>
			)}
		</div>
	);
};

const CategoryItem = ({
	category,
	groups,
}: {
	category: Category;
	groups: CategoryGroup[];
}) => {
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
						{!category.is_income && groups.length > 0 && (
							<GroupSelect
								label={`Group for ${category.name}`}
								groups={groups}
								value={category.group_id}
								disabled={update.isPending}
								onChange={(id) =>
									update.mutate({
										id: category.id,
										input: { group_id: id },
									})
								}
							/>
						)}
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
};

const RenameForm = ({
	category,
	onDone,
}: {
	category: Category;
	onDone: () => void;
}) => {
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
};

/** A native select of groups, with "No group" for null. */
const GroupSelect = ({
	label,
	groups,
	value,
	onChange,
	disabled,
}: {
	label: string;
	groups: CategoryGroup[];
	value: number | null;
	onChange: (id: number | null) => void;
	disabled?: boolean;
}) => {
	return (
		<select
			aria-label={label}
			className={styles.groupSelect}
			disabled={disabled}
			value={value === null ? '' : String(value)}
			onChange={(e) =>
				onChange(e.target.value === '' ? null : Number(e.target.value))
			}
		>
			<option value=''>No group</option>
			{groups.map((g) => (
				<option key={g.id} value={g.id}>
					{g.name}
				</option>
			))}
		</select>
	);
};

/** Add, rename, reorder and delete groups. Deleting one ungroups its categories. */
const GroupsPanel = ({
	groups,
	error,
}: {
	groups: CategoryGroup[];
	error?: string;
}) => {
	const [name, setName] = useState('');
	const create = useCreateCategoryGroup();
	const reorder = useReorderCategoryGroups();

	const move = (index: number, by: -1 | 1) => {
		const ids = groups.map((g) => g.id);
		[ids[index], ids[index + by]] = [ids[index + by], ids[index]];
		reorder.mutate(ids);
	};

	return (
		<div className={styles.list}>
			<h2 className={styles.heading}>Groups</h2>
			<form
				className={styles.rename}
				aria-label='New group'
				onSubmit={(e) => {
					e.preventDefault();
					create.mutate(name.trim(), { onSuccess: () => setName('') });
				}}
			>
				<input
					aria-label='Group name'
					placeholder='New group'
					required
					value={name}
					onChange={(e) => setName(e.target.value)}
				/>
				<button
					type='submit'
					className={buttons.small}
					disabled={create.isPending}
				>
					Add
				</button>
				{create.error && (
					<p className={styles.error} role='alert'>
						{create.error.message}
					</p>
				)}
			</form>
			{error && (
				<p className={styles.error} role='alert'>
					Couldn’t load groups: {error}
				</p>
			)}
			{groups.length === 0 ? (
				<p className={styles.muted}>
					Groups like Housing or Wealth building organize your spending
					categories.
				</p>
			) : (
				<ul>
					{groups.map((g, i) => (
						<GroupItem
							key={g.id}
							group={g}
							canMoveUp={i > 0}
							canMoveDown={i < groups.length - 1}
							moving={reorder.isPending}
							onMove={(by) => move(i, by)}
						/>
					))}
				</ul>
			)}
			{reorder.error && (
				<p className={styles.error} role='alert'>
					{reorder.error.message}
				</p>
			)}
		</div>
	);
};

const GroupItem = ({
	group,
	canMoveUp,
	canMoveDown,
	moving,
	onMove,
}: {
	group: CategoryGroup;
	canMoveUp: boolean;
	canMoveDown: boolean;
	moving: boolean;
	onMove: (by: -1 | 1) => void;
}) => {
	const [renaming, setRenaming] = useState(false);
	const [name, setName] = useState(group.name);
	const update = useUpdateCategoryGroup();
	const remove = useDeleteCategoryGroup();
	const error = update.error ?? remove.error;

	if (renaming) {
		const done = () => {
			setRenaming(false);
			setName(group.name);
		};
		return (
			<li>
				<form
					className={styles.rename}
					aria-label={`Rename ${group.name}`}
					onSubmit={(e) => {
						e.preventDefault();
						if (name.trim() === group.name) return done();
						update.mutate(
							{ id: group.id, input: { name: name.trim() } },
							{ onSuccess: () => setRenaming(false) },
						);
					}}
					onKeyDown={(e) => {
						if (e.key === 'Escape') done();
					}}
				>
					<input
						aria-label='Group name'
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
						onClick={done}
					>
						Cancel
					</button>
					{update.error && (
						<p className={styles.error} role='alert'>
							{update.error.message}
						</p>
					)}
				</form>
			</li>
		);
	}

	return (
		<li>
			<div className={styles.item}>
				<span className={styles.name}>{group.name}</span>
				<span className={styles.actions}>
					<button
						type='button'
						className={cx(buttons.quiet, buttons.small)}
						aria-label={`Move ${group.name} up`}
						disabled={!canMoveUp || moving}
						onClick={() => onMove(-1)}
					>
						↑
					</button>
					<button
						type='button'
						className={cx(buttons.quiet, buttons.small)}
						aria-label={`Move ${group.name} down`}
						disabled={!canMoveDown || moving}
						onClick={() => onMove(1)}
					>
						↓
					</button>
					<button
						type='button'
						className={cx(buttons.quiet, buttons.small)}
						aria-label={`Rename ${group.name}`}
						onClick={() => setRenaming(true)}
					>
						Rename
					</button>
					<ConfirmButton
						aria-label={`Delete ${group.name}`}
						confirmLabel='Confirm'
						disabled={remove.isPending}
						onConfirm={() => remove.mutate(group.id)}
					>
						Delete
					</ConfirmButton>
				</span>
			</div>
			{error && (
				<p className={styles.error} role='alert'>
					{error.message}
				</p>
			)}
		</li>
	);
};
