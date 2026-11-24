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
	PagePreview,
	PersonInfoList,
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
	PfdComposition,
	PfdStepKey,
	DocumentRowInput,
	RowDifferenceList,
	Change,
	ChangeList,
	ChangeVerificationInput,
	ChangeImpactsInput,
	YokotenList,
	PrintRequest,
	PrintLogList,
	SignedScan,
	RevisionVerification,
	PrintLayout,
	PrintLayoutList,
	PrintLayoutInput,
	CustomerPrintLayoutList,
	SqamChangeInput,
	AccountImport,
	MasterDataImport,
	LogoChangeInput,
	PendingLogoChange,
	Yokoten,
	NewYokoten,
	YokotenSearch,
	YokotenTargetList,
	YokotenDecision,
	PropagationPreview,
	PropagationInput,
	DocumentTemplateList,
	DocumentTemplateModeInput,
	DocumentRevisionList,
	DocumentDiff,
	NewChange,
	NewDocumentSection,
	DocumentSectionUpdate,
	NewReferencedDocument,
	NewReferencedDocumentLink,
	ReferencedDocument,
	ReferencedDocumentList,
	ReferencedDocumentUpdate,
	PfmeaCpComparison,
	NewDocument,
	Company,
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
	SubmissionControl,
	SubmissionEntry,
	AdminHealth,
	WaitingSignatureList,
	ErrorReportList,
	ProblemReportList,
	CreatedId,
	SignaturePlan,
	SignatureRequest,
	SignatureRequestList,
	SignatureOverview,
	SignatureSend,
	NewRole,
	RoleChange,
	Session,
	TemporaryPassword,
	ExcelComparison,
	HistoryEntry,
	LegacyDocument,
	LinkSuggestionList,
	Metrics,
	MigratableNumberList,
	MigratedSet,
	MigrationOverview,
	LegacySet,
	RowLink,
	PermissionList,
	Role,
	IssuedServiceToken,
	NewServiceAccount,
	ServiceAccount,
	ServiceAccountList,
	EmailAddress,
	NotificationOverview,
	NewPeriodicReview,
	Performance,
	PeriodicReview,
	PeriodicReviewList,
	ReviewSetList,
	Inbox,
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
// dan status aktif (E-11).
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

