import { describe, expect, it } from 'vitest';
import { checkHealth } from './health.ts';

// fakeFetch membuat pengganti fetch yang selalu menjawab dengan respons tertentu,
// sekaligus mencatat alamat yang diminta.
function fakeFetch(response: Response, calls: string[] = []): typeof fetch {
	return async (input) => {
		calls.push(String(input));
		return response;
	};
}

describe('checkHealth', () => {
	it('memanggil GET /health', async () => {
		const calls: string[] = [];
		await checkHealth(fakeFetch(Response.json({ status: 'ok', database: 'ok' }), calls));
		expect(calls).toEqual(['/health']);
	});

	it('mengembalikan ok bila server dan database hidup', async () => {
		const result = await checkHealth(fakeFetch(Response.json({ status: 'ok', database: 'ok' })));
		expect(result).toBe('ok');
	});

	it('mengembalikan database_unreachable bila server hidup tapi database tidak', async () => {
		const body = { status: 'unavailable', database: 'unreachable' };
		const result = await checkHealth(fakeFetch(Response.json(body, { status: 503 })));
		expect(result).toBe('database_unreachable');
	});

	it('mengembalikan error bila jawaban 503 bukan JSON dari kontrak', async () => {
		const result = await checkHealth(fakeFetch(new Response('down', { status: 503 })));
		expect(result).toBe('error');
	});

	it('mengembalikan error bila status di JSON bukan ok', async () => {
		const result = await checkHealth(fakeFetch(Response.json({ status: 'starting' })));
		expect(result).toBe('error');
	});

	it('mengembalikan error bila server tidak bisa dihubungi', async () => {
		const failingFetch: typeof fetch = async () => {
			throw new TypeError('fetch failed');
		};
		expect(await checkHealth(failingFetch)).toBe('error');
	});
});
