import { afterEach, describe, expect, it, vi } from 'vitest';
import { copyText } from './clipboard.ts';

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('copyText', () => {
	it('menyalin teks dan melaporkan berhasil', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		vi.stubGlobal('navigator', { clipboard: { writeText } });
		expect(await copyText('rahasia sementara')).toBe(true);
		expect(writeText).toHaveBeenCalledWith('rahasia sementara');
	});

	it('melaporkan gagal bila browser menolak', async () => {
		vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('ditolak')) } });
		expect(await copyText('x')).toBe(false);
	});

	it('melaporkan gagal bila clipboard tidak tersedia', async () => {
		vi.stubGlobal('navigator', {});
		expect(await copyText('x')).toBe(false);
	});
});
