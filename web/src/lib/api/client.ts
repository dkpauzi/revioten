// Pemanggil API yang ditulis tangan. Bentuk data dan error diambil dari tipe
// hasil generate di gen/ (dibuat dari api/openapi.yaml), jadi bila kontrak
// berubah, TypeScript langsung menunjukkan kode yang harus ikut diubah.
//
// Cookie sesi dikirim browser secara otomatis, karena alamat API sama dengan
// alamat halaman. Kode ini tidak pernah membaca cookie itu (HttpOnly).
import type {
	AccountInfo,
	AccountList,
	ApprovalRoute,
	ApprovalRouteInput,
	ApprovalRouteList,
	NewPositionHolder,
	Position,
	PositionHolder,
	PositionList,
	SystemSettings,
	SystemSettingsUpdate,
	AuditEntryList,
	AuditVerification,
	ChangeSource,
	ChangeSourceList,
	ChangeSourceUpdate,
	ChangeType,
	ChangeTypeList,
	DocumentType,
	DocumentTypeInput,
	DocumentTypeList,
	FourMCategory,
	FourMCategoryList,
	NewSpecialCode,
	NumberSettings,
	NumberSettingsUpdate,
	NewDocumentNumber,
	NewDocumentNumberSet,
	NextDocumentNumber,
	ApRule,
	Document,
	DocumentList,
	ComposeInput,
	Composition,
	DocumentRowInput,
	NewDocument,
	FormSet,
	FormSetList,
	FormSetSheets,
	NewFormSet,
	FormatFieldUpdate,
	FormatHighlights,
	FormatSymbol,
	FormatTemplate,
	FormatTemplateList,
	FormatVersion,
	NewFormatField,
	NewFormatTemplate,
	NewSubprocess,
	NewTemplatePoint,
	ProcessTemplate,
	ProcessTemplateList,
	ProcessTemplateVersion,
	SubprocessUpdate,
	TemplatePointUpdate,
	NewPointVariant,
	PointVariantUpdate,
	ResolvedPoints,
	RatingCriterionInput,
	NumberImportReport,
	NumberImportRequest,
	NumberRegistration,
	DocumentNumber,
	DocumentNumberList,
	SpecialCode,
	SpecialCodeList,
	Error as ApiError,
	CreatedAccount,
	Credentials,
	Customer,
	CustomerList,
	CustomerUpdate,
	Model,
	ModelGroup,
	ModelGroupList,
	ModelList,
	ModelUpdate,
	NewModel,
	NewModelGroup,
	NewPartNumber,
	NewProcess,
	PartNumber,
	PartNumberList,
	PartNumberUpdate,
	Process,
	ProcessList,
	ProcessUpdate,
	ProjectGroup,
	ProjectGroupInput,
	ProjectGroupList,
	NewAccount,
	NewCustomer,
	PasswordChange,
	RoleGrant,
	RoleList,
	Session,
	TemporaryPassword
} from './gen/index.ts';

// ApiFailure menjelaskan panggilan yang gagal: status HTTP (0 bila server
// tidak bisa dihubungi) dan isi error dari kontrak, bila ada.
export type ApiFailure = { status: number; body: ApiError | null };

// ApiResult adalah hasil satu panggilan: berhasil dengan data, atau gagal.
// Dengan bentuk ini, kode layar dipaksa menangani kedua kemungkinan.
export type ApiResult<T> = { ok: true; data: T } | { ok: false; error: ApiFailure };

// onUnauthenticated dipanggil setiap kali API menjawab 401 (sesi berakhir),
// kecuali saat login. Layout mengisinya dengan "pindah ke halaman login".
let onUnauthenticated: (() => void) | null = null;

// setUnauthenticatedHandler mengganti penanganan sesi berakhir (null untuk melepas).
export function setUnauthenticatedHandler(handler: (() => void) | null): void {
	onUnauthenticated = handler;
}

// listCustomers memanggil GET /api/v1/customers.
// Parameter fetchFn bisa diganti di test; di browser nilainya fetch bawaan.
export function listCustomers(fetchFn: typeof fetch = fetch): Promise<ApiResult<CustomerList>> {
	return call<CustomerList>(fetchFn, '/api/v1/customers', { method: 'GET' });
}

