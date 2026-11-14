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
				'{roleFormApprover}: menerbitkan form set dan tabel AP; untuk saat ini PPAP Supervisor. {roleVerifier}, {rolePfmeaCpChecker}, dan {roleMigration} untuk layar yang dibangun nanti.',
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
'Tekan {pointContent} pada poin atau varian untuk mengisi apa yang ditulisnya ke setiap kolom form, misalnya failure mode, akibatnya, dan severity untuk poin PFMEA. Kolom yang dimiliki dua form cukup diisi sekali dan berlaku untuk keduanya. Nilai inilah yang menjadi isi awal baris dokumen saat poin dipakai di dokumen.',
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
				'Versi baru memakai ID field dan poin yang sama, jadi dokumen tetap tersambung.',
				'Sesudah versi process template terbit, {propagationPreview} di bagian propagasi menunjukkan dokumen mana yang memakai template itu, baris mana yang berubah, dan mana yang sudah diubah lokal. Propagasi membuat satu perubahan per customer yang menunggu verifikasi; isi baru masuk ke draft saat perubahan diterima.'
			]
		},
		{
			id: 'form-sets',
			title: 'Form set',
			who: 'Admin, dan penyetuju form untuk menerbitkan',
			intro: 'Form set adalah satu paket form untuk PFD, PFMEA, dan Control Plan. Sistem punya dua: set AIAG 4th (RPN) dan set AIAG-VDA (Action Priority). Setiap customer memakai satu form set untuk dokumen barunya.',
			steps: [
				'Buka {navGroupTemplates}, lalu {navFormSets}.',
				'Setiap form set menampilkan tiga sheet-nya: {formSetSheetPfd}, {formSetSheetPfmea}, dan {formSetSheetCp}. Admin mengganti sheet versi draft dan menekan {formSetSaveSheets}.',
				'Terbitkan dulu format template ketiga sheet itu, di {navFormatTemplates}.',
				'Penyetuju form membuka form set itu dan menekan [[Terbitkan versi 1]]. Orang yang menyusun draft tidak boleh menerbitkannya. Bawaannya semua penyetuju form harus setuju; daftarnya menampilkan {approvalsDone} dan {approvalsPending}. Admin bisa mengubah aturan ini di {navSettings}.',
				'Di bagian {formSetCustomersTitle}, admin memilih form set setiap customer. Tanpa pilihan, customer memakai form set bawaan.',
				'Untuk membuat paket baru, pakai {formSetCreateTitle} di bagian bawah halaman.'
			],
			image: 'form-sets',
			imageAlt: 'Halaman Form set dengan dua form set dan form set per customer.',
			tips: [
				'Format PFMEA dengan Action Priority juga diterbitkan penyetuju form, karena tabel AP-nya menentukan prioritas tindakan.',
				'Admin bisa mengubah tabel AP versi draft: tekan {apEdit} di format template, klik satu prioritas untuk mengubahnya, lalu tekan {apSave}.',
				'Berikan peran {roleFormApprover} ke akun lain di {navAccounts} bila penyetujunya bertambah, misalnya para Project Leader.'
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
			id: 'documents',
			title: 'Dokumen',
			who: 'Penyusun',
			intro: 'Dokumen adalah satu PFD, PFMEA, atau Control Plan dengan nomornya sendiri. Dokumen memakai form set customer-nya yang sudah terbit, jadi langkah terbit di bab Form set dikerjakan lebih dulu.',
			steps: [
				'Daftarkan dulu nomornya di {navNumbers}. Dokumen butuh nomor berstatus {numberStatusReserved}.',
				'Buka {navGroupDocuments}, lalu {navDocuments}, dan pilih {documentCustomer}.',
				'Di bagian {documentCreateTitle}, pilih {documentNumber}, centang {documentModels} yang dicakup dokumen, lalu tekan {documentCreate}. Nomornya menjadi In use.',
				'Buka dokumen itu dari daftar. Isi field header dan tekan {documentHeaderSave}.',
				'Tekan {documentRowAdd} untuk menambah baris kosong di akhir sheet.',
'Bila satu model punya beberapa PCB dengan alur proses sendiri dalam satu nomor dokumen (misalnya PCB A, Heatsink Assy R/L, dan Base Assy R/L), tekan {sectionSplit}, beri PCB pertama nama dan part number-nya, lalu {sectionAdd} untuk PCB berikutnya. PFD, PFMEA, dan CP satu set mendapat bagian yang sama. Pilih bagian lewat tabnya di atas tabel: tabel, {documentRowAdd}, {composeOpen}, {orderOpen}, tab {checkTab}, dan diagram lalu berlaku untuk bagian itu saja, dan nomor operasi mulai lagi dari 10 di setiap bagian. Baris yang sudah ada masuk ke bagian pertama. Hanya bagian kosong yang bisa dihapus.',
'Untuk membawa baris dari file Excel, tekan {pasteOpen}. Salin sel di Excel, tempel di kotaknya, lalu tekan {pastePreview}. Centang {pasteHasHeader} bila baris judul ikut disalin; kolom lalu dicocokkan menurut judulnya. Di atas setiap kolom Anda memilih ke mana kolom itu masuk. Sel yang tidak cocok dengan kolomnya, misalnya severity yang bukan angka, ditandai dan dikosongkan; barisnya tetap ditambahkan. Belum ada yang disimpan sampai Anda menekan tombol Impor.',
'Untuk menyusun baris dari pustaka proses tanpa mengetiknya, tekan {composeOpen}. Pilih proses di sebelah kiri dan centang sub-proses serta poin yang dipakai model ini; sebelah kanan menampilkan baris yang akan didapat setiap dokumen di set. Tekan {composeApply}. Poin yang dicentang menjadi baris di PFD, PFMEA, atau CP yang tetap tertaut ke template. Poin tanpa centang dicatat sebagai tidak dipakai, dan alasannya boleh ditulis di sebelahnya. Poin yang ditambahkan versi template yang lebih baru bertanda {composeUndecided} sampai Anda memutuskannya.',
'Kolom Relevant Document di Control Plan diisi dari daftar di Template library, {navReferencedDocuments}. Di sana pengelola template mencatat setiap WI, SOP, atau QEP sekali (nomor, judul, level, revisi) dan, lewat {refDocLinks}, menjadikannya bawaan proses yang memakainya, untuk semua model atau hanya untuk satu customer, group model, atau model. Saat Anda merakit, baris pertama setiap proses mendapat nomor-nomor itu, satu per baris. Sel yang sudah berisi tidak pernah ditimpa. Untuk baris yang Anda ketik sendiri, tekan {refFill}.',
'Model baru yang mirip model yang sudah ada tidak perlu mulai dari nol. Selama dokumen belum punya baris, tekan {copyOpen}, pilih dokumen model yang serupa, lalu tekan {copyRun}. Baris, urutannya, dan poin yang tidak dipakai disalin untuk seluruh set (PFD, PFMEA, dan CP). Sesudah itu salinan berdiri sendiri: mengubahnya tidak mengubah model lain itu.',
'Baris yang dirakit dari process template tetap tertaut ke template itu. Tekan {compareOpen} untuk melihat, baris demi baris, apakah baris masih sama dengan poin template-nya atau sudah diubah di dokumen ini; untuk baris yang diubah, kolom yang berbeda ditampilkan dengan nilai template dan nilai baris. Bila satu baris harus berdiri sendiri, tekan {compareDetach}, tulis alasannya, lalu konfirmasi. Baris yang tautannya diputus tetap ada di dokumen, tetapi tidak dibandingkan lagi.',
'Bila set dokumen punya PFMEA dan CP, tab sheet diakhiri {checkTab}. Tab ini menampilkan kedua dokumen berdampingan, satu proses demi satu proses, supaya Anda bisa memastikan setiap failure mode dan kontrol di PFMEA tercakup di CP. Sesudah itu PPAP Engineer menekan {checkMark}. Tanda itu menunjukkan siapa yang mengecek dan kapan, dan berlaku untuk revisi saat ini. Bila nanti ada baris proses itu yang ditambah, diubah, atau dihapus, tanda menyebut proses berubah sesudah dicek; cek lagi lalu tekan {checkMark} sekali lagi.',
'Di PFD, tekan {diagramOpen} untuk melihat alur proses yang digambar dari barisnya: satu kotak per baris menurut urutan, bentuknya mengikuti Flow Symbol, dengan NG Disposition di samping langkahnya. Diagram hanya untuk dilihat; ubah barisnya di tabel dan diagram ikut berubah.',
'Untuk mengubah urutan proses, tekan {orderOpen}. Setiap butir adalah satu langkah: baris bersebelahan dengan proses yang sama. Seret langkah ke tempat barunya, atau pakai tombol panah, lalu tekan {orderSave}. Bila penomoran otomatis dicentang, langkah diberi nomor 10, 20, 30, dan seterusnya; nomor yang Anda ketik sendiri di tabel tidak diubah. Untuk kembali ke nomor otomatis, kosongkan selnya lalu simpan urutan lagi.',
'Dua orang bisa bekerja di dokumen yang sama. Bila orang lain lebih dulu menyimpan baris yang sama, perubahan Anda tidak disimpan: pesan muncul, isi terbaru ditampilkan, dan Anda memasukkan lagi perubahan Anda.',
				'Klik satu sel lalu ketik, seperti di Excel. Tekan Enter atau klik di tempat lain, dan barisnya tersimpan. Di kolom nomor operasi dan proses, Anda memilih nilai milik baris itu sendiri.',
				'Tekan {documentRowDelete} pada satu baris untuk menghapusnya. Baris sesudahnya bergeser naik.',
				'Tab di atas tabel membuka dokumen lain di set yang sama: PFD, PFMEA, dan Control Plan.'
			],
			tips: [
				'Nomor operasi dan nama proses adalah kolom yang diisi dari baris; keduanya tidak diketik di kolomnya sendiri.',
				'RPN dan Action Priority dihitung dan tidak bisa diketik.',
				'Kolom abu-abu diisi atau dihitung sistem dan tidak bisa diketik.'
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
			id: 'changes',
			title: 'Perubahan',
			who: 'Penyusun, Verifikator',
			intro: 'Setiap revisi PFD, PFMEA, atau CP berangkat dari satu perubahan: pemicunya, jenis perubahan, alasannya, dan dokumen yang terdampak.',
			steps: [
				'Buka {navGroupDocuments}, lalu {navChanges}, dan pilih customer.',
				'Isi {changeTrigger} (misalnya Customer claim), {changeType}, dan bila perlu {changeFourM} serta {changeReference}.',
				'Tulis {changeReason}, centang setiap dokumen terdampak di {changeDocuments}, lalu tekan {changeSubmit}.',
				'Perubahan mendapat nomor seperti [[CHG-0001]]. Buka perubahan itu untuk menambah atau melepas dokumen terdampak selama masih diajukan.',
				'Untuk revisi SQAM baru yang berlaku juga bagi dokumen yang sudah ada, pilih customer lalu tekan {sqamOpen}: tulis revisi SQAM, tanggal berlaku, jenis perubahan, dan alasannya. Perubahan ini mencakup semua dokumen aktif customer.',
				'Di {changeAttachments}, pilih berkas di {changeAttachmentAdd} lalu tekan {changeAttachmentUpload}, misalnya laporan 8D atau foto NG. Tekan nama berkas untuk mengunduhnya.',
				'Di {impactTitle}, catat untuk setiap dokumen terkait (PFD, PFMEA, CP, WI, standar inspeksi) apakah terdampak, dengan alasan bila tidak, lalu tekan {impactSave}. WI atau standar inspeksi dipilih dari master dokumen terkait, jadi nomornya sama persis dengan sumbernya; ketik sendiri hanya bila tidak ada di daftar. Daftar ini masih bisa dilengkapi sesudah perubahan diterima.',
				'Verifikator (Project Engineer atau Project Leader dari Project Group customer, atau APQP/PPAP Controller) membuka perubahan, menulis {changeVerifyNote}, lalu menekan {changeAccept} atau {changeReject}. Pemohon tidak bisa memverifikasi perubahannya sendiri.'
			],
			tips: ['Perubahan selalu menunjuk paling sedikit satu dokumen, jadi dokumen terakhir tidak bisa dilepas.', 'Lampiran paling besar 20 MB. Lampiran yang dilepas tetap tercatat di audit log.', 'Saat perubahan diterima, setiap dokumen terdampak mendapat draft: dokumen yang sudah punya draft terbuka menerima perubahan itu ke draft-nya, dan dokumen yang sudah terbit membuka revisi berikutnya. Revisi yang berlaku tetap sampai draft itu terbit.', 'Sesudah revisi ditandatangani di kertas, pemegang dokumen kontrol membuka dokumen dan mencatat rilisnya di bagian Revisi. Isinya dibekukan dengan hash SHA-256; orang yang menyusun draft tidak bisa mencatat rilisnya.', 'Di Bandingkan revisi, pilih revisi terbit dan draft terbuka atau revisi terbit lain, dan bila perlu satu proses, untuk melihat setiap baris yang ditambah, dihapus, diubah, atau dipindah beserta isi sel sebelum dan sesudahnya.']
		},
		{
			id: 'yokoten',
			title: 'Yokoten',
			who: 'Penyusun, Project Engineer, Project Leader',
			intro: 'Yokoten menyebarkan perbaikan yang sudah diterima ke model lain, dengan keputusan per model dan papan status.',
			steps: [
				'Buka perubahan yang sudah diterima, lalu tekan {yokotenSpread}.',
				'Pilih {yokotenTargets}: semua model customer, satu group model, atau model pilihan. Persempit dengan proses atau teks di baris, lalu tekan {yokotenSearch}.',
				'Periksa model dan dokumen yang ditemukan, tulis alasannya, lalu buat yokoten.',
				'Di papan status, pemilik setiap target menekan {yokotenDecision_apply}, {yokotenDecision_adapt}, atau {yokotenDecision_not_applicable} dengan alasan. Terapkan dan sesuaikan membuat perubahan untuk dokumen model itu, yang diverifikasi seperti perubahan lain.',
				'Ikuti kemajuan per model: menunggu verifikasi, dalam draft, terbit, atau tidak berlaku.'
			],
			tips: ['Model yang dokumennya sudah ada di perubahan sumber tidak ikut.', 'Setiap target hanya tampil untuk akun yang boleh melihat customer-nya.']
		},
		{
			id: 'printing',
			title: 'Cetak dan rilis',
			who: 'Penyusun, Dokumen kontrol, Admin untuk layout',
			intro: 'Cetakan berasal dari isi yang sama dengan layar, dalam Excel atau PDF, A3 atau A4 sesuai layout. Approval tetap tanda tangan basah di kertas.',
			steps: [
				'Buka dokumen. Di {printTitle}, pilih revisi dan {printPurpose}, lalu tekan {printExcel} atau {printPdf}.',
				'Untuk approval di kertas, cetak {printPurpose_signature}. Tanggal rilis terisi, dan cover memuat slot approval dari rute.',
				'Setelah semua slot ditandatangani, dokumen kontrol mengunggah scan di {revisions} dengan {scanUploadButton}, lalu mencatat rilis. Rilis ditolak tanpa scan.',
				'Revisi terbit dicetak sebagai {printPurpose_reference} atau {printPurpose_controlled}. Salinan terkendali bernomor, dan setiap cetakan tampil di {printLog}.',
				'Admin mengubah kertas, huruf, margin, dan lebar kolom di {navGroupTemplates}, lalu {navPrintLayouts}, dan bisa memberi satu customer layout lain.'
			],
			tips: ['Draft selalu dicetak dengan UNCONTROLLED COPY.', 'Baris yang berubah di revisi yang dicetak diberi segitiga dengan nomor revisi.', 'PDF butuh Chrome atau Edge di komputer server; Excel selalu bisa.']
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
