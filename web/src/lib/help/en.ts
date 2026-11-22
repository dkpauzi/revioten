// Buku panduan pengguna, bahasa Inggris (D-141). Bentuk dan penandanya
// dijelaskan di content.ts. Setiap perubahan di sini dibuat juga di id.ts.
import type { HelpContent } from './content.ts';

export const helpEn: HelpContent = {
	title: 'User guide',
	intro:
		'This guide follows the order of real work: the admin prepares accounts, the PPAP Engineer prepares master data and templates, and then registers document numbers. Names in bold are the names you see on the screen.',
	contentsTitle: 'Contents',
	whoLabel: 'Who',
	tipsTitle: 'Good to know',
	print: 'Print this guide',
	chapters: [
		{
			id: 'login',
			title: 'Log in and change your password',
			who: 'Everyone',
			intro: 'Every page needs a login. A new account gets a temporary password from the admin and must change it at the first login.',
			steps: [
				'Open the address of the system in your browser. The {loginTitle} page appears.',
				'Fill in {loginName} and {loginPassword}, then press {loginSubmit}. Press the eye icon ({loginShow}) in the password box to check what you typed.',
				'At the first login the {passwordTitle} page opens by itself. Fill in {passwordCurrent} with the temporary password.',
				'Fill in {passwordNew} and {passwordConfirm} with your own password of at least 8 characters, then press {passwordSave}.',
				'To change your password later, press {navAccount} in the top bar.',
				'When you are done, press {logout}.'
			],
			image: 'login',
			imageAlt: 'The login page with the login name and password fields.',
			tips: [
				'A short sentence is easy to remember and hard to guess, for example four everyday words.',
				'A password with your login name, your name, or a very common password is refused.',
				'After 10 wrong passwords in a row the account is locked for 15 minutes.'
			]
		},
		{
			id: 'screen',
			title: 'The screen: menu, language, and notifications',
			who: 'Everyone',
			intro: 'The top bar is the same on every page. The menu only shows what your account may open.',
			steps: [
				'The menu has groups: {navHome}, {navGroupDocuments}, {navGroupTemplates}, {navGroupMaster}, and, for the admin, {navGroupAdmin}. Press a group to see its pages.',
				'On a phone or tablet the menu is folded. Press {navMenu} to open it.',
				'Press [[EN]] or [[ID]] in the top bar to change the language of the screen.',
				'After you save something, a message appears in the page and as a pop-up in the top right corner. Green means it worked; red means it did not. Click a pop-up to close it.',
				'Press {navHelp} in the top bar to open this guide.'
			],
			image: 'home',
			imageAlt: 'The home page with the menu groups in the top bar.'
		},
		{
			id: 'accounts',
			title: 'Accounts and roles',
			who: 'Admin',
			intro: 'Every person works with their own account. Roles decide what an account may do; every account can already view.',
			steps: [
				'Open {navGroupAdmin}, then {navAccounts}.',
				'Choose {accountPersonNew} and fill in {accountPersonName}. For someone who already has an account, choose {accountPersonExisting} instead.',
				'Choose {accountKindWork} for daily work. Choose {accountKindAdmin} only for admin tasks.',
				'Fill in {accountLogin}, for example [[ppap.engineer]].',
				'Tick the {accountRoles} of the person and choose the scope of each role: {scopeAll} or {scopeOwnProjectGroup}.',
				'Press {accountCreate}.',
				'The temporary password appears at the top of the page and in a pop-up, and is copied to the clipboard automatically. Paste it where you send it to the person, or press {accountCopyPassword}.',
				'In the list of accounts, use {accountEditRoles}, {accountResetPassword}, or {accountDisable} for an existing account.'
			],
			image: 'accounts',
			imageAlt: 'The Accounts page with the box of the temporary password above the form.',
			tips: [
				'{rolePpapEngineer}: composes, prints, records releases, manages master data, templates, and numbers, and enters legacy documents. {rolePpapEngineerSupervisor}: the same, and also verifies changes and publishes form sets and AP tables.',
				'{roleProjectLeader}: composes, prints, and verifies for the customers of the own project group; the scope is chosen automatically. {roleManager}: signs and views the performance report. An account without a role can view and add genba notes.',
				'The admin decides what each role holds under {navRoles}, and can add a role there.',
				'A {roleProjectLeader} is limited to the own project group by default. For a Project Leader who helps another group, choose {scopeAll}: the person can then also sign and verify for that group.',
				'The temporary password is shown only once. If it is lost, reset the password of the account.',
				'An admin account holds no work role. The person who holds it uses a separate work account for daily work.',
				'An account is never deleted, only disabled.'
			]
		},
		{
			id: 'customers',
			title: 'Project groups, customers, and models',
			who: 'PPAP Engineer',
			intro: 'Customers and models are used everywhere else, so they come first. A project group must exist before its customers.',
			steps: [
				'Open {navGroupMaster}, then {navCustomers}.',
				'Under {projectGroupsTitle}, fill in {projectGroupCode} (for example [[GR1]]) and {projectGroupName}, then press {projectGroupAdd}.',
				'Under {customerAddTitle}, fill in {customerCode} with two letters, fill in {customerName}, choose the {customerProjectGroup}, then press {customerSave}.',
				'Open {navGroupMaster}, then {navModels}, and choose the {modelsCustomer}.',
				'Fill in {modelGroupName} and press {modelGroupAdd}. A model group is optional.',
				'Fill in {modelName}, choose its {modelGroup} or leave it without a group, then press {modelAdd}.',
				'To add a part number, choose the model, fill in {partNumber}, then press {partNumberAdd}.'
			],
			image: 'customers',
			imageAlt: 'The Customers page with the project groups and the list of customers.',
			tips: [
				'The customer code becomes part of every document number, for example SC-AB-001.',
				'A customer is never deleted. Use {customerEdit} to make it inactive, or {customerChangeCode} to change its code; the old code stays on record.'
			]
		},
		{
			id: 'positions',
			title: 'Positions and approval routes',
			who: 'PPAP Engineer',
			intro: 'Positions follow the organization chart. Approval routes list who signs a printed document, in signing order.',
			steps: [
				'Open {navGroupMaster}, then {navPositions}.',
				'Choose the {positionName} and the {holderPerson}. For Project Leader and Project Engineer also choose the {holderGroup}. Then press {holderAdd}.',
				'Open {navGroupMaster}, then {navApprovalRoutes}, to see the routes. Press {routeEdit} to change the slots of a route.',
				'On the same page, choose the default route of each customer.'
			],
			image: 'positions',
			imageAlt: 'The Positions page with positions and their holders.',
			tips: [
				'An account with the scope {scopeOwnProjectGroup} only works for the customers of the project group where its person is recorded as Project Leader or Project Engineer.'
			]
		},
		{
			id: 'document-settings',
			title: 'Document settings',
			who: 'PPAP Engineer; everyone can see them',
			intro: 'Settings for documents and prints. The safeguards (migration mode and the rule for form approval) stay with the admin under {navGroupAdmin}, then {navSettings}.',
			steps: [
				'Open {navGroupMaster}, then {navDocumentSettings}.',
				'Tick {settingsSignDates} when every signature slot needs its own date. By default only the release date is recorded.',
				'Tick {settingsQr} to print a QR code that opens the verification page. By default it is off.',
				'Fill in {settingsSubmissionFormNumber}, for example [[QR-ENG-24-K015 (Rev.01)]].',
				'Press {settingsSave}.'
			],
			image: 'document-settings',
			imageAlt: 'The Document settings page with the sign date switch, the QR switch, and the form number.',
			tips: ['Every change is recorded in the audit log with the old and the new value.']
		},
		{
			id: 'format-templates',
			title: 'Format templates',
			who: 'PPAP Engineer',
			intro: 'A format template sets the header fields and columns of a PFD, PFMEA, or Control Plan form. The system starts with the forms in use.',
			steps: [
				'Open {navGroupTemplates}, then {navFormatTemplates}, and choose one under {formatChoose}.',
				'Press {formatEdit} on a field to change its label, hide it, or move it. A field is hidden, never deleted.',
				'Under {formatAddFieldTitle}, add a new header field or column, then press {formatAddField}.',
				'Under {formatRatingsTitle}, fill in the criteria of severity, occurrence, and detection from your form, then press {formatRatingsSave}.',
				'A template with the risk method AP shows the {apTableTitle}. Use {apTryTitle} to see the priority of one severity, occurrence, and detection.',
				'To make a new format, use {formatCreateTitle} at the bottom of the page. Choose {formatCopyFrom} to start from an existing one.'
			],
			image: 'format-templates',
			imageAlt: 'The Format templates page with the columns of the PFMEA form.',
			tips: [
				'The system starts with five forms: PFD v1, PFMEA v1 (AIAG 4th, RPN), and Control Plan v1 (the form in use today), plus PFMEA v2 (AIAG-VDA, Action Priority) and Control Plan v2 (AIAG 2024). PFMEA v2 and Control Plan v2 are hidden for now; tick {showHidden} to see them.',
				'{formatHide} takes a template out of new choices (sheets of a form set, copy sources). Documents that already use it do not change. {formatShow} brings it back.',
				'The operation number, the process name, the process flow, the document number, RPN, and Action Priority are filled in by the system and are never typed.',
				'Changing a label never changes the data of documents, because documents refer to the fixed ID of a field.'
			]
		},
		{
			id: 'processes',
			title: 'Processes',
			who: 'PPAP Engineer',
			intro: 'The process list is the master of process names. Documents refer to a process by its fixed ID, so a process can be renamed safely.',
			steps: [
				'Open {navGroupTemplates}, then {navProcesses}.',
				'Fill in {processName} with the name used in the work instruction.',
				'Under {processAliases}, write other names of the same process, one per line. This is optional.',
				'Press {processAdd}.',
				'Press {processEdit} on a process to change its name or aliases.'
			],
			image: 'processes',
			imageAlt: 'The Processes page with the list of processes and their aliases.'
		},
		{
			id: 'process-templates',
			title: 'Process templates: sub-processes, points, and variants',
			who: 'PPAP Engineer',
			intro: 'A process template lists the sub-processes of one process and their points for the PFMEA and the Control Plan. One process has one template. The PFD has no template points: it is typed per document.',
			steps: [
				'Open {navGroupTemplates}, then {navProcessTemplates}, and choose the {processTemplateChoose}.',
				'If the process has no template yet, press {processTemplateCreate}.',
				'Fill in {subprocessName} and press {subprocessAdd} for each sub-process.',
				'Press {subprocessShowPoints} on a sub-process to see and add its points.',
				'Choose the {pointDocumentType}, fill in {pointText}, and choose {pointScope}: {scopeGeneral} if the point is for every model, or a customer, model group, or model. Then press {pointAdd}.',
				'When one customer or model needs different text, press {variantAdd} on the point, choose the narrower scope, fill in {variantText}, and press {variantAdd} again.',
'Press {pointContent} on a point or a variant to fill in what it writes into each column of the form, for example the failure mode, its effect, and the severity for a PFMEA point. A column that two forms share is filled once and serves both. These are the values a document row starts with when the point is used in a document.',
				'Under {previewTitle}, choose a customer and a model to see which points that model gets.'
			],
			image: 'process-templates',
			imageAlt: 'The Process templates page with a point and its variant for one model.',
			tips: [
				'The most specific text wins: model, then model group, then customer, then general.',
				'The scope of a point cannot be changed later. If it is wrong, hide the point and add a new one.',
				'Sub-processes and points are hidden, never deleted.'
			]
		},
		{
			id: 'versions',
			title: 'Publish a version and make a new one',
			who: 'PPAP Engineer',
			intro: 'A new template is a draft. A draft can be changed freely. A published version can never be changed, so changes after that go into a new version.',
			steps: [
				'Open the template and look at the {versionChoose} row above its content.',
				'Press [[Publish version 1]] and confirm. The label changes to {versionStatusPublished}, and the buttons to add and edit disappear.',
				'To change a published template, press {versionNew}. The system makes the next draft as a copy of the published version.',
				'Change the draft, then publish it in the same way.',
				'Use the {versionChoose} box to open an older version. Older versions can only be read.'
			],
			image: 'versions',
			imageAlt: 'The version row of a published process template with the New version button.',
			tips: [
				'A template has at most one draft, so {versionNew} only appears when the last version is published.',
				'A new version keeps the same IDs of fields and points, so documents stay linked.',
				'After a process template version is published, {propagationPreview} under the propagation section shows which documents use the template, which rows change, and which were changed locally. Propagating makes one change per customer that waits for verification; the new content goes into the drafts when the change is accepted.'
			]
		},
		{
			id: 'form-sets',
			title: 'Form sets',
			who: 'PPAP Engineer, and the PPAP Engineer Supervisor to publish',
			intro: 'A form set is one package of forms for PFD, PFMEA, and Control Plan. The system has two: the AIAG 4th set (PFD v1, PFMEA v1, Control Plan v1) and the AIAG-VDA set (PFD v1, PFMEA v2, Control Plan v2). The AIAG-VDA set is hidden for now. Each customer uses one form set for its new documents.',
			steps: [
				'Open {navGroupTemplates}, then {navFormSets}.',
				'Each form set shows its three sheets: {formSetSheetPfd}, {formSetSheetPfmea}, and {formSetSheetCp}. The PPAP Engineer changes the sheets of a draft version and presses {formSetSaveSheets}.',
				'Publish the format templates of the three sheets first, under {navFormatTemplates}.',
				'The form approver opens the form set and presses [[Publish version 1]]. The person who prepared the draft cannot publish it. By default every form approver must approve; the list shows {approvalsDone} and {approvalsPending}. The admin can change this rule under {navGroupAdmin}, then {navSettings}.',
				'Under {formSetCustomersTitle}, the PPAP Engineer chooses the form set of each customer. Without a choice the customer uses the default form set.',
				'To make a new package, use {formSetCreateTitle} at the bottom of the page.',
				'A hidden form set cannot be chosen for a customer. Tick {showHidden} to see it, then press its show button. The default form set and a form set that a customer uses cannot be hidden.'
			],
			image: 'form-sets',
			imageAlt: 'The Form sets page with the two form sets and the form set per customer.',
			tips: [
				'A PFMEA format with Action Priority is also published by the form approver, because its AP table decides the priority of actions.',
				'The PPAP Engineer can change the AP table of a draft version: press {apEdit} on the format template, click a priority to change it, then press {apSave}.',
				'Every {rolePpapEngineerSupervisor} can publish. To let more people approve, tick V-3 for their role under {navRoles}.'
			]
		},
		{
			id: 'numbers',
			title: 'Register document numbers',
			who: 'PPAP Engineer, Project Leader',
			intro: 'The system gives every PFD, PFMEA, and Control Plan number. A number is never reused or deleted.',
			steps: [
				'Open {navGroupDocuments}, then {navNumbers}.',
				'Under {numberRegisterTitle}, choose the {numberCustomer}.',
				'Leave the set box ticked to get one running number for PFD, PFMEA, and Control Plan together. Untick it to register one number, then choose the {numberType}.',
				'Leave {numberInitialRevision} at 0, unless a legacy document enters at a later revision.',
				'Press {numberRegister}. The new numbers appear in the register with the status {numberStatusReserved}.',
				'To find a number, use {numberFilterStatus} or {numberFilterSearch} above the register, then press {numberFilterApply}.'
			],
			image: 'numbers',
			imageAlt: 'The Numbers page with the registration form and the register.',
			tips: [
				'When a set skips a number to keep the three documents aligned, the skipped number stays in the register as {numberStatusSkipped}.'
			]
		},
		{
			id: 'documents',
			title: 'Documents',
			who: 'PPAP Engineer, Project Leader',
			intro: 'A document is one PFD, PFMEA, or Control Plan with its own number. It uses the published form set of its customer, so the publish steps under Form sets come first.',
			steps: [
				'Register the number first under {navNumbers}. A document needs a number with the status {numberStatusReserved}.',
				'Open {navGroupDocuments}, then {navDocuments}, and choose the {documentCustomer}.',
				'Under {documentCreateTitle}, choose the {documentNumber}, tick the {documentModels} the document covers, and press {documentCreate}. The number becomes In use.',
				'Open the document from the list. It opens as a workbook: the {nextStepTitle} box shows the one thing to do next, the toolbar groups ({wbGroupRows}, {wbGroupTemplate}, {wbGroupReview}, {wbGroupOutput}) open their panel beside the sheet, and the tabs at the bottom switch between {sheetCover}, PFD, PFMEA, Control Plan, and {checkTab}.',
				'Fill in the header fields and press {documentHeaderSave}.',
				'Press {wbShortPreview} in the toolbar to see the document page by page as it prints. In a Control Plan the {previewFlow} column is drawn from the PFD: one symbol per step with its number, arrows between the steps, and connectors A, B, C where the flow continues on the next page. NG goes to a box on the left: SCRAP and NCP as a predefined process (a box with double side lines), REWORK as a plain box. OK is written only under an inspection (diamond). Adding or removing rows moves them by itself. The PDF and the Excel file print exactly these pages; in Excel the flow is drawn as shapes.',
				'A PFD prints as a diagram on A4 portrait, like the reference PFD: the legend of the symbols, then one box per step with its number and process name, an inspection as a diamond around the box, and NG to SCRAP, NCP, or REWORK on the left. The flow goes down in two columns and continues through connector A, B, C on the next column or page. You still fill in the PFD in the table.',
				'Press {documentRowAdd} to add an empty row at the end of the sheet.',
'When one model has several PCBs, each with its own process flow, under one document number (for example PCB A, Heatsink Assy R/L, and Base Assy R/L), press {sectionSplit} and give the first PCB a name and its part numbers, then {sectionAdd} for the next ones. The PFD, PFMEA, and CP of the set get the same sections. Choose a section with its tab above the table: the table, {documentRowAdd}, {composeOpen}, {orderOpen}, the {checkTab} tab, and the diagram then work for that section only, and operation numbers start again at 10 in each section. The rows that were already there go into the first section. Only an empty section can be removed.',
'To bring rows from an Excel file, press {pasteOpen}. Copy the cells in Excel, paste them in the box, and press {pastePreview}. Tick {pasteHasHeader} when you copied the title row too; the columns are then matched by title. Above each column you choose where it goes. A cell that does not fit its column, such as a severity that is not a number, is marked and left empty; the row is still added. Nothing is saved until you press the Import button.',
'In a PFMEA or a Control Plan whose set has a PFD, fill in the PFD first: one row per step with its step number, process, and Flow Symbol. Then press {pfdComposeOpen}. Each PFD step is listed with the published process template of its process and what the PFMEA and the Control Plan will get: {pfdStatePresent}, rows to add, or {pfdStateNoTemplate} for a step whose process has no template. Tick the steps and press {pfdApply}. The rows get the step number and process of the PFD and follow its order; rows already there are not changed. Below the list, {pfdCheckTitle} shows PFD steps that are not yet in a document and step numbers that the PFD does not have. The same check appears above the pages of {wbShortPreview}. The PFD itself is typed per document and has no compose button.',
'To build the rows from the process library instead of typing them, press {composeOpen}. Choose a process on the left and tick the sub-processes and points this model uses; the right side shows the rows each document of the set will get. Press {composeApply}. A ticked point becomes a row in the PFMEA or CP that stays linked to the template. The PFD is never composed from templates; it is typed per document. A point you leave unticked is recorded as left out, and you may write the reason next to it. A point that a newer template version added shows {composeUndecided} until you decide.',
'The Relevant Document column of a Control Plan is filled from the list under Template library, {navReferencedDocuments}. There a template maintainer records each WI, SOP, or QEP once (number, title, level, revision) and, with {refDocLinks}, makes it a default of the processes that use it, for every model or only for a customer, model group, or model. When you compose, the first row of each process gets those numbers, one per line. Cells that already hold text are never overwritten. For rows you typed yourself, press {refFill}.',
'A new model that is close to an existing one does not start from nothing. While the document has no rows, press {copyOpen}, choose the document of the similar model, and press {copyRun}. The rows, their order, and the points left out are copied for the whole set (PFD, PFMEA, and CP). After that the copy stands alone: changing it does not change the other model.',
'Rows composed from a process template stay linked to it. Press {compareOpen} to see, row by row, whether a row is still the same as its template point or was changed in this document; for a changed row the columns that differ are shown with the template value and the row value. When a row should stand on its own, press {compareDetach}, write the reason, and confirm. A detached row stays in the document but is no longer compared.',
'When a document set has both a PFMEA and a CP, the sheet tabs end with {checkTab}. It shows the two documents side by side, one process at a time, so you can confirm that every failure mode and control in the PFMEA is covered in the CP. The PPAP Engineer then presses {checkMark}. The mark shows who checked and when, and holds for the current revisions. When a row of that process is added, changed, or removed later, the mark says the process changed after the check; check it again and press {checkMark} once more.',
'In a PFD, press {diagramOpen} to see the process flow drawn from the rows: one box per row in their order, shaped by Flow Symbol, with NG Disposition beside the step. The diagram only shows; change the rows in the table and it follows.',
'To change the order of the processes, press {orderOpen}. Each line is one step: rows next to each other with the same process. Drag a step to its new place, or use the arrows, then press {orderSave}. With the automatic numbering ticked, the steps are numbered 10, 20, 30 and so on; a number you typed yourself in the table is kept. To go back to the automatic number, empty the cell and save the order again.',
'Two people can work in the same document. When someone else saved the same row before you, your change is not saved: a message appears, the latest content is shown, and you enter your change again.',
				'Click a cell and type, like in Excel. Press Enter or click elsewhere, and the row is saved. In the operation number and process columns you choose the values of the row itself.',
				'Press {documentRowDelete} on a row to remove it. The rows after it move up.',
				'The tabs above the table open the other documents of the same set: PFD, PFMEA, and Control Plan.'
			],
			tips: [
				'The operation number and the process name are columns filled from the row; they are never typed in the column itself.',
				'RPN and Action Priority are calculated and cannot be typed.',
				'Grey columns are filled or calculated by the system and cannot be typed.'
			],
			image: 'documents',
			imageAlt: 'The workbook of a PFD with one composed row.'
		},
		{
			id: 'document-control',
			title: 'Void, obsolete, and legacy numbers',
			who: 'PPAP Engineer',
			intro: 'The PPAP Engineer, who now also does document control (D-199), changes the status of numbers and brings in the numbers of the old register.',
			steps: [
				'Open {navGroupDocuments}, then {navNumbers}.',
				'On the row of the number, press {numberVoid} or {numberObsolete}, give the reason, and press {numberStatusConfirm}.',
				'To bring in old numbers, open {navGroupDocuments}, then {numberImportLink}.',
				'Choose the customer and the {numberImportStatus}, then paste the numbers under {numberImportText}: up to three columns (PFD, PFMEA, CP), one set per line. With {numberImportOneSet}, one number on a line also registers the other two.',
				'Press {numberImportPreview} and read the report: which lines will be imported, which are refused and why.',
				'When the report is right, press {numberImportRun}.'
			],
			tips: ['A voided number stays in the register with its reason. It is never given to another document.'],
			image: 'document-control',
			imageAlt: 'The Revisions section with a released revision and its signed scan.'
		},
		{
			id: 'changes',
			title: 'Changes',
			who: 'PPAP Engineer and Project Leader request; PPAP Engineer Supervisor and Project Leader verify',
			intro: 'Every revision of a PFD, PFMEA, or CP starts from a change: what triggered it, the type of change, the reason, and the documents it affects.',
			steps: [
				'Open {navGroupDocuments}, then {navChanges}, and choose the customer.',
				'Fill in {changeTrigger} (for example Customer claim), {changeType}, and, when it applies, {changeFourM} and {changeReference}.',
				'Write the {changeReason}, tick every affected document under {changeDocuments}, and press {changeSubmit}.',
				'The change gets a number such as [[CHG-0001]]. Open it to add or remove affected documents while it is still submitted.',
				'For a new SQAM revision that applies to existing documents, choose the customer and press {sqamOpen}: write the SQAM revision, the effective date, the change type, and the reason. The change covers every active document of the customer.',
				'Under {changeAttachments}, choose a file in {changeAttachmentAdd} and press {changeAttachmentUpload}, for example the 8D report or a photo of the defect. Press the file name to download it.',
				'Under {impactTitle}, record for each related document (PFD, PFMEA, CP, WI, inspection standard) whether it is affected, with a reason when it is not, and press {impactSave}. A WI or inspection standard is picked from the master list of referenced documents, so its number stays exactly as in the source; type it only when it is not in the list. The checklist can still be completed after the change is accepted.',
				'A verifier (the Project Engineer or the Project Leader of the customer’s project group, or the APQP/PPAP Controller) opens the change, writes a {changeVerifyNote}, and presses {changeAccept} or {changeReject}. The requester cannot verify their own change.'
			],
			tips: ['A change always affects at least one document, so the last one cannot be removed.', 'An attachment can be at most 20 MB. A removed attachment stays in the audit log.', 'When a change is accepted, every affected document gets a draft: a document that already has an open draft takes the change into it, and a released document opens the next revision. The revision in force stays as it is until the draft is released.', 'After the revision is signed on paper, a document control holder opens the document and records the release under Revisions. The content is frozen with a SHA-256 hash; the person who prepared the draft cannot record its release.', 'Under Compare revisions, choose a released revision and the open draft or another released revision, and optionally one process, to see every added, removed, changed, or moved row with the cells before and after.'],
			image: 'changes',
			imageAlt: 'The Changes page with the list of changes and the form for a new change.'
		},
		{
			id: 'yokoten',
			title: 'Yokoten',
			who: 'PPAP Engineer, Project Leader',
			intro: 'Yokoten spreads an accepted improvement to other models, with a decision per model and a status board.',
			steps: [
				'Open an accepted change and press {yokotenSpread}.',
				'Choose the {yokotenTargets}: every model of the customer, a model group, or chosen models. Narrow them with a process or a text in the rows, and press {yokotenSearch}.',
				'Check the models and documents found, write the reason, and make the yokoten.',
				'On the status board, the owner of each target presses {yokotenDecision_apply}, {yokotenDecision_adapt}, or {yokotenDecision_not_applicable} with a reason. Apply and adapt make a change for the documents of that model, which is verified like any other change.',
				'Follow the progress per model: waiting for verification, in draft, released, or not applicable.'
			],
			tips: ['Models whose documents are already in the source change are left out.', 'Each target only shows for accounts that may view its customer.']
		},
		{
			id: 'printing',
			title: 'Print and release',
			who: 'PPAP Engineer, Project Leader',
			intro: 'Prints come from the same content as the screen, in Excel or PDF, A3 or A4 as the layout says. Approval is a wet signature on paper, or a digital signature (see Digital signature).',
			steps: [
				'Open the document and press {printTitle} in the toolbar. Choose the revision and the {printPurpose}, then press {printExcel} or {printPdf}.',
				'For the paper approval, print {printPurpose_signature}. The release date is filled in, and the cover carries the approval slots of the route.',
				'After every slot is signed, document control uploads the scan under {revisions} with {scanUploadButton}, then records the release. The release is refused without the scan.',
				'Released revisions print as {printPurpose_reference} or {printPurpose_controlled}. Controlled copies are numbered, and every print shows in the {printLog}.',
				'The PPAP Engineer changes paper, fonts, margins, and column widths under {navGroupTemplates}, then {navPrintLayouts}, and can give one customer another layout.'
			],
			tips: ['Drafts always print with UNCONTROLLED COPY.', 'Rows changed in the printed revision carry a triangle with the revision number.', 'PDF needs Chrome or Edge on the server computer; Excel always works.'],
			image: 'printing',
			imageAlt: 'The Print section of a document with the purpose and the Excel and PDF buttons.'
		},
		{
			id: 'signatures',
			title: 'Digital signature',
			who: 'PPAP Engineer and Project Leader send; the people of the route sign; another PPAP Engineer records the release',
			intro: 'Signers of the approval route sign in the system instead of on paper (D-198). The content is frozen when it is sent, signing goes in route order, and the revision date is the date it was sent.',
			steps: [
				'Open the draft and press {sigTitle} in the toolbar. Choose who signs each slot: {sigPreparedHint} For the Engineer slot, {sigCheckedHint} Slots held by one person are filled in already.',
				'Type your password and press {sigSend}. The draft is locked and the first signer gets an email and a task on Home.',
				'Each signer opens {navGroupDocuments}, then {navSignatures}, presses {sigOpenSign}, reads the document, types the password, and presses {signButton}. {signReject} needs a reason and opens the draft again.',
				'The sender can press {sigWithdraw} while signatures are still missing.',
				'When every slot is signed, another PPAP Engineer opens the document and records the release under {revisions}. No scan is needed.'
			],
			tips: [
				'The PPAP Engineer decides per customer under {navGroupMaster}, then {navCustomers}: {signatureModeDigital} or {signatureModeWetOnly}. Wet-only customers keep the paper flow under {printTitle}.',
				'Every document, also one signed digitally, can still be printed for wet signature. The revision date on the print never becomes the print date.',
				'Signatures cannot be changed or deleted, also not by the admin.'
			],
			image: 'signatures',
			imageAlt: 'The Digital signature section of a draft with the people chosen for each slot.'
		},
		{
			id: 'submission-control',
			title: 'Form Control Submission',
			who: 'PPAP Engineer fills it in and sets the defaults; everyone with draft access reads it',
			intro: 'Each document has its own control list: the PFD, the PFMEA, and the Control Plan each have one, with one row per revision. It replaces the paper Form Control Submission Document.',
			steps: [
				'Open a document and press {subTitle} in the toolbar. The buttons at the top switch between the PFD, PFMEA, and Control Plan of the set.',
				'{subReason}, {subStart}, {subFinish}, and {subTopSign} are filled by the system from the change, the release, and the signatures.',
				'Press {subEdit} on a revision. Fill in {subSubmittedOn}, {subPackage}, {subResultLabel}, {subResultOn}, and {subRemarkLabel}, then press {subSave}. Upload the customer approval under {subScan} if you have it.',
				'Untick {subNeedsSubmission} for a revision that is not sent to the customer, and write why.',
				'Press {subDownloadPdf} or {subDownloadExcel} to print the form of this document only.'
			],
			tips: [
				'The PPAP Engineer sets per customer whether revisions need customer submission ({navGroupMaster}, then {navCustomers}, column {customerSubmission}), and the form number under {navGroupMaster}, then {navDocumentSettings}.',
				'Untick {subShowAll} to hide revisions that are not sent to the customer.'
			],
			image: 'submission-control',
			imageAlt: 'The Submission control section of a document with one row per revision and the print buttons.'
		},
		{
			id: 'migration',
			title: 'Legacy documents',
			who: 'PPAP Engineer, Admin for the switch',
			intro: 'A document that already exists on paper enters at its current revision and is released after another person checks it against the signed hard copy. Nothing is cleaned up on the way in.',
			steps: [
				'The admin switches on {settingsMigration} under {navGroupAdmin}, then {navSettings}.',
				'Open {navGroupDocuments}, then {navLegacy}, and choose the customer. Under {legacyReadyTitle}, every line needs a tick; a line without one links to the page that fixes it.',
				'Under {legacySetTitle}, keep {legacyOneSet} ticked when PFD, PFMEA, and CP share one running number, and type one number, for example SH-TR-020. A revision typed with it, such as SH-TR-020-02, moves to {migrateRevision}. Untick it when the numbers are not aligned, for example SC-TR-005, SF-TR-005, SH-TR-004, and type each number.',
				'Fill in {migrateRevision}, {migrateEffectiveDate}, and the names on the hard copy for each document, tick the models, and press {legacySetSubmit}. Numbers that are not in the register yet are registered as Released, as one set.',
				'Open the document from {legacyBoardTitle}. Type or compose the content until it equals the hard copy. Press {migrationTitle} in the toolbar; under {historyTitle}, press {historyAdd} for each line of the old history table, then {historySave}.',
				'To check your typing, upload the old file under {excelCompareTitle} and press {excelCompareButton}. The file is only compared, never imported.',
				'Under {linkTitle}, press {linkFind}, tick the pairs that are right, and press {linkConfirm}. Linked rows receive later template updates.',
				'Another migration officer opens the document, compares it with the signed hard copy, fills in {migrationEffort}, and presses {migrationVerify}. The status becomes {documentStatusReleasedMigrated}.'
			],
			tips: ['{legacyBoardTitle} shows each legacy document as {legacyStateEmpty}, {legacyStateToVerify}, or {documentStatusReleasedMigrated}.', 'The person who entered a document cannot verify it.', 'A scan of the hard copy is optional. To keep one as evidence, upload it under {revisions} before the verification.', 'After verification the baseline and its history cannot be changed; the next change goes through a normal change request.', 'Switch migration mode off when the migration is finished.'],
			image: 'migration',
			imageAlt: 'The Legacy documents page with the readiness checks, the set of three numbers, and the status board.'
		},
		{
			id: 'dashboard',
			title: 'Dashboard',
			who: 'Everyone',
			intro: 'The dashboard counts rows that follow a template, documents by status, yokoten targets, revision time, and migration effort. Every number comes from the data at that moment.',
			steps: [
				'Open {navGroupDocuments}, then {navDashboard}.',
				'Choose a customer, or leave {dashboardAllCustomers}.',
				'Read {dashboardRows}: the share of rows that are linked to a template point.',
				'Read {dashboardRevision} and {dashboardMigration} for the time numbers.'
			],
			tips: ['Revision time counts only revisions opened by an accepted change.', 'Migration minutes are the ones typed in at verification.'],
			image: 'dashboard',
			imageAlt: 'The dashboard with cards for rows, documents, yokoten, revision time, and migration.'
		},
		{
			id: 'released-documents',
			title: 'Released documents',
			who: 'Everyone, also Viewer accounts (internal audit, QA, QC, production)',
			intro: 'Read the revision in force of any released PFD, PFMEA, or Control Plan, exactly as it was signed. Drafts are never shown here.',
			steps: [
				'Open {navGroupDocuments}, then {navReleasedDocuments}.',
				'Choose a customer, then type a number or model in {releasedSearch} to narrow the list.',
				'Press a number to read the document. The tabs open the other documents of the set.',
				'Press {checkTab} to read the released PFMEA and CP side by side, with the check marks.'
			],
			tips: ['Nothing can be changed on these pages.', 'A document that is being revised still shows its released revision until the new one is released.'],
			image: 'released-documents',
			imageAlt: 'The Released documents page with the list of released documents of one customer.'
		},
		{
			id: 'periodic-review',
			title: '3-year review',
			who: 'PPAP Engineer records, a verifier confirms; everyone sees the warnings',
			intro: 'A PFD, PFMEA, and Control Plan set with no revision for 3 years must be reviewed. The warning appears on {navHome}, on the dashboard, on the document page, in {navPeriodicReviews}, and as an email to the authors of the customer: yellow 90 days before the due date, red after it.',
			steps: [
				'Open {navGroupDocuments}, then {navPeriodicReviews}. The cards count the sets that are overdue, due soon, due within a year, and reviewed this year.',
				'Choose {reviewTabDue}, {reviewTabYear}, or {reviewTabAll}. Filter by customer or engineer, and tick {reviewShowEol} to see models that are end of life. {reviewDownload} saves the list.',
				'Press {reviewStart} on a set. Choose {reviewChangesNeeded} and press {reviewOpenChange}: the change request opens with the trigger Periodic review and the documents of the set.',
				'Or choose {reviewNoChange}, fill in {reviewDate}, {reviewParticipants}, every checklist item, and {reviewConclusion}, then press {reviewSave}. Another person with the right V-1, for example the {rolePpapEngineerSupervisor}, presses {reviewConfirm} under {reviewWaitingTitle}; the clock then starts again.'
			],
			tips: ['Each document has its own clock: its last release, or the last confirmed review of its set.', 'The PPAP Engineer or the Project Leader marks a model end of life in Master data → Models ({modelEolMark}). Its sets leave the review.', 'The person who recorded a review cannot confirm it. A confirmed review cannot be changed.'],
			image: 'periodic-review',
			imageAlt: 'The Periodic review page with a document set that is overdue.'
		},
		{
			id: 'performance',
			title: 'PPAP performance',
			who: 'PPAP Engineer, supervisor, and manager',
			intro: 'The weekly, monthly, or yearly numbers of the PPAP work, counted from the recorded changes, prints, signed scans, and releases. A week runs from Friday to Thursday, so the Friday report covers the week that just ended.',
			steps: [
				'Open {navGroupDocuments}, then {navPerformance}.',
				'Choose {performanceWeek}, {performanceMonth}, or {performanceYear}. The arrows move one period; choose an engineer or a customer to narrow it. The download button saves the report as Excel.',
				'Read {performanceReceived}, {performanceReleased}, and {performanceApprovedToReleased}, each compared with the period before.',
				'Read {performanceWhere}: {performanceStageEdit}, {performanceStageWaiting}, {performanceStagePrint}, and {performanceStageRelease}; then the triggers, the trend, yokoten, and {performancePeopleTitle}.'
			],
			tips: ['No targets are set: the report only collects how long each stage takes.', 'Press {revisionReady} in the Revisions section of a draft when it is complete, so {performanceStageWaiting} can be measured.'],
			image: 'performance',
			imageAlt: 'The PPAP performance report for one week with the change counts and the time per stage.'
		},
		{
			id: 'roles',
			title: 'Roles and rights',
			who: 'Admin',
			intro: 'A role is a set of rights. The catalog of rights is fixed; the admin decides which rights each work role holds, and can add roles, so access can change without changing the program.',
			steps: [
				'Open {navGroupAdmin}, then {navRoles}. Each column is a role and each line is a right; under the name you see how many accounts hold it.',
				'Tick or untick a right in the column of a work role, then press {rolesSave} under that column.',
				'To add a role, fill in {rolesNameEn} and {rolesNameId} under {rolesAddTitle}, choose {rolesCopyFrom} to start from another role, and press {rolesAdd}.',
				'Press {rolesRename} under a column to change its name. Press {rolesDeactivate} when no account holds the role any more; tick {rolesShowInactive} to see it again.',
				'Give the role to people under {navAccounts}. The change applies at once to every account that holds the role.'
			],
			tips: [
				'Request a change is the right E-3: untick it to take it from a role, or tick it to give it to another role.',
				'S-1 (digital signature) and G-1 (genba notes) can already be set; their screens come later.',
				'Roles are never deleted, so the audit log stays readable. The old roles from before D-199 are inactive.',
				'The Admin and Viewer roles are fixed, and admin rights (A-1 to A-4) cannot be given to a work role.',
				'The basic rights L-1 to L-3 always stay ticked.',
				'Some rules cannot be switched off by any role: a requester never verifies their own request, and the person who prepared a draft never records its release.',
				'Documents of a customer outside your project group do not show at all, and their address answers "not found".'
			],
			image: 'roles',
			imageAlt: 'The Roles page with one column per role and one line per right.'
		},
		{
			id: 'service-accounts',
			title: 'Other applications (integration API)',
			who: 'Admin',
			intro: 'Another application, for example a MES, can read released documents and propose changes through the integration API. It never edits a document directly: a proposal becomes a change that a verifier accepts or rejects as usual.',
			steps: [
				'Open {navGroupAdmin}, then {navServiceAccounts}. Under {serviceAccountAdd}, fill in the name, the owning team, the contact, and the purpose.',
				'Tick only the scopes the application needs, and the customers whose documents it may see. Press {serviceAccountCreate}.',
				'Press {serviceTokenIssue}. The token is shown once and copied to the clipboard. Give it to the owning team through a safe channel.',
				'To rotate, issue a new token, let the team switch to it, then press {serviceTokenRevoke} on the old one.',
				'If the application is no longer used, press {serviceAccountDisable}. All its tokens are refused at once.'
			],
			tips: [
				'A token expires after 90 days unless you choose another lifetime.',
				'The application sees only released revisions, never drafts.',
				'Proposals from an application appear under {navChanges} with the application as requester. The verifier must not be the person the application named.',
				'The contract for the other team is the file api/integration.yaml.'
			],
			image: 'service-accounts',
			imageAlt: 'The Service accounts page with one application, its scopes, customers, and tokens.'
		},
		{
			id: 'notifications',
			title: 'Email notifications',
			who: 'Everyone for the own address, Admin for the settings',
			intro: 'The system sends an email when something waits for someone: a change to verify, a decision on your change, a signed scan ready to release, a yokoten to decide, and more. Approval itself stays on paper.',
			steps: [
				'Set your own address: press {passwordTitle} in the top bar, then fill in {accountEmailTitle} and press {accountEmailSave}. The admin can also set it under {navAccounts} with {accountEmail}.',
				'The admin opens {navGroupAdmin}, then {navNotifications}, to see how emails are sent and to switch events on or off.',
				'Press {notifyTest} to check that an email reaches your address.',
				'The {notifyOutbox} shows every email with its status. {notifyStatus_skipped} with {notifyReason_no_email} means the person has no address yet.'
			],
			tips: [
				'IT sets the email server (Microsoft 365) in the server settings. Until then emails are recorded but not sent.',
				'You never get an email about your own action.',
				'Emails never contain passwords or tokens.'
			],
			image: 'notifications',
			imageAlt: 'The Email notifications page with the email server, the list of events, and the outbox.'
		},
		{
			id: 'admin-health',
			title: 'Web health (admin)',
			who: 'Admin',
			intro: 'The Home of an admin account shows how the web application is doing and where work is stuck. Document content is not shown.',
			steps: [
				'Log in with the admin account. Home shows the cards: signatures waiting, server errors, problem reports, email, accounts, the last backup, and the audit log chain.',
				'Read {adminTopSignatures}: the five digital signatures that wait the longest, with the person who is awaited.',
				'Open {navGroupAdmin}, then {navAdminHealth} for every waiting signature, every server error with its request ID, and every problem report.',
				'For a problem report, choose the {adminStatus}, write an {adminNote}, and press {adminSave}.',
				'{adminProcessed} counts the revisions released this report week and this month, per trigger.'
			],
			tips: [
				'A server error shows the same request ID the user saw on the screen.',
				'Waiting time counts from the previous signature, or from the time the document was sent when the first slot waits.'
			],
			image: 'admin-health',
			imageAlt: 'The Home of the admin account with the health cards and the signatures waiting the longest.'
		},
		{
			id: 'audit',
			title: 'Audit log',
			who: 'Admin',
			intro: 'Every change of data is recorded with who, when, and what. Nobody can change or delete these records.',
			steps: [
				'Open {navGroupAdmin}, then {navAuditLog}.',
				'Read the list from the newest entry. Each entry shows the account and the person behind it.',
				'Press {auditVerify} to check that no entry was changed.'
			],
			image: 'audit',
			imageAlt: 'The Audit log page with the list of entries.'
		},
		{
			id: 'problems',
			title: 'When something goes wrong',
			who: 'Everyone',
			intro: 'Most problems have one of these causes.',
			steps: [
				'A menu or button is missing: your account does not have that role. Ask the admin to add the role.',
				'You forgot your password, or the account is locked: ask the admin to press {accountResetPassword} for your account.',
				'A red message appears under a field: correct that field and save again. Nothing was saved yet.',
				'A message says the version cannot be changed: the version is published. Press {versionNew} and change the new draft.',
				'The page looks old after an update: reload the page in the browser.',
				'Something else: press {reportProblem} in the top bar and describe it. The admin sees the page and the last request ID.'
			]
		}
	]
};
