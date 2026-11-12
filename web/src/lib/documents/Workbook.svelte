<script lang="ts">
	// Tabel baris satu sheet workbook (M4.2, D-145), memakai Tabulator: sel
	// diubah langsung di tabel seperti Excel, dan setiap perubahan disimpan ke
	// server sebagai satu baris utuh. Kolom terisi otomatis (D-122, D-123)
	// diubah lewat kolom nomor operasi dan proses milik baris; kolom yang
	// dihitung (RPN, AP) hanya dibaca.
	import { onDestroy, onMount } from 'svelte';
	import { TabulatorFull as Tabulator, type CellComponent, type ColumnDefinition } from 'tabulator-tables';
	import 'tabulator-tables/dist/css/tabulator_simple.min.css';
	import type { Document, DocumentRow, DocumentRowInput, FormatVersion, Process } from '#lib/api/gen/index.ts';
	import { cellText, gridRow, rowInputFrom } from '#lib/documents.ts';
	import { t } from '#lib/i18n/index.ts';

	let {
		doc,
		format,
		processes,
		canEdit,
		onsave,
		ondelete
	}: {
		doc: Document;
		format: FormatVersion;
		processes: Process[];
		canEdit: boolean;
		// onsave menyimpan satu baris. Jawabannya baris yang tersimpan, 'reloaded'
		// bila halaman memuat ulang isi terbaru karena konflik (M4.3), atau null
		// bila server menolaknya.
		onsave: (rowId: string, input: DocumentRowInput) => Promise<DocumentRow | 'reloaded' | null>;
		ondelete: (rowId: string) => void;
	} = $props();

	let element: HTMLDivElement;
	let table: Tabulator | null = null;
	// Antrean simpan: penyimpanan berjalan satu per satu, supaya perubahan
	// kedua pada baris yang sama memakai versi hasil penyimpanan pertama.
	let queue: Promise<void> = Promise.resolve();
	// built: Tabulator selesai dibangun. Sebelum itu isi tabel tidak boleh
	// diganti; pending menyimpan isi terbaru yang datang terlalu awal.
	let built = false;
	let pending: ReturnType<typeof gridRow>[] | null = null;
	// shown: daftar baris dari server yang sedang tampil di tabel. Tabel hanya
	// diisi ulang bila daftarnya berganti, karena mengisi ulang membatalkan sel
	// yang sedang diketik.
	let shown: Document['rows'] | null = null;
	// Lebar tabel bisa berubah saat sel dibuka (scrollbar halaman muncul).
	// Menggambar ulang saat itu membuang sel yang sedang diketik, jadi gambar
	// ulang otomatis Tabulator dimatikan dan diurus di sini: ditunda selama
	// ada sel yang diedit.
	let editing = false;
	let redrawWaiting = false;
	let lastWidth = 0;
	let observer: ResizeObserver | null = null;

	function redrawWhenIdle() {
		if (editing) {
			redrawWaiting = true;
			return;
		}
		redrawWaiting = false;
		if (built) {
			table?.redraw();
		}
	}

	function editEnded() {
		editing = false;
		if (redrawWaiting) {
			redrawWhenIdle();
		}
	}

	const typed = ['text', 'date', 'select', 'symbol', 'rating'];

	// columns menyusun kolom Tabulator dari versi format.
	function columns(): ColumnDefinition[] {
		const processNames = Object.fromEntries(processes.map((p) => [p.id, p.name]));
		const cols: ColumnDefinition[] = [{ title: '#', field: 'position', width: 52, headerSort: false, frozen: true }];
		for (const f of format.fields.filter((x) => x.area === 'row' && !x.hidden)) {
			const col: ColumnDefinition = { title: f.label, field: f.id, headerSort: false, minWidth: 120, formatter: 'textarea' };
			if (f.kind === 'operation_number') {
				col.field = '_operation';
				if (canEdit) col.editor = 'input';
			} else if (f.kind === 'process') {
				col.field = '_process';
				col.formatter = (cell: CellComponent) => processNames[cell.getValue() as string] ?? '';
				if (canEdit) col.editor = 'list';
				col.editorParams = { values: processNames, clearable: true };
			} else if (typed.includes(f.kind) && canEdit) {
				if (f.kind === 'rating') {
					col.editor = 'number';
					col.editorParams = { min: 1, max: 10, step: 1 };
					col.width = 64;
				} else if (f.kind === 'select' || f.kind === 'symbol') {
					const options = f.kind === 'select' ? (f.options ?? []).map((o) => [o.key, o.label]) : format.symbols.filter((s) => !s.automatic).map((s) => [s.key, s.name]);
					col.editor = 'list';
					col.editorParams = { values: Object.fromEntries(options), clearable: true };
					col.formatter = (cell: CellComponent) => cellText(f, { content: { [f.id]: cell.getValue() } });
				} else if (f.kind === 'date') {
					col.editor = 'date';
				} else {
					col.editor = 'textarea';
				}
			} else if (!typed.includes(f.kind)) {
				// Kolom yang dihitung atau terisi dari PFD: hanya dibaca (D-122).
				col.cssClass = 'readonly';
			}
			cols.push(col);
		}
		if (canEdit) {
			cols.push({
				title: '',
				field: '_actions',
				headerSort: false,
				width: 96,
				formatter: () => `<button type="button" class="row-delete">${t('documentRowDelete')}</button>`,
				cellClick: (event: UIEvent, cell: CellComponent) => {
					if ((event.target as HTMLElement).closest('.row-delete')) {
						ondelete(cell.getRow().getData().id as string);
					}
				}
			});
		}
		return cols;
	}

	onMount(() => {
		shown = doc.rows;
		table = new Tabulator(element, {
			data: doc.rows.map(gridRow),
			columns: columns(),
			layout: 'fitDataStretch',
			index: 'id',
			height: 'auto',
			autoResize: false,
			placeholder: t('documentRowsEmpty')
		});
		// Setiap sel yang berubah menyimpan barisnya. Bila server menolak, sel
		// kembali ke nilai lama, dan pesannya tampil di halaman.
		table.on('tableBuilt', () => {
			built = true;
			if (pending) {
				table?.replaceData(pending);
				pending = null;
			}
		});
		table.on('cellEditing', () => {
			editing = true;
		});
		table.on('cellEditCancelled', editEnded);
		lastWidth = element.clientWidth;
		observer = new ResizeObserver(() => {
			if (element.clientWidth !== lastWidth) {
				lastWidth = element.clientWidth;
				redrawWhenIdle();
			}
		});
		observer.observe(element);
		table.on('cellEdited', (cell: CellComponent) => {
			editEnded();
			// catch: satu penyimpanan yang gagal tidak menghentikan antrean.
			queue = queue.catch(() => {}).then(async () => {
				const row = cell.getRow();
				const data = row.getData();
				const saved = await onsave(data.id as string, rowInputFrom(format, data));
				if (saved === 'reloaded') {
					return; // isi tabel sudah diganti dengan isi terbaru dari server
				}
				if (saved === null) {
					cell.restoreOldValue();
					return;
				}
				// Versi baru dicatat di baris, untuk penyimpanan berikutnya (R-DOC-07).
				await row.update({ _version: saved.version });
			});
		});
	});

	onDestroy(() => {
		observer?.disconnect();
		table?.destroy();
	});

	// Isi tabel mengikuti dokumen terbaru dari server (sesudah tambah atau hapus baris).
	$effect(() => {
		const source = doc.rows;
		if (!table || source === shown) {
			return; // tabel belum dibuat, atau isinya sudah tampil
		}
		shown = source;
		const rows = source.map(gridRow);
		if (built) {
			table.replaceData(rows);
		} else {
			pending = rows;
		}
	});
</script>

<div class="workbook" bind:this={element}></div>

<style>
	.workbook {
		font-size: 14px;
	}

	.workbook :global(.tabulator-cell.readonly) {
		background: var(--surface-2);
		color: var(--text-3);
	}

	.workbook :global(.row-delete) {
		min-height: 28px;
		padding: 0 8px;
		font-size: 13px;
	}
</style>
