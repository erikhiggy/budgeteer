/** Joins truthy class names: cx(styles.row, isActive && styles.active). */
export function cx(...names: Array<string | false | null | undefined>): string {
	return names.filter(Boolean).join(' ');
}
