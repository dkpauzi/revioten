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
	'model_id:not_allowed': 'modelCustomerInactive'
};

// ProblemScope adalah formulir tempat masalah ditampilkan. Field yang sama bisa
// butuh teks berbeda, misalnya code:taken di formulir Project Group.
export type ProblemScope = 'projectGroup' | 'customerCode' | 'modelGroup' | 'model' | 'partNumber';

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
	'partNumber.number:taken': 'partNumberTaken'
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
