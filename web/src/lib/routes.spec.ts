import { describe, expect, it } from 'vitest';
import type { ApprovalRoute } from './api/gen/index.ts';
import { en } from './i18n/en.ts';
import { moveSlot, positionNames, routeInputOf, stageLabel } from './routes.ts';

const route: ApprovalRoute = {
	id: 'r-1',
	name: 'Up to GM',
	created_at: '2026-10-28T00:00:00Z',
	slots: [
		{
			stage: 'prepared',
			print_title: 'Process. Eng',
			positions: [
				{ id: 'pos-1', name: 'PPAP Engineer' },
				{ id: 'pos-2', name: 'PPAP Staff' }
			]
		},
		{ stage: 'checked', print_title: 'Engineer', positions: [{ id: 'pos-3', name: 'Project Engineer' }] },
		{ stage: 'approved', print_title: 'NPI Manager', positions: [{ id: 'pos-6', name: 'NPI Manager' }] }
	]
};

describe('rute approval di layar', () => {
	it('menamai tahap tanda tangan', () => {
		expect(stageLabel('prepared')).toBe(en.stagePrepared);
		expect(stageLabel('checked')).toBe(en.stageChecked);
		expect(stageLabel('approved')).toBe(en.stageApproved);
	});

	it('menulis jabatan satu slot', () => {
		expect(positionNames(route.slots[0])).toBe('PPAP Engineer, PPAP Staff');
	});

	// Isian formulir ubah dibuat dari rute yang tampil, lalu dikirim apa adanya.
	it('mengubah rute menjadi isian formulir', () => {
		expect(routeInputOf(route)).toEqual({
			name: 'Up to GM',
			slots: [
				{ stage: 'prepared', print_title: 'Process. Eng', position_ids: ['pos-1', 'pos-2'] },
				{ stage: 'checked', print_title: 'Engineer', position_ids: ['pos-3'] },
				{ stage: 'approved', print_title: 'NPI Manager', position_ids: ['pos-6'] }
			]
		});
	});

	it('memindah slot ke atas atau ke bawah, dan diam di ujung', () => {
		const slots = routeInputOf(route).slots;
		expect(moveSlot(slots, 2, -1).map((s) => s.print_title)).toEqual(['Process. Eng', 'NPI Manager', 'Engineer']);
		expect(moveSlot(slots, 0, -1)).toEqual(slots);
		expect(moveSlot(slots, 2, 1)).toEqual(slots);
		// Daftar asli tidak berubah.
		expect(slots.map((s) => s.print_title)).toEqual(['Process. Eng', 'Engineer', 'NPI Manager']);
	});
});
