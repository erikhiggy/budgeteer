import { describe, expect, it } from 'vitest';
import { cx } from './cx';

describe('cx', () => {
	it('joins truthy names and skips falsy ones', () => {
		expect(cx('a', false, 'b', null, undefined, '')).toBe('a b');
	});

	it('returns an empty string when nothing is truthy', () => {
		expect(cx(false, null)).toBe('');
	});
});
