// Buku panduan pengguna, bahasa Indonesia (D-141). Bentuk dan penandanya
// dijelaskan di content.ts. Setiap perubahan di sini dibuat juga di en.ts.
import type { HelpContent } from './content.ts';

export const helpId: HelpContent = {
	title: 'Panduan pengguna',
	intro:
		'Panduan ini mengikuti urutan kerja yang sebenarnya: admin menyiapkan akun dan master data, pengelola template menyusun template, dan penyusun mendaftarkan nomor dokumen. Nama yang dicetak tebal adalah nama yang Anda lihat di layar.',
	contentsTitle: 'Daftar isi',
	whoLabel: 'Siapa',
	tipsTitle: 'Perlu diketahui',
	print: 'Cetak panduan ini',
	chapters: [
		{
			id: 'login',
			title: 'Login dan mengganti kata sandi',
			who: 'Semua pengguna',
			intro: 'Semua halaman butuh login. Akun baru mendapat kata sandi sementara dari admin dan wajib menggantinya saat login pertama.',
			steps: [
				'Buka alamat sistem di browser. Halaman {loginTitle} tampil.',
				'Isi {loginName} dan {loginPassword}, lalu tekan {loginSubmit}.',
				'Saat login pertama, halaman {passwordTitle} terbuka sendiri. Isi {passwordCurrent} dengan kata sandi sementara.',
				'Isi {passwordNew} dan {passwordConfirm} dengan kata sandi Anda sendiri, minimal 8 karakter, lalu tekan {passwordSave}.',
				'Untuk mengganti kata sandi di lain waktu, tekan {navAccount} di bilah atas.',
				'Bila sudah selesai, tekan {logout}.'
			],
			image: 'login',
			imageAlt: 'Halaman login dengan isian nama login dan kata sandi.',
			tips: [
				'Kalimat pendek mudah diingat dan sulit ditebak, misalnya empat kata sehari-hari.',
				'Kata sandi yang memuat nama login, nama Anda, atau kata sandi yang sangat umum akan ditolak.',
				'Setelah 10 kali salah kata sandi berturut-turut, akun dikunci 15 menit.'
			]
		},
		{
			id: 'screen',
			title: 'Layar: menu, bahasa, dan notifikasi',
			who: 'Semua pengguna',
			intro: 'Bilah atas sama di setiap halaman. Menu hanya menampilkan halaman yang boleh dibuka akun Anda.',
			steps: [
				'Menu punya kelompok: {navHome}, {navGroupDocuments}, {navGroupTemplates}, {navGroupMaster}, dan, untuk admin, {navGroupAdmin}. Tekan satu kelompok untuk melihat halamannya.',
				'Di HP atau tablet menu dilipat. Tekan {navMenu} untuk membukanya.',
				'Tekan [[EN]] atau [[ID]] di bilah atas untuk mengganti bahasa layar.',
				'Setelah Anda menyimpan sesuatu, pesan muncul di halaman dan sebagai pop-up di pojok kanan atas. Hijau berarti berhasil; merah berarti gagal. Klik pop-up untuk menutupnya.',
				'Tekan {navHelp} di bilah atas untuk membuka panduan ini.'
			],
			image: 'home',
			imageAlt: 'Halaman beranda dengan kelompok menu di bilah atas.'
		},
		{
			id: 'customers',
			title: 'Project group, customer, dan model',
			who: 'Admin',
			intro: 'Customer dan model dipakai di semua bagian lain, jadi dibuat lebih dulu. Project group harus ada sebelum customernya.',
			steps: [
				'Buka {navGroupMaster}, lalu {navCustomers}.',
				'Di bagian {projectGroupsTitle}, isi {projectGroupCode} (misalnya [[GR1]]) dan {projectGroupName}, lalu tekan {projectGroupAdd}.',
				'Di bagian {customerAddTitle}, isi {customerCode} dengan dua huruf, isi {customerName}, pilih {customerProjectGroup}, lalu tekan {customerSave}.',
				'Buka {navGroupMaster}, lalu {navModels}, dan pilih {modelsCustomer}.',
				'Isi {modelGroupName} dan tekan {modelGroupAdd}. Group model boleh tidak dibuat.',
				'Isi {modelName}, pilih {modelGroup} atau biarkan tanpa group, lalu tekan {modelAdd}.',
				'Untuk menambah part number, pilih modelnya, isi {partNumber}, lalu tekan {partNumberAdd}.'
			],
			image: 'customers',
			imageAlt: 'Halaman Customers dengan project group dan daftar customer.',
			tips: [
				'Kode customer menjadi bagian dari setiap nomor dokumen, misalnya SC-AB-001.',
				'Customer tidak pernah dihapus. Pakai {customerEdit} untuk menonaktifkannya, atau {customerChangeCode} untuk mengganti kodenya; kode lama tetap tercatat.'
			]
		},
		{
			id: 'accounts',
			title: 'Akun dan peran',
			who: 'Admin',
			intro: 'Setiap orang bekerja dengan akunnya sendiri. Peran menentukan apa yang boleh dilakukan sebuah akun; semua akun sudah bisa melihat.',
			steps: [
				'Buka {navGroupAdmin}, lalu {navAccounts}.',
				'Pilih {accountPersonNew} dan isi {accountPersonName}. Untuk orang yang sudah punya akun, pilih {accountPersonExisting}.',
				'Pilih {accountKindWork} untuk pekerjaan sehari-hari. Pilih {accountKindAdmin} hanya untuk tugas admin.',
				'Isi {accountLogin}, misalnya [[ppap.engineer]].',
				'Centang {accountRoles} orang itu dan pilih cakupan setiap peran: {scopeAll} atau {scopeOwnProjectGroup}.',
				'Tekan {accountCreate}.',
				'Kata sandi sementara tampil di bagian atas halaman dan di pop-up, dan otomatis disalin ke clipboard. Tempel di tempat Anda mengirimkannya ke orangnya, atau tekan {accountCopyPassword}.',
				'Di daftar akun, pakai {accountEditRoles}, {accountResetPassword}, atau {accountDisable} untuk akun yang sudah ada.'
			],
			image: 'accounts',
			imageAlt: 'Halaman Accounts dengan kotak kata sandi sementara di atas formulir.',
			tips: [
				'{roleAuthor}: mendaftarkan nomor dokumen. {roleTemplateMaintainer}: mengelola proses dan process template. {roleDocumentControl}: membatalkan nomor, menjadikannya obsolete, dan mengimpor nomor lama.',
				'{roleVerifier}, {rolePfmeaCpChecker}, dan {roleMigration} untuk layar yang dibangun nanti.',
				'Kata sandi sementara hanya tampil satu kali. Bila hilang, atur ulang kata sandi akun itu.',
				'Akun admin tidak memegang peran kerja. Pemegangnya memakai akun kerja terpisah untuk pekerjaan sehari-hari.',
				'Akun tidak pernah dihapus, hanya dinonaktifkan.'
			]
		},
		{
			id: 'positions',
			title: 'Jabatan dan rute approval',
			who: 'Admin',
			intro: 'Jabatan mengikuti bagan organisasi. Rute approval berisi siapa yang menandatangani dokumen cetak, sesuai urutan tanda tangan.',
			steps: [
				'Buka {navGroupMaster}, lalu {navPositions}.',
				'Pilih {positionName} dan {holderPerson}. Untuk Project Leader dan Project Engineer pilih juga {holderGroup}. Lalu tekan {holderAdd}.',
				'Buka {navGroupMaster}, lalu {navApprovalRoutes}, untuk melihat rute. Tekan {routeEdit} untuk mengubah slot sebuah rute.',
				'Di halaman yang sama, pilih rute bawaan setiap customer.'
			],
			image: 'positions',
			imageAlt: 'Halaman Positions dengan jabatan dan pemegangnya.',
			tips: [
				'Akun dengan cakupan {scopeOwnProjectGroup} hanya bekerja untuk customer di project group tempat orangnya tercatat sebagai Project Leader atau Project Engineer.'
			]
		},
		{
			id: 'format-templates',
			title: 'Format template',
			who: 'Admin',
			intro: 'Format template mengatur field header dan kolom form PFD, PFMEA, atau Control Plan. Sistem dimulai dengan form yang dipakai sekarang.',
			steps: [
				'Buka {navGroupTemplates}, lalu {navFormatTemplates}, dan pilih satu di {formatChoose}.',
				'Tekan {formatEdit} pada satu field untuk mengganti labelnya, menyembunyikannya, atau memindahkannya. Field disembunyikan, tidak pernah dihapus.',
				'Di bagian {formatAddFieldTitle}, tambahkan field header atau kolom baru, lalu tekan {formatAddField}.',
				'Di bagian {formatRatingsTitle}, isi kriteria severity, occurrence, dan detection dari form Anda, lalu tekan {formatRatingsSave}.',
				'Template dengan metode risiko AP menampilkan {apTableTitle}. Pakai {apTryTitle} untuk melihat prioritas satu severity, occurrence, dan detection.',
				'Untuk membuat format baru, pakai {formatCreateTitle} di bagian bawah halaman. Pilih {formatCopyFrom} untuk memulai dari format yang sudah ada.'
			],
			image: 'format-templates',
			imageAlt: 'Halaman Format templates dengan kolom form PFMEA.',
			tips: [
				'Sistem dimulai dengan dua set form: PFMEA AIAG 4th (RPN) dengan Control Plan yang dipakai sekarang, dan PFMEA AIAG-VDA 2019 (Action Priority) dengan Control Plan AIAG 2024. Form PFD sama untuk keduanya.',
				'Nomor operasi, nama proses, process flow, nomor dokumen, RPN, dan Action Priority diisi sistem dan tidak pernah diketik.',
				'Mengganti label tidak mengubah data dokumen, karena dokumen menunjuk ID tetap sebuah field.'
			]
		},
		{
			id: 'processes',
			title: 'Proses',
			who: 'Pengelola template',
			intro: 'Daftar proses adalah master nama proses. Dokumen menunjuk proses lewat ID tetapnya, jadi nama proses aman diganti.',
			steps: [
				'Buka {navGroupTemplates}, lalu {navProcesses}.',
				'Isi {processName} dengan nama yang dipakai di work instruction.',
				'Di {processAliases}, tulis nama lain proses yang sama, satu per baris. Ini boleh dikosongkan.',
				'Tekan {processAdd}.',
				'Tekan {processEdit} pada satu proses untuk mengganti nama atau aliasnya.'
			],
			image: 'processes',
			imageAlt: 'Halaman Processes dengan daftar proses dan aliasnya.'
		},
		{
			id: 'process-templates',
			title: 'Process template: sub-proses, poin, dan varian',
			who: 'Pengelola template',
			intro: 'Process template berisi sub-proses satu proses dan poinnya untuk PFD, PFMEA, dan Control Plan. Satu proses punya satu template.',
			steps: [
				'Buka {navGroupTemplates}, lalu {navProcessTemplates}, dan pilih {processTemplateChoose}.',
				'Bila proses itu belum punya template, tekan {processTemplateCreate}.',
				'Isi {subprocessName} dan tekan {subprocessAdd} untuk setiap sub-proses.',
				'Tekan {subprocessShowPoints} pada satu sub-proses untuk melihat dan menambah poinnya.',
				'Pilih {pointDocumentType}, isi {pointText}, dan pilih {pointScope}: {scopeGeneral} bila poin berlaku untuk semua model, atau customer, group model, atau model tertentu. Lalu tekan {pointAdd}.',
				'Bila satu customer atau model butuh teks berbeda, tekan {variantAdd} pada poinnya, pilih cakupan yang lebih sempit, isi {variantText}, dan tekan {variantAdd} lagi.',
				'Di bagian {previewTitle}, pilih customer dan model untuk melihat poin yang didapat model itu.'
			],
			image: 'process-templates',
			imageAlt: 'Halaman Process templates dengan satu poin dan variannya untuk satu model.',
			tips: [
				'Teks yang paling spesifik menang: model, lalu group model, lalu customer, lalu umum.',
				'Cakupan poin tidak bisa diubah kemudian. Bila salah, sembunyikan poinnya dan tambahkan poin baru.',
				'Sub-proses dan poin disembunyikan, tidak pernah dihapus.'
			]
		},
		{
			id: 'versions',
			title: 'Menerbitkan versi dan membuat versi baru',
			who: 'Pengelola template, dan admin untuk format template',
			intro: 'Template baru berstatus draft. Draft bebas diubah. Versi yang sudah terbit tidak bisa diubah lagi, jadi perubahan sesudahnya masuk ke versi baru.',
			steps: [
				'Buka templatenya dan lihat baris {versionChoose} di atas isinya.',
				'Tekan [[Terbitkan versi 1]] dan setujui. Labelnya berubah menjadi {versionStatusPublished}, dan tombol tambah serta ubah hilang.',
				'Untuk mengubah template yang sudah terbit, tekan {versionNew}. Sistem membuat draft berikutnya sebagai salinan versi terbit.',
				'Ubah draft itu, lalu terbitkan dengan cara yang sama.',
				'Pakai kotak {versionChoose} untuk membuka versi lama. Versi lama hanya bisa dibaca.'
			],
			image: 'versions',
			imageAlt: 'Baris versi process template yang sudah terbit, dengan tombol versi baru.',
			tips: [
				'Satu template paling banyak punya satu draft, jadi {versionNew} hanya muncul bila versi terakhir sudah terbit.',
				'Versi baru memakai ID field dan poin yang sama, jadi dokumen tetap tersambung.'
			]
		},
		{
			id: 'numbers',
			title: 'Mendaftarkan nomor dokumen',
			who: 'Penyusun',
			intro: 'Sistem yang memberi setiap nomor PFD, PFMEA, dan Control Plan. Nomor tidak pernah dipakai ulang atau dihapus.',
			steps: [
				'Buka {navGroupDocuments}, lalu {navNumbers}.',
				'Di bagian {numberRegisterTitle}, pilih {numberCustomer}.',
				'Biarkan kotak set tercentang untuk mendapat satu nomor berjalan bagi PFD, PFMEA, dan Control Plan sekaligus. Hilangkan centangnya untuk mendaftarkan satu nomor, lalu pilih {numberType}.',
				'Biarkan {numberInitialRevision} tetap 0, kecuali dokumen lama masuk di revisi yang lebih tinggi.',
				'Tekan {numberRegister}. Nomor baru muncul di register dengan status {numberStatusReserved}.',
				'Untuk mencari nomor, pakai {numberFilterStatus} atau {numberFilterSearch} di atas register, lalu tekan {numberFilterApply}.'
			],
			image: 'numbers',
			imageAlt: 'Halaman Numbers dengan formulir pendaftaran dan register.',
			tips: [
				'Bila satu set melewati satu nomor supaya ketiga dokumen tetap sejajar, nomor yang dilewati tetap ada di register sebagai {numberStatusSkipped}.'
			]
		},
		{
			id: 'document-control',
			title: 'Void, obsolete, dan nomor lama',
			who: 'Dokumen kontrol',
			intro: 'Dokumen kontrol mengubah status nomor dan memasukkan nomor dari register lama.',
			steps: [
				'Buka {navGroupDocuments}, lalu {navNumbers}.',
				'Pada baris nomornya, tekan {numberVoid} atau {numberObsolete}, isi alasannya, dan tekan {numberStatusConfirm}.',
				'Untuk memasukkan nomor lama, buka {navGroupDocuments}, lalu {numberImportLink}.',
				'Pilih customer dan {numberImportStatus}, lalu tempel nomornya di {numberImportText}.',
				'Tekan {numberImportPreview} dan baca laporannya: baris mana yang akan diimpor, mana yang ditolak dan alasannya.',
				'Bila laporannya sudah benar, tekan {numberImportRun}.'
			],
			tips: ['Nomor yang dibatalkan tetap ada di register beserta alasannya. Nomor itu tidak pernah diberikan ke dokumen lain.']
		},
		{
			id: 'audit',
			title: 'Audit log',
			who: 'Admin',
			intro: 'Setiap perubahan data dicatat: siapa, kapan, dan apa. Tidak ada yang bisa mengubah atau menghapus catatan ini.',
			steps: [
				'Buka {navGroupAdmin}, lalu {navAuditLog}.',
				'Baca daftarnya dari entri terbaru. Setiap entri menampilkan akun dan orang pemegangnya.',
				'Tekan {auditVerify} untuk memeriksa bahwa tidak ada entri yang diubah.'
			],
			image: 'audit',
			imageAlt: 'Halaman Audit log dengan daftar entri.'
		},
		{
			id: 'problems',
			title: 'Bila ada masalah',
			who: 'Semua pengguna',
			intro: 'Sebagian besar masalah punya salah satu penyebab berikut.',
			steps: [
				'Menu atau tombol tidak ada: akun Anda tidak punya peran itu. Minta admin menambahkan perannya.',
				'Lupa kata sandi, atau akun terkunci: minta admin menekan {accountResetPassword} untuk akun Anda.',
				'Pesan merah muncul di bawah satu isian: betulkan isian itu dan simpan lagi. Belum ada yang tersimpan.',
				'Pesan menyebut versi tidak bisa diubah: versinya sudah terbit. Tekan {versionNew} dan ubah draft barunya.',
				'Halaman terlihat lama setelah pembaruan: muat ulang halaman di browser.'
			]
		}
	]
};
