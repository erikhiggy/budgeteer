import type { Ref } from 'react';
import type { Category } from '../api/types';

type Props = {
	categories: Category[];
	/** A category id as a string, or '' for none selected. */
	value: string;
	onChange: (value: string) => void;
	invalid?: boolean;
	className?: string;
	ref?: Ref<HTMLSelectElement>;
};

/** Native select (keeps type-to-jump and Tab) grouped into expense and income. */
export function CategorySelect({
	categories,
	value,
	onChange,
	invalid,
	className,
	ref,
}: Props) {
	const expense = categories.filter((c) => !c.is_income);
	const income = categories.filter((c) => c.is_income);

	return (
		<select
			ref={ref}
			aria-label='Category'
			aria-invalid={invalid || undefined}
			className={className}
			value={value}
			onChange={(e) => onChange(e.target.value)}
		>
			<option value=''>Category…</option>
			{expense.length > 0 && (
				<optgroup label='Spending'>
					{expense.map((c) => (
						<option key={c.id} value={c.id}>
							{c.name}
						</option>
					))}
				</optgroup>
			)}
			{income.length > 0 && (
				<optgroup label='Income'>
					{income.map((c) => (
						<option key={c.id} value={c.id}>
							{c.name}
						</option>
					))}
				</optgroup>
			)}
		</select>
	);
}
