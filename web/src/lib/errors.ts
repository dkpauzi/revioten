// Menerjemahkan error dari API ke teks layar.
import type { ApiFailure } from './api/client.ts';
import type { ErrorDetail } from './api/gen/index.ts';
import { t, type MessageKey } from './i18n/index.ts';

// problemKeys memetakan "field:kode masalah" dari API ke kunci teks layar.
// Dengan cara ini teks layar tetap bisa diterjemahkan, walau pesan dari server
// berbahasa Inggris.
const problemKeys: Record<string, MessageKey> = {
	'code:required': 'customerCodeRequired',
	'code:invalid_format': 'customerCodeInvalid',
	'code:reserved': 'customerCodeReserved',
	'code:taken': 'customerCodeTaken',
	'name:required': 'customerNameRequired',
	'name:too_long': 'customerNameTooLong',
	'name:invalid_format': 'customerNameInvalid',
	'current_password:incorrect': 'passwordCurrentIncorrect',
	'new_password:too_short': 'passwordTooShort',
	'new_password:too_long': 'passwordTooLong',
	'new_password:too_common': 'passwordTooCommon',
	'person_name:required': 'accountPersonRequired',
	'login:required': 'accountLoginRequired',
	'login:invalid_format': 'accountLoginInvalid',
	'login:taken': 'accountLoginTaken',
	'kind:taken': 'accountKindTaken',
	'employee_number:taken': 'accountEmployeeNumberTaken',
	'code:not_allowed': 'customerCodeSame',
	'project_group_id:required': 'customerProjectGroupRequired',
	'customer_id:not_allowed': 'customerInactive',
	'model_group_id:not_allowed': 'modelGroupOtherCustomer',
	'model_id:required': 'partNumberModelRequired',
	'model_id:not_allowed': 'modelCustomerInactive',
	'aliases:taken': 'processAliasTaken',
	'aliases:too_long': 'processAliasTooLong',
	'aliases:invalid_format': 'processAliasInvalid',
	'factory_code:invalid_format': 'factoryCodeInvalid',
	'letter:taken': 'documentTypeLetterTaken',
	'letter:invalid_format': 'documentTypeLetterInvalid',
	'active:not_allowed': 'changeSourceSystemActive',
	'person_id:required': 'holderPersonRequired',
	'person_id:taken': 'holderPersonTaken',
	'slots:required': 'routeSlotsRequired',
	'slots:too_long': 'routeSlotsTooMany',
	'slots:invalid_format': 'routeSlotsOrder',
	'print_title:required': 'routePrintTitleRequired',
	'print_title:too_long': 'routePrintTitleTooLong',
	'position_ids:required': 'routeSlotPositionsRequired',
	'position_ids:invalid_format': 'routeSlotPositionsInvalid',
	'route_id:required': 'routeIdRequired',
	'route_id:invalid_format': 'routeIdInvalid',
	'pattern:invalid_format': 'numberPatternInvalid',
	'revision_digits:invalid_format': 'numberRevisionDigitsInvalid',
	'customer_id:required': 'numberCustomerRequired',
	'document_type_id:required': 'numberTypeRequired',
	'document_type_id:not_allowed': 'numberTypeNotNumbered',
	'document_type_id:taken': 'numberTaken',
	'initial_revision:invalid_format': 'numberInitialRevisionInvalid',
	'idempotency_key:taken': 'numberKeyReused',
	'reason:required': 'numberReasonRequired',
	'status:not_allowed': 'numberStatusNotAllowed',
	'status:required': 'numberImportStatusRequired',
	'status:invalid_format': 'numberImportStatusInvalid',
	'text:required': 'numberImportTextRequired',
	'text:too_long': 'numberImportTextTooLong',
	'text:taken': 'numberImportRetry',
	'risk_method:required': 'formatRiskRequired',
	'copy_from_version_id:not_allowed': 'formatCopyNotAllowed',
	'label:required': 'formatLabelRequired',
	'label:too_long': 'formatLabelTooLong',
	'label:taken': 'formatLabelTaken',
	'kind:not_allowed': 'formatKindNotAllowed',
	'rating_scale:required': 'formatScaleRequired',
	'rating_scale:taken': 'formatScaleTaken',
	'rating_set:taken': 'formatSetTaken',
	'options:required': 'formatOptionsRequired',
	'options:taken': 'formatOptionsTaken',
	'options:not_allowed': 'formatOptionsNotAllowed',
	'required:not_allowed': 'formatHiddenRequired',
	'symbols:taken': 'formatSymbolsTaken',
	'symbols:not_allowed': 'formatSymbolsNotAllowed',
	'file:not_allowed': 'changeAttachmentNotAllowed',
	'file:required': 'changeAttachmentEmpty',
	'file:too_long': 'changeAttachmentTooLarge',
	'verifier:not_allowed': 'changeVerifierNotAllowed',
	'note:required': 'changeVerifyNoteRequired',
	'note:too_long': 'changeNoteTooLong',
	'recorder:not_allowed': 'revisionRecorderNotAllowed',
	'to:required': 'diffNoRelease',
	'impact.reason:required': 'impactReasonRequired',
	'impact.decision:required': 'impactDecisionRequired',
	'impact.kind:invalid_format': 'impactKindInvalid',
	'impact.document_reference:too_long': 'impactReferenceTooLong',
	'impact.reason:too_long': 'impactReasonTooLong',
	'impact.referenced_document_id:invalid_format': 'impactMasterInvalid',
	'items:too_long': 'impactTooMany',
	'targets:required': 'yokotenNoTargets',
	'scan:required': 'scanRequired',
	'route:required': 'printRouteRequired',
	'purpose:not_allowed': 'printPurposeNotAllowed',
	'sqam_reference:required': 'sqamReferenceRequired',
	'effective_date:required': 'sqamDateRequired'
};

