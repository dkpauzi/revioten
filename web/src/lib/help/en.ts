// Buku panduan pengguna, bahasa Inggris (D-141). Bentuk dan penandanya
// dijelaskan di content.ts. Setiap perubahan di sini dibuat juga di id.ts.
import type { HelpContent } from './content.ts';

export const helpEn: HelpContent = {
	title: 'User guide',
	intro:
		'This guide follows the order of real work: the admin prepares accounts and master data, the template maintainer builds the templates, and the author registers document numbers. Names in bold are the names you see on the screen.',
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
				'Fill in {loginName} and {loginPassword}, then press {loginSubmit}.',
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
			id: 'customers',
			title: 'Project groups, customers, and models',
			who: 'Admin',
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
				'{roleAuthor}: registers document numbers. {roleTemplateMaintainer}: manages processes and process templates. {roleDocumentControl}: voids numbers, makes them obsolete, and imports legacy numbers.',
				'{roleFormApprover}: publishes form sets and AP tables; for now the PPAP Supervisor. {roleVerifier}, {rolePfmeaCpChecker}, and {roleMigration} are for screens that come later.',
				'The temporary password is shown only once. If it is lost, reset the password of the account.',
				'An admin account holds no work role. The person who holds it uses a separate work account for daily work.',
				'An account is never deleted, only disabled.'
			]
		},
		{
			id: 'positions',
			title: 'Positions and approval routes',
			who: 'Admin',
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
			id: 'format-templates',
			title: 'Format templates',
			who: 'Admin',
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
				'The system starts with two sets of forms: PFMEA AIAG 4th (RPN) with the current Control Plan, and PFMEA AIAG-VDA 2019 (Action Priority) with Control Plan AIAG 2024. The PFD form is the same for both.',
				'The operation number, the process name, the process flow, the document number, RPN, and Action Priority are filled in by the system and are never typed.',
				'Changing a label never changes the data of documents, because documents refer to the fixed ID of a field.'
			]
		},
		{
			id: 'processes',
			title: 'Processes',
			who: 'Template maintainer',
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
			who: 'Template maintainer',
			intro: 'A process template lists the sub-processes of one process and their points for PFD, PFMEA, and Control Plan. One process has one template.',
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
			who: 'Template maintainer, and the admin for format templates',
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
				'A new version keeps the same IDs of fields and points, so documents stay linked.'
			]
		},
		{
			id: 'form-sets',
			title: 'Form sets',
			who: 'Admin, and the form approver to publish',
			intro: 'A form set is one package of forms for PFD, PFMEA, and Control Plan. The system has two: the AIAG 4th set (RPN) and the AIAG-VDA set (Action Priority). Each customer uses one form set for its new documents.',
			steps: [
				'Open {navGroupTemplates}, then {navFormSets}.',
				'Each form set shows its three sheets: {formSetSheetPfd}, {formSetSheetPfmea}, and {formSetSheetCp}. The admin changes the sheets of a draft version and presses {formSetSaveSheets}.',
				'Publish the format templates of the three sheets first, under {navFormatTemplates}.',
				'The form approver opens the form set and presses [[Publish version 1]]. The person who prepared the draft cannot publish it. By default every form approver must approve; the list shows {approvalsDone} and {approvalsPending}. The admin can change this rule under {navSettings}.',
				'Under {formSetCustomersTitle}, the admin chooses the form set of each customer. Without a choice the customer uses the default form set.',
				'To make a new package, use {formSetCreateTitle} at the bottom of the page.'
			],
			image: 'form-sets',
			imageAlt: 'The Form sets page with the two form sets and the form set per customer.',
			tips: [
				'A PFMEA format with Action Priority is also published by the form approver, because its AP table decides the priority of actions.',
				'The admin can change the AP table of a draft version: press {apEdit} on the format template, click a priority to change it, then press {apSave}.',
				'Give the role {roleFormApprover} to more accounts under {navAccounts} when more people approve, for example the Project Leaders.'
			]
		},
		{
			id: 'numbers',
			title: 'Register document numbers',
			who: 'Author',
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
			who: 'Author',
			intro: 'A document is one PFD, PFMEA, or Control Plan with its own number. It uses the published form set of its customer, so the publish steps under Form sets come first.',
			steps: [
				'Register the number first under {navNumbers}. A document needs a number with the status {numberStatusReserved}.',
				'Open {navGroupDocuments}, then {navDocuments}, and choose the {documentCustomer}.',
				'Under {documentCreateTitle}, choose the {documentNumber}, tick the {documentModels} the document covers, and press {documentCreate}. The number becomes In use.',
				'Open the document from the list. Fill in the header fields and press {documentHeaderSave}.',
				'Press {documentRowAdd} to add an empty row at the end of the sheet.',
'To bring rows from an Excel file, press {pasteOpen}. Copy the cells in Excel, paste them in the box, and press {pastePreview}. Tick {pasteHasHeader} when you copied the title row too; the columns are then matched by title. Above each column you choose where it goes. A cell that does not fit its column, such as a severity that is not a number, is marked and left empty; the row is still added. Nothing is saved until you press the Import button.',
'To build the rows from the process library instead of typing them, press {composeOpen}. Choose a process on the left and tick the sub-processes and points this model uses; the right side shows the rows each document of the set will get. Press {composeApply}. A ticked point becomes a row in the PFD, PFMEA, or CP that stays linked to the template. A point you leave unticked is recorded as left out, and you may write the reason next to it. A point that a newer template version added shows {composeUndecided} until you decide.',
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
			]
		},
		{
			id: 'document-control',
			title: 'Void, obsolete, and legacy numbers',
			who: 'Document control',
			intro: 'Document control changes the status of numbers and brings in the numbers of the old register.',
			steps: [
				'Open {navGroupDocuments}, then {navNumbers}.',
				'On the row of the number, press {numberVoid} or {numberObsolete}, give the reason, and press {numberStatusConfirm}.',
				'To bring in old numbers, open {navGroupDocuments}, then {numberImportLink}.',
				'Choose the customer and the {numberImportStatus}, then paste the numbers under {numberImportText}.',
				'Press {numberImportPreview} and read the report: which lines will be imported, which are refused and why.',
				'When the report is right, press {numberImportRun}.'
			],
			tips: ['A voided number stays in the register with its reason. It is never given to another document.']
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
				'The page looks old after an update: reload the page in the browser.'
			]
		}
	]
};