// createCustomer memanggil POST /api/v1/customers.
export function createCustomer(
	input: NewCustomer,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Customer>> {
	return call<Customer>(fetchFn, '/api/v1/customers', jsonRequest('POST', input));
}

// updateCustomer memanggil PUT /api/v1/customers/{id}: nama, Project Group,
// dan status aktif (A-2).
export function updateCustomer(
	id: string,
	input: CustomerUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Customer>> {
	return call<Customer>(fetchFn, itemPath('customers', id), jsonRequest('PUT', input));
}

// changeCustomerCode memanggil POST /api/v1/customers/{id}/code. Kode lama
// tetap tercatat di riwayat kode customer (E-11).
export function changeCustomerCode(
	id: string,
	code: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Customer>> {
	return call<Customer>(fetchFn, itemPath('customers', id) + '/code', jsonRequest('POST', { code }));
}

// listProjectGroups memanggil GET /api/v1/project-groups.
export function listProjectGroups(fetchFn: typeof fetch = fetch): Promise<ApiResult<ProjectGroupList>> {
	return call<ProjectGroupList>(fetchFn, '/api/v1/project-groups', { method: 'GET' });
}

// createProjectGroup memanggil POST /api/v1/project-groups (A-2).
export function createProjectGroup(
	input: ProjectGroupInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProjectGroup>> {
	return call<ProjectGroup>(fetchFn, '/api/v1/project-groups', jsonRequest('POST', input));
}

// updateProjectGroup memanggil PUT /api/v1/project-groups/{id} (A-2).
export function updateProjectGroup(
	id: string,
	input: ProjectGroupInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProjectGroup>> {
	return call<ProjectGroup>(fetchFn, itemPath('project-groups', id), jsonRequest('PUT', input));
}

// listModelGroups memanggil GET /api/v1/model-groups?customer_id=...
export function listModelGroups(
	customerId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ModelGroupList>> {
	return call<ModelGroupList>(fetchFn, listPath('model-groups', customerId), { method: 'GET' });
}

// createModelGroup memanggil POST /api/v1/model-groups (A-2).
export function createModelGroup(
	input: NewModelGroup,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ModelGroup>> {
	return call<ModelGroup>(fetchFn, '/api/v1/model-groups', jsonRequest('POST', input));
}

// renameModelGroup memanggil PUT /api/v1/model-groups/{id} (A-2).
export function renameModelGroup(
	id: string,
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ModelGroup>> {
	return call<ModelGroup>(fetchFn, itemPath('model-groups', id), jsonRequest('PUT', { name }));
}

// listModels memanggil GET /api/v1/models?customer_id=...
export function listModels(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<ModelList>> {
	return call<ModelList>(fetchFn, listPath('models', customerId), { method: 'GET' });
}

// createModel memanggil POST /api/v1/models (A-2).
export function createModel(input: NewModel, fetchFn: typeof fetch = fetch): Promise<ApiResult<Model>> {
	return call<Model>(fetchFn, '/api/v1/models', jsonRequest('POST', input));
}

// updateModel memanggil PUT /api/v1/models/{id} (A-2). Tanpa model_group_id
// berarti model dikeluarkan dari group-nya.
export function updateModel(
	id: string,
	input: ModelUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Model>> {
	return call<Model>(fetchFn, itemPath('models', id), jsonRequest('PUT', input));
}

// listPartNumbers memanggil GET /api/v1/part-numbers?customer_id=...
export function listPartNumbers(
	customerId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<PartNumberList>> {
	return call<PartNumberList>(fetchFn, listPath('part-numbers', customerId), { method: 'GET' });
}

// createPartNumber memanggil POST /api/v1/part-numbers (A-2).
export function createPartNumber(
	input: NewPartNumber,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<PartNumber>> {
	return call<PartNumber>(fetchFn, '/api/v1/part-numbers', jsonRequest('POST', input));
}

// updatePartNumber memanggil PUT /api/v1/part-numbers/{id} (A-2).
export function updatePartNumber(
	id: string,
	input: PartNumberUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<PartNumber>> {
	return call<PartNumber>(fetchFn, itemPath('part-numbers', id), jsonRequest('PUT', input));
}

// listProcesses memanggil GET /api/v1/processes (M1.5).
export function listProcesses(fetchFn: typeof fetch = fetch): Promise<ApiResult<ProcessList>> {
	return call<ProcessList>(fetchFn, '/api/v1/processes', { method: 'GET' });
}

// createProcess memanggil POST /api/v1/processes (E-6).
export function createProcess(input: NewProcess, fetchFn: typeof fetch = fetch): Promise<ApiResult<Process>> {
	return call<Process>(fetchFn, '/api/v1/processes', jsonRequest('POST', input));
}

// updateProcess memanggil PUT /api/v1/processes/{id} (E-6). Daftar alias
// menggantikan semua alias lama, jadi selalu dikirim lengkap.
export function updateProcess(
	id: string,
	input: ProcessUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Process>> {
	return call<Process>(fetchFn, itemPath('processes', id), jsonRequest('PUT', input));
}

// getNumberSettings memanggil GET /api/v1/number-settings (M1.6).
export function getNumberSettings(fetchFn: typeof fetch = fetch): Promise<ApiResult<NumberSettings>> {
	return call<NumberSettings>(fetchFn, '/api/v1/number-settings', { method: 'GET' });
}

// updateNumberSettings memanggil PUT /api/v1/number-settings (A-2). pattern dan
// revision_digits yang tidak dikirim berarti tetap (D-111).
export function updateNumberSettings(
	input: NumberSettingsUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<NumberSettings>> {
	return call<NumberSettings>(fetchFn, '/api/v1/number-settings', jsonRequest('PUT', input));
}

// registerDocumentNumber memanggil POST /api/v1/document-numbers (E-1, M2.1).
// key adalah Idempotency-Key: dibuat sekali untuk setiap pendaftaran baru, dan
// dipakai lagi bila permintaan yang sama dikirim ulang, supaya tombol yang
// ditekan dua kali tidak menghasilkan dua nomor (R-NUM-03, M2.2).
export function registerDocumentNumber(
	input: NewDocumentNumber,
	key: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<NumberRegistration>> {
	const init = jsonRequest('POST', input);
	init.headers = { 'Content-Type': 'application/json', 'Idempotency-Key': key };
	return call<NumberRegistration>(fetchFn, '/api/v1/document-numbers', init);
}

// registerDocumentNumberSet memanggil POST /api/v1/document-numbers/sets
// (E-1, M2.4): PFD, PFMEA, dan CP dengan satu nomor berjalan. key seperti di
// registerDocumentNumber.
export function registerDocumentNumberSet(
	input: NewDocumentNumberSet,
	key: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<NumberRegistration>> {
	const init = jsonRequest('POST', input);
	init.headers = { 'Content-Type': 'application/json', 'Idempotency-Key': key };
	return call<NumberRegistration>(fetchFn, '/api/v1/document-numbers/sets', init);
}

// NumberFilter adalah saringan register (M2.3). Field kosong tidak dikirim.
export type NumberFilter = {
	customer_id?: string;
	document_type_id?: string;
	status?: string;
	q?: string;
	before?: string;
	limit?: number;
};

// listDocumentNumbers memanggil GET /api/v1/document-numbers (L-2, M2.3).
export function listDocumentNumbers(
	filter: NumberFilter,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<DocumentNumberList>> {
	const query = new URLSearchParams();
	for (const [name, value] of Object.entries(filter)) {
		if (value !== undefined && value !== '') {
			query.set(name, String(value));
		}
	}
	const text = query.toString();
	return call<DocumentNumberList>(fetchFn, '/api/v1/document-numbers' + (text ? '?' + text : ''), { method: 'GET' });
}

// voidDocumentNumber memanggil POST /api/v1/document-numbers/{id}/void (E-8).
export function voidDocumentNumber(
	id: string,
	reason: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<DocumentNumber>> {
	return call<DocumentNumber>(fetchFn, itemPath('document-numbers', id) + '/void', jsonRequest('POST', { reason }));
}

// makeDocumentNumberObsolete memanggil POST /api/v1/document-numbers/{id}/obsolete (E-8).
export function makeDocumentNumberObsolete(
	id: string,
	reason: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<DocumentNumber>> {
	return call<DocumentNumber>(fetchFn, itemPath('document-numbers', id) + '/obsolete', jsonRequest('POST', { reason }));
}

// importDocumentNumbers memanggil POST /api/v1/document-number-imports (E-8,
// M2.5): nomor lama satu customer beserta laporannya. Dengan dry_run true,
// hanya laporan (pratinjau) dan tidak ada yang disimpan.
export function importDocumentNumbers(
	input: NumberImportRequest,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<NumberImportReport>> {
	return call<NumberImportReport>(fetchFn, '/api/v1/document-number-imports', jsonRequest('POST', input));
}

// getNextDocumentNumber memanggil GET /api/v1/document-numbers/next: nomor yang
// akan didapat pendaftaran berikutnya, tanpa mendaftarkannya (L-2).
export function getNextDocumentNumber(
	customerId: string,
	documentTypeId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<NextDocumentNumber>> {
	const query = `customer_id=${encodeURIComponent(customerId)}&document_type_id=${encodeURIComponent(documentTypeId)}`;
	return call<NextDocumentNumber>(fetchFn, `/api/v1/document-numbers/next?${query}`, { method: 'GET' });
}

// listDocumentTypes memanggil GET /api/v1/document-types.
export function listDocumentTypes(fetchFn: typeof fetch = fetch): Promise<ApiResult<DocumentTypeList>> {
	return call<DocumentTypeList>(fetchFn, '/api/v1/document-types', { method: 'GET' });
}

// createDocumentType memanggil POST /api/v1/document-types (A-2).
export function createDocumentType(
	input: DocumentTypeInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<DocumentType>> {
	return call<DocumentType>(fetchFn, '/api/v1/document-types', jsonRequest('POST', input));
}

// updateDocumentType memanggil PUT /api/v1/document-types/{id} (A-2).
export function updateDocumentType(
	id: string,
	input: DocumentTypeInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<DocumentType>> {
	return call<DocumentType>(fetchFn, itemPath('document-types', id), jsonRequest('PUT', input));
}

// listSpecialCodes memanggil GET /api/v1/special-codes.
export function listSpecialCodes(fetchFn: typeof fetch = fetch): Promise<ApiResult<SpecialCodeList>> {
	return call<SpecialCodeList>(fetchFn, '/api/v1/special-codes', { method: 'GET' });
}

// createSpecialCode memanggil POST /api/v1/special-codes (A-2).
export function createSpecialCode(
	input: NewSpecialCode,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<SpecialCode>> {
	return call<SpecialCode>(fetchFn, '/api/v1/special-codes', jsonRequest('POST', input));
}

// updateSpecialCode memanggil PUT /api/v1/special-codes/{id} (A-2). Kodenya
// tidak bisa diubah, hanya namanya.
export function updateSpecialCode(
	id: string,
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<SpecialCode>> {
	return call<SpecialCode>(fetchFn, itemPath('special-codes', id), jsonRequest('PUT', { name }));
}

// listChangeTypes memanggil GET /api/v1/change-types.
export function listChangeTypes(fetchFn: typeof fetch = fetch): Promise<ApiResult<ChangeTypeList>> {
	return call<ChangeTypeList>(fetchFn, '/api/v1/change-types', { method: 'GET' });
}

// updateChangeType memanggil PUT /api/v1/change-types/{id} (A-2).
export function updateChangeType(
	id: string,
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ChangeType>> {
	return call<ChangeType>(fetchFn, itemPath('change-types', id), jsonRequest('PUT', { name }));
}

// listChangeSources memanggil GET /api/v1/change-sources.
export function listChangeSources(fetchFn: typeof fetch = fetch): Promise<ApiResult<ChangeSourceList>> {
	return call<ChangeSourceList>(fetchFn, '/api/v1/change-sources', { method: 'GET' });
}

// createChangeSource memanggil POST /api/v1/change-sources (A-2).
export function createChangeSource(
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ChangeSource>> {
	return call<ChangeSource>(fetchFn, '/api/v1/change-sources', jsonRequest('POST', { name }));
}

// updateChangeSource memanggil PUT /api/v1/change-sources/{id} (A-2).
export function updateChangeSource(
	id: string,
	input: ChangeSourceUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ChangeSource>> {
	return call<ChangeSource>(fetchFn, itemPath('change-sources', id), jsonRequest('PUT', input));
}

// listFourMCategories memanggil GET /api/v1/four-m-categories.
export function listFourMCategories(fetchFn: typeof fetch = fetch): Promise<ApiResult<FourMCategoryList>> {
	return call<FourMCategoryList>(fetchFn, '/api/v1/four-m-categories', { method: 'GET' });
}

// updateFourMCategory memanggil PUT /api/v1/four-m-categories/{id} (A-2).
export function updateFourMCategory(
	id: string,
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FourMCategory>> {
	return call<FourMCategory>(fetchFn, itemPath('four-m-categories', id), jsonRequest('PUT', { name }));
}

// listPositions memanggil GET /api/v1/positions (M1.8).
export function listPositions(fetchFn: typeof fetch = fetch): Promise<ApiResult<PositionList>> {
	return call<PositionList>(fetchFn, '/api/v1/positions', { method: 'GET' });
}

// updatePosition memanggil PUT /api/v1/positions/{id} (A-1).
export function updatePosition(
	id: string,
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Position>> {
	return call<Position>(fetchFn, itemPath('positions', id), jsonRequest('PUT', { name }));
}

// addPositionHolder memanggil POST /api/v1/positions/{id}/holders (A-1).
export function addPositionHolder(
	positionId: string,
	input: NewPositionHolder,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<PositionHolder>> {
	return call<PositionHolder>(fetchFn, `${itemPath('positions', positionId)}/holders`, jsonRequest('POST', input));
}

// removePositionHolder memanggil DELETE /api/v1/positions/{id}/holders/{holderId} (A-1).
export function removePositionHolder(
	positionId: string,
	holderId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<undefined>> {
	const url = `${itemPath('positions', positionId)}/holders/${encodeURIComponent(holderId)}`;
	return call<undefined>(fetchFn, url, { method: 'DELETE' });
}

// listFormatTemplates memanggil GET /api/v1/format-templates (M3.1).
export function listFormatTemplates(fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatTemplateList>> {
	return call<FormatTemplateList>(fetchFn, '/api/v1/format-templates', { method: 'GET' });
}

// createFormatTemplate memanggil POST /api/v1/format-templates (A-2).
export function createFormatTemplate(
	input: NewFormatTemplate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatTemplate>> {
	return call<FormatTemplate>(fetchFn, '/api/v1/format-templates', jsonRequest('POST', input));
}

// renameFormatTemplate memanggil PUT /api/v1/format-templates/{id} (A-2).
export function renameFormatTemplate(
	id: string,
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatTemplate>> {
	return call<FormatTemplate>(fetchFn, itemPath('format-templates', id), jsonRequest('PUT', { name }));
}

// getFormatVersion memanggil GET /api/v1/format-template-versions/{id}.
// Dokumen (M4.1).

// listDocuments memanggil GET /api/v1/documents?customer_id= (L-4).
export function listDocuments(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<DocumentList>> {
	return call<DocumentList>(fetchFn, `/api/v1/documents?customer_id=${encodeURIComponent(customerId)}`, { method: 'GET' });
}

// createDocument memanggil POST /api/v1/documents (E-2).
export function createDocument(input: NewDocument, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, '/api/v1/documents', jsonRequest('POST', input));
}

// getDocument memanggil GET /api/v1/documents/{id} (L-4).
export function getDocument(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id), { method: 'GET' });
}

// setDocumentHeader memanggil PUT /api/v1/documents/{id}/header (E-2).
// headerVersion adalah versi header yang dibaca (M4.3).
export function setDocumentHeader(id: string, header: Record<string, unknown>, headerVersion: number, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/header', jsonRequest('PUT', { header, header_version: headerVersion }));
}

// addDocumentRow memanggil POST /api/v1/documents/{id}/rows (E-2).
export function addDocumentRow(id: string, input: DocumentRowInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/rows', jsonRequest('POST', input));
}

// addDocumentRows memanggil POST /api/v1/documents/{id}/rows/batch (E-2, M4.4).
export function addDocumentRows(id: string, rows: DocumentRowInput[], fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/rows/batch', jsonRequest('POST', { rows }));
}

// getDocumentComposition memanggil GET /api/v1/documents/{id}/composition (L-4, M4.5).
export function getDocumentComposition(id: string, versionId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Composition>> {
	const path = itemPath('documents', id) + '/composition?process_template_version_id=' + encodeURIComponent(versionId);
	return call<Composition>(fetchFn, path, { method: 'GET' });
}

// composeDocument memanggil PUT /api/v1/documents/{id}/composition (E-2, M4.5).
export function composeDocument(id: string, input: ComposeInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/composition', jsonRequest('PUT', input));
}

// reorderDocumentRows memanggil PUT /api/v1/documents/{id}/order (E-2, M4.6).
export function reorderDocumentRows(id: string, rowIds: string[], renumber: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/order', jsonRequest('PUT', { row_ids: rowIds, renumber }));
}

// copyDocumentComposition memanggil POST /api/v1/documents/{id}/copy (E-2, M4.7).
export function copyDocumentComposition(id: string, sourceId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/copy', jsonRequest('POST', { source_document_id: sourceId }));
}

// updateDocumentRow memanggil PUT /api/v1/documents/{id}/rows/{rowId} (E-2).
export function updateDocumentRow(id: string, rowId: string, input: DocumentRowInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, `${itemPath('documents', id)}/rows/${encodeURIComponent(rowId)}`, jsonRequest('PUT', input));
}

// deleteDocumentRow memanggil DELETE /api/v1/documents/{id}/rows/{rowId} (E-2, M4.2).
export function deleteDocumentRow(id: string, rowId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, `${itemPath('documents', id)}/rows/${encodeURIComponent(rowId)}`, { method: 'DELETE' });
}

// Form set (langkah D, D-133, D-139).

// listFormSets memanggil GET /api/v1/form-sets (L-2).
export function listFormSets(fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSetList>> {
	return call<FormSetList>(fetchFn, '/api/v1/form-sets', { method: 'GET' });
}

// createFormSet memanggil POST /api/v1/form-sets (A-2).
export function createFormSet(input: NewFormSet, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, '/api/v1/form-sets', jsonRequest('POST', input));
}

// setFormSetSheets memanggil PUT /api/v1/form-set-versions/{id} (A-2, versi draft).
export function setFormSetSheets(versionId: string, sheets: FormSetSheets, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, itemPath('form-set-versions', versionId), jsonRequest('PUT', sheets));
}

// createFormSetVersion memanggil POST /api/v1/form-sets/{id}/versions (A-2).
export function createFormSetVersion(formSetId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, itemPath('form-sets', formSetId) + '/versions', { method: 'POST' });
}

// publishFormSetVersion memanggil POST /api/v1/form-set-versions/{id}/publish (V-3).
export function publishFormSetVersion(versionId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, itemPath('form-set-versions', versionId) + '/publish', { method: 'POST' });
}

// setCustomerFormSet memanggil PUT /api/v1/customers/{id}/default-form-set (A-2).
// formSetId kosong berarti customer memakai set bawaan.
export function setCustomerFormSet(customerId: string, formSetId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<void>> {
	const body = formSetId === '' ? {} : { form_set_id: formSetId };
	return call<void>(fetchFn, `${itemPath('customers', customerId)}/default-form-set`, jsonRequest('PUT', body));
}

// setFormatApRules memanggil PUT /api/v1/format-template-versions/{id}/ap-rules (A-2, versi draft).
export function setFormatApRules(versionId: string, items: ApRule[], fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', versionId) + '/ap-rules', jsonRequest('PUT', { items }));
}

// publishFormatVersion memanggil POST .../format-template-versions/{id}/publish (A-2, M3.4).
export function publishFormatVersion(versionId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatVersion>> {
	const path = itemPath('format-template-versions', versionId) + '/publish';
	return call<FormatVersion>(fetchFn, path, { method: 'POST' });
}

// createFormatVersion memanggil POST .../format-templates/{id}/versions (A-2, M3.4):
// versi draft berikutnya, salinan versi terbit terakhir.
export function createFormatVersion(templateId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatVersion>> {
	const path = itemPath('format-templates', templateId) + '/versions';
	return call<FormatVersion>(fetchFn, path, { method: 'POST' });
}

export function getFormatVersion(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', id), { method: 'GET' });
}

// updateFormatHighlights memanggil PUT /api/v1/format-template-versions/{id} (A-2).
export function updateFormatHighlights(
	id: string,
	input: FormatHighlights,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', id), jsonRequest('PUT', input));
}

// addFormatField memanggil POST /api/v1/format-template-versions/{id}/fields (A-2).
export function addFormatField(
	versionId: string,
	input: NewFormatField,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', versionId) + '/fields', jsonRequest('POST', input));
}

// updateFormatField memanggil PUT /api/v1/format-template-versions/{id}/fields/{fieldId} (A-2).
export function updateFormatField(
	versionId: string,
	fieldId: string,
	input: FormatFieldUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	const path = itemPath('format-template-versions', versionId) + '/fields/' + encodeURIComponent(fieldId);
	return call<FormatVersion>(fetchFn, path, jsonRequest('PUT', input));
}

// setFormatRatingCriteria memanggil PUT /api/v1/format-template-versions/{id}/rating-criteria (A-2).
export function setFormatRatingCriteria(
	versionId: string,
	items: RatingCriterionInput[],
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	const path = itemPath('format-template-versions', versionId) + '/rating-criteria';
	return call<FormatVersion>(fetchFn, path, jsonRequest('PUT', { items }));
}

// saveFormatSymbols memanggil PUT /api/v1/format-template-versions/{id}/symbols (A-2).
export function saveFormatSymbols(
	versionId: string,
	items: FormatSymbol[],
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', versionId) + '/symbols', jsonRequest('PUT', { items }));
}

// listProcessTemplates memanggil GET /api/v1/process-templates (M3.2).
export function listProcessTemplates(fetchFn: typeof fetch = fetch): Promise<ApiResult<ProcessTemplateList>> {
	return call<ProcessTemplateList>(fetchFn, '/api/v1/process-templates', { method: 'GET' });
}

// createProcessTemplate memanggil POST /api/v1/process-templates (E-6).
export function createProcessTemplate(processId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<ProcessTemplate>> {
	return call<ProcessTemplate>(fetchFn, '/api/v1/process-templates', jsonRequest('POST', { process_id: processId }));
}

// getProcessTemplateVersion memanggil GET /api/v1/process-template-versions/{id}.
// publishProcessTemplateVersion memanggil POST .../process-template-versions/{id}/publish (E-6, M3.4).
export function publishProcessTemplateVersion(
	versionId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	const path = itemPath('process-template-versions', versionId) + '/publish';
	return call<ProcessTemplateVersion>(fetchFn, path, { method: 'POST' });
}

// createProcessTemplateVersion memanggil POST .../process-templates/{id}/versions (E-6, M3.4):
// versi draft berikutnya, salinan versi terbit terakhir.
export function createProcessTemplateVersion(
	templateId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	const path = itemPath('process-templates', templateId) + '/versions';
	return call<ProcessTemplateVersion>(fetchFn, path, { method: 'POST' });
}

export function getProcessTemplateVersion(
	id: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	return call<ProcessTemplateVersion>(fetchFn, itemPath('process-template-versions', id), { method: 'GET' });
}

// addSubprocess memanggil POST /api/v1/process-template-versions/{id}/subprocesses (E-6).
export function addSubprocess(
	versionId: string,
	input: NewSubprocess,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	const path = itemPath('process-template-versions', versionId) + '/subprocesses';
	return call<ProcessTemplateVersion>(fetchFn, path, jsonRequest('POST', input));
}

// updateSubprocess memanggil PUT /api/v1/process-template-versions/{id}/subprocesses/{subprocessId} (E-6).
// Isian yang tidak dikirim tetap (D-111).
export function updateSubprocess(
	versionId: string,
	subprocessId: string,
	input: SubprocessUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	const path = itemPath('process-template-versions', versionId) + '/subprocesses/' + encodeURIComponent(subprocessId);
	return call<ProcessTemplateVersion>(fetchFn, path, jsonRequest('PUT', input));
}

// addTemplatePoint memanggil POST /api/v1/process-template-versions/{id}/points (E-6).
export function addTemplatePoint(
	versionId: string,
	input: NewTemplatePoint,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	const path = itemPath('process-template-versions', versionId) + '/points';
	return call<ProcessTemplateVersion>(fetchFn, path, jsonRequest('POST', input));
}

// updateTemplatePoint memanggil PUT /api/v1/process-template-versions/{id}/points/{pointId} (E-6).
// Isian yang tidak dikirim tetap (D-111).
export function updateTemplatePoint(
	versionId: string,
	pointId: string,
	input: TemplatePointUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	const path = itemPath('process-template-versions', versionId) + '/points/' + encodeURIComponent(pointId);
	return call<ProcessTemplateVersion>(fetchFn, path, jsonRequest('PUT', input));
}

// addPointVariant memanggil POST /api/v1/process-template-versions/{id}/points/{pointId}/variants (E-6, M3.3).
export function addPointVariant(
	versionId: string,
	pointId: string,
	input: NewPointVariant,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	const path = itemPath('process-template-versions', versionId) + '/points/' + encodeURIComponent(pointId) + '/variants';
	return call<ProcessTemplateVersion>(fetchFn, path, jsonRequest('POST', input));
}

// updatePointVariant memanggil PUT /api/v1/process-template-versions/{id}/variants/{variantId} (E-6, M3.3).
// Isian yang tidak dikirim tetap (D-111).
export function updatePointVariant(
	versionId: string,
	variantId: string,
	input: PointVariantUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProcessTemplateVersion>> {
	const path = itemPath('process-template-versions', versionId) + '/variants/' + encodeURIComponent(variantId);
	return call<ProcessTemplateVersion>(fetchFn, path, jsonRequest('PUT', input));
}

// resolveProcessTemplatePoints memanggil GET /api/v1/process-template-versions/{id}/resolved?model_id=
// untuk pratinjau poin yang berlaku bagi satu model (M3.3).
export function resolveProcessTemplatePoints(
	versionId: string,
	modelId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ResolvedPoints>> {
	const path = itemPath('process-template-versions', versionId) + '/resolved?model_id=' + encodeURIComponent(modelId);
	return call<ResolvedPoints>(fetchFn, path, { method: 'GET' });
}

// listApprovalRoutes memanggil GET /api/v1/approval-routes.
export function listApprovalRoutes(fetchFn: typeof fetch = fetch): Promise<ApiResult<ApprovalRouteList>> {
	return call<ApprovalRouteList>(fetchFn, '/api/v1/approval-routes', { method: 'GET' });
}

// createApprovalRoute memanggil POST /api/v1/approval-routes (A-2).
export function createApprovalRoute(
	input: ApprovalRouteInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ApprovalRoute>> {
	return call<ApprovalRoute>(fetchFn, '/api/v1/approval-routes', jsonRequest('POST', input));
}

// updateApprovalRoute memanggil PUT /api/v1/approval-routes/{id} (A-2).
// Daftar slot menggantikan semua slot lama.
export function updateApprovalRoute(
	id: string,
	input: ApprovalRouteInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ApprovalRoute>> {
	return call<ApprovalRoute>(fetchFn, itemPath('approval-routes', id), jsonRequest('PUT', input));
}

// setCustomerDefaultRoute memanggil PUT /api/v1/customers/{id}/default-route (A-2).
export function setCustomerDefaultRoute(
	customerId: string,
	routeId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Customer>> {
	const url = `${itemPath('customers', customerId)}/default-route`;
	return call<Customer>(fetchFn, url, jsonRequest('PUT', { route_id: routeId }));
}

// setChangeTypeRoute memanggil PUT /api/v1/change-types/{id}/approval-route
// (A-2). null berarti rute default setiap customer.
export function setChangeTypeRoute(
	changeTypeId: string,
	routeId: string | null,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ChangeType>> {
	const url = `${itemPath('change-types', changeTypeId)}/approval-route`;
	return call<ChangeType>(fetchFn, url, jsonRequest('PUT', routeId === null ? {} : { route_id: routeId }));
}

// getSystemSettings memanggil GET /api/v1/system-settings.
export function getSystemSettings(fetchFn: typeof fetch = fetch): Promise<ApiResult<SystemSettings>> {
	return call<SystemSettings>(fetchFn, '/api/v1/system-settings', { method: 'GET' });
}

// updateSystemSettings memanggil PUT /api/v1/system-settings (A-3).
export function updateSystemSettings(
	input: SystemSettingsUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<SystemSettings>> {
	return call<SystemSettings>(fetchFn, '/api/v1/system-settings', jsonRequest('PUT', input));
}

// getSession memanggil GET /api/v1/session: siapa yang sedang login.
export function getSession(fetchFn: typeof fetch = fetch): Promise<ApiResult<Session>> {
	return call<Session>(fetchFn, '/api/v1/session', { method: 'GET' });
}

// login memanggil POST /api/v1/session. Bila berhasil, server memasang cookie sesi.
export function login(
	credentials: Credentials,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Session>> {
	// 401 di sini berarti nama login atau password salah, bukan sesi berakhir.
	return call<Session>(fetchFn, '/api/v1/session', jsonRequest('POST', credentials), false);
}

// logout memanggil DELETE /api/v1/session. Server menghapus sesi dan cookie-nya.
export function logout(fetchFn: typeof fetch = fetch): Promise<ApiResult<undefined>> {
	return call<undefined>(fetchFn, '/api/v1/session', { method: 'DELETE' });
}

// changePassword memanggil PUT /api/v1/account/password. Server memasang
// cookie sesi baru, dan sesi lain milik akun ini berakhir.
export function changePassword(
	input: PasswordChange,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<undefined>> {
	return call<undefined>(fetchFn, '/api/v1/account/password', jsonRequest('PUT', input));
}

// listRoles memanggil GET /api/v1/roles: katalog peran (A-1).
export function listRoles(fetchFn: typeof fetch = fetch): Promise<ApiResult<RoleList>> {
	return call<RoleList>(fetchFn, '/api/v1/roles', { method: 'GET' });
}

// listAccounts memanggil GET /api/v1/accounts (A-1).
export function listAccounts(fetchFn: typeof fetch = fetch): Promise<ApiResult<AccountList>> {
	return call<AccountList>(fetchFn, '/api/v1/accounts', { method: 'GET' });
}

// createAccount memanggil POST /api/v1/accounts. Jawabannya memuat password
// sementara, yang hanya dikirim sekali.
export function createAccount(
	input: NewAccount,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<CreatedAccount>> {
	return call<CreatedAccount>(fetchFn, '/api/v1/accounts', jsonRequest('POST', input));
}

// setAccountRoles memanggil PUT /api/v1/accounts/{id}/roles.
export function setAccountRoles(
	accountId: string,
	roles: RoleGrant[],
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<AccountInfo>> {
	return call<AccountInfo>(fetchFn, accountPath(accountId, 'roles'), jsonRequest('PUT', { roles }));
}

// resetAccountPassword memanggil POST /api/v1/accounts/{id}/password-reset.
export function resetAccountPassword(
	accountId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<TemporaryPassword>> {
	return call<TemporaryPassword>(fetchFn, accountPath(accountId, 'password-reset'), { method: 'POST' });
}

// disableAccount memanggil POST /api/v1/accounts/{id}/disable.
export function disableAccount(
	accountId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<AccountInfo>> {
	return call<AccountInfo>(fetchFn, accountPath(accountId, 'disable'), { method: 'POST' });
}

// enableAccount memanggil POST /api/v1/accounts/{id}/enable.
export function enableAccount(
	accountId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<AccountInfo>> {
	return call<AccountInfo>(fetchFn, accountPath(accountId, 'enable'), { method: 'POST' });
}

// listAuditLog memanggil GET /api/v1/audit-log: entri terbaru dulu (A-4).
// Untuk entri yang lebih lama, isi before dengan next_before dari jawaban sebelumnya.
export function listAuditLog(
	before?: number,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<AuditEntryList>> {
	const query = before === undefined ? '' : `?before=${before}`;
	return call<AuditEntryList>(fetchFn, `/api/v1/audit-log${query}`, { method: 'GET' });
}

// verifyAuditLog memanggil GET /api/v1/audit-log/verification: server memeriksa
// seluruh rantai hash audit log (A-4).
export function verifyAuditLog(fetchFn: typeof fetch = fetch): Promise<ApiResult<AuditVerification>> {
	return call<AuditVerification>(fetchFn, '/api/v1/audit-log/verification', { method: 'GET' });
}

// accountPath menyusun alamat operasi satu akun. ID dikodekan, supaya karakter
// seperti / atau ? tidak mengubah alamat.
function accountPath(accountId: string, action: string): string {
	return `/api/v1/accounts/${encodeURIComponent(accountId)}/${action}`;
}

// itemPath menyusun alamat satu data, misalnya /api/v1/models/{id}, dengan ID dikodekan.
function itemPath(collection: string, id: string): string {
	return `/api/v1/${collection}/${encodeURIComponent(id)}`;
}

// listPath menyusun alamat daftar milik satu customer, dengan ID dikodekan.
function listPath(collection: string, customerId: string): string {
	return `/api/v1/${collection}?customer_id=${encodeURIComponent(customerId)}`;
}

// jsonRequest menyusun permintaan dengan isi JSON.
function jsonRequest(method: string, body: unknown): RequestInit {
	return { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) };
}

// call menjalankan satu permintaan dan membaca jawabannya sebagai JSON.
async function call<T>(
	fetchFn: typeof fetch,
	url: string,
	init: RequestInit,
	notifyUnauthenticated = true
): Promise<ApiResult<T>> {
	let response: Response;
	try {
		response = await fetchFn(url, init);
	} catch {
		// fetch gagal sama sekali, misalnya server Go tidak berjalan.
		return { ok: false, error: { status: 0, body: null } };
	}

	// 204 No Content: berhasil tanpa isi.
	if (response.status === 204) {
		return { ok: true, data: undefined as T };
	}
	if (response.status === 401 && notifyUnauthenticated) {
		onUnauthenticated?.();
	}

	// Isi yang bukan JSON (misalnya halaman HTML) dibaca sebagai null.
	const body: unknown = await response.json().catch(() => null);
	if (response.ok && body !== null) {
		// Server dan klien memakai kontrak yang sama, jadi bentuknya dipercaya.
		return { ok: true, data: body as T };
	}
	return { ok: false, error: { status: response.status, body: isApiError(body) ? body : null } };
}

// isApiError memeriksa bahwa isi jawaban berbentuk Error dari kontrak.
function isApiError(body: unknown): body is ApiError {
	return (
		typeof body === 'object' &&
		body !== null &&
		'code' in body &&
		typeof body.code === 'string' &&
		'request_id' in body &&
		typeof body.request_id === 'string'
	);
}
