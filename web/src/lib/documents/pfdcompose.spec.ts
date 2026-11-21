// Test rakit dari PFD (C4, D-211).
import { describe, expect, it } from 'vitest';
import type { PfdComposition } from '#lib/api/gen/index.ts';
import { chosenSteps, initialChoice, stepKey, warnings } from './pfdcompose.ts';

const step = (operation: string, process: string, pfmea: string, cp: string) => ({
	section_key: '',
	operation,
	process_id: 'p-' + process,
	process_name: process,
	symbol: 'process',
	ng: '',
	template_version_id: '',
	template_version: 0,
	targets: [
		{ document_id: 'm', document_type_key: 'pfmea', points: 1, state: pfmea },
		{ document_id: 'c', document_type_key: 'cp', points: 1, state: cp }
	]
});

const view = {
	pfd_id: 'pfd',
	documents: [],
	steps: [step('1', 'Part Receiving', 'present', 'present'), step('2', 'IQC', 'to_add', 'no_template'), step('3', 'SMT', 'not_draft', 'present')],
	extra: [{ document_id: 'c', document_type_key: 'cp', section_key: '', operation: '99', process_name: 'Packing' }]
} as PfdComposition;

describe('rakit dari PFD', () => {
	it('mencentang langkah yang belum ada', () => {
		const chosen = initialChoice(view);
		expect(chosen[stepKey(view.steps[0])]).toBe(false);
		expect(chosen[stepKey(view.steps[1])]).toBe(true);
		expect(chosen[stepKey(view.steps[2])]).toBe(false);
	});

	it('hanya mengirim langkah yang dicentang dan masih terbuka', () => {
		const chosen = { ...initialChoice(view), [stepKey(view.steps[0])]: true };
		expect(chosenSteps(view, chosen)).toEqual([{ section_key: '', operation: '2', process_id: 'p-IQC' }]);
	});

	it('memeriksa langkah yang hilang dan yang tidak ada di PFD', () => {
		expect(warnings(view).map((w) => `${w.kind} ${w.documentType} ${w.operation}`)).toEqual([
			'missing pfmea 2',
			'missing cp 2',
			'missing pfmea 3',
			'extra cp 99'
		]);
	});
});
