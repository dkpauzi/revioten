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
	'new_password:too_common': 'passwordTooCommon'
};

// problemText memberi teks layar untuk satu masalah input. Masalah yang belum
// ada di daftar memakai pesan dari server.
export function problemText(detail: ErrorDetail): string {
	const key = problemKeys[`${detail.field}:${detail.code}`];
	return key ? t(key) : detail.message;
}

// failureText memberi teks layar untuk kegagalan yang bukan masalah per field.
export function failureText(failure: ApiFailure): string {
	if (failure.status === 0) {
		return t('errorNetwork');
	}
	if (failure.body) {
		// Request ID membantu admin menemukan detail error di log server.
		return t('errorUnexpected', { requestId: failure.body.request_id });
	}
	return t('errorUnknown');
}
