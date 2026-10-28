// Bantuan untuk layar rute approval (M1.8): nama tahap, jabatan per slot, dan
// isian formulir ubah rute. Aturan bentuk rute tetap diperiksa server.
import type { ApprovalRoute, ApprovalRouteInput, RouteSlot, RouteSlotInput } from './api/gen/index.ts';
import { t, type MessageKey } from './i18n/index.ts';

const stageKeys: Record<string, MessageKey> = {
	prepared: 'stagePrepared',
	checked: 'stageChecked',
	approved: 'stageApproved'
};

// stageLabel memberi nama tampilan tahap tanda tangan (D-45).
export function stageLabel(stage: string): string {
	const key = stageKeys[stage];
	return key ? t(key) : stage;
}

// positionNames menulis jabatan yang boleh mengisi satu slot.
export function positionNames(slot: RouteSlot): string {
	return slot.positions.map((p) => p.name).join(', ');
}

// routeInputOf mengubah rute yang tampil menjadi isian formulir ubah.
export function routeInputOf(route: ApprovalRoute): ApprovalRouteInput {
	return {
		name: route.name,
		slots: route.slots.map((s) => ({
			stage: s.stage,
			print_title: s.print_title,
			position_ids: s.positions.map((p) => p.id)
		}))
	};
}

// moveSlot memindah slot di index sejauh delta (-1 ke atas, 1 ke bawah) dan
// mengembalikan daftar baru. Slot di ujung tidak bergerak.
export function moveSlot(slots: RouteSlotInput[], index: number, delta: number): RouteSlotInput[] {
	const target = index + delta;
	if (target < 0 || target >= slots.length) {
		return slots;
	}
	const moved = [...slots];
	[moved[index], moved[target]] = [moved[target], moved[index]];
	return moved;
}