// createProjectGroup memanggil POST /api/v1/project-groups (E-11).
export function createProjectGroup(
	input: ProjectGroupInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ProjectGroup>> {
	return call<ProjectGroup>(fetchFn, '/api/v1/project-groups', jsonRequest('POST', input));
}

// updateProjectGroup memanggil PUT /api/v1/project-groups/{id} (E-11).
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

// createModelGroup memanggil POST /api/v1/model-groups (E-11).
export function createModelGroup(
	input: NewModelGroup,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ModelGroup>> {
	return call<ModelGroup>(fetchFn, '/api/v1/model-groups', jsonRequest('POST', input));
}

// renameModelGroup memanggil PUT /api/v1/model-groups/{id} (E-11).
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

// createModel memanggil POST /api/v1/models (E-11).
export function createModel(input: NewModel, fetchFn: typeof fetch = fetch): Promise<ApiResult<Model>> {
	return call<Model>(fetchFn, '/api/v1/models', jsonRequest('POST', input));
}

// updateModel memanggil PUT /api/v1/models/{id} (E-11). Tanpa model_group_id
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

// createPartNumber memanggil POST /api/v1/part-numbers (E-11).
export function createPartNumber(
	input: NewPartNumber,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<PartNumber>> {
	return call<PartNumber>(fetchFn, '/api/v1/part-numbers', jsonRequest('POST', input));
}

// updatePartNumber memanggil PUT /api/v1/part-numbers/{id} (E-11).
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

// updateNumberSettings memanggil PUT /api/v1/number-settings (E-11). pattern dan
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

// importDocumentNumbersFile memanggil POST /api/v1/document-number-imports/file
// (E-8, I4): nomor lama dari berkas Excel. commit false hanya memberi laporan.
export function importDocumentNumbersFile(
	file: File,
	commit: boolean,
	oneSet: boolean,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<NumberImportReport>> {
	const form = new FormData();
	form.append('file', file);
	const query = `?commit=${String(commit)}&one_set=${String(oneSet)}`;
	return call<NumberImportReport>(fetchFn, '/api/v1/document-number-imports/file' + query, { method: 'POST', body: form });
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

// createDocumentType memanggil POST /api/v1/document-types (E-11).
export function createDocumentType(
	input: DocumentTypeInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<DocumentType>> {
	return call<DocumentType>(fetchFn, '/api/v1/document-types', jsonRequest('POST', input));
}

// updateDocumentType memanggil PUT /api/v1/document-types/{id} (E-11).
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

// createSpecialCode memanggil POST /api/v1/special-codes (E-11).
export function createSpecialCode(
	input: NewSpecialCode,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<SpecialCode>> {
	return call<SpecialCode>(fetchFn, '/api/v1/special-codes', jsonRequest('POST', input));
}

// updateSpecialCode memanggil PUT /api/v1/special-codes/{id} (E-11). Kodenya
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

// updateChangeType memanggil PUT /api/v1/change-types/{id} (E-11).
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

// createChangeSource memanggil POST /api/v1/change-sources (E-11).
export function createChangeSource(
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ChangeSource>> {
	return call<ChangeSource>(fetchFn, '/api/v1/change-sources', jsonRequest('POST', { name }));
}

// updateChangeSource memanggil PUT /api/v1/change-sources/{id} (E-11).
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

// updateFourMCategory memanggil PUT /api/v1/four-m-categories/{id} (E-11).
export function updateFourMCategory(
	id: string,
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FourMCategory>> {
	return call<FourMCategory>(fetchFn, itemPath('four-m-categories', id), jsonRequest('PUT', { name }));
}

// listPositionCandidates memanggil GET /api/v1/position-candidates (E-11, D-209):
// orang dengan akun kerja aktif, hanya ID dan nama.
export function listPositionCandidates(fetchFn: typeof fetch = fetch): Promise<ApiResult<PersonInfoList>> {
	return call<PersonInfoList>(fetchFn, '/api/v1/position-candidates', { method: 'GET' });
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

// createFormatTemplate memanggil POST /api/v1/format-templates (E-11).
export function createFormatTemplate(
	input: NewFormatTemplate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatTemplate>> {
	return call<FormatTemplate>(fetchFn, '/api/v1/format-templates', jsonRequest('POST', input));
}

// renameFormatTemplate memanggil PUT /api/v1/format-templates/{id} (E-11).
export function renameFormatTemplate(
	id: string,
	name: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatTemplate>> {
	return call<FormatTemplate>(fetchFn, itemPath('format-templates', id), jsonRequest('PUT', { name }));
}

// setFormatTemplateHidden memanggil PUT /api/v1/format-templates/{id}/hidden (E-11, D-218).
export function setFormatTemplateHidden(
	id: string,
	hidden: boolean,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatTemplate>> {
	return call<FormatTemplate>(fetchFn, itemPath('format-templates', id) + '/hidden', jsonRequest('PUT', { hidden }));
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

// Tahap 7: migrasi dokumen lama, penautan baris, pembanding Excel, dan dasbor.

// listMigratableNumbers memanggil GET /api/v1/migration/numbers (E-10, M7.1).
export function listMigratableNumbers(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<MigratableNumberList>> {
	return call<MigratableNumberList>(fetchFn, `/api/v1/migration/numbers?customer_id=${encodeURIComponent(customerId)}`, { method: 'GET' });
}

// migrateDocument memanggil POST /api/v1/migration/documents (E-10, M7.1).
export function migrateDocument(input: LegacyDocument, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, '/api/v1/migration/documents', jsonRequest('POST', input));
}

// setDocumentHistory memanggil PUT /api/v1/documents/{id}/history (E-10, M7.2).
export function setDocumentHistory(id: string, entries: HistoryEntry[], fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/history', jsonRequest('PUT', { entries }));
}

// getMigrationOverview memanggil GET /api/v1/migration/overview (E-10, D-215).
export function getMigrationOverview(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<MigrationOverview>> {
	return call<MigrationOverview>(fetchFn, `/api/v1/migration/overview?customer_id=${encodeURIComponent(customerId)}`, { method: 'GET' });
}

// migrateSet memanggil POST /api/v1/migration/sets (E-10, D-215, D-216).
export function migrateSet(input: LegacySet, fetchFn: typeof fetch = fetch): Promise<ApiResult<MigratedSet>> {
	return call<MigratedSet>(fetchFn, '/api/v1/migration/sets', jsonRequest('POST', input));
}

// verifyMigration memanggil POST /api/v1/documents/{id}/migration/verify (E-10, M7.5).
export function verifyMigration(id: string, effortMinutes: number | undefined, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/migration/verify', jsonRequest('POST', { effort_minutes: effortMinutes }));
}

// listLinkSuggestions memanggil GET /api/v1/documents/{id}/link-suggestions (L-4, M7.4).
export function listLinkSuggestions(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<LinkSuggestionList>> {
	return call<LinkSuggestionList>(fetchFn, itemPath('documents', id) + '/link-suggestions', { method: 'GET' });
}

// linkDocumentRows memanggil POST /api/v1/documents/{id}/links (E-2, M7.4).
export function linkDocumentRows(id: string, links: RowLink[], fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/links', jsonRequest('POST', { links }));
}

// importMasterData memanggil POST /api/v1/master-data/import (E-11, I5).
// commit false hanya memberi laporan (Preview).
export function importMasterData(file: File, commit: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<MasterDataImport>> {
	const form = new FormData();
	form.append('file', file);
	return call<MasterDataImport>(fetchFn, '/api/v1/master-data/import?commit=' + String(commit), { method: 'POST', body: form });
}

// importAccounts memanggil POST /api/v1/accounts/import (A-1, I1, D-224).
// commit false hanya memberi laporan (Preview).
export function importAccounts(file: File, commit: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<AccountImport>> {
	const form = new FormData();
	form.append('file', file);
	return call<AccountImport>(fetchFn, '/api/v1/accounts/import?commit=' + String(commit), { method: 'POST', body: form });
}

// compareDocumentWithExcel memanggil POST /api/v1/documents/{id}/excel-comparison
// (L-4, M7.3). sheet dikirim sebelum file, karena server membaca berurutan.
export function compareDocumentWithExcel(id: string, file: File, sheet: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<ExcelComparison>> {
	const form = new FormData();
	if (sheet !== '') {
		form.append('sheet', sheet);
	}
	form.append('file', file);
	return call<ExcelComparison>(fetchFn, itemPath('documents', id) + '/excel-comparison', { method: 'POST', body: form });
}

// getMetrics memanggil GET /api/v1/metrics (L-2, M7.7). customerId kosong
// berarti semua customer.
export function getMetrics(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Metrics>> {
	const query = customerId === '' ? '' : `?customer_id=${encodeURIComponent(customerId)}`;
	return call<Metrics>(fetchFn, '/api/v1/metrics' + query, { method: 'GET' });
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

// getPagePreview memanggil GET /api/v1/documents/{id}/page-preview (C3): rencana
// halaman yang sama dengan PDF dan Excel, dengan SVG alur dari server.
export function getPagePreview(id: string, revision?: number, fetchFn: typeof fetch = fetch): Promise<ApiResult<PagePreview>> {
	const query = revision === undefined ? '' : `?revision=${revision}`;
	return call<PagePreview>(fetchFn, itemPath('documents', id) + '/page-preview' + query, { method: 'GET' });
}

// addDocumentRows memanggil POST /api/v1/documents/{id}/rows/batch (E-2, M4.4).
export function addDocumentRows(id: string, rows: DocumentRowInput[], fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/rows/batch', jsonRequest('POST', { rows }));
}

// getDocumentComposition memanggil GET /api/v1/documents/{id}/composition (L-4, M4.5).
// sectionKey memilih bagian (D-155); kosong untuk dokumen tanpa bagian.
export function getDocumentComposition(id: string, versionId: string, sectionKey = '', fetchFn: typeof fetch = fetch): Promise<ApiResult<Composition>> {
	let path = itemPath('documents', id) + '/composition?process_template_version_id=' + encodeURIComponent(versionId);
	if (sectionKey !== '') {
		path += '&section_key=' + encodeURIComponent(sectionKey);
	}
	return call<Composition>(fetchFn, path, { method: 'GET' });
}

// composeDocument memanggil PUT /api/v1/documents/{id}/composition (E-2, M4.5).
export function composeDocument(id: string, input: ComposeInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/composition', jsonRequest('PUT', input));
}

// getPfdComposition memanggil GET /api/v1/documents/{id}/pfd-composition (L-4, C4).
export function getPfdComposition(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<PfdComposition>> {
	return call<PfdComposition>(fetchFn, itemPath('documents', id) + '/pfd-composition', { method: 'GET' });
}

// composeFromPfd memanggil POST /api/v1/documents/{id}/pfd-composition (E-2, C4).
export function composeFromPfd(id: string, steps: PfdStepKey[], fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/pfd-composition', jsonRequest('POST', { steps }));
}

// reorderDocumentRows memanggil PUT /api/v1/documents/{id}/order (E-2, M4.6).
// sectionId memilih bagian yang diurutkan (D-155); kosong untuk dokumen tanpa bagian.
export function reorderDocumentRows(id: string, rowIds: string[], renumber: boolean, sectionId = '', fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/order', jsonRequest('PUT', { row_ids: rowIds, renumber, section_id: sectionId || undefined }));
}

// copyDocumentComposition memanggil POST /api/v1/documents/{id}/copy (E-2, M4.7).
export function copyDocumentComposition(id: string, sourceId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/copy', jsonRequest('POST', { source_document_id: sourceId }));
}

// getDocumentDifferences memanggil GET /api/v1/documents/{id}/differences (L-4, M4.8).
export function getDocumentDifferences(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<RowDifferenceList>> {
	return call<RowDifferenceList>(fetchFn, itemPath('documents', id) + '/differences', { method: 'GET' });
}

// detachDocumentRow memanggil POST /api/v1/documents/{id}/rows/{rowId}/detach (E-2, M4.8).
export function detachDocumentRow(id: string, rowId: string, reason: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, `${itemPath('documents', id)}/rows/${encodeURIComponent(rowId)}/detach`, jsonRequest('POST', { reason }));
}

// getPfmeaCpComparison memanggil GET /api/v1/documents/{id}/pfmea-cp (L-4, M4.9).
export function getPfmeaCpComparison(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<PfmeaCpComparison>> {
	return call<PfmeaCpComparison>(fetchFn, itemPath('documents', id) + '/pfmea-cp', { method: 'GET' });
}

// markPfmeaCpChecked memanggil PUT /api/v1/documents/{id}/pfmea-cp (V-2, D-65).
export function markPfmeaCpChecked(id: string, processId: string, checked: boolean, sectionKey = '', fetchFn: typeof fetch = fetch): Promise<ApiResult<PfmeaCpComparison>> {
	return call<PfmeaCpComparison>(fetchFn, itemPath('documents', id) + '/pfmea-cp', jsonRequest('PUT', { process_id: processId, checked, section_key: sectionKey || undefined }));
}

// Master dokumen terkait (D-130, D-156).

// listReferencedDocuments memanggil GET /api/v1/referenced-documents (L-2).
export function listReferencedDocuments(fetchFn: typeof fetch = fetch): Promise<ApiResult<ReferencedDocumentList>> {
	return call<ReferencedDocumentList>(fetchFn, '/api/v1/referenced-documents', { method: 'GET' });
}

// createReferencedDocument memanggil POST /api/v1/referenced-documents (E-6).
export function createReferencedDocument(input: NewReferencedDocument, fetchFn: typeof fetch = fetch): Promise<ApiResult<ReferencedDocument>> {
	return call<ReferencedDocument>(fetchFn, '/api/v1/referenced-documents', jsonRequest('POST', input));
}

// updateReferencedDocument memanggil PUT /api/v1/referenced-documents/{id} (E-6).
export function updateReferencedDocument(id: string, input: ReferencedDocumentUpdate, fetchFn: typeof fetch = fetch): Promise<ApiResult<ReferencedDocument>> {
	return call<ReferencedDocument>(fetchFn, itemPath('referenced-documents', id), jsonRequest('PUT', input));
}

// addReferencedDocumentLink memanggil POST /api/v1/referenced-documents/{id}/links (E-6).
export function addReferencedDocumentLink(id: string, input: NewReferencedDocumentLink, fetchFn: typeof fetch = fetch): Promise<ApiResult<ReferencedDocument>> {
	return call<ReferencedDocument>(fetchFn, itemPath('referenced-documents', id) + '/links', jsonRequest('POST', input));
}

// removeReferencedDocumentLink memanggil DELETE /api/v1/referenced-documents/{id}/links/{linkId} (E-6).
export function removeReferencedDocumentLink(id: string, linkId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<ReferencedDocument>> {
	return call<ReferencedDocument>(fetchFn, `${itemPath('referenced-documents', id)}/links/${encodeURIComponent(linkId)}`, { method: 'DELETE' });
}

// fillDocumentReferences memanggil POST /api/v1/documents/{id}/references (E-2, D-130).
export function fillDocumentReferences(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/references', { method: 'POST' });
}

// Bagian per PCB (D-155).

// addDocumentSection memanggil POST /api/v1/documents/{id}/sections (E-2).
export function addDocumentSection(id: string, input: NewDocumentSection, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/sections', jsonRequest('POST', input));
}

// updateDocumentSection memanggil PUT /api/v1/documents/{id}/sections/{sectionId} (E-2).
export function updateDocumentSection(id: string, sectionId: string, input: DocumentSectionUpdate, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, `${itemPath('documents', id)}/sections/${encodeURIComponent(sectionId)}`, jsonRequest('PUT', input));
}

// removeDocumentSection memanggil DELETE /api/v1/documents/{id}/sections/{sectionId} (E-2).
export function removeDocumentSection(id: string, sectionId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, `${itemPath('documents', id)}/sections/${encodeURIComponent(sectionId)}`, { method: 'DELETE' });
}

// Perubahan (Tahap 5).

// listChanges memanggil GET /api/v1/changes?customer_id= (L-4, M5.1).
export function listChanges(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<ChangeList>> {
	return call<ChangeList>(fetchFn, '/api/v1/changes?customer_id=' + encodeURIComponent(customerId), { method: 'GET' });
}

// createChange memanggil POST /api/v1/changes (E-3, M5.1).
export function createChange(input: NewChange, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, '/api/v1/changes', jsonRequest('POST', input));
}

// getChange memanggil GET /api/v1/changes/{id} (L-4).
export function getChange(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, itemPath('changes', id), { method: 'GET' });
}

// addChangeDocument memanggil POST /api/v1/changes/{id}/documents (E-3).
export function addChangeDocument(id: string, documentId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, itemPath('changes', id) + '/documents', jsonRequest('POST', { document_id: documentId }));
}

// removeChangeDocument memanggil DELETE /api/v1/changes/{id}/documents/{documentId} (E-3).
export function removeChangeDocument(id: string, documentId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, `${itemPath('changes', id)}/documents/${encodeURIComponent(documentId)}`, { method: 'DELETE' });
}

// addChangeAttachment memanggil POST /api/v1/changes/{id}/attachments (E-3, M5.2).
// Berkas dikirim sebagai multipart; browser mengisi Content-Type beserta batasnya.
export function addChangeAttachment(id: string, file: File, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	const form = new FormData();
	form.append('file', file);
	return call<Change>(fetchFn, itemPath('changes', id) + '/attachments', { method: 'POST', body: form });
}

// removeChangeAttachment memanggil DELETE /api/v1/changes/{id}/attachments/{attachmentId} (E-3).
export function removeChangeAttachment(id: string, attachmentId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, changeAttachmentUrl(id, attachmentId), { method: 'DELETE' });
}

// saveChangeImpacts memanggil PUT /api/v1/changes/{id}/impacts (E-3, M5.7).
export function saveChangeImpacts(id: string, input: ChangeImpactsInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, itemPath('changes', id) + '/impacts', jsonRequest('PUT', input));
}

// previewPropagation memanggil GET /api/v1/process-template-versions/{id}/propagation (E-6, M5.8).
export function previewPropagation(versionId: string, scope: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<PropagationPreview>> {
	return call<PropagationPreview>(fetchFn, `${itemPath('process-template-versions', versionId)}/propagation?scope=${encodeURIComponent(scope)}`, { method: 'GET' });
}

// propagateTemplate memanggil POST /api/v1/process-template-versions/{id}/propagation (E-6, M5.8).
export function propagateTemplate(versionId: string, input: PropagationInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<ChangeList>> {
	return call<ChangeList>(fetchFn, itemPath('process-template-versions', versionId) + '/propagation', jsonRequest('POST', input));
}

// listDocumentTemplates memanggil GET /api/v1/documents/{id}/templates (L-4, M5.8).
export function listDocumentTemplates(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<DocumentTemplateList>> {
	return call<DocumentTemplateList>(fetchFn, itemPath('documents', id) + '/templates', { method: 'GET' });
}

// setDocumentTemplateMode memanggil PUT /api/v1/documents/{id}/templates/{templateId} (E-2, M5.8).
export function setDocumentTemplateMode(
	id: string,
	templateId: string,
	input: DocumentTemplateModeInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<DocumentTemplateList>> {
	return call<DocumentTemplateList>(fetchFn, `${itemPath('documents', id)}/templates/${encodeURIComponent(templateId)}`, jsonRequest('PUT', input));
}

// submitSQAMChange memanggil POST /api/v1/sqam-changes (E-3, M5.10).
export function submitSQAMChange(input: SqamChangeInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, '/api/v1/sqam-changes', jsonRequest('POST', input));
}

// PrintedFile adalah hasil cetak yang siap disimpan (Tahap 6).
export type PrintedFile = { name: string; blob: Blob };

// printDocument memanggil POST /api/v1/documents/{id}/print (M6.2 sampai M6.6).
// Jawabannya berkas, bukan JSON; nama berkas diambil dari Content-Disposition.
export async function printDocument(id: string, input: PrintRequest, fetchFn: typeof fetch = fetch): Promise<ApiResult<PrintedFile>> {
	let response: Response;
	try {
		response = await fetchFn(itemPath('documents', id) + '/print', jsonRequest('POST', input));
	} catch {
		return { ok: false, error: { status: 0, body: null } };
	}
	if (!response.ok) {
		if (response.status === 401) onUnauthenticated?.();
		const body = (await response.json().catch(() => null)) as ApiError | null;
		return { ok: false, error: { status: response.status, body } };
	}
	const disposition = response.headers.get('Content-Disposition') ?? '';
	const match = /filename\*=UTF-8''([^;]+)/.exec(disposition);
	const name = match ? decodeURIComponent(match[1]) : 'document';
	return { ok: true, data: { name, blob: await response.blob() } };
}

// saveFile meminta browser menyimpan berkas hasil cetak.
export function saveFile(file: PrintedFile): void {
	const url = URL.createObjectURL(file.blob);
	const a = document.createElement('a');
	a.href = url;
	a.download = file.name;
	document.body.append(a);
	a.click();
	a.remove();
	setTimeout(() => URL.revokeObjectURL(url), 1000);
}

// listDocumentPrints memanggil GET /api/v1/documents/{id}/prints (L-4, M6.6).
export function listDocumentPrints(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<PrintLogList>> {
	return call<PrintLogList>(fetchFn, itemPath('documents', id) + '/prints', { method: 'GET' });
}

// uploadSignedScan memanggil POST /api/v1/documents/{id}/scan (E-7, M6.7).
export function uploadSignedScan(id: string, file: File, fetchFn: typeof fetch = fetch): Promise<ApiResult<SignedScan>> {
	const form = new FormData();
	form.append('file', file);
	return call<SignedScan>(fetchFn, itemPath('documents', id) + '/scan', { method: 'POST', body: form });
}

// signedScanUrl adalah alamat unduhan scan satu revisi (L-4, M6.7).
export function signedScanUrl(id: string, revision: number): string {
	return `${itemPath('documents', id)}/revisions/${revision}/scan`;
}

// verifyRevision memanggil GET /api/v1/verify/{revisionId} (L-1, M6.5).
export function verifyRevision(revisionId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<RevisionVerification>> {
	return call<RevisionVerification>(fetchFn, itemPath('verify', revisionId), { method: 'GET' });
}

// listPrintLayouts memanggil GET /api/v1/print-layouts (L-2, M6.1).
export function listPrintLayouts(fetchFn: typeof fetch = fetch): Promise<ApiResult<PrintLayoutList>> {
	return call<PrintLayoutList>(fetchFn, '/api/v1/print-layouts', { method: 'GET' });
}

// createPrintLayout memanggil POST /api/v1/print-layouts (E-11, M6.1).
export function createPrintLayout(input: PrintLayoutInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<PrintLayout>> {
	return call<PrintLayout>(fetchFn, '/api/v1/print-layouts', jsonRequest('POST', input));
}

// updatePrintLayout memanggil PUT /api/v1/print-layouts/{id} (E-11, M6.1).
export function updatePrintLayout(id: string, input: PrintLayoutInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<PrintLayout>> {
	return call<PrintLayout>(fetchFn, itemPath('print-layouts', id), jsonRequest('PUT', input));
}

// listCustomerPrintLayouts memanggil GET /api/v1/customers/{id}/print-layouts (L-2, M6.1).
export function listCustomerPrintLayouts(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<CustomerPrintLayoutList>> {
	return call<CustomerPrintLayoutList>(fetchFn, itemPath('customers', customerId) + '/print-layouts', { method: 'GET' });
}

// setCustomerPrintLayout memanggil PUT /api/v1/customers/{id}/print-layouts/{typeId} (E-11, M6.1).
export function setCustomerPrintLayout(
	customerId: string,
	documentTypeId: string,
	layoutId: string | undefined,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<CustomerPrintLayoutList>> {
	return call<CustomerPrintLayoutList>(
		fetchFn,
		`${itemPath('customers', customerId)}/print-layouts/${encodeURIComponent(documentTypeId)}`,
		jsonRequest('PUT', { layout_id: layoutId })
	);
}

// listYokotens memanggil GET /api/v1/yokotens (L-4, M5.9).
export function listYokotens(fetchFn: typeof fetch = fetch): Promise<ApiResult<YokotenList>> {
	return call<YokotenList>(fetchFn, '/api/v1/yokotens', { method: 'GET' });
}

// getYokoten memanggil GET /api/v1/yokotens/{id} (L-4, M5.9).
export function getYokoten(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Yokoten>> {
	return call<Yokoten>(fetchFn, itemPath('yokotens', id), { method: 'GET' });
}

// createYokoten memanggil POST /api/v1/yokotens (E-3, M5.9).
export function createYokoten(input: NewYokoten, fetchFn: typeof fetch = fetch): Promise<ApiResult<Yokoten>> {
	return call<Yokoten>(fetchFn, '/api/v1/yokotens', jsonRequest('POST', input));
}

// searchYokotenTargets memanggil POST /api/v1/yokoten-targets/search (E-3, M5.9); tidak menyimpan.
export function searchYokotenTargets(input: YokotenSearch, fetchFn: typeof fetch = fetch): Promise<ApiResult<YokotenTargetList>> {
	return call<YokotenTargetList>(fetchFn, '/api/v1/yokoten-targets/search', jsonRequest('POST', input));
}

// decideYokotenTarget memanggil POST /api/v1/yokotens/{id}/targets/{targetId}/decision (E-3, M5.9).
export function decideYokotenTarget(id: string, targetId: string, input: YokotenDecision, fetchFn: typeof fetch = fetch): Promise<ApiResult<Yokoten>> {
	return call<Yokoten>(fetchFn, `${itemPath('yokotens', id)}/targets/${encodeURIComponent(targetId)}/decision`, jsonRequest('POST', input));
}

// verifyChange memanggil POST /api/v1/changes/{id}/verification (V-1, M5.3).
export function verifyChange(id: string, input: ChangeVerificationInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, itemPath('changes', id) + '/verification', jsonRequest('POST', input));
}

// changeAttachmentUrl adalah alamat unduhan satu lampiran (L-4).
export function changeAttachmentUrl(id: string, attachmentId: string): string {
	return `${itemPath('changes', id)}/attachments/${encodeURIComponent(attachmentId)}`;
}

// releaseDocument memanggil POST /api/v1/documents/{id}/release (E-7, M5.5).
export function releaseDocument(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('documents', id) + '/release', { method: 'POST' });
}

// listDocumentRevisions memanggil GET /api/v1/documents/{id}/revisions (L-4, M5.5).
export function listDocumentRevisions(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<DocumentRevisionList>> {
	return call<DocumentRevisionList>(fetchFn, itemPath('documents', id) + '/revisions', { method: 'GET' });
}

// getDocumentDiff memanggil GET /api/v1/documents/{id}/diff (L-4, M5.6). to
// kosong berarti draft yang sedang dikerjakan.
export function getDocumentDiff(
	id: string,
	from: number,
	to?: number,
	processId?: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<DocumentDiff>> {
	const query = new URLSearchParams({ from: String(from) });
	if (to !== undefined) query.set('to', String(to));
	if (processId) query.set('process_id', processId);
	return call<DocumentDiff>(fetchFn, `${itemPath('documents', id)}/diff?${query}`, { method: 'GET' });
}

// listDocumentChanges memanggil GET /api/v1/documents/{id}/changes (L-4).
export function listDocumentChanges(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<ChangeList>> {
	return call<ChangeList>(fetchFn, itemPath('documents', id) + '/changes', { method: 'GET' });
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

// setFormSetHidden memanggil PUT /api/v1/form-sets/{id}/hidden (E-11, D-218).
export function setFormSetHidden(id: string, hidden: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, itemPath('form-sets', id) + '/hidden', jsonRequest('PUT', { hidden }));
}

// createFormSet memanggil POST /api/v1/form-sets (E-11).
export function createFormSet(input: NewFormSet, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, '/api/v1/form-sets', jsonRequest('POST', input));
}

// setFormSetSheets memanggil PUT /api/v1/form-set-versions/{id} (E-11, versi draft).
export function setFormSetSheets(versionId: string, sheets: FormSetSheets, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, itemPath('form-set-versions', versionId), jsonRequest('PUT', sheets));
}

// createFormSetVersion memanggil POST /api/v1/form-sets/{id}/versions (E-11).
export function createFormSetVersion(formSetId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, itemPath('form-sets', formSetId) + '/versions', { method: 'POST' });
}

// publishFormSetVersion memanggil POST /api/v1/form-set-versions/{id}/publish (V-3).
export function publishFormSetVersion(versionId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormSet>> {
	return call<FormSet>(fetchFn, itemPath('form-set-versions', versionId) + '/publish', { method: 'POST' });
}

// setCustomerFormSet memanggil PUT /api/v1/customers/{id}/default-form-set (E-11).
// formSetId kosong berarti customer memakai set bawaan.
export function setCustomerFormSet(customerId: string, formSetId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<void>> {
	const body = formSetId === '' ? {} : { form_set_id: formSetId };
	return call<void>(fetchFn, `${itemPath('customers', customerId)}/default-form-set`, jsonRequest('PUT', body));
}

// setFormatApRules memanggil PUT /api/v1/format-template-versions/{id}/ap-rules (E-11, versi draft).
export function setFormatApRules(versionId: string, items: ApRule[], fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', versionId) + '/ap-rules', jsonRequest('PUT', { items }));
}

// publishFormatVersion memanggil POST .../format-template-versions/{id}/publish (E-11, M3.4).
export function publishFormatVersion(versionId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatVersion>> {
	const path = itemPath('format-template-versions', versionId) + '/publish';
	return call<FormatVersion>(fetchFn, path, { method: 'POST' });
}

// createFormatVersion memanggil POST .../format-templates/{id}/versions (E-11, M3.4):
// versi draft berikutnya, salinan versi terbit terakhir.
export function createFormatVersion(templateId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatVersion>> {
	const path = itemPath('format-templates', templateId) + '/versions';
	return call<FormatVersion>(fetchFn, path, { method: 'POST' });
}

export function getFormatVersion(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', id), { method: 'GET' });
}

// updateFormatHighlights memanggil PUT /api/v1/format-template-versions/{id} (E-11).
export function updateFormatHighlights(
	id: string,
	input: FormatHighlights,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', id), jsonRequest('PUT', input));
}

// addFormatField memanggil POST /api/v1/format-template-versions/{id}/fields (E-11).
export function addFormatField(
	versionId: string,
	input: NewFormatField,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	return call<FormatVersion>(fetchFn, itemPath('format-template-versions', versionId) + '/fields', jsonRequest('POST', input));
}

// updateFormatField memanggil PUT /api/v1/format-template-versions/{id}/fields/{fieldId} (E-11).
export function updateFormatField(
	versionId: string,
	fieldId: string,
	input: FormatFieldUpdate,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	const path = itemPath('format-template-versions', versionId) + '/fields/' + encodeURIComponent(fieldId);
	return call<FormatVersion>(fetchFn, path, jsonRequest('PUT', input));
}

// setFormatRatingCriteria memanggil PUT /api/v1/format-template-versions/{id}/rating-criteria (E-11).
export function setFormatRatingCriteria(
	versionId: string,
	items: RatingCriterionInput[],
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<FormatVersion>> {
	const path = itemPath('format-template-versions', versionId) + '/rating-criteria';
	return call<FormatVersion>(fetchFn, path, jsonRequest('PUT', { items }));
}

// saveFormatSymbols memanggil PUT /api/v1/format-template-versions/{id}/symbols (E-11).
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

// createApprovalRoute memanggil POST /api/v1/approval-routes (E-11).
export function createApprovalRoute(
	input: ApprovalRouteInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ApprovalRoute>> {
	return call<ApprovalRoute>(fetchFn, '/api/v1/approval-routes', jsonRequest('POST', input));
}

// updateApprovalRoute memanggil PUT /api/v1/approval-routes/{id} (E-11).
// Daftar slot menggantikan semua slot lama.
export function updateApprovalRoute(
	id: string,
	input: ApprovalRouteInput,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<ApprovalRoute>> {
	return call<ApprovalRoute>(fetchFn, itemPath('approval-routes', id), jsonRequest('PUT', input));
}

// setCustomerDefaultRoute memanggil PUT /api/v1/customers/{id}/default-route (E-11).
export function setCustomerDefaultRoute(
	customerId: string,
	routeId: string,
	fetchFn: typeof fetch = fetch
): Promise<ApiResult<Customer>> {
	const url = `${itemPath('customers', customerId)}/default-route`;
	return call<Customer>(fetchFn, url, jsonRequest('PUT', { route_id: routeId }));
}

// setChangeTypeRoute memanggil PUT /api/v1/change-types/{id}/approval-route
// (E-11). null berarti rute default setiap customer.
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

// Nama perusahaan dan logo cetakan (bagian C, E-144).

// getCompany memanggil GET /api/v1/company.
export function getCompany(fetchFn: typeof fetch = fetch): Promise<ApiResult<Company>> {
	return call<Company>(fetchFn, '/api/v1/company', { method: 'GET' });
}

// setCompanyName memanggil PUT /api/v1/company (admin).
export function setCompanyName(name: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Company>> {
	return call<Company>(fetchFn, '/api/v1/company', jsonRequest('PUT', { name }));
}

// setCompanyLogo memanggil PUT /api/v1/company/logo dengan isi berkas PNG (admin).
// revise adalah jawaban admin apakah dokumen direvisi karena logo baru (D-223).
export function setCompanyLogo(file: Blob, revise: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<Company>> {
	return call<Company>(fetchFn, '/api/v1/company/logo?revise=' + String(revise), {
		method: 'PUT',
		headers: { 'Content-Type': 'image/png' },
		body: file
	});
}

// resetCompanyLogo memanggil DELETE /api/v1/company/logo (admin), revise seperti di atas.
export function resetCompanyLogo(revise: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<Company>> {
	return call<Company>(fetchFn, '/api/v1/company/logo?revise=' + String(revise), { method: 'DELETE' });
}

// getPendingLogoChange memanggil GET /api/v1/customers/{id}/pending-logo-change (D-223).
export function getPendingLogoChange(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<PendingLogoChange>> {
	return call<PendingLogoChange>(fetchFn, itemPath('customers', customerId) + '/pending-logo-change', { method: 'GET' });
}

// submitLogoChange memanggil POST /api/v1/logo-changes (E-3, D-223).
export function submitLogoChange(input: LogoChangeInput, fetchFn: typeof fetch = fetch): Promise<ApiResult<Change>> {
	return call<Change>(fetchFn, '/api/v1/logo-changes', jsonRequest('POST', input));
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

// listPermissions memanggil GET /api/v1/permissions: katalog hak (A-1, M8.1).
export function listPermissions(fetchFn: typeof fetch = fetch): Promise<ApiResult<PermissionList>> {
	return call<PermissionList>(fetchFn, '/api/v1/permissions', { method: 'GET' });
}

// setRolePermissions memanggil PUT /api/v1/roles/{id}/permissions (A-1, M8.1).
export function setRolePermissions(roleId: string, permissions: string[], fetchFn: typeof fetch = fetch): Promise<ApiResult<Role>> {
	return call<Role>(fetchFn, itemPath('roles', roleId) + '/permissions', jsonRequest('PUT', { permissions }));
}

// D-199: admin menambah, mengganti nama, dan menonaktifkan peran.

// createRole memanggil POST /api/v1/roles.
export function createRole(input: NewRole, fetchFn: typeof fetch = fetch): Promise<ApiResult<Role>> {
	return call<Role>(fetchFn, '/api/v1/roles', jsonRequest('POST', input));
}

// updateRole memanggil PATCH /api/v1/roles/{id}: nama dan cakupan Project Group.
export function updateRole(roleId: string, change: RoleChange, fetchFn: typeof fetch = fetch): Promise<ApiResult<Role>> {
	return call<Role>(fetchFn, itemPath('roles', roleId), jsonRequest('PATCH', change));
}

// setRoleActive memanggil PUT /api/v1/roles/{id}/active.
export function setRoleActive(roleId: string, active: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<Role>> {
	return call<Role>(fetchFn, itemPath('roles', roleId) + '/active', jsonRequest('PUT', { active }));
}

// Fase 3: tanda tangan digital (D-198, D-203, D-204).

// setCustomerSignatureMode memanggil PUT /api/v1/customers/{id}/signature-mode (E-11).
export function setCustomerSignatureMode(customerId: string, mode: 'digital' | 'wet_only', fetchFn: typeof fetch = fetch): Promise<ApiResult<Customer>> {
	return call<Customer>(fetchFn, itemPath('customers', customerId) + '/signature-mode', jsonRequest('PUT', { mode }));
}

// getSignaturePlan memanggil GET /api/v1/documents/{id}/signature-plan (E-5).
export function getSignaturePlan(documentId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<SignaturePlan>> {
	return call<SignaturePlan>(fetchFn, itemPath('documents', documentId) + '/signature-plan', { method: 'GET' });
}

// listDocumentSignatureRequests memanggil GET /api/v1/documents/{id}/signature-requests.
export function listDocumentSignatureRequests(documentId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<SignatureRequestList>> {
	return call<SignatureRequestList>(fetchFn, itemPath('documents', documentId) + '/signature-requests', { method: 'GET' });
}

// sendForSignature memanggil POST /api/v1/documents/{id}/signature-requests.
export function sendForSignature(documentId: string, input: SignatureSend, fetchFn: typeof fetch = fetch): Promise<ApiResult<SignatureRequest>> {
	return call<SignatureRequest>(fetchFn, itemPath('documents', documentId) + '/signature-requests', jsonRequest('POST', input));
}

// listSignatureRequests memanggil GET /api/v1/signature-requests.
export function listSignatureRequests(fetchFn: typeof fetch = fetch): Promise<ApiResult<SignatureOverview>> {
	return call<SignatureOverview>(fetchFn, '/api/v1/signature-requests', { method: 'GET' });
}

// getSignatureRequest memanggil GET /api/v1/signature-requests/{id}.
export function getSignatureRequest(requestId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<SignatureRequest>> {
	return call<SignatureRequest>(fetchFn, itemPath('signature-requests', requestId), { method: 'GET' });
}

// signSignatureRequest memanggil POST /api/v1/signature-requests/{id}/sign.
export function signSignatureRequest(requestId: string, password: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<SignatureRequest>> {
	return call<SignatureRequest>(fetchFn, itemPath('signature-requests', requestId) + '/sign', jsonRequest('POST', { password }));
}

// rejectSignatureRequest memanggil POST /api/v1/signature-requests/{id}/reject.
export function rejectSignatureRequest(requestId: string, reason: string, password: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<SignatureRequest>> {
	return call<SignatureRequest>(fetchFn, itemPath('signature-requests', requestId) + '/reject', jsonRequest('POST', { reason, password }));
}

// withdrawSignatureRequest memanggil POST /api/v1/signature-requests/{id}/withdraw.
export function withdrawSignatureRequest(requestId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<SignatureRequest>> {
	return call<SignatureRequest>(fetchFn, itemPath('signature-requests', requestId) + '/withdraw', { method: 'POST' });
}

// Fase 4: dasbor admin (D-205).

// lastFailure menyimpan request ID error API terakhir, supaya laporan masalah
// bisa menyertakannya tanpa pengguna mencarinya sendiri.
export const lastFailure = { requestId: '' };

// getAdminHealth memanggil GET /api/v1/admin/health (A-3).
export function getAdminHealth(fetchFn: typeof fetch = fetch): Promise<ApiResult<AdminHealth>> {
	return call<AdminHealth>(fetchFn, '/api/v1/admin/health', { method: 'GET' });
}

// listWaitingSignatures memanggil GET /api/v1/admin/waiting-signatures (A-3).
export function listWaitingSignatures(fetchFn: typeof fetch = fetch): Promise<ApiResult<WaitingSignatureList>> {
	return call<WaitingSignatureList>(fetchFn, '/api/v1/admin/waiting-signatures', { method: 'GET' });
}

// listErrorReports memanggil GET /api/v1/admin/error-reports (A-3).
export function listErrorReports(fetchFn: typeof fetch = fetch): Promise<ApiResult<ErrorReportList>> {
	return call<ErrorReportList>(fetchFn, '/api/v1/admin/error-reports', { method: 'GET' });
}

// listProblemReports memanggil GET /api/v1/admin/problem-reports (A-3).
export function listProblemReports(fetchFn: typeof fetch = fetch): Promise<ApiResult<ProblemReportList>> {
	return call<ProblemReportList>(fetchFn, '/api/v1/admin/problem-reports', { method: 'GET' });
}

// updateProblemReport memanggil PATCH /api/v1/admin/problem-reports/{id} (A-3).
export function updateProblemReport(id: string, status: 'new' | 'checking' | 'done', note: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<void>> {
	return call<void>(fetchFn, itemPath('admin/problem-reports', id), jsonRequest('PATCH', { status, note }));
}

// reportProblem memanggil POST /api/v1/problem-reports (setiap akun).
export function reportProblem(page: string, description: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<CreatedId>> {
	const body: { page: string; description: string; last_request_id?: string } = { page, description };
	if (lastFailure.requestId) body.last_request_id = lastFailure.requestId;
	return call<CreatedId>(fetchFn, '/api/v1/problem-reports', jsonRequest('POST', body));
}

// D-207: Form Control Submission per dokumen.

// getSubmissionControl memanggil GET /api/v1/documents/{id}/submission-control (L-4).
export function getSubmissionControl(documentId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<SubmissionControl>> {
	return call<SubmissionControl>(fetchFn, itemPath('documents', documentId) + '/submission-control', { method: 'GET' });
}

// saveSubmissionEntry memanggil PUT /api/v1/documents/{id}/submission-control/{revision} (E-7).
export function saveSubmissionEntry(documentId: string, revision: number, entry: SubmissionEntry, fetchFn: typeof fetch = fetch): Promise<ApiResult<SubmissionControl>> {
	return call<SubmissionControl>(fetchFn, `${itemPath('documents', documentId)}/submission-control/${revision}`, jsonRequest('PUT', entry));
}

// uploadSubmissionScan memanggil POST /api/v1/documents/{id}/submission-control/{revision}/scan (E-7).
export function uploadSubmissionScan(documentId: string, revision: number, file: File, fetchFn: typeof fetch = fetch): Promise<ApiResult<SubmissionControl>> {
	const form = new FormData();
	form.append('file', file);
	return call<SubmissionControl>(fetchFn, `${itemPath('documents', documentId)}/submission-control/${revision}/scan`, { method: 'POST', body: form });
}

// submissionScanUrl adalah alamat unduhan bukti persetujuan customer satu revisi.
export function submissionScanUrl(documentId: string, revision: number): string {
	return `${itemPath('documents', documentId)}/submission-control/${revision}/scan`;
}

// submissionExportUrl adalah alamat unduhan form (xlsx atau pdf).
export function submissionExportUrl(documentId: string, format: 'xlsx' | 'pdf'): string {
	return `${itemPath('documents', documentId)}/submission-control/export?format=${format}`;
}

// setCustomerSubmissionRequired memanggil PUT /api/v1/customers/{id}/submission-required (E-11).
export function setCustomerSubmissionRequired(customerId: string, required: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<Customer>> {
	return call<Customer>(fetchFn, itemPath('customers', customerId) + '/submission-required', jsonRequest('PUT', { required }));
}

// D-194: email notifikasi.

// getOwnEmail memanggil GET /api/v1/account/email.
export function getOwnEmail(fetchFn: typeof fetch = fetch): Promise<ApiResult<EmailAddress>> {
	return call<EmailAddress>(fetchFn, '/api/v1/account/email', { method: 'GET' });
}

// setOwnEmail memanggil PUT /api/v1/account/email.
export function setOwnEmail(email: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<EmailAddress>> {
	return call<EmailAddress>(fetchFn, '/api/v1/account/email', jsonRequest('PUT', { email }));
}

// setAccountEmail memanggil PUT /api/v1/accounts/{id}/email (A-1).
export function setAccountEmail(accountId: string, email: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<EmailAddress>> {
	return call<EmailAddress>(fetchFn, `/api/v1/accounts/${encodeURIComponent(accountId)}/email`, jsonRequest('PUT', { email }));
}

// getNotifications memanggil GET /api/v1/notifications (A-3).
export function getNotifications(fetchFn: typeof fetch = fetch): Promise<ApiResult<NotificationOverview>> {
	return call<NotificationOverview>(fetchFn, '/api/v1/notifications', { method: 'GET' });
}

// setNotificationEvent memanggil PUT /api/v1/notifications/events/{event} (A-3).
export function setNotificationEvent(event: string, enabled: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<void>> {
	return call<void>(fetchFn, `/api/v1/notifications/events/${encodeURIComponent(event)}`, jsonRequest('PUT', { enabled }));
}

// sendTestEmail memanggil POST /api/v1/notifications/test (A-3).
export function sendTestEmail(fetchFn: typeof fetch = fetch): Promise<ApiResult<void>> {
	return call<void>(fetchFn, '/api/v1/notifications/test', { method: 'POST' });
}

// Tahap 9: service account API integrasi (A-1).

// listServiceAccounts memanggil GET /api/v1/service-accounts.
export function listServiceAccounts(fetchFn: typeof fetch = fetch): Promise<ApiResult<ServiceAccountList>> {
	return call<ServiceAccountList>(fetchFn, '/api/v1/service-accounts', { method: 'GET' });
}

// createServiceAccount memanggil POST /api/v1/service-accounts.
export function createServiceAccount(input: NewServiceAccount, fetchFn: typeof fetch = fetch): Promise<ApiResult<ServiceAccount>> {
	return call<ServiceAccount>(fetchFn, '/api/v1/service-accounts', jsonRequest('POST', input));
}

// issueServiceToken memanggil POST /api/v1/service-accounts/{id}/tokens. days
// kosong berarti 90 hari.
export function issueServiceToken(id: string, days: number | undefined, fetchFn: typeof fetch = fetch): Promise<ApiResult<IssuedServiceToken>> {
	return call<IssuedServiceToken>(fetchFn, itemPath('service-accounts', id) + '/tokens', jsonRequest('POST', { days }));
}

// revokeServiceToken memanggil DELETE /api/v1/service-accounts/{id}/tokens/{tokenId}.
export function revokeServiceToken(id: string, tokenId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<void>> {
	return call<void>(fetchFn, `${itemPath('service-accounts', id)}/tokens/${encodeURIComponent(tokenId)}`, { method: 'DELETE' });
}

// disableServiceAccount memanggil POST /api/v1/service-accounts/{id}/disable.
export function disableServiceAccount(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<void>> {
	return call<void>(fetchFn, itemPath('service-accounts', id) + '/disable', { method: 'POST' });
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
	if (isApiError(body) && body.request_id) {
		lastFailure.requestId = body.request_id;
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

// D-195: dokumen terbit untuk pembaca dengan L-1 (Viewer). Isinya revisi yang
// berlaku, dari snapshot; draft tidak pernah terlihat.

// listReleasedDocuments memanggil GET /api/v1/released-documents?customer_id= (L-1).
export function listReleasedDocuments(customerId: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<DocumentList>> {
	return call<DocumentList>(fetchFn, `/api/v1/released-documents?customer_id=${encodeURIComponent(customerId)}`, { method: 'GET' });
}

// getReleasedDocument memanggil GET /api/v1/released-documents/{id} (L-1).
export function getReleasedDocument(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<Document>> {
	return call<Document>(fetchFn, itemPath('released-documents', id), { method: 'GET' });
}

// getReleasedPfmeaCpComparison memanggil GET /api/v1/released-documents/{id}/pfmea-cp (L-1).
export function getReleasedPfmeaCpComparison(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<PfmeaCpComparison>> {
	return call<PfmeaCpComparison>(fetchFn, itemPath('released-documents', id) + '/pfmea-cp', { method: 'GET' });
}

// D-196: review berkala 3 tahun.

// listReviewSets memanggil GET /api/v1/periodic-reviews/sets (L-2). Tanpa all,
// hanya set yang akan atau sudah lewat jatuh tempo.
export type ReviewSetFilter = { customerId?: string; documentId?: string; all?: boolean; eol?: boolean };

function reviewQuery(filter: ReviewSetFilter): string {
	const query = new URLSearchParams();
	if (filter.customerId) query.set('customer_id', filter.customerId);
	if (filter.documentId) query.set('document_id', filter.documentId);
	if (filter.all) query.set('all', 'true');
	if (filter.eol) query.set('eol', 'true');
	const text = query.toString();
	return text ? '?' + text : '';
}

export function listReviewSets(filter: ReviewSetFilter, fetchFn: typeof fetch = fetch): Promise<ApiResult<ReviewSetList>> {
	return call<ReviewSetList>(fetchFn, '/api/v1/periodic-reviews/sets' + reviewQuery(filter), { method: 'GET' });
}

// reviewExportUrl memberi alamat unduhan Excel daftar review (D-202). Unduhan
// lewat tautan biasa: browser mengirim cookie sesi sendiri.
export function reviewExportUrl(filter: ReviewSetFilter): string {
	return '/api/v1/periodic-reviews/sets/export' + reviewQuery(filter);
}

// listPeriodicReviews memanggil GET /api/v1/periodic-reviews (L-2).
export function listPeriodicReviews(openOnly: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<PeriodicReviewList>> {
	return call<PeriodicReviewList>(fetchFn, '/api/v1/periodic-reviews' + (openOnly ? '?open_only=true' : ''), { method: 'GET' });
}

// recordPeriodicReview memanggil POST /api/v1/periodic-reviews (E-3).
export function recordPeriodicReview(input: NewPeriodicReview, fetchFn: typeof fetch = fetch): Promise<ApiResult<PeriodicReview>> {
	return call<PeriodicReview>(fetchFn, '/api/v1/periodic-reviews', jsonRequest('POST', input));
}

// confirmPeriodicReview memanggil POST /api/v1/periodic-reviews/{id}/confirm (V-1).
export function confirmPeriodicReview(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<PeriodicReview>> {
	return call<PeriodicReview>(fetchFn, itemPath('periodic-reviews', id) + '/confirm', { method: 'POST' });
}

// D-197: laporan kinerja PPAP Engineer.

// getPerformance memanggil GET /api/v1/performance (L-5). date YYYY-MM-DD
// memilih periode yang memuatnya; kosong berarti hari ini.
export type PerformanceQuery = { period: 'week' | 'month' | 'year'; date: string; customerId: string; personId: string };

function performanceQuery(q: PerformanceQuery): string {
	const query = new URLSearchParams({ period: q.period });
	if (q.date) query.set('date', q.date);
	if (q.customerId) query.set('customer_id', q.customerId);
	if (q.personId) query.set('person_id', q.personId);
	return '?' + query.toString();
}

export function getPerformance(q: PerformanceQuery, fetchFn: typeof fetch = fetch): Promise<ApiResult<Performance>> {
	return call<Performance>(fetchFn, '/api/v1/performance' + performanceQuery(q), { method: 'GET' });
}

// performanceExportUrl memberi alamat unduhan Excel laporan kinerja (D-202).
export function performanceExportUrl(q: PerformanceQuery): string {
	return '/api/v1/performance/export' + performanceQuery(q);
}

// getInbox memanggil GET /api/v1/inbox: pekerjaan yang menunggu di Home (D-202).
export function getInbox(fetchFn: typeof fetch = fetch): Promise<ApiResult<Inbox>> {
	return call<Inbox>(fetchFn, '/api/v1/inbox', { method: 'GET' });
}

// setModelEol memanggil PUT /api/v1/models/{id}/eol (E-2, D-200).
export function setModelEol(id: string, eol: boolean, fetchFn: typeof fetch = fetch): Promise<ApiResult<Model>> {
	return call<Model>(fetchFn, itemPath('models', id) + '/eol', jsonRequest('PUT', { eol }));
}

// markDocumentReady memanggil POST /api/v1/documents/{id}/ready (E-2, D-202).
export function markDocumentReady(id: string, fetchFn: typeof fetch = fetch): Promise<ApiResult<DocumentRevisionList>> {
	return call<DocumentRevisionList>(fetchFn, itemPath('documents', id) + '/ready', { method: 'POST' });
}
