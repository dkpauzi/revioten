// Menyiapkan server untuk test E2E (dipanggil Playwright lewat webServer):
//   1. membuat ulang database revioten_e2e di container PostgreSQL Docker
//      Compose, supaya setiap putaran test mulai dari database kosong;
//   2. membangun binary Go (yang sudah berisi hasil vite build);
//   3. menjalankan binary itu di port 8090 dengan database revioten_e2e.
// Syarat: docker compose up -d sudah dijalankan dan file .env sudah diisi.
import { execFileSync, spawn } from 'node:child_process';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = fileURLToPath(new URL('../..', import.meta.url));

// psql dijalankan di dalam container, jadi tidak butuh password dari .env.
// Variabel $POSTGRES_USER dibaca shell di dalam container.
const resetSQL = 'DROP DATABASE IF EXISTS revioten_e2e WITH (FORCE);\nCREATE DATABASE revioten_e2e;\n';
execFileSync(
	'docker',
	['compose', 'exec', '-T', 'db', 'sh', '-c', 'psql -v ON_ERROR_STOP=1 -q -U "$POSTGRES_USER" -d postgres'],
	{ cwd: root, input: resetSQL, stdio: ['pipe', 'inherit', 'inherit'] }
);

const binary = join(root, 'bin', process.platform === 'win32' ? 'revioten-e2e.exe' : 'revioten-e2e');
execFileSync('go', ['build', '-o', binary, './cmd/revioten'], { cwd: root, stdio: 'inherit' });

// POSTGRES_DB di environment menang atas isi .env (lihat cmd/revioten/config.go).
const server = spawn(binary, ['-addr', '127.0.0.1:8090'], {
	cwd: root,
	env: { ...process.env, POSTGRES_DB: 'revioten_e2e' },
	stdio: 'inherit'
});
server.on('exit', (code) => process.exit(code ?? 0));

// Sinyal berhenti dari Playwright diteruskan ke server Go.
/** @type {NodeJS.Signals[]} */
const signals = ['SIGINT', 'SIGTERM'];
for (const signal of signals) {
	process.on(signal, () => server.kill(signal));
}
