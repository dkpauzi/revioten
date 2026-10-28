import { afterEach, describe, expect, it } from 'vitest';
import { can, session } from './session.svelte.ts';

describe('can', () => {
	afterEach(() => {
		session.current = null;
	});

	it('melaporkan hak yang dimiliki sesi', () => {
		session.current = {
			account: { id: 'a', login: 'admin.uji', kind: 'admin' },
			person: { id: 'p', name: 'Admin Uji' },
			must_change_password: false,
			permissions: ['L-1', 'A-1', 'A-2']
		};
		expect(can('A-2')).toBe(true);
		expect(can('E-2')).toBe(false);
	});

	it('tanpa sesi tidak punya hak apa pun', () => {
		expect(can('L-1')).toBe(false);
	});
});
