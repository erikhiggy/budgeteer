import { describe, expect, it } from 'vitest';
import type { Payee } from '../api/types';
import { matchPayees } from './PayeeCombobox';

const p = (payee: string): Payee => ({ payee, category_id: null });

describe('matchPayees', () => {
	const payees = ['Cafe Rio', 'Blue Bottle Cafe', 'Costco', 'cafe'].map(p);

	it('puts names that start with the text before names that contain it', () => {
		expect(matchPayees(payees, 'caf').map((x) => x.payee)).toEqual([
			'Cafe Rio',
			'cafe',
			'Blue Bottle Cafe',
		]);
	});

	it('ignores case and surrounding spaces', () => {
		expect(matchPayees(payees, '  COST ').map((x) => x.payee)).toEqual([
			'Costco',
		]);
	});

	it('suggests nothing for empty text or a name typed out exactly', () => {
		expect(matchPayees(payees, '  ')).toEqual([]);
		expect(matchPayees(payees, 'Costco')).toEqual([]);
	});

	it('shows at most eight', () => {
		const many = Array.from({ length: 20 }, (_, i) => p(`Shop ${i}`));
		expect(matchPayees(many, 'shop')).toHaveLength(8);
	});
});
