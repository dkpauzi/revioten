import { describe, expect, it } from 'vitest';
import { waited } from './health.ts';

describe('waited', () => {
	it('menulis jam di bawah sehari, hari dan jam sesudahnya', () => {
		expect(waited(0.4)).toBe('0 h');
		expect(waited(5.9)).toBe('5 h');
		expect(waited(51)).toBe('2 d 3 h');
	});
});