// ProblemScope adalah formulir tempat masalah ditampilkan. Field yang sama bisa
// butuh teks berbeda, misalnya code:taken di formulir Project Group.
export type ProblemScope =
	| 'projectGroup'
	| 'customerCode'
	| 'modelGroup'
	| 'model'
	| 'partNumber'
	| 'process'
	| 'documentType'
	| 'specialCode'
	| 'changeList'
	| 'position'
	| 'holder'
	| 'route'
	| 'formatTemplate'
	| 'processTemplate'
	| 'formSet'
	| 'refDoc';

// scopedKeys memetakan "formulir.field:kode masalah" ke kunci teks layar.
const scopedKeys: Record<string, MessageKey> = {
	'projectGroup.code:required': 'projectGroupCodeRequired',
	'projectGroup.code:invalid_format': 'projectGroupCodeInvalid',
	'projectGroup.code:taken': 'projectGroupCodeTaken',
	'projectGroup.name:required': 'projectGroupNameRequired',
	'customerCode.code:taken': 'customerCodeUsedBefore',
	'modelGroup.name:required': 'modelGroupNameRequired',
	'modelGroup.name:taken': 'modelGroupNameTaken',
	'model.name:required': 'modelNameRequired',
	'model.name:taken': 'modelNameTaken',
	'partNumber.number:required': 'partNumberRequired',
	'partNumber.number:taken': 'partNumberTaken',
	'process.name:required': 'processNameRequired',
	'process.name:taken': 'processNameTaken',
	'documentType.name:taken': 'documentTypeNameTaken',
	'specialCode.code:taken': 'specialCodeTaken',
	'specialCode.code:invalid_format': 'specialCodeInvalid',
	'changeList.name:taken': 'changeListNameTaken',
	'position.name:taken': 'positionNameTaken',
	'holder.project_group_id:required': 'holderGroupRequired',
	'holder.project_group_id:taken': 'holderGroupTaken',
	'holder.project_group_id:not_allowed': 'holderGroupNotAllowed',
	'holder.position_id:taken': 'holderPositionTaken',
	'route.name:taken': 'routeNameTaken',
	'formatTemplate.name:required': 'formatNameRequired',
	'formatTemplate.name:taken': 'formatNameTaken',
	'formatTemplate.document_type_id:not_allowed': 'formatTypeNotAllowed',
	'processTemplate.process_id:required': 'processTemplateProcessRequired',
	'processTemplate.process_id:taken': 'processTemplateTaken',
	'processTemplate.name:required': 'subprocessNameRequired',
	'processTemplate.name:taken': 'subprocessNameTaken',
	'processTemplate.name:not_allowed': 'subprocessLimit',
	'processTemplate.text:required': 'pointTextRequired',
	'processTemplate.subprocess_id:invalid_format': 'pointSubprocessInvalid',
	'processTemplate.subprocess_id:not_allowed': 'pointLimit',
	'processTemplate.document_type_id:required': 'pointTypeRequired',
	'processTemplate.document_type_id:not_allowed': 'pointTypeNotAllowed',
	// Cakupan poin dan varian (M3.3, D-129).
	'processTemplate.scope_level:invalid_format': 'scopeLevelInvalid',
	'processTemplate.scope_level:not_allowed': 'variantNotNarrower',
	'processTemplate.scope_target_id:required': 'scopeTargetRequired',
	'processTemplate.scope_target_id:invalid_format': 'scopeTargetInvalid',
	'processTemplate.scope_target_id:not_allowed': 'variantOutsidePoint',
	'processTemplate.scope_target_id:taken': 'variantTaken',
	'processTemplate.model_id:invalid_format': 'previewModelRequired',
	// Versi terbit dan versi baru (M3.4): status versi tidak cocok dengan aksinya.
	'processTemplate.status:not_allowed': 'versionStatusNotAllowed',
	'formatTemplate.status:not_allowed': 'versionStatusNotAllowed',
	// Form set (langkah D, D-139).
	'formSet.name:required': 'formSetNameRequired',
	'formSet.name:taken': 'formSetNameTaken',
	// Master dokumen terkait (D-130).
	'refDoc.number:taken': 'refDocNumberTaken',
	'refDoc.number:invalid_format': 'refDocNumberInvalid',
	'refDoc.process_id:taken': 'refDocLinkTaken',
	'formSet.pfd_format_version_id:required': 'formSetSheetRequired',
	'formSet.pfmea_format_version_id:required': 'formSetSheetRequired',
	'formSet.cp_format_version_id:required': 'formSetSheetRequired',
	'formSet.pfd_format_version_id:not_allowed': 'formSetSheetWrongType',
	'formSet.pfmea_format_version_id:not_allowed': 'formSetSheetWrongType',
	'formSet.cp_format_version_id:not_allowed': 'formSetSheetWrongType',
	'formSet.sheets:not_allowed': 'formSetSheetsNotPublished',
	'formSet.approver:not_allowed': 'formSetSameApprover',
	'formSet.approver:taken': 'approvalTaken',
	'formatTemplate.approver:taken': 'approvalTaken',
	'formSet.status:not_allowed': 'versionStatusNotAllowed',
	// Dokumen (M4.1): pesan dari server dipakai bila tidak ada teks khusus.
	'formatTemplate.approver:not_allowed': 'formSetSameApprover'
};

// problemText memberi teks layar untuk satu masalah input: teks khusus
// formulir bila ada, lalu teks umum field itu. Masalah yang belum ada di
// kedua daftar memakai pesan dari server.
export function problemText(detail: ErrorDetail, scope?: ProblemScope): string {
	const field = `${detail.field}:${detail.code}`;
	const key = (scope && scopedKeys[`${scope}.${field}`]) || problemKeys[field];
	return key ? t(key) : detail.message;
}

// failureText memberi teks layar untuk kegagalan yang bukan masalah per field.
export function failureText(failure: ApiFailure): string {
	if (failure.status === 0) {
		return t('errorNetwork');
	}
	if (failure.body?.code === 'forbidden') {
		return t('errorForbidden');
	}
	if (failure.body) {
		// Request ID membantu admin menemukan detail error di log server.
		return t('errorUnexpected', { requestId: failure.body.request_id });
	}
	return t('errorUnknown');
}
