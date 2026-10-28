// Teks layar dalam bahasa Inggris (Q-14, keputusan pemilik 28 Oktober 2026).
//
// Setiap teks punya kunci tetap. Kode layar tidak menulis teks langsung, tapi
// memanggil t('kunci'). Untuk bahasa lain cukup dibuat file baru dengan kunci
// yang sama, misalnya id.ts untuk bahasa Indonesia (lihat index.ts).
// Tempat seperti {code} diisi saat dipanggil: t('customerSaved', { code: 'AB' }).
//
// ASUMSI E-19: istilah Inggris diambil dari brief bagian 3 dan 4 (dan dari form
// serta WI), sampai glosarium U1 punya kolom istilah layar berbahasa Inggris.
export const en = {
	appTitle: 'PFD, PFMEA, and CP System',
	navHome: 'Home',
	navCustomers: 'Customers',
	navAbout: 'About',
	homeTitle: 'Home',
	healthChecking: 'Checking the server…',
	healthOk: 'Server status: OK',
	healthDatabaseDown: 'Server status: running, but the database is not reachable',
	healthError: 'Server status: not reachable',
	healthRetry: 'Check again',
	aboutTitle: 'About',
	aboutText:
		'An internal web system to prepare, revise, number, print, and record the release of PFD, PFMEA, and Control Plan documents. Approval stays on paper with wet signatures.',
	customersTitle: 'Customers',
	customersListTitle: 'All customers',
	customersLoading: 'Loading customers…',
	customersEmpty: 'No customers yet.',
	customerCode: 'Customer code',
	customerCodeHint: 'Two letters A to Z, as used in document numbers such as SC-AB-001.',
	customerName: 'Customer name',
	customerCreated: 'Added',
	customerAddTitle: 'Add a customer',
	customerSave: 'Save',
	customerSaving: 'Saving…',
	customerSaved: 'Customer {code} was added.',
	customerCodeRequired: 'Enter the customer code.',
	customerCodeInvalid: 'Use two letters A to Z.',
	customerCodeReserved: 'ZZ is reserved for general documents.',
	customerCodeTaken: 'This code is already used by another customer.',
	customerNameRequired: 'Enter the customer name.',
	customerNameTooLong: 'Use at most 200 characters.',
	customerNameInvalid: 'Use one line without control characters.',
	errorNetwork: 'The server cannot be reached. Check that it is running.',
	errorUnexpected: 'Something went wrong. Give this request ID to the administrator: {requestId}',
	errorUnknown: 'Something went wrong.'
};

// MessageKey adalah semua kunci teks yang tersedia.
export type MessageKey = keyof typeof en;

// Messages adalah bentuk satu katalog bahasa: setiap kunci wajib punya teks.
export type Messages = Record<MessageKey, string>;
