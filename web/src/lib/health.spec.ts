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
		await checkHealth(fakeFetch(Response.json({ status: 'ok' }), calls));
		expect(calls).toEqual(['/health']);
	});

	it('mengembalikan ok bila server menjawab {"status":"ok"}', async () => {
		const result = await checkHealth(fakeFetch(Response.json({ status: 'ok' })));
		expect(result).toBe('ok');
	});

	it('mengembalikan error bila server menjawab 503', async () => {
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
