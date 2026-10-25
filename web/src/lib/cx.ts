/** Joins truthy class names: cx(styles.row, isActive && styles.active). */
export const cx = (
	...names: Array<string | false | null | undefined>
): string => {
	return names.filter(Boolean).join(' ');
};
