#!/usr/bin/env node
'use strict';

/**
 * Dependency-free static audit for the Sales Network plugin.
 *
 * This intentionally does not pretend to replace a WordPress/PHP/MySQL smoke
 * test. It catches packaging and wiring regressions before a build reaches
 * Staging and runs PHP lint automatically when a PHP CLI is available.
 */

const fs = require('fs');
const path = require('path');
const childProcess = require('child_process');
const vm = require('vm');

const root = path.resolve(__dirname, '..');
const errors = [];
const warnings = [];
const checks = {};

function filesIn(directory, extension) {
  const absolute = path.join(root, directory);
  if (!fs.existsSync(absolute)) return [];
  return fs.readdirSync(absolute)
    .filter((name) => name.endsWith(extension))
    .sort()
    .map((name) => path.join(directory, name));
}

function filesRecursive(directory, extension) {
  const absolute = path.join(root, directory);
  if (!fs.existsSync(absolute)) return [];
  return fs.readdirSync(absolute, { withFileTypes: true }).flatMap((entry) => {
    const relative = path.join(directory, entry.name);
    return entry.isDirectory() ? filesRecursive(relative, extension) : (entry.name.endsWith(extension) ? [relative] : []);
  }).sort();
}

const phpFiles = ['sales-network.php'].concat(filesRecursive('includes', '.php'));
const jsFiles = filesIn('assets/js', '.js');
const cssFiles = filesIn('assets/css', '.css');

function source(file) {
  return fs.readFileSync(path.join(root, file), 'utf8');
}

function addError(message) {
  errors.push(message);
}

function addWarning(message) {
  warnings.push(message);
}

function commandExists(command) {
  const result = childProcess.spawnSync('sh', ['-c', 'command -v "$1" >/dev/null 2>&1', 'sh', command]);
  return result.status === 0;
}

function extractFunctions(text) {
  const output = new Map();
  const pattern = /function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\([^)]*\)\s*(?::\s*[^\{]+)?\{/g;
  let match;
  while ((match = pattern.exec(text))) {
    const open = text.indexOf('{', match.index);
    let depth = 0;
    let state = 'code';
    for (let index = open; index < text.length; index++) {
      const current = text[index];
      const next = text[index + 1];
      if (state === 'single') {
        if (current === '\\') index++;
        else if (current === "'") state = 'code';
        continue;
      }
      if (state === 'double') {
        if (current === '\\') index++;
        else if (current === '"') state = 'code';
        continue;
      }
      if (state === 'line-comment') {
        if (current === '\n') state = 'code';
        continue;
      }
      if (state === 'block-comment') {
        if (current === '*' && next === '/') {
          state = 'code';
          index++;
        }
        continue;
      }
      if (current === "'") state = 'single';
      else if (current === '"') state = 'double';
      else if (current === '/' && next === '/') {
        state = 'line-comment';
        index++;
      } else if (current === '/' && next === '*') {
        state = 'block-comment';
        index++;
      } else if (current === '#') state = 'line-comment';
      else if (current === '{') depth++;
      else if (current === '}') {
        depth--;
        if (depth === 0) {
          output.set(match[1], text.slice(open + 1, index));
          pattern.lastIndex = index + 1;
          break;
        }
      }
    }
  }
  return output;
}

function methodNames(text) {
  return new Set(Array.from(
    text.matchAll(/function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(/g),
    (match) => match[1]
  ));
}

function stripCssStringsAndComments(text) {
  return text
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/"(?:\\.|[^"\\])*"/g, '""')
    .replace(/'(?:\\.|[^'\\])*'/g, "''");
}

// Required package files and version parity.
for (const file of [
  'sales-network.php',
  'includes/class-sn-plugin.php',
  'includes/class-sn-activator.php',
	'includes/class-sn-customer-portal.php',
	'includes/class-sn-operations-flow.php',
	'includes/class-sn-operations-execution.php',
	'includes/class-sn-hr-transfer.php',
	'includes/class-sn-seller-flow.php',
  'assets/js/hr-transfer.js',
  'assets/css/hr-transfer.css',
	'assets/js/customer-portal.js',
	'assets/css/customer-portal.css',
	'assets/js/operations-flow.js',
	'assets/css/operations-flow.css',
]) {
  if (!fs.existsSync(path.join(root, file))) addError(`Missing required file: ${file}`);
}
const bootstrap = source('sales-network.php');
const headerVersion = (bootstrap.match(/^[ \t]*\* Version:\s*([^\s]+)/m) || [])[1] || '';
const constantVersion = (bootstrap.match(/define\(\s*'SN_VERSION'\s*,\s*'([^']+)'/) || [])[1] || '';
if (!headerVersion || headerVersion !== constantVersion) {
  addError(`Plugin version mismatch: header=${headerVersion || 'missing'} constant=${constantVersion || 'missing'}`);
}
checks.version = constantVersion;

// The synchronous Zibal CSV export must remain read-only and avoid loading the
// complete HR graph before LiteSpeed can send the download response.
const corePlugin = source('includes/class-sn-plugin.php');
const zibalCoreFunctions = extractFunctions(corePlugin);
const zibalExportHandler = zibalCoreFunctions.get('handle_financial_gateway_export') || '';
const zibalBuildRows = zibalCoreFunctions.get('sn_zibal_build_finance_export_rows') || '';
const zibalHrChain = zibalCoreFunctions.get('sn_zibal_hr_chain_map') || '';
const zibalExportLog = zibalCoreFunctions.get('sn_zibal_export_log') || '';
const zibalReportFetch = zibalCoreFunctions.get('sn_zibal_report_fetch_transactions') || '';
const zibalNormalizeTransaction = zibalCoreFunctions.get('sn_zibal_normalize_report_transaction') || '';
const zibalColumns = zibalCoreFunctions.get('sn_zibal_export_columns') || '';
const zibalSelectedColumns = zibalCoreFunctions.get('sn_zibal_export_selected_column_keys') || '';
for (const required of [
  "sn_zibal_export_log($request_id, 'remote_report'",
  "sn_zibal_export_log($request_id, 'merge'",
  "sn_zibal_export_log($request_id, 'build_rows'",
  'catch (Throwable $error)',
]) {
  if (!zibalExportHandler.includes(required)) addError(`Zibal export diagnostic guard is missing: ${required}`);
}
for (const required of ['WHERE p.user_id IN', 'child_profile_id IN', 'array_chunk($user_ids, 400)', '$depth < 12']) {
  if (!zibalHrChain.includes(required)) addError(`Zibal bounded HR resolver invariant is missing: ${required}`);
}
if (!zibalBuildRows.includes('sn_zibal_hr_chain_map($seller_ids)')) addError('Zibal export does not use its bounded HR resolver.');
if (zibalBuildRows.includes('sn_admin_invoice_hr_chain_map($seller_ids)')) addError('Zibal export still loads the shared full HR graph.');
if (/token|authorization/i.test(zibalExportLog)) addError('Zibal diagnostic logger must never receive or log API credentials.');
for (const forbidden of ['INSERT ', 'UPDATE ', 'DELETE ', 'dbDelta(', 'migrate(']) {
  if (zibalExportHandler.includes(forbidden)) addError(`Zibal export must remain read-only: ${forbidden}`);
}
checks.zibalExportBoundedHr = true;

// 2.0.96: preserve the selectable native Zibal export merged from the Finance
// branch without weakening the bounded/read-only export guarantees above.
const declaredZibalColumns = Array.from(zibalColumns.matchAll(/'([a-z_]+)'\s*=>\s*\[\s*'label'/g), (match) => match[1]);
if (declaredZibalColumns.length !== 28) addError(`Zibal export must expose 28 selectable columns; found ${declaredZibalColumns.length}.`);
for (const required of [
  "'seller_id' => ['label' => 'شناسه فروشنده'",
  "'sales_manager_id' => ['label' => 'شناسه مدیر فروش'",
  "'sales_deputy_id' => ['label' => 'شناسه معاون فروش'",
]) {
  if (!zibalColumns.includes(required)) addError(`Zibal selectable column invariant is missing: ${required}`);
}
for (const required of ['sn_gateway_export_columns_submitted', 'sanitize_key', 'isset($columns[$key])']) {
  if (!zibalSelectedColumns.includes(required)) addError(`Zibal selected-column allowlist invariant is missing: ${required}`);
}
for (const required of [
  "get_option('sn_zibal_merchant'",
  "'merchantId' => $merchant_id",
  "$request_body['merchantId'] = $merchant_id",
]) {
  if (!zibalReportFetch.includes(required)) addError(`Zibal native merchant request invariant is missing: ${required}`);
}
for (const required of ['terminalNumber', 'cardHolderIP', "'psp'"]) {
  if (!zibalNormalizeTransaction.includes(required)) addError(`Zibal live field alias is missing: ${required}`);
}
for (const required of [
  'fwrite($out, "\\xEF\\xBB\\xBF")',
  "fopen('php://output', 'wb')",
  'sn_gateway_export_columns[]',
  "preg_match('/^[0-9*.+:\\/ -]+$/', $text_value)",
]) {
  if (!corePlugin.includes(required)) addError(`Zibal UTF-8/selectable export invariant is missing: ${required}`);
}
for (const required of ['sales_manager_name', 'sales_manager_id', 'deputy_name', 'deputy_id']) {
  if (!zibalBuildRows.includes(required)) addError(`Zibal HR export field is missing: ${required}`);
}
checks.zibalSelectableUtf8Export = true;

// Stage-one seller workflow wiring and invariants.
const sellerFlow = source('includes/class-sn-seller-flow.php');
const sellerJs = source('assets/js/public-seller.js');
const stageOneLabels = ['جواب نداده', 'تماس مجدد', 'تکراری', 'عدم خرید', 'پیش‌فاکتور'];
const statusBody = extractFunctions(sellerFlow).get('statuses') || '';
const declaredStageLabels = Array.from(statusBody.matchAll(/'label'\s*=>\s*'([^']+)'/g), (match) => match[1]);
for (const label of stageOneLabels) {
  if (!sellerFlow.includes(`'label' => '${label}'`)) addError(`Seller stage-one status is missing: ${label}`);
}
if (declaredStageLabels.length !== stageOneLabels.length || declaredStageLabels.some((label, index) => label !== stageOneLabels[index])) {
  addError(`Seller stage-one status contract differs from the required statuses: ${declaredStageLabels.join(', ')}`);
}
for (const required of ['MAX_NO_ANSWER_ATTEMPTS = 3', 'ARCHIVE_IDLE_DAYS = 3', 'sn_not_purchase_reason_required', 'sn_pre_invoice_requires_invoice', 'seller_reassigned_detected', 'live_source_snapshot', "! empty( $state['idempotent'] ) ? 'ROLLBACK' : 'COMMIT'"]) {
  if (!sellerFlow.includes(required)) addError(`Seller stage-one server invariant is missing: ${required}`);
}
for (const required of [
  "require_once SN_PLUGIN_DIR . 'includes/class-sn-seller-flow.php'",
  'SN_Seller_Flow::instance()->register_hooks()',
]) {
  if (!bootstrap.includes(required)) addError(`Seller flow bootstrap wiring is missing: ${required}`);
}
for (const required of [
  'sn_seller_flow_no_answer_attempt',
  'sn-not-purchase-reason',
  "changedField: 'lead_status'",
]) {
  if (!sellerJs.includes(required)) addError(`Seller stage-one UI wiring is missing: ${required}`);
}
// Seller stage tabs must filter by canonical workflow slugs. Comparing Persian
// display labels caused successfully saved `no_answer` rows to disappear from
// the corresponding tab when the underlying source used a legacy/MIS variant.
for (const required of [
  "var filterKey = String(s.slug || snSellerFlowSlug(s.label) || s.label || '')",
  "activeFilter = String(savedPayload.seller_flow_status || snSellerFlowSlug(savedPayload.lead_status) || 'no-status')",
]) {
  if (!sellerJs.includes(required)) addError(`Seller canonical status-tab invariant is missing: ${required}`);
}
for (const required of [
  "$canonical_stage_filter = $flow_filter ? $flow_filter->status_slug($active_filter) : ''",
  "$flow_status = sanitize_key((string) ($lead['seller_flow_status'] ?? ''))",
  "$matches_filter = $flow_status === $active_filter",
  '$lead_has_no_status = static function (array $lead): bool',
  "$matches_filter = $lead_has_no_status($lead)",
]) {
  if (!corePlugin.includes(required)) addError(`Seller server canonical-filter invariant is missing: ${required}`);
}
for (const forbidden of [
  "(string) ($lead['lead_status'] ?? '') === '' || ! empty($lead['is_extra_number_request'])",
  "! empty($lead['is_extra_number_request']) || ($flow_status === '' && $status === '')",
]) {
  if (corePlugin.includes(forbidden)) addError(`Seller no-status filter still contains the legacy extra-number exception: ${forbidden}`);
}
for (const required of [
  'function sellerLeadHasNoStatus(lead)',
  'allLeads.filter(sellerLeadHasNoStatus).length',
]) {
  if (!sellerJs.includes(required)) addError(`Seller no-status UI invariant is missing: ${required}`);
}
const sellerNoStatusPredicate = (lead) => !String((lead && lead.seller_flow_status) || '').trim()
  && !String((lead && lead.lead_status) || '').trim();
const sellerNoStatusCases = [
  [{}, true, 'empty ordinary lead'],
  [{ is_extra_number_request: true }, true, 'empty extra-number lead'],
  [{ is_extra_number_request: true, seller_flow_status: 'no_answer', lead_status: 'جواب نداده' }, false, 'statused extra-number lead'],
  [{ seller_flow_status: 'no_answer' }, false, 'canonical-only status'],
  [{ lead_status: 'جواب نداده' }, false, 'legacy/display-only status'],
];
for (const [lead, expected, label] of sellerNoStatusCases) {
  if (sellerNoStatusPredicate(lead) !== expected) addError(`Seller no-status predicate failed: ${label}`);
}
const dotFlowNoAnswerAudit = source('includes/class-sn-dot-flow.php');
for (const required of [
  "$contact_filters = [ 'new', 'no_answer', 'contacted', 'follow_up'",
  "$allowed = [ 'no_answer', 'follow_up', 'customer_declined', 'payment_link' ]",
]) {
  if (!dotFlowNoAnswerAudit.includes(required)) addError(`Converter no-answer canonical invariant is missing: ${required}`);
}
const operationsExecution = source('includes/class-sn-operations-execution.php');
for (const required of [
  "status='no_answer'",
  'value="no_answer"',
]) {
  if (!operationsExecution.includes(required)) addError(`Operations-execution no-answer invariant is missing: ${required}`);
}
const projectsFlow = source('includes/class-sn-projects.php');
for (const required of [
  "'no_answer' => 'جواب نداده'",
  "mi.workflow_status='no_answer'",
]) {
  if (!projectsFlow.includes(required)) addError(`Projects no-answer canonical invariant is missing: ${required}`);
}
checks.noAnswerCanonicalRouting = true;
if (!source('includes/class-sn-plugin.php').includes("'manager-archives' => 'بایگانی‌ها'")) addError('Sales manager archive tab is missing.');
if (!source('includes/class-sn-activator.php').includes('product_type VARCHAR(30) DEFAULT NULL')) addError('Invoice item product-type snapshot schema is missing.');
if (!source('includes/class-sn-plugin.php').includes('ثبت یکپارچه وضعیت مرحله فروشنده ناموفق بود')) addError('Invoice and seller-flow state are not wired into one transaction.');
checks.sellerStageOneStatuses = stageOneLabels.length;

// Supervisor self-assignment must be available without widening access to
// another supervisor's or converter's cases.
const dotFlow = source('includes/class-sn-dot-flow.php');
const plugin = source('includes/class-sn-plugin.php');
for (const required of [
  'private function is_supervisor(',
  'private function converter_assignment_targets(',
  "$targets[ $supervisor_id ] = $supervisor",
  "(int) $case->converter_id !== $user_id",
  "(int) $case->supervisor_id === $user_id",
  'converter_id=%d AND supervisor_id=%d',
  "'پنل تبدیل‌کننده (تخصیص‌های خودم)'",
]) {
  if (!dotFlow.includes(required)) addError(`Supervisor self-converter invariant is missing: ${required}`);
}
if (!plugin.includes("$this->sn_user_has_hr_position($user_id, ['converter', 'supervisor', 'senior_supervisor'])")) {
  addError('Supervisor navigation access to the converter workspace is missing.');
}
if (dotFlow.includes("$can_issue_invoice = ( $is_converter_user || $is_supervisor )")) {
  addError('Supervisor self-converter must not inherit unrestricted manual-invoice issuance.');
}
checks.supervisorSelfConverter = true;

// Phase-two assessment lifecycle: product typing, two timed archives, assignment
// before customer selection, mandatory decline reason, and late-payment recovery.
for (const required of [
  'private const ASSESSMENT_UNPAID_DAYS = 3',
  'private const SUBSCRIPTION_UNPAID_DAYS = 5',
  "'archives' => $wpdb->prefix . 'sn_dot_operational_archives'",
  'converter_decline_reason TEXT DEFAULT NULL',
  'operational_archived_at DATETIME DEFAULT NULL',
  "'archive_type' => 'assessment_unpaid'",
  "'archive_type' => 'subscription_unpaid'",
  "'status' => 'archived_unpaid_subscription'",
  'resolve_operational_archive(',
  "meta_value' => 'assessment'",
]) {
  if (!dotFlow.includes(required)) addError(`Assessment lifecycle invariant is missing: ${required}`);
}
const assignBody = extractFunctions(dotFlow).get('handle_assign_converter') || '';
const bulkAssignBody = extractFunctions(dotFlow).get('handle_bulk_assign_converter') || '';
if (assignBody.includes('empty( $case->selected_option_key )') || bulkAssignBody.includes('empty( $case->selected_option_key )')) {
  addError('A paid assessment case without customer selection is still blocked from assignment.');
}
const converterUpdateBody = extractFunctions(dotFlow).get('handle_converter_update') || '';
for (const required of ["'no_answer'", 'decline_reason', 'واردکردن علت انصراف اجباری است']) {
  if (!converterUpdateBody.includes(required)) addError(`Converter outcome invariant is missing: ${required}`);
}
for (const required of ['عدم پرداخت اعتبارسنجی', 'عدم پرداخت اشتراک', 'mark_assessment_invoice_unpaid', 'resolve_assessment_invoice_unpaid']) {
  if (!sellerFlow.includes(required)) addError(`Sales manager archive invariant is missing: ${required}`);
}
const managerScript = source('assets/js/public-manager.js');
for (const required of ['data-sn-manager-archive-tab', 'data-sn-manager-archive-panel', 'snManagerArchiveTab']) {
  if (!sellerFlow.includes(required) && !managerScript.includes(required)) addError(`Ajax-safe manager archive subtab invariant is missing: ${required}`);
}
if (!plugin.includes('آماده‌های تبدیل') || !plugin.includes("'product_type' => $subscription_form ? 'subscription' : 'assessment'")) {
  addError('Assessment UI label or Marketing invoice product-type snapshot is missing.');
}
checks.assessmentLifecycle = true;

// Assessment conversion ownership is additive: missing/legacy values retain
// the supervisor queue, while seller-owned cases cannot be reassigned.
for (const required of [
  'conversion_route VARCHAR(30) DEFAULT NULL',
  "[ 'supervisor_queue', 'seller_self' ]",
  "? $this->normalize_conversion_route( (string) $link->conversion_route )",
  ": 'supervisor_queue'",
  "'converter_id' => $seller_self ? (int) $invoice->seller_id : null",
  'private function is_seller_self_case(',
  "render_converter_panel( string $context = '' )",
  "conversion_route='seller_self'",
  "conversion_route<>'seller_self'",
]) {
  if (!dotFlow.includes(required)) addError(`Assessment conversion ownership invariant is missing: ${required}`);
}
for (const required of [
  "dot_conversion_route",
  'ارسال برای سرپرست / تبدیل توسط خودم',
  'ارسال پیامک نتیجه اعتبارسنجی',
  'تبدیل توسط خودم',
  'data-tab="my-conversions"',
  "render_converter_panel('supervisor')",
]) {
  if (!plugin.includes(required)) addError(`Assessment conversion UI/server wiring is missing: ${required}`);
}
for (const required of [
  'dot_conversion_route:dotConversionRoute',
  "context:'seller'",
  'snLoadSellerConversions',
]) {
  if (!sellerJs.includes(required)) addError(`Seller self-conversion UI wiring is missing: ${required}`);
}
if (!source('assets/js/public-dot.js').includes("context: String($current.attr('data-sn-context') || '')")) {
  addError('Embedded conversion workspace refresh does not preserve its owner context.');
}
for (const body of [assignBody, bulkAssignBody]) {
  if (!body.includes('is_seller_self_case')) addError('Seller-owned assessment cases are not protected from supervisor reassignment.');
}
checks.assessmentConversionOwnership = true;

// Staged payment remains additive and cannot advance the business flow before
// the full invoice amount has been approved.
const pluginCore = extractFunctions(plugin).get('sn_create_invoice_core') || '';
const finalizeStage = extractFunctions(plugin).get('sn_finalize_current_payment_stage') || '';
const followInvoice = extractFunctions(plugin).get('handle_paid_referral_create_follow_invoice') || '';
for (const required of [
  '$initial_due - $total > 0.5',
  "if ( abs( $initial_due - $total ) <= 0.5 ) { $payment_plan = 'full';",
]) {
  if (!pluginCore.includes(required)) addError(`Initial staged-payment invariant is missing: ${required}`);
}
for (const required of [
  '$new_remaining < 0.5',
  "'payment_workflow_status'=>'awaiting_assignment'",
  "do_action('sn_invoice_paid'",
]) {
  if (!finalizeStage.includes(required)) addError(`Payment completion gate is missing: ${required}`);
}
if (!sellerJs.includes('prepaymentAmount>invoiceTotal')) addError('Seller UI does not allow a custom stage up to the invoice total.');
checks.stagedPaymentCompletionGate = true;

// Converter-issued physical-product invoices must expose and submit the same
// shipping fields that the shared server-side invoice validator requires.
const dotFlowPhp = source('includes/class-sn-dot-flow.php');
const converterInvoiceJs = source('assets/js/public-converter.js');
for (const required of ['sn-converter-cust-address', 'sn-converter-cust-postal', 'sn-converter-shipping-field']) {
  if (!dotFlowPhp.includes(required)) addError(`Converter shipping form field is missing: ${required}`);
}
for (const required of ['customer_address: address', 'customer_postal_code: postalCode', 'hasPhysicalProduct()', "postalCode.length !== 10"]) {
  if (!converterInvoiceJs.includes(required)) addError(`Converter shipping payload/validation invariant is missing: ${required}`);
}
checks.converterPhysicalAddress = true;

// The supervisor repeat-action form has a narrow internal permission path,
// visible feedback, and duplicate-submit protection.
for (const required of [
  '$this->sn_create_invoice_core($request, true)',
  'sn_paid_referral_action_notice_',
  'sn_paid_referral_invoice_create_lock_',
  'repeat_action_changed_after_lock',
  'WHERE referral_item_id=%d',
  "'customer_address' =>",
  "'customer_postal_code' =>",
]) {
  if (!followInvoice.includes(required)) addError(`Controlled supervisor invoice invariant is missing: ${required}`);
}
if (!pluginCore.includes('! $allow_controlled_workflow_invoice && ! $this->sn_user_can_create_manual_invoice')) {
  addError('Controlled supervisor invoice bypass is not isolated from ordinary manual invoice permissions.');
}
const supervisorJs = source('assets/js/public-supervisor.js');
for (const required of ['snPaidReferralCreate', 'sn-paid-referral-create-form', 'در حال صدور فاکتور']) {
  if (!supervisorJs.includes(required)) addError(`Supervisor repeat-action UI safeguard is missing: ${required}`);
}
checks.supervisorRepeatInvoice = true;
if (!source('includes/class-sn-activator.php').includes('referral_item_id BIGINT UNSIGNED DEFAULT NULL')) {
  addError('Controlled invoice idempotency column is missing from the installer schema.');
}

// Callback metadata is additive, shared by every conversion workspace, and
// schedules a one-time reminder without removing the in-panel overdue marker.
const dotJs = source('assets/js/public-dot.js');
for (const required of [
  'converter_followup_at DATETIME DEFAULT NULL',
  'converter_followup_reminded_at DATETIME DEFAULT NULL',
  'normalize_followup_datetime',
  'schedule_followup_reminder',
  'maybe_process_due_followup_reminders',
]) {
  if (!dotFlow.includes(required)) addError(`Conversion callback invariant is missing: ${required}`);
}
if (!source('includes/class-sn-helpers.php').includes('زمان تماس مجدد باید در آینده باشد') || !dotFlow.includes('normalize_jalali_tehran_datetime')) {
  addError('Conversion callback must use the shared future Tehran datetime validator.');
}
for (const required of ['syncConverterStatusFields', 'input[name="followup_date"]', 'input[name="followup_time"]', 'followupComplete', 'textarea[name="decline_reason"]']) {
  if (!dotJs.includes(required)) addError(`Conversion callback UI invariant is missing: ${required}`);
}
checks.conversionCallbackReminder = true;

// Conversion outcomes are shared by seller/self, supervisor/self and regular
// converters. Conditional fields must stay closed until their outcome is picked.
for (const required of [
  'converter_no_answer_attempts INT UNSIGNED NOT NULL DEFAULT 0',
  "[ 'no_answer', 'follow_up', 'customer_declined', 'payment_link' ]",
  'برای ارسال لینک پرداخت، ابتدا محصول نهایی را انتخاب کنید',
  'class="sn-jalali-date" name="followup_date"',
  'SN_Helpers::normalize_jalali_tehran_datetime',
]) {
  if (!dotFlow.includes(required)) addError(`Unified conversion outcome invariant is missing: ${required}`);
}
for (const required of ['data-sn-dot-payment-step', "status === 'payment_link'", "status === 'follow_up' && !followupComplete"]) {
  if (!dotJs.includes(required)) addError(`Conditional conversion field wiring is missing: ${required}`);
}
checks.unifiedConversionOutcomes = true;

// Supervisor and senior-supervisor repeat referrals share the Dot conversion
// queue. Legacy origin rows are retained and linked idempotently.
for (const required of [
  'referral_item_id BIGINT UNSIGNED DEFAULT NULL',
  'UNIQUE KEY referral_item_id (referral_item_id)',
  'ensure_paid_referral_conversion_case',
  'migrate_paid_referrals_for_actor',
  "'status' => 'converted_to_dot_case'",
  "'status' => 'ready_for_conversion'",
  'repair_paid_assessment_cases_for_supervisor',
]) {
  if (!dotFlow.includes(required)) addError(`Unified referral conversion invariant is missing: ${required}`);
}
const supervisorLazyBody = extractFunctions(plugin).get('ajax_supervisor_lazy_tab') || '';
const seniorLazyBody = extractFunctions(plugin).get('ajax_senior_supervisor_lazy_tab') || '';
if (!supervisorLazyBody.includes("case 'needs-action':") || !supervisorLazyBody.includes('render_supervisor_ready_tab')) {
  addError('Supervisor ready-conversion tab is not wired to the shared conversion queue.');
}
if (!seniorLazyBody.includes("case 'ss-repeat-actions':") || !seniorLazyBody.includes('render_supervisor_ready_tab')) {
  addError('Senior-supervisor ready-conversion tab is not wired to the shared conversion queue.');
}
const readyTabBody = extractFunctions(dotFlow).get('render_supervisor_ready_tab') || '';
if (!readyTabBody.includes('repair_paid_assessment_cases_for_supervisor') || readyTabBody.includes('selected_option_key IS NOT NULL')) {
  addError('Paid assessments without a customer option are not safely repaired into the supervisor queue.');
}
if (!readyTabBody.includes('ORDER BY created_at DESC,id DESC') || readyTabBody.includes('ORDER BY selected_at IS NULL')) {
  addError('Supervisor ready-conversion queue is not sorted by immutable panel-entry time, newest first.');
}
if (!readyTabBody.includes('زمان ورود به صف تبدیل') || !readyTabBody.includes("$case->created_at")) {
  addError('Supervisor ready-conversion rows do not display their queue-entry timestamp.');
}
checks.supervisorReadyNewestFirst = true;
const paidAssessmentRepairBody = extractFunctions(dotFlow).get('repair_paid_assessment_cases_for_supervisor') || '';
for (const required of [
  "dl.flow_kind='assessment_source'",
  'dl.supervisor_id=%d',
  "COALESCE(i.payment_workflow_status,'')='completed'",
  'COALESCE(i.remaining_amount,0)<=0.5',
  'create_case_for_source_invoice',
]) {
  if (!paidAssessmentRepairBody.includes(required)) addError(`Paid assessment queue repair invariant is missing: ${required}`);
}
const paidHookBody = extractFunctions(dotFlow).get('on_invoice_paid') || '';
if (paidHookBody.includes("empty( $this->config()['enabled'] )")) {
  addError('A later Dot settings change can still strand a paid assessment outside the supervisor queue.');
}
if (!supervisorJs.includes("target === 'needs-action' || target === 'my-conversions'") || !supervisorJs.includes("target === 'ss-repeat-actions' || target === 'ss-my-conversions'")) {
  addError('Conversion queues are not refreshed when their tabs are reopened.');
}
checks.unifiedReferralConversionQueue = true;

// Regular Dot Flow includes every supervisor, while the operational conversion
// queue belongs to the seller's nearest ordinary supervisor. Senior supervisors
// retain a seller-scoped aggregate read-only view.
const settingsPanelBody = extractFunctions(dotFlow).get('render_settings_panel') || '';
if (settingsPanelBody.includes('name="sn_dot[product_supervisors]')) {
  addError('Regular Dot settings still expose a per-product supervisor selector.');
}
if (!settingsPanelBody.includes('همه سرپرست‌ها مشمول‌اند')) {
  addError('Regular Dot settings do not explain the all-supervisors scope.');
}
const prepareAssessmentBody = extractFunctions(dotFlow).get('prepare_invoice_context') || '';
if (!prepareAssessmentBody.includes('resolve_default_supervisor_for_seller') || prepareAssessmentBody.includes('resolve_supervisor_for_seller')) {
  addError('Regular assessment invoices are not routed by the seller direct supervisor.');
}
const productFilterBody = extractFunctions(dotFlow).get('filter_products_for_user') || '';
if (productFilterBody.includes('resolve_supervisor_for_seller')) {
  addError('Assessment product visibility is still restricted by selected supervisors.');
}
const reconcileSupervisorBody = extractFunctions(dotFlow).get('reconcile_regular_assessment_link_supervisor') || '';
for (const required of [
  "flow_kind ?? '' ) !== 'assessment_source'",
  'marketing_form_id',
  'is_marketing_seller_id',
  'resolve_default_supervisor_for_seller',
  "conversion_route ?? $link->conversion_route ?? '' ) === 'seller_self'",
  'supervisor_scope_reconciled',
]) {
  if (!reconcileSupervisorBody.includes(required)) addError(`Direct-supervisor reconciliation invariant is missing: ${required}`);
}
if (!paidHookBody.includes('reconcile_regular_assessment_link_supervisor')) {
  addError('Finance-paid assessment hook does not reconcile the direct supervisor before creating its case.');
}
const reconcileSupervisorQueueBody = extractFunctions(dotFlow).get('reconcile_paid_assessment_scope_for_supervisor') || '';
for (const required of [
  'scope_visible_seller_ids',
  "COALESCE(dl.marketing_form_id,'')=''",
  'COALESCE(dl.supervisor_id,0)<>%d',
  'COALESCE(source_case.converter_id,0)=0',
  "COALESCE(i.payment_workflow_status,'')='completed'",
  'reconcile_regular_assessment_link_supervisor',
]) {
  if (!reconcileSupervisorQueueBody.includes(required)) addError(`Legacy direct-supervisor queue repair invariant is missing: ${required}`);
}
if (!readyTabBody.includes('reconcile_paid_assessment_scope_for_supervisor')) {
  addError('Supervisor ready queue does not recover older paid rows routed to another configured supervisor.');
}
for (const required of [
  'private function is_senior_supervisor(',
  '$this->is_senior_supervisor( $seller_id )',
  "(string) $parent->slug === 'senior_supervisor'",
  'bool $read_only = false',
  'فقط مشاهده',
]) {
  if (!dotFlow.includes(required)) addError(`Supervisor queue ownership/oversight invariant is missing: ${required}`);
}
for (const required of [
  "render_supervisor_ready_tab($user_id, true)",
  "render_supervisor_ready_tab($user_id);",
  "case 'ss-converter-stats':",
  "case 'ss-my-conversions':",
  "render_converter_panel('supervisor')",
]) {
  if (!plugin.includes(required)) addError(`Supervisor queue/senior oversight wiring is missing: ${required}`);
}
checks.allSupervisorsDotScope = true;

// The seller's assessment-SMS choice is an immutable invoice-link snapshot.
// Legacy rows default to enabled; disabled rows enter conversion directly and
// every SMS entry point must refuse to send or schedule them.
for (const required of [
  'access_sms_enabled TINYINT(1) NOT NULL DEFAULT 1',
  'send_assessment_sms TINYINT(1) NOT NULL DEFAULT 1',
  "'send_assessment_sms' => $send_assessment_sms ? 1 : 0",
  "'status' => $send_access_sms ? 'awaiting_access_sms' : 'ready_for_conversion'",
  "'sms_due_at' => $send_access_sms ? $due_at : null",
  'repair_paid_assessment_cases_for_seller',
]) {
  if (!dotFlow.includes(required)) addError(`Assessment SMS choice invariant is missing: ${required}`);
}
const createAssessmentCaseBody = extractFunctions(dotFlow).get('create_case_for_source_invoice') || '';
if (!createAssessmentCaseBody.includes('if ( $send_access_sms ) { $this->dispatch_case_sms( $case_id ); }')) {
  addError('SMS-disabled assessments can still dispatch an access SMS after case creation.');
}
const dispatchAssessmentSmsBody = extractFunctions(dotFlow).get('dispatch_case_sms') || '';
const sendAssessmentSmsBody = extractFunctions(dotFlow).get('send_access_sms') || '';
const dueAssessmentSmsBody = extractFunctions(dotFlow).get('process_due_access_sms') || '';
if (!dispatchAssessmentSmsBody.includes('access_sms_enabled') || !sendAssessmentSmsBody.includes('access_sms_enabled') || !dueAssessmentSmsBody.includes('access_sms_enabled=1')) {
  addError('Assessment SMS opt-out is not enforced at every queue/dispatch/send boundary.');
}
const sellerRepairBody = extractFunctions(dotFlow).get('repair_paid_assessment_cases_for_seller') || '';
for (const required of ["dl.conversion_route='seller_self'", 'i.seller_id=%d', 'create_case_for_source_invoice']) {
  if (!sellerRepairBody.includes(required)) addError(`Seller conversion queue repair invariant is missing: ${required}`);
}
const converterPanelBody = extractFunctions(dotFlow).get('render_converter_panel') || '';
if (!converterPanelBody.includes("'seller' === $context") || !converterPanelBody.includes('repair_paid_assessment_cases_for_seller')) {
  addError('Seller ready-conversion tab does not repair fully-paid seller_self assessment links.');
}
for (const required of ['id="sn-dot-send-assessment-sms"', 'name="dot_send_assessment_sms" value="0"', 'dot_send_assessment_sms_label']) {
  if (!plugin.includes(required) && !sellerJs.includes(required)) addError(`Assessment SMS seller UI invariant is missing: ${required}`);
}
for (const required of ['dot_assessment_decision_ui', 'dot_route_choice_present', 'ابتدا وضعیت «ارسال پیامک نتیجه اعتبارسنجی»']) {
  if (!plugin.includes(required)) addError(`Assessment decision-order server invariant is missing: ${required}`);
}
for (const required of ['sn-dot-conversion-owner-options', "prop('disabled',!smsChoiceMade)", "typeof dotSmsChoice==='undefined'", 'dot_assessment_decision_ui:hasAssessment?1:0']) {
  if (!sellerJs.includes(required) && !plugin.includes(required)) addError(`Assessment decision-order UI invariant is missing: ${required}`);
}
checks.assessmentSmsChoice = true;

// Drafting a WooCommerce product removes it from every new-sale selector, but
// historical invoices and migrated conversion rows retain stored snapshots.
const helpers = source('includes/class-sn-helpers.php');
if (!helpers.includes("'post_status'    => 'publish'")) addError('Draft products can leak into the new-sale product selector.');
for (const required of [
  "get_post_status($pid) !== 'publish'",
  "'product_name' => $product->get_name()",
  'SELECT product_name,unit_price,total_price,qty',
  'سابقه پرونده حفظ شده است',
]) {
  if (!plugin.includes(required) && !dotFlow.includes(required)) addError(`Draft-safe product history invariant is missing: ${required}`);
}
checks.draftProductHistorySafety = true;

// Product reclassification freezes missing historical snapshots, while wallet
// posting is safe against partial invoices, legacy credits and concurrent runs.
for (const required of [
  'UPDATE {$items_table} SET product_type=%s',
  'Existing non-empty snapshots are immutable',
  'تغییر این گزینه فقط روی فاکتورهای جدید اثر دارد',
]) {
  if (!plugin.includes(required)) addError(`Product reclassification safety invariant is missing: ${required}`);
}
for (const required of [
  'idempotency_key VARCHAR(191) DEFAULT NULL',
  'UNIQUE KEY idempotency_key (idempotency_key)',
  "type IN ('commission_engine','seller_commission')",
  "in_array($workflow, ['partial_paid', 'awaiting_next_payment', 'payment_stage_issued']",
  'on_invoice_financial_rejected_commission',
]) {
  if (!plugin.includes(required)) addError(`Commission update-safety invariant is missing: ${required}`);
}
for (const required of ['Asia/Tehran', 'tehran_timezone', 'setTimezone(self::tehran_timezone())']) {
  if (!helpers.includes(required)) addError(`Tehran/Jalali date invariant is missing: ${required}`);
}
for (const required of [
  'tehran_now',
  'tehran_format',
  'site_mysql_from_timestamp',
  'site_mysql_timestamp',
  'tehran_today_gregorian',
  'normalize_jalali_tehran_datetime',
  "new DateTimeZone('Asia/Tehran')",
]) {
  if (!helpers.includes(required)) addError(`Shared Tehran time safety helper is missing: ${required}`);
}
if (plugin.includes('type="datetime-local"')) addError('A Gregorian/browser-local datetime input remains in the CRM panels.');
if (plugin.includes("seller_next_followup_at ?? '—'")) addError('Seller follow-up still renders a raw Gregorian database value.');
for (const file of phpFiles) {
  const runtimeDates = source(file).replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/[^\n]*/g, '');
  if (/\b(?:date|gmdate|strtotime|wp_date)\s*\(/.test(runtimeDates)) {
    addError(`Timezone-sensitive date function remains in ${file}; use the shared Tehran/storage helpers.`);
  }
}
for (const file of jsFiles) {
  if (source(file).includes("new Date(new Date().toLocaleString('en-US',{timeZone:'Asia/Tehran'}))")) {
    addError(`Locale-string Tehran parsing remains in ${file}; use Intl.formatToParts.`);
  }
}
const invoiceAjax = extractFunctions(plugin).get('ajax_supervisor_invoices') || '';
for (const required of [
  "['created_at', 'approved_at', 'paid_at', 'updated_at', 'date']",
  "$r[$date_field . '_jalali'] = SN_Helpers::gregorian_to_jalali_date",
]) {
  if (!invoiceAjax.includes(required)) addError(`Invoice Jalali/Tehran API invariant is missing: ${required}`);
}
if (!managerScript.includes('i.approved_at_jalali||i.paid_at_jalali||i.created_at_jalali||i.updated_at_jalali||i.date_jalali')) {
  addError('Sales-manager invoice table still does not use the Jalali/Tehran API fields.');
}
if (!supervisorJs.includes('i.created_at_jalali || i.approved_at_jalali || i.paid_at_jalali || i.updated_at_jalali || i.date_jalali')) {
  addError('Supervisor invoice table still does not use the Jalali/Tehran API fields.');
}
if (managerScript.includes("var date=i.approved_at||i.paid_at||i.created_at||i.date||'—'") ||
    supervisorJs.includes("var date = i.created_at || i.approved_at || i.paid_at || i.updated_at || i.date || '—'")) {
  addError('A manager/supervisor invoice table still renders a raw database datetime.');
}
checks.managerSupervisorInvoiceJalaliTehran = true;
checks.jalaliTehranTimeSafety = true;
checks.commissionAndDateSafety = true;

// Manager archive counts and rows must both exclude resolved records. Legacy
// source-table absence must not break the whole archive tab.
for (const required of [
  "'resolved_at IS NULL'",
  "'archive_type=%s AND resolved_at IS NULL'",
  "'SHOW TABLES LIKE %s'",
  'maybe_process_unpaid_archives( 5 )',
]) {
  if (!sellerFlow.includes(required) && !dotFlow.includes(required)) addError(`Sales-manager archive invariant is missing: ${required}`);
}
checks.salesManagerArchives = true;

// Project membership routing may override the legacy/default project manager
// by the original seller's sales-manager branch. The schema is additive and
// historic membership assignments must never be backfilled or rewritten.
const projects = source('includes/class-sn-projects.php');
for (const required of [
  'sn_project_manager_sales_routes',
  'UNIQUE KEY content_sales_manager (content_product_id,sales_manager_user_id)',
  'sn_project_manager_routes_present',
  'resolve_source_sales_manager',
  '( $invoice->original_seller_id ?? 0 )',
  'c.converter_id,c.seller_id',
  'apply_sales_branch_routes',
  "'manager_route' =>",
  'source_sales_manager_user_id',
  "'manager_routes_updated' => $route_controls_present",
]) {
  if (!projects.includes(required)) addError(`Project sales-manager routing invariant is missing: ${required}`);
}
if (/UPDATE[\s\S]{0,200}SET\s+source_sales_manager_user_id/i.test(projects)) {
  addError('Historic project memberships must not be backfilled to a current sales-manager branch.');
}
checks.projectSalesManagerRouting = true;

// The customer-facing Biawin portal must remain separate from the internal
// Customer 360 screen, scope reads to the authenticated customer and send one
// post-membership invite only after the immutable membership snapshots commit.
const customerPortal = source('includes/class-sn-customer-portal.php');
const activator = source('includes/class-sn-activator.php');
for (const required of [
  "require_once SN_PLUGIN_DIR . 'includes/class-sn-customer-portal.php'",
  'SN_Customer_Portal::instance()->register_hooks()',
]) {
  if (!bootstrap.includes(required)) addError(`Customer portal bootstrap wiring is missing: ${required}`);
}
for (const required of [
  'sn_biavin_customer_portal_page_id',
  "'slug' => 'biawin'",
  "'shortcode' => '[sn_biavin_customer_portal]'",
]) {
  if (!activator.includes(required)) addError(`Customer portal system-page invariant is missing: ${required}`);
}
for (const required of [
  'sn_customer_portal_send_otp',
  'sn_customer_portal_verify_otp',
  'wp_hash_password( $code )',
  'sn_customer_portal_active_otp_',
  'OTP_MAX_ATTEMPTS = 5',
  "in_array( 'customer', $roles, true )",
  'COALESCE(m.customer_wp_id,0)=0',
  'upgrade_options_snapshot_json',
  'customer_portal_invite_sent',
  "portal_invite_status='sending'",
  'portal_invite_sent_at IS NULL',
	'portal_invite_attempts',
	'INVITE_MAX_ATTEMPTS = 3',
	'sn_customer_portal_retry_invite',
  "add_filter( 'wp_robots'",
]) {
  if (!customerPortal.includes(required)) addError(`Customer portal security/lifecycle invariant is missing: ${required}`);
}
if ((projects.match(/do_action\( 'sn_project_membership_activated'/g) || []).length !== 2) {
  addError('Customer portal invite hook must run for both subscription and Product* membership commits.');
}
if (!projects.includes('portal_invite_sent_at DATETIME DEFAULT NULL') || !projects.includes('portal_invite_attempts INT UNSIGNED NOT NULL DEFAULT 0')) {
  addError('Customer portal invite idempotency columns are missing from the membership schema.');
}
if (!plugin.includes("'shortcode' => 'sn_biavin_customer_portal'")) {
  addError('Customer portal is missing from the system-page registry.');
}
checks.customerPortal = true;

// Customer OTP must remain on the lightweight bootstrap. Loading the complete
// CRM action registry or firing extension login hooks inside the verification
// response can exceed upstream time limits and surface as HTTP 502.
for (const required of [
  "'sn_customer_portal_send_otp'",
  "'sn_customer_portal_verify_otp'",
]) {
  if (!bootstrap.includes(required)) addError(`Customer portal OTP lightweight bootstrap invariant is missing: ${required}`);
}
const verifyCustomerOtp = extractFunctions(customerPortal).get('ajax_verify_otp') || '';
if (verifyCustomerOtp.includes("do_action( 'sn_customer_portal_login'")) {
  addError('Customer portal OTP verification still fires an extension hook before returning JSON.');
}
for (const required of [
  'try {',
  'wp_set_auth_cookie',
  'Consume the OTP only after the authentication cookie has been created',
]) {
  if (!verifyCustomerOtp.includes(required)) addError(`Customer portal OTP session safety invariant is missing: ${required}`);
}
checks.customerPortalOtpLightweight = true;
for (const required of [
  'private function reusable_otp',
  'کد قبلی هنوز معتبر است',
  "'reused' => true",
  'sn_customer_otp_v2_cd_',
]) {
  if (!customerPortal.includes(required)) addError(`Customer portal reusable OTP invariant is missing: ${required}`);
}
const customerPortalJs = source('assets/js/customer-portal.js');
if (!customerPortalJs.includes('requestingCode') || !customerPortalJs.includes('if (state.requestingCode)')) {
  addError('Customer portal does not suppress duplicate client-side OTP send requests.');
}
checks.customerPortalOtpReuse = true;
const smsService = source('includes/class-sn-sms.php');
for (const required of [
  'send_customer_portal_otp',
  'sn_faraz_pattern_customer_otp',
  'sn_faraz_pattern_customer_otp_variable',
  'send_faraz_pattern_once',
  "'timeout'   => 10",
]) {
  if (!smsService.includes(required)) addError(`Customer portal OTP pattern invariant is missing: ${required}`);
}
for (const required of [
  "'sn_faraz_pattern_customer_otp'",
  "'sn_faraz_pattern_customer_otp_variable'",
  'کد پترن ورود پروفایل مشتری',
]) {
  if (!plugin.includes(required)) addError(`Customer portal OTP pattern setting is missing: ${required}`);
}
const sendCustomerOtp = extractFunctions(customerPortal).get('ajax_send_otp') || '';
if (!sendCustomerOtp.includes('send_otp_sms') || sendCustomerOtp.includes("wp_send_json( [ 'success' => false, 'message' => 'ارسال پیامک انجام نشد؛ تنظیمات سرویس پیامک را بررسی کنید.' ], 502 )")) {
  addError('Customer portal OTP still uses the generic simple-SMS failure path.');
}
checks.customerPortalOtpPattern = true;

// Customer payment history must be derived only from invoices already scoped
// to the authenticated customer. The portal UI also owns its heading colors
// and compact transaction layout so theme-level typography cannot hide content.
for (const required of [
  'private function customer_payments',
  'p.invoice_id IN ({$placeholders})',
  'INNER JOIN {$invoice_table} i ON i.id=p.invoice_id',
  'سوابق پرداخت',
  'مجموع مبالغ تأییدشده',
  'payment_status_kind',
  'legacy_fallback',
]) {
  if (!customerPortal.includes(required)) addError(`Customer portal payment-history invariant is missing: ${required}`);
}
const customerPortalCss = source('assets/css/customer-portal.css');
for (const required of [
  '.sn-customer-section-head h2',
  'color: var(--sncp-ink) !important',
  '.sn-customer-payment-list',
  '.sn-customer-payment-row.is-success',
  'min-height: 0 !important',
  'repeat(auto-fit,minmax(min(100%,420px),1fr))',
]) {
  if (!customerPortalCss.includes(required)) addError(`Customer portal visual/payment invariant is missing: ${required}`);
}
checks.customerPortalPaymentHistory = true;

// The refreshed customer profile resolves location from account metadata with
// an invoice fallback, keeps notices compact, and exposes clear section links.
for (const required of [
  'private function customer_location',
  'province,city,COALESCE',
  '$this->customer_location( $user, $invoices, $memberships )',
  "array_map( static fn( $row ) => absint( $row['source_invoice_id']",
  "array_map( static fn( $row ) => absint( $row['dot_case_id']",
  'SELECT id,wc_order_id,province,city',
  'get_billing_state()',
  'get_billing_city()',
  'id="sn-customer-cards"',
	'id="sn-customer-purchases"',
  'id="sn-customer-payments"',
  "'ناموفق' : 'موفق'",
]) {
  if (!customerPortal.includes(required)) addError(`Customer profile refresh invariant is missing: ${required}`);
}
for (const required of [
  '.sn-customer-nav-links',
  '.sn-customer-identity i',
  'Customer profile refresh v5',
  'Customer profile and asset-card UX v7',
  '.sn-customer-card-dialog',
]) {
  if (!customerPortalCss.includes(required)) addError(`Customer profile refresh CSS is missing: ${required}`);
}
if (!customerPortalJs.includes("compact = 'ناموفق — ' + compact")) addError('Customer login notices are not compactly classified.');
for (const required of ['data-sn-customer-card-open', 'data-sn-customer-card-modal', 'data-sn-customer-card-close', 'sn-customer-modal-open']) {
  if (!customerPortal.includes(required) && !customerPortalJs.includes(required)) addError(`Customer card modal invariant is missing: ${required}`);
}
if (!customerPortal.includes("$card_state = $operations_stage === 'customer_code_issued'") || !customerPortal.includes('در حال پیگیری')) {
  addError('Customer cards do not expose a human-readable customer status.');
}
if (!customerPortalJs.includes("event.key === 'Tab'") || !customerPortalJs.includes("aria-expanded")) {
  addError('Customer card modal keyboard/focus handling is incomplete.');
}
checks.customerProfileRefresh = true;

// Card Operations is isolated from historic project state. New cards are
// eligible for inactivity routing, while lazily discovered legacy cards are not.
const operationsFlow = source('includes/class-sn-operations-flow.php');
const migrationService = source('includes/class-sn-migration-service.php');
for (const required of [
  'sn_project_operations',
  'auto_route_eligible TINYINT(1) NOT NULL DEFAULT 0',
  "stage VARCHAR(40) NOT NULL DEFAULT 'awaiting_customer'",
  "'sales_manager_queue'",
  "'executive_manager_queue'",
  "'sn_operations_sales_manager'",
	"'sn_operations_sales_supervisor'",
  "'sn_operations_sales_expert'",
  "'sn_operations_executive_manager'",
  'normalize_followup_datetime',
  'follow_up_reminder_attempts',
  "product_type'=>'project_upgrade'",
  'operations_upgrade_paid_routed_to_execution',
  'credit_operations_upgrade_commission',
]) {
  if (!operationsFlow.includes(required)) addError(`Card Operations invariant is missing: ${required}`);
}
const seedOperations = extractFunctions(operationsFlow).get('seed_membership_cards') || '';
const lazyOperations = extractFunctions(operationsFlow).get('ensure_operation') || '';
if (!seedOperations.includes('auto_route_eligible') || !seedOperations.includes("VALUES (%d,%s,1")) addError('New membership cards are not explicitly eligible for inactivity routing.');
if (!lazyOperations.includes('$auto_eligible ? 1 : 0')) addError('Legacy/lazy card safety flag is missing.');
const operationsPayment = extractFunctions(operationsFlow).get('on_invoice_paid') || '';
if (!operationsPayment.includes("payment_state='paid'") || !operationsPayment.includes("stage='executive_manager_queue'")) addError('Paid upgrade is not atomically routed to execution.');
for (const required of [
  'FOR UPDATE',
  "'payment_reversed','cancelled_continue_normal','cancelled_by_customer'",
  'credit_operations_upgrade_commission',
]) {
  if (!operationsPayment.includes(required)) addError(`Operations paid-upgrade idempotency invariant is missing: ${required}`);
}
if (operationsPayment.includes("status<>'paid'")) addError('A repeated paid hook can regress a fulfilled Operations action back to paid.');
const operationsInvoiceSms = extractFunctions(operationsFlow).get('send_invoice_sms') || '';
const sharedPaymentSms = extractFunctions(dotFlow).get('send_sales_cycle_payment_sms') || '';
if (!operationsInvoiceSms.includes('send_invoice_link') || !operationsInvoiceSms.includes('new SN_SMS') || operationsInvoiceSms.includes('send_sales_cycle_payment_sms')) {
  addError('Operations upgrade invoice SMS does not use the canonical Sales invoice SMS path.');
}
if (!sharedPaymentSms.includes("send_configured_sms( 'payment'")) {
  addError('Shared payment SMS boundary does not use the configured Sales Cycle payment pattern.');
}
const operationsReversal = extractFunctions(operationsFlow).get('on_invoice_reversed') || '';
for (const required of [
  'START TRANSACTION',
  'FOR UPDATE',
  "status='payment_reversed'",
  "'payment_state' => 'rejected'",
  "'current_credit' => (float) $op->base_credit",
]) {
  if (!operationsReversal.includes(required)) addError(`Operations financial-reversal consistency invariant is missing: ${required}`);
}
const operationsReconcile = extractFunctions(operationsFlow).get('reconcile_paid_upgrades') || '';
for (const required of [
  "o.stage='upgrade_payment'",
  "COALESCE(o.payment_state,'')<>'paid'",
  'COALESCE(i.remaining_amount,0)<=0.5',
  '$this->on_invoice_paid',
]) {
  if (!operationsReconcile.includes(required)) addError(`Operations interrupted-payment recovery invariant is missing: ${required}`);
}
const projectPaid = extractFunctions(source('includes/class-sn-projects.php')).get('on_invoice_paid') || '';
if (!projectPaid.includes('is_project_action_invoice')) addError('Upgrade invoice can create a duplicate project membership.');
const activateFromInvoice = extractFunctions(projects).get('activate_from_invoice') || '';
if (!activateFromInvoice.includes('$activated_single_cards') || !activateFromInvoice.includes('|| $activated_single_cards')) {
  addError('A paid invoice containing multiple Product* rows can lose all but its first card.');
}
const gatewayCallback = extractFunctions(plugin).get('handle_zarinpal_callback') || '';
if (!gatewayCallback.includes('customer_return_url') || !gatewayCallback.includes('$is_operations_upgrade')) addError('Upgrade gateway callback does not return to the customer card.');
for (const required of ['sn_operations_sales_manager_panel_page_id', 'sn_operations_sales_supervisor_panel_page_id', 'sn_operations_sales_expert_panel_page_id', 'sn_operations_executive_manager_panel_page_id']) {
  if (!source('includes/class-sn-activator.php').includes(required) || !plugin.includes(required)) addError(`Operations system page wiring is missing: ${required}`);
}
for (const required of ['no_answer_count=no_answer_count+1', 'normalize_followup_datetime', "stage='sales_manager_queue'", 'cancelled_continue_normal']) {
  if (!operationsFlow.includes(required)) addError(`Operations outcome wiring is missing: ${required}`);
}
checks.cardOperationsFlow = true;

// Operations hierarchy routing: every future card snapshots the originating
// Sales Manager plus the resolved Operations manager/supervisor. Exact branch
// rules take precedence over the card default, while historic rows stay put.
for (const required of [
  "'operations_managers' => $wpdb->prefix . 'sn_project_operations_manager_products'",
  "'operations_supervisor_routes' => $wpdb->prefix . 'sn_project_operations_supervisor_sales_routes'",
  'operations_supervisor_user_id BIGINT UNSIGNED NOT NULL',
  'UNIQUE KEY content_sales_manager (content_product_id,sales_manager_user_id)',
  'sn_operations_supervisor_routes_present',
  'sn_projects[operations_managers]',
  'sn_projects[operations_supervisor_routes]',
  'sn_operations_supervisor_route_duplicate',
  "'operations_supervisor_routes_updated'",
]) {
  if (!projects.includes(required)) addError(`Operations supervisor routing setting is missing: ${required}`);
}
const resolveOperationsManager = extractFunctions(operationsFlow).get('resolve_operations_manager') || '';
for (const required of [
  'content_product_id=%d AND sales_manager_user_id=%d',
  "'source' => 'sales_manager_supervisor_route'",
  "'source' => 'sales_manager_route'",
  "'source' => 'card_default'",
  "'source' => 'unconfigured'",
]) {
  if (!resolveOperationsManager.includes(required)) addError(`Operations destination resolver precedence is missing: ${required}`);
}
const sourceSalesManagerResolver = extractFunctions(projects).get('resolve_source_sales_manager') || '';
const sourceCandidatesAt = sourceSalesManagerResolver.indexOf('$candidates = [');
if (sourceCandidatesAt < 0 || sourceSalesManagerResolver.indexOf('$fallback_user_id', sourceCandidatesAt) > sourceSalesManagerResolver.indexOf('(int) ( $invoice->seller_id', sourceCandidatesAt)) {
  addError('Dot case seller must take precedence over the later invoice issuer when resolving the source Sales Manager.');
}
for (const required of [
  'source_sales_manager_user_id BIGINT UNSIGNED DEFAULT NULL',
  'operations_sales_manager_user_id BIGINT UNSIGNED DEFAULT NULL',
  'sales_supervisor_user_id BIGINT UNSIGNED DEFAULT NULL',
  "routing_source VARCHAR(30) NOT NULL DEFAULT 'legacy'",
  "'operations_route_snapshotted'",
]) {
  if (!operationsFlow.includes(required)) addError(`Operations immutable routing snapshot is missing: ${required}`);
}
if (!seedOperations.includes('source_sales_manager_user_id') || !seedOperations.includes('operations_sales_manager_user_id') || !seedOperations.includes("'awaiting_customer'")) {
  addError('New membership cards do not snapshot both Sales and Operations ownership before customer action.');
}
const inactivityDaysBody = extractFunctions(operationsFlow).get('inactivity_days') || '';
const inactivityDueBody = extractFunctions(operationsFlow).get('inactivity_due_at') || '';
const inactivityRouterBody = extractFunctions(operationsFlow).get('route_inactive_cards') || '';
const routeOperationBody = extractFunctions(operationsFlow).get('route_operation_to_sales_manager') || '';
const customerActionBody = extractFunctions(operationsFlow).get('handle_customer_action') || '';
const customerControlsBody = extractFunctions(operationsFlow).get('render_customer_card_controls') || '';
for (const required of [
  'max( 0, absint(',
  "get_option( 'sn_operations_customer_inactivity_days', 3 )",
]) {
  if (!inactivityDaysBody.includes(required)) addError(`Operations inactivity setting does not support zero safely: ${required}`);
}
if (!inactivityDueBody.includes('SN_Helpers::site_mysql_from_timestamp') || !inactivityDueBody.includes('time() +') || inactivityDueBody.includes("current_time( 'timestamp', true )")) {
  addError('Operations inactivity deadline still mixes UTC timestamps with database-local time.');
}
for (const required of [
  '0 === $inactivity_days',
  '$this->route_operation_to_sales_manager( $operation, 0 )',
]) {
  if (!seedOperations.includes(required)) addError(`Zero-day Operations cards are not routed immediately at creation: ${required}`);
}
for (const required of [
  "o.auto_route_eligible=1 ORDER BY o.id ASC",
  'o.decision_due_at<=%s',
  '$this->route_operation_to_sales_manager',
]) {
  if (!inactivityRouterBody.includes(required)) addError(`Operations inactivity router branch is incomplete: ${required}`);
}
if (!routeOperationBody.includes("AND stage='awaiting_customer' AND auto_route_eligible=1")) {
  addError('Operations automatic routing can mutate a historical or already-processed card.');
}
if (!customerActionBody.includes('$this->maybe_route_operation_if_due( $op )') || !customerControlsBody.includes('$this->maybe_route_operation_if_due($op)')) {
  addError('Expired or zero-delay cards can still expose or submit a customer activation choice.');
}
for (const required of [
  'name="sn_operations_customer_inactivity_days" min="0"',
  'max( 0, absint( get_option(',
  "SN_Operations_Flow::instance()->route_inactive_cards( 500 )",
]) {
  if (!projects.includes(required)) addError(`Operations zero-day settings UI/save invariant is missing: ${required}`);
}
checks.operationsInactivityPolicy = true;
for (const required of [
  "'sn_operations_sales_supervisor'",
  "'sn_supervise_operations_sales'",
  "'sn_operations_sales_supervisor_panel'",
  "stage='sales_supervisor'",
  "stage='sales_expert'",
  "'assign_supervisor'",
  "'assign_expert'",
  "valid_direct_report( $manager_id, $supervisor_id, 'operations_sales_supervisor' )",
  "valid_direct_report( $supervisor_id, $expert_id, 'operations_sales_expert' )",
]) {
  if (!operationsFlow.includes(required)) addError(`Manager-to-supervisor-to-expert chain is missing: ${required}`);
}
const panelRows = extractFunctions(operationsFlow).get('panel_rows') || '';
for (const required of [
  'o.operations_sales_manager_user_id=%d',
  'o.sales_supervisor_user_id=%d',
  'o.sales_expert_user_id=%d',
]) {
  if (!panelRows.includes(required)) addError(`Operations panel scope is missing: ${required}`);
}
if (panelRows.includes('operations_sales_manager_user_id IS NULL') || panelRows.includes('COALESCE(o.operations_sales_manager_user_id,0)=0')) {
  addError('Unowned legacy Operations rows are exposed to every Operations Sales Manager.');
}
for (const required of [
  "'operations_sales_supervisor' => ['operations_sales_manager']",
  "'operations_sales_expert' => ['operations_sales_supervisor']",
  "'operations_sales_supervisors' => ['child' => 'operations_sales_supervisor'",
  "'operations_sales_experts' => ['child' => 'operations_sales_expert'",
]) {
  if (!plugin.includes(required)) addError(`HR Operations hierarchy management is missing: ${required}`);
}
for (const required of [
  "'operations_sales_manager' => [ 'label' => 'مدیر فروش عملیات'",
  "'operations_sales_supervisor' => [ 'label' => 'سرپرست فروش عملیات'",
  "'operations_sales_expert' => [ 'label' => 'کارشناس فروش عملیات'",
  "'sn_operations_sales_manager' => 'operations_sales_manager'",
  "'sn_operations_sales_supervisor' => 'operations_sales_supervisor'",
  "'sn_operations_sales_expert' => 'operations_sales_expert'",
]) {
  if (!migrationService.includes(required)) addError(`Fresh-install HR seed is missing an Operations Sales contract: ${required}`);
}
for (const required of [
  "'operations_sales_manager' => ['label' => 'مدیر فروش عملیات'",
  "'operations_sales_supervisor' => ['label' => 'سرپرست فروش عملیات'",
  "'operations_sales_expert' => ['label' => 'کارشناس فروش عملیات'",
	"'مدیر فروش عملیات' => 'operations_sales_manager'",
	"'سرپرست فروش عملیات' => 'operations_sales_supervisor'",
	"'کارشناس فروش عملیات' => 'operations_sales_expert'",
]) {
  if (!plugin.includes(required)) addError(`HR runtime self-repair is missing an Operations Sales position: ${required}`);
}
for (const required of [
  '! $this->hr_contract_is_complete()',
  "'sn_operations_sales_manager' => 'operations_sales_manager'",
  "'sn_operations_sales_supervisor' => 'operations_sales_supervisor'",
  "'sn_operations_sales_expert' => 'operations_sales_expert'",
  "$wpdb->update( $mappings",
]) {
  if (!operationsFlow.includes(required)) addError(`Operations HR self-heal is incomplete: ${required}`);
}
checks.operationsFreshInstallHrRoles = true;
for (const required of [
	"'no_answer', 'cancel', 'refer_normal', 'create_upgrade'",
	"$action === 'follow_up'",
  'cancel_reason',
  "stage='cancelled'",
  'normalize_followup_datetime',
  "'timezone' => 'Asia/Tehran'",
  'wp_clear_scheduled_hook',
  'operations_upgrade_paid_routed_to_execution',
]) {
  if (!operationsFlow.includes(required)) addError(`Operations expert outcome is missing: ${required}`);
}
const followupScheduler = extractFunctions(operationsFlow).get('schedule_existing_followup') || '';
const followupReminder = extractFunctions(operationsFlow).get('send_followup_reminder') || '';
if (followupScheduler.includes('wp_schedule_single_event') || followupReminder.includes('wp_schedule_single_event') || followupReminder.includes('new SN_SMS') || followupReminder.includes('->send(')) {
  addError('Operations callback date still schedules or sends an SMS reminder.');
}
if (operationsFlow.includes("add_action( self::FOLLOWUP_HOOK") || !operationsFlow.includes('wp_unschedule_hook( self::FOLLOWUP_HOOK )')) {
  addError('Legacy Operations callback SMS cron hooks are not fully disabled.');
}
if (!customerPortal.includes('خرید شما با موفقیت وارد چرخه بیاوین شد') || !customerPortal.includes('$this->portal_url()') || !customerPortal.includes('SET customer_wp_id=%d') || !customerPortal.includes('COALESCE(customer_wp_id,0)=0')) {
  addError('Paid-card customer account/link SMS lifecycle is incomplete.');
}
if (!customerPortal.includes("'operations_upgrade_payment_reversed' =>")) {
  addError('Customer timeline does not disclose an Operations upgrade payment reversal.');
}
const portalInvite = extractFunctions(customerPortal).get('send_membership_invite') || '';
if (!portalInvite.includes('portal_invite_attempts<%d') || !portalInvite.includes('portal_invite_attempted_at<%s')) {
  addError('Customer portal invitation retry is not atomically rate-limited.');
}
if (portalInvite.includes('strtotime') || portalInvite.includes('DateTimeImmutable')) {
  addError('Customer portal invitation retry compares database-local timestamps in a timezone-unsafe runtime parser.');
}
const operationsJs = source('assets/js/operations-flow.js');
for (const required of [
  'Asia/Tehran',
  'gregorianToJalali',
  'jalaliToGregorian',
  'jalaliMonthLength',
  'data-sn-ops-followup-submit',
  "document.querySelectorAll('.sn-ops-jalali-date')",
  'submit.disabled = !parseJalali(date.value)',
  "document.body.appendChild(picker)",
  "picker.className = 'sn-ops-jalali-picker sn-ops-jalali-floating'",
  "button.classList.add('is-past')",
]) {
  if (!operationsJs.includes(required)) addError(`Operations Jalali follow-up UI invariant is missing: ${required}`);
}
if (!operationsFlow.includes('class="sn-ops-jalali-date"') || !operationsFlow.includes('data-sn-ops-followup-submit')) {
  addError('Operations follow-up form is not connected to the Jalali date guard.');
}
if (operationsFlow.includes('name="followup_time"') || operationsJs.includes('input[name="followup_time"]')) {
  addError('An Operations callback form still asks for an hour instead of date-only input.');
}
const normalizeOperationsFollowup = extractFunctions(operationsFlow).get('normalize_followup_datetime') || '';
for (const required of ['SN_Helpers::normalize_jalali_tehran_datetime', "'09:00'", "['jalali']"]) {
  if (!normalizeOperationsFollowup.includes(required)) addError(`Operations date-only storage invariant is missing: ${required}`);
}
for (const required of [
  "$_POST['followup_date']",
  'تاریخ مناسب تماس را انتخاب کنید',
  "private function jalali_date",
  'SN_Helpers::normalize_jalali_tehran_datetime',
  'sn-customer-callback-time',
  "'assign_self'",
  'operations_supervisor_self_assigned',
  "position === 'operations_sales_supervisor'",
  'wp_logout_url',
  'data-sn-ops-status',
  'data-status=',
  'مدیر فروش مبدأ',
]) {
  if (!operationsFlow.includes(required)) addError(`Operations callback/retention panel invariant is missing: ${required}`);
}
for (const required of [
  "o.operations_sales_manager_user_id=%d AND o.stage<>'awaiting_customer'",
  "'o.sales_supervisor_user_id=%d'",
  "'o.sales_expert_user_id=%d'",
]) {
  if (!panelRows.includes(required)) addError(`Operations retained ownership query is missing: ${required}`);
}
for (const required of [
  "panel.querySelector('[data-sn-ops-status]')",
  "card.getAttribute('data-status') === selectedStatus",
  'matchesSearch && matchesStatus',
]) {
  if (!operationsJs.includes(required)) addError(`Operations combined status filter is missing: ${required}`);
}
if (!customerPortal.includes("'operations_supervisor_self_assigned' =>")) {
  addError('Customer timeline does not disclose Supervisor self-assignment.');
}
checks.operationsCallbackRetention = true;

// Operations bulk assignment is limited to actionable queue rows and the
// actor's HR branch. The UI selects only visible/filter-matched cards, while
// the server revalidates every selected operation against current state.
const bulkAssignmentBody = extractFunctions(operationsFlow).get('handle_bulk_assignment') || '';
const staffActionBody = extractFunctions(operationsFlow).get('handle_staff_action') || '';
const operationsPanelBody = extractFunctions(operationsFlow).get('render_panel') || '';
const caseWorkActionsBody = extractFunctions(operationsFlow).get('render_case_work_actions') || '';
for (const required of [
  "'bulk_assign_supervisor'",
  "'bulk_assign_expert'",
  'count( $raw_ids ) > 500',
  '$this->can_manage_case( $op, $actor )',
  '$this->can_supervise_case( $op, $actor )',
  '$this->valid_direct_report( $manager_id, $target_id, $target_position )',
  '$this->valid_direct_report( $supervisor_id, $target_id, $target_position )',
  "stage !== 'sales_manager_queue'",
  "stage !== 'sales_supervisor'",
  "AND stage='sales_manager_queue' AND operations_sales_manager_user_id=%d",
  "AND stage='sales_supervisor' AND sales_supervisor_user_id=%d",
]) {
  if (!bulkAssignmentBody.includes(required)) addError(`Operations bulk-assignment server invariant is missing: ${required}`);
}
if (!staffActionBody.includes("check_admin_referer( 'sn_operations_bulk_assign' )") || !staffActionBody.includes('$this->handle_bulk_assignment( $action, $actor )')) {
  addError('Operations bulk assignment is not protected by its dedicated nonce/handler boundary.');
}
for (const required of [
  'name="operation_ids[]"',
  'data-sn-ops-bulk-checkbox',
  'data-sn-ops-select-visible',
  'data-sn-ops-bulk-target',
  "(string) $row->stage === 'sales_manager_queue'",
  "(string) $row->stage === 'sales_supervisor'",
]) {
  if (!operationsPanelBody.includes(required)) addError(`Operations bulk-assignment panel wiring is missing: ${required}`);
}
for (const required of [
  'data-sn-ops-case-status',
  'data-sn-ops-status-select',
  'data-sn-ops-status-fields="no_answer"',
  'data-sn-ops-status-fields="follow_up"',
  'data-sn-ops-status-fields="cancel"',
  'name="cancel_reason"',
  'number_format_i18n( (int) $row->no_answer_count )',
  "value=\"refer_normal\"",
  "value=\"create_upgrade\"",
  "value=\"retry_payment\"",
  "value=\"revert_normal\"",
]) {
  if (!caseWorkActionsBody.includes(required)) addError(`Unified Operations case-status selector is missing: ${required}`);
}
for (const required of [
  "if (!card.hidden && checkbox && !checkbox.disabled)",
  "bulkCheckboxes().filter(function (checkbox) { return checkbox.checked; })",
  "group.getAttribute('data-sn-ops-status-fields') === action",
  "action === 'follow_up'",
  "action === 'cancel'",
  "action === 'create_upgrade'",
]) {
  if (!operationsJs.includes(required)) addError(`Operations bulk/status client invariant is missing: ${required}`);
}
checks.operationsBulkAssignmentAndStatus = true;

for (const required of ['data-sn-ops-select-all', 'visibleSelectable', 'selectAll.indeterminate']) {
  if (!operationsJs.includes(required) && !operationsFlow.includes(required)) addError(`Operations Select All invariant is missing: ${required}`);
}
checks.operationsSelectAll = true;

for (const required of ["$requested_lead_id < 1 && ! $this->sn_seller_manual_invoice_enabled()", 'پیش‌فاکتور فقط از داخل پرونده', '$show_manual_invoice_tab = $can_issue_invoice && $manual_invoice_enabled', 'sn_effective_manual_invoice_access_mode']) {
  if (!plugin.includes(required)) addError(`HR-controlled manual invoice access is missing: ${required}`);
}
checks.supervisorInvoiceSourcePolicy = true;

for (const required of ['wp_ajax_sn_full_data_reset', 'ajax_full_data_reset', 'حذف کامل داده‌ها', 'sn_last_full_data_reset_report', 'sn-settings-reset']) {
  if (!plugin.includes(required)) addError(`Protected full reset tool is missing: ${required}`);
}
checks.protectedFullReset = true;
try {
  const instrumentedCalendar = operationsJs.replace(
    /\}\(\)\);\s*$/,
    "globalThis.__snOpsCalendar={gregorianToJalali:gregorianToJalali,jalaliToGregorian:jalaliToGregorian,jalaliMonthLength:jalaliMonthLength,parseJalali:parseJalali,updateCaseStatusForm:updateCaseStatusForm};}());"
  );
  if (instrumentedCalendar === operationsJs) throw new Error('calendar export point not found');
  const calendarContext = {
    document: { querySelectorAll: () => [], addEventListener: () => {} },
    Intl,
    Date,
    Event: function Event() {},
  };
  vm.runInNewContext(instrumentedCalendar, calendarContext, { filename: 'operations-flow.js' });
  const calendar = calendarContext.__snOpsCalendar;
  const knownDates = [
    [2023, 3, 21, 1402, 1, 1],
    [2024, 3, 20, 1403, 1, 1],
    [2025, 3, 21, 1404, 1, 1],
    [2026, 3, 21, 1405, 1, 1],
  ];
  for (const [gy, gm, gd, jy, jm, jd] of knownDates) {
    const j = calendar.gregorianToJalali(gy, gm, gd);
    const g = calendar.jalaliToGregorian(jy, jm, jd);
    if (j.jy !== jy || j.jm !== jm || j.jd !== jd || g.gy !== gy || g.gm !== gm || g.gd !== gd) {
      throw new Error(`date mismatch for ${gy}-${gm}-${gd}`);
    }
  }
  if (calendar.jalaliMonthLength(1403, 12) !== 30 || calendar.jalaliMonthLength(1404, 12) !== 29) throw new Error('Esfand leap length mismatch');
  if (!calendar.parseJalali('۱۴۰۳/۱۲/۳۰') || calendar.parseJalali('۱۴۰۴/۱۲/۳۰')) throw new Error('Jalali day validation mismatch');

  const makeField = (value, required) => ({
    value: value || '', disabled: true, required: false,
    hasAttribute: (name) => name === 'data-sn-ops-required' && Boolean(required),
  });
  const actionSelect = { value: 'cancel' };
  const submit = { disabled: false };
  const reason = makeField('', true);
  const dateField = makeField('', true);
  const groups = [
    { key: 'no_answer', fields: [], hidden: false },
    { key: 'follow_up', fields: [dateField], hidden: false },
    { key: 'cancel', fields: [reason], hidden: false },
  ].map((group) => ({
    hidden: group.hidden,
    getAttribute: () => group.key,
    querySelectorAll: () => group.fields,
  }));
  const statusForm = {
    querySelectorAll: () => groups,
    querySelector: (selector) => ({
      '[data-sn-ops-status-select]': actionSelect,
      '[data-sn-ops-status-submit]': submit,
      'textarea[name="cancel_reason"]': reason,
      'input[name="followup_date"]': dateField,
    }[selector] || null),
  };
  calendar.updateCaseStatusForm(statusForm);
  if (!submit.disabled || reason.disabled || !reason.required) throw new Error('mandatory cancellation reason guard mismatch');
  reason.value = 'علت'; calendar.updateCaseStatusForm(statusForm);
  if (submit.disabled) throw new Error('valid cancellation reason remains blocked');
  actionSelect.value = 'follow_up'; dateField.value = '۱۴۰۵/۰۶/۲۳'; calendar.updateCaseStatusForm(statusForm);
  if (submit.disabled || dateField.disabled || !dateField.required) throw new Error('follow-up date guard mismatch');
  checks.operationsJalaliCalendar = true;
} catch (error) {
  addError(`Operations Jalali calendar executable test failed: ${error.message}`);
}
const hierarchyService = source('includes/class-sn-hierarchy-service.php');
for (const required of [
  "'operations_sales_supervisor' => [ 'operations_sales_manager' ]",
  "'operations_sales_expert' => [ 'operations_sales_supervisor' ]",
]) {
  if (!hierarchyService.includes(required)) addError(`Hierarchy service Operations parent rule is missing: ${required}`);
}
checks.operationsHierarchyRouting = true;

// WordPress supplies a legacy empty-string placeholder to no-payload actions.
// The init fallback must reject hook arguments at registration and normalize
// direct-call input before passing it to the strictly typed worker.
if (!dotFlow.includes("add_action( 'init', [ $this, 'maybe_process_unpaid_archives' ], 26, 0 );")) {
  addError('Unpaid-archive init callback must accept zero WordPress hook arguments.');
}
if (!dotFlow.includes('$limit = is_numeric( $limit ) ? (int) $limit : 50;')) {
  addError('Unpaid-archive init callback does not normalize incidental input.');
}
if (/function\s+maybe_process_unpaid_archives\s*\(\s*int\s+\$limit/.test(dotFlow)) {
  addError('Unpaid-archive init callback retains a crash-prone scalar type declaration.');
}
if (!dotFlow.includes("add_action( 'init', [ $this, 'maybe_process_due_followup_reminders' ], 27, 0 );")) {
  addError('Follow-up reminder init callback must accept zero WordPress hook arguments.');
}
checks.wordpressInitCallbackSafety = true;

// Every literal plugin include/asset reference must exist.
let literalReferences = 0;
for (const file of phpFiles) {
  const text = source(file);
  for (const match of text.matchAll(/SN_PLUGIN_(?:URL|DIR)\s*\.\s*['"]([^'"]+)['"]/g)) {
    literalReferences++;
    if (!fs.existsSync(path.join(root, match[1]))) addError(`Missing referenced file: ${file} -> ${match[1]}`);
  }
}
checks.literalFileReferences = literalReferences;

// JavaScript parser check.
let checkedJavaScript = 0;
for (const file of jsFiles) {
  const result = childProcess.spawnSync(process.execPath, ['--check', path.join(root, file)], { encoding: 'utf8' });
  if (result.status !== 0) addError(`JavaScript syntax failed: ${file}: ${(result.stderr || result.stdout).trim()}`);
  checkedJavaScript++;
}
checks.javascriptFiles = checkedJavaScript;

// Lightweight CSS structural check.
for (const file of cssFiles) {
  const clean = stripCssStringsAndComments(source(file));
  let balance = 0;
  for (const character of clean) {
    if (character === '{') balance++;
    if (character === '}') balance--;
    if (balance < 0) break;
  }
  if (balance !== 0) addError(`Unbalanced CSS braces: ${file}`);
}
checks.cssFiles = cssFiles.length;

// PHP method calls and hook callbacks must resolve inside their class file.
let phpMethods = 0;
let thisCalls = 0;
let staticClassCalls = 0;
let duplicateMethods = 0;
const classMethods = new Map();
for (const file of phpFiles.filter((name) => name.startsWith('includes/'))) {
  const text = source(file);
  const methods = methodNames(text);
  const methodOccurrences = new Map();
  for (const match of text.matchAll(/function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(/g)) {
    methodOccurrences.set(match[1], (methodOccurrences.get(match[1]) || 0) + 1);
  }
  for (const [method, count] of methodOccurrences) {
    if (count > 1) {
      duplicateMethods++;
      addError(`Duplicate method declaration: ${file} -> ${method}() (${count})`);
    }
  }
  phpMethods += methods.size;
  const classMatch = text.match(/(?:final\s+)?class\s+(SN_[A-Za-z0-9_]+)/);
  if (classMatch) classMethods.set(classMatch[1], methods);
  for (const call of text.matchAll(/\$this->([A-Za-z_][A-Za-z0-9_]*)\s*\(/g)) {
    thisCalls++;
    if (!methods.has(call[1])) addError(`Undefined method call: ${file} -> ${call[1]}()`);
  }
  for (const callback of text.matchAll(/\[\s*\$this\s*,\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]/g)) {
    if (!methods.has(callback[1])) addError(`Undefined hook callback: ${file} -> ${callback[1]}()`);
  }
}
for (const file of phpFiles) {
  for (const call of source(file).matchAll(/\b(SN_[A-Za-z0-9_]+)::([A-Za-z_][A-Za-z0-9_]*)\s*\(/g)) {
    staticClassCalls++;
    if (classMethods.has(call[1]) && !classMethods.get(call[1]).has(call[2])) {
      addError(`Undefined static call: ${file} -> ${call[1]}::${call[2]}()`);
    }
  }
  for (const callback of source(file).matchAll(/\[\s*['"](SN_[A-Za-z0-9_]+)['"]\s*,\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]/g)) {
    if (classMethods.has(callback[1]) && !classMethods.get(callback[1]).has(callback[2])) {
      addError(`Undefined static callback: ${file} -> ${callback[1]}::${callback[2]}()`);
    }
  }
}
checks.phpMethods = phpMethods;
checks.thisMethodCalls = thisCalls;
checks.staticClassCalls = staticClassCalls;
checks.duplicateMethods = duplicateMethods;

// Literal localization targets and plugin script dependencies must reference a
// handle that is enqueued/registered somewhere in the package. WordPress core
// handles are explicitly allowlisted.
const scriptHandles = new Set(['jquery', 'jquery-ui-core', 'jquery-ui-sortable', 'wp-api-fetch']);
const localizedHandles = new Set();
const scriptDependencies = new Set();
for (const file of phpFiles) {
  const text = source(file);
  for (const match of text.matchAll(/wp_(?:enqueue|register)_script\(\s*['"]([^'"]+)['"]/g)) scriptHandles.add(match[1]);
  for (const match of text.matchAll(/wp_localize_script\(\s*['"]([^'"]+)['"]/g)) localizedHandles.add(match[1]);
  for (const call of text.matchAll(/wp_(?:enqueue|register)_script\(\s*['"][^'"]+['"]\s*,[\s\S]*?\[([^\]]*)\]/g)) {
    for (const dependency of call[1].matchAll(/['"]([^'"]+)['"]/g)) scriptDependencies.add(dependency[1]);
  }
}
for (const handle of localizedHandles) {
  if (!scriptHandles.has(handle)) addError(`Localized script handle is never registered/enqueued: ${handle}`);
}
for (const handle of scriptDependencies) {
  if (handle.startsWith('sn-') && !scriptHandles.has(handle)) addError(`Plugin script dependency is never registered/enqueued: ${handle}`);
}
checks.scriptHandles = scriptHandles.size;
checks.localizedHandles = localizedHandles.size;

// Bootstrap and core shortcode registries must remain in parity.
const lightBody = extractFunctions(bootstrap).get('sn_bootstrap_shortcodes') || '';
const coreText = source('includes/class-sn-plugin.php');
const coreFunctions = extractFunctions(coreText);
const coreBody = coreFunctions.get('register_shortcodes') || '';
const lightShortcodes = new Set(Array.from(lightBody.matchAll(/['"](sn_[a-z0-9_]+)['"]\s*=>/g), (match) => match[1]));
const coreShortcodes = new Set(Array.from(coreBody.matchAll(/add_shortcode\(\s*['"](sn_[a-z0-9_]+)['"]/g), (match) => match[1]));
for (const shortcode of coreShortcodes) {
  if (!lightShortcodes.has(shortcode)) addError(`Core shortcode missing from light bootstrap: ${shortcode}`);
}
for (const shortcode of lightShortcodes) {
  if (!coreShortcodes.has(shortcode)) addError(`Light shortcode missing from core registry: ${shortcode}`);
}
checks.shortcodes = lightShortcodes.size;

// HR panel resolver and portal registry must point only to registered shortcodes.
const allRegisteredShortcodes = new Set(coreShortcodes);
for (const file of phpFiles) {
  for (const match of source(file).matchAll(/add_shortcode\(\s*['"](sn_[a-z0-9_]+)['"]/g)) {
    allRegisteredShortcodes.add(match[1]);
  }
}
const hrService = source('includes/class-sn-hr-service.php');
const routedShortcodes = new Set(Array.from(
  hrService.matchAll(/['"]shortcode['"]\s*=>\s*['"](sn_[a-z0-9_]+)['"]/g),
  (match) => match[1]
));
for (const shortcode of routedShortcodes) {
  if (!allRegisteredShortcodes.has(shortcode)) addError(`HR panel route uses an unregistered shortcode: ${shortcode}`);
}
const registryStart = coreText.indexOf('private function sn_portal_page_registry');
const registryEnd = coreText.indexOf('private function sn_portal_page_ids', registryStart);
const registryText = registryStart >= 0 && registryEnd > registryStart ? coreText.slice(registryStart, registryEnd) : '';
if (!registryText) addError('Portal page registry could not be located.');
const portalShortcodes = new Set(Array.from(
  registryText.matchAll(/['"]shortcode['"]\s*=>\s*['"](sn_[a-z0-9_]+)['"]/g),
  (match) => match[1]
));
for (const shortcode of portalShortcodes) {
  if (!allRegisteredShortcodes.has(shortcode)) addError(`Portal registry uses an unregistered shortcode: ${shortcode}`);
}
checks.panelRoutes = routedShortcodes.size;
checks.portalPages = portalShortcodes.size;

// Every operational WordPress role must resolve to the same HR position in
// activation, migration, full-scope and fast-login paths. A mismatch here can
// expose the wrong panel or leave a legitimate role with an empty dashboard.
const activatorText = source('includes/class-sn-activator.php');
const migrationText = source('includes/class-sn-migration-service.php');
const scopeText = source('includes/class-sn-scope-service.php');
const rolePositionContract = {
  sn_seller: 'seller',
  sn_supervisor: 'supervisor',
  sn_senior_supervisor: 'senior_supervisor',
  sn_sales_manager: 'sales_manager',
  sn_sales_deputy: 'sales_deputy',
  sn_converter: 'converter',
  sn_finance: 'finance',
  sn_hr: 'hr',
  sn_mis: 'mis',
  sn_after_sales: 'after_sales',
	  sn_shipping_expert: 'shipping_expert',
	  sn_operations_execution_expert: 'operations_execution_expert',
	};
const hrFallbackBody = extractFunctions(hrService).get('fallback_position_from_legacy_role') || '';
const scopeFallbackBody = extractFunctions(scopeText).get('legacy_position_slug') || '';
const fastPositionBody = coreFunctions.get('sn_fast_hr_position_for_user') || '';
const panelUrlBody = coreFunctions.get('sn_get_user_panel_url') || '';
	for (const [role, position] of Object.entries(rolePositionContract)) {
  if (!activatorText.includes(`add_role( '${role}'`)) addError(`Operational role is not registered on activation: ${role}`);
  if (!migrationText.includes(`'${role}' => '${position}'`)) addError(`Migration role mapping is missing: ${role} -> ${position}`);
  if (!hrService.includes(`'${role}'`)) addError(`HR role resolver does not know ${role}`);
  if (!hrFallbackBody.includes(`'${role}'`) || !hrFallbackBody.includes(`'${position}'`)) addError(`HR fallback mapping is missing: ${role} -> ${position}`);
  if (!scopeFallbackBody.includes(`'${role}'`) || !scopeFallbackBody.includes(`'${position}'`)) addError(`Scope fallback mapping is missing: ${role} -> ${position}`);
  if (!fastPositionBody.includes(`'${role}'`) || !fastPositionBody.includes(`'${position}'`)) addError(`Fast panel mapping is missing: ${role} -> ${position}`);
	}
	for (const retiredRole of ['sn_project_manager', 'sn_project_expert']) {
	  if (activatorText.includes(`add_role( '${retiredRole}'`)) addError(`Retired project role is still registered on activation: ${retiredRole}`);
	}
for (const [role, option] of Object.entries({
  sn_senior_supervisor: 'sn_senior_supervisor_panel_page_id',
  sn_sales_deputy: 'sn_sales_deputy_panel_page_id',
  sn_finance: 'sn_financial_panel_page_id',
  sn_hr: 'sn_hr_panel_page_id',
  sn_mis: 'sn_mis_panel_page_id',
})) {
  if (!panelUrlBody.includes(`'${role}' => '${option}'`)) addError(`Role-only panel URL fallback is missing: ${role}`);
}
const applyRoleBody = coreFunctions.get('sn_hr_apply_user_access_for_position') || '';
if (!applyRoleBody.includes('$user->remove_role($managed_role)') || !applyRoleBody.includes('clean_user_cache($user_id)') || !applyRoleBody.includes("$this->sn_hr_position_for_user_cache['fast:' . $user_id]")) {
  addError('HR position changes do not revoke stale plugin roles and refresh the user cache.');
}
for (const financeInvariant of ["$position_slug === 'finance'", "'sn_view_finance'", "'sn_approve_payment'", "'sn_reject_payment'", '$user->remove_cap($finance_cap)']) {
  if (!applyRoleBody.includes(financeInvariant)) addError(`Finance-position role synchronization is incomplete: ${financeInvariant}`);
}
for (const method of ['ajax_supervisor_lazy_tab', 'ajax_senior_supervisor_lazy_tab', 'ajax_sales_manager_lazy_tab']) {
  const body = coreFunctions.get(method) || '';
  if (!body.includes("check_ajax_referer('sn_public', 'nonce'")) addError(`${method} is missing its public nonce gate.`);
  if (!body.includes('sn_current_user_can_')) addError(`${method} is missing its panel permission gate.`);
}
checks.crossPanelRoleRouting = Object.keys(rolePositionContract).length;
checks.lazyPanelAuthorization = 3;

// 2.0.88: the Sales Deputy panel must expose the same guarded V4 allocation
// engine used by manager/supervisor panels. Keep the UI route and backend
// descendant-scope contract tied together so a future refactor cannot leave a
// visible but non-functional tab (or broaden recipients beyond HR scope).
const salesDeputyPanelBody = coreFunctions.get('render_sales_deputy_panel') || '';
const recipientPositionsBody = coreFunctions.get('sn_distribution_recipient_positions_for_actor') || '';
if (!salesDeputyPanelBody.includes("'sd-distribution' => 'تخصیص شماره'")) addError('Sales Deputy panel is missing the number-allocation tab.');
if (!salesDeputyPanelBody.includes("sn_render_supervisor_progressive_assign_tab($viewer_id)")) addError('Sales Deputy allocation tab is not wired to the guarded progressive allocation workflow.');
for (const required of ["$actor_position === 'sales_deputy'", "'sales_manager'", "'senior_supervisor'", "'supervisor'", "'seller'"]) {
  if (!recipientPositionsBody.includes(required)) addError(`Sales Deputy distribution scope contract is incomplete: ${required}`);
}
checks.salesDeputyDistribution = 1;

// Product* normal/upsell contract. Live rules are additive configuration;
// each new membership item receives an immutable credit/options snapshot.
for (const required of [
  "'upgrade_rules' => $wpdb->prefix . 'sn_project_card_upgrade_rules'",
  'UNIQUE KEY source_target (source_product_id,target_product_id)',
  'base_credit_snapshot DECIMAL(18,2) DEFAULT NULL',
  'upgrade_options_snapshot_json LONGTEXT DEFAULT NULL',
  'public function save_product_upgrade_rules(',
  "return new WP_Error( 'sn_upgrade_duplicate'",
  "return new WP_Error( 'sn_upgrade_amount'",
  "$content['upgrade_options'] = $this->product_upgrade_rules( $content_id )",
  "'upgrade_options_snapshot_json' => wp_json_encode(",
  '$base_credit = $this->item_base_credit( $item )',
  "$amount = max( 0, (float) ( $matched_upgrade['price']",
  "$out['upgrade_rules_locked'] =",
]) {
  if (!projects.includes(required)) addError(`Product* upgrade invariant is missing: ${required}`);
}
if (projects.includes("return new WP_Error( 'sn_upgrade_credit'") || projects.includes('$target_credit <= $source_credit')) {
  addError('Product* upgrade rules still reject target credit at or below source credit.');
}
const upgradeOptionBody = extractFunctions(source('includes/class-sn-operations-flow.php')).get('upgrade_option') || '';
if (!upgradeOptionBody.includes("'credit' => (float) $raw_credit") || upgradeOptionBody.includes('base_credit_snapshot')) {
  addError('Customer upgrade selection does not accept the configured target credit.');
}
for (const required of [
  'sn_project_upgrade_rules_present',
  'حالت‌های افزایشی این کارت',
  'SN_Projects::instance()->save_product_upgrade_rules',
  '_sn_upgrade_rules_save_error',
]) {
  if (!plugin.includes(required)) addError(`Product* upgrade metabox wiring is missing: ${required}`);
}
const projectJs = source('assets/js/public-projects.js');
for (const required of ['item.upgrade_rules_locked', 'item.upgrade_options', 'data-price=', 'sn-project-action-amount']) {
  if (!projectJs.includes(required)) addError(`Product* upgrade action UI is missing: ${required}`);
}
checks.productStarUpgradeSnapshots = 4;

// 2.0.57 customer profile and exact subscription repurchase awareness.
const releaseCustomerPortal = source('includes/class-sn-customer-portal.php');
const customerCss = source('assets/css/customer-portal.css');
const invoiceJs = source('assets/js/public-invoice.js');
const converterJs = source('assets/js/public-converter.js');
const publicCss = source('assets/css/public.css');
const profileRenderBody = extractFunctions(releaseCustomerPortal).get('render') || '';
for (const removed of ['id="sn-customer-timeline"', 'href="#sn-customer-timeline"', '$timeline =']) {
  if (profileRenderBody.includes(removed)) addError(`Customer timeline is still rendered: ${removed}`);
}
for (const required of ['دلیل پرداخت', "'reason' =>", 'sn-customer-payment-reason', 'sn-customer-credit-wallet', 'sn-customer-purchase-list']) {
  if (!releaseCustomerPortal.includes(required) && !customerCss.includes(required)) addError(`Customer profile refresh is missing: ${required}`);
}
for (const required of ['purchased_subscription_product_ids(', 'ajax_customer_purchased_subscriptions', 'sn_customer_purchased_subscriptions']) {
  if (!releaseCustomerPortal.includes(required)) addError(`Subscription purchase lookup is missing: ${required}`);
}
const duplicateWarning = 'مشتری گرامی، شما قبلا این اشتراک را خریداری کرده اید و در حال خرید دوباره همین نوع اشتراک می باشید';
for (const required of ['duplicate_subscription_purchase', 'duplicate_subscription_product_ids', 'duplicate_subscription_warning', duplicateWarning]) {
  if (!plugin.includes(required)) addError(`Public invoice duplicate-subscription warning is missing: ${required}`);
}
if (!invoiceJs.includes('sn-duplicate-subscription-warning') || !publicCss.includes('.sn-duplicate-subscription-warning')) {
  addError('Public invoice duplicate-subscription warning is not rendered or styled.');
}
for (const script of [sellerJs, converterJs]) {
  for (const required of ['sn_customer_purchased_subscriptions', 'data-sn-purchased', 'خریداری‌شده']) {
    if (!script.includes(required)) addError(`Seller/converter purchased-subscription tag is missing: ${required}`);
  }
}
checks.customerProfilePurchaseAwareness = true;

// Seller assessment visibility, HR number-request permission and staged Dot
// subscription duplicate-purchase safety, merged with the Zibal export branch.
const sellerConversionResult = extractFunctions(dotFlow).get('completed_results_for_seller_invoices') || '';
for (const required of ['seller_id=%d', 'source_invoice_id IN', '$has_selected_subscription', "'sale_state'", "'subscription_sold'", "'conversion_status_label'", 'اشتراک فروخته شد', 'اشتراک فروخته نشد']) {
  if (!sellerConversionResult.includes(required)) addError(`Seller assessment conversion result is incomplete: ${required}`);
}
for (const required of ['نتیجه تبدیل اعتبارسنجی', 'sale_state_label', 'conversion_status_label', 'is-not-sold', 'is-progress']) {
  if (!sellerJs.includes(required)) addError(`Seller assessment result UI is incomplete: ${required}`);
}
const requestExtraNumber = extractFunctions(plugin).get('ajax_seller_request_extra_number') || '';
for (const required of ['sn_extra_number_request_access', 'sn_user_can_request_extra_number', 'extra_number_request_access_updated', 'دسترسی درخواست شماره']) {
  if (!plugin.includes(required)) addError(`HR extra-number permission wiring is missing: ${required}`);
}
if (!requestExtraNumber.includes('sn_user_can_request_extra_number')) {
  addError('Extra-number permission is hidden in UI but not enforced server-side.');
}
const purchasedSubscriptions = extractFunctions(releaseCustomerPortal).get('purchased_subscription_product_ids') || '';
for (const required of ['sn_dot_invoice_links', "flow_kind='conversion_payment'", 'NOT EXISTS']) {
  if (!purchasedSubscriptions.includes(required)) addError(`Staged Dot duplicate-purchase guard is missing: ${required}`);
}
checks.release2073SalesSafety = true;

// 2.0.77 supervisor receipt upload: one scoped handler, staged-payment
// visibility, and a shared server-side final-status guard.
const supervisorReceiptUpload = extractFunctions(plugin).get('ajax_supervisor_upload_receipt') || '';
const supervisorReceiptStatusGuard = extractFunctions(plugin).get('sn_supervisor_can_upload_receipt_for_status') || '';
if (supervisorJs.includes('.sn-supervisor-receipt-file[data-id="')) {
  addError('Obsolete supervisor receipt handler can still throw before the scoped upload handler.');
}
for (const required of [
  'click.snSupReceiptUpload',
  "$('#sn-supervisor-receipt-file')[0]",
  'sn-supervisor-receipt-open',
  "fd.append('invoice_id',invoiceId)",
  "'partial_paid'",
  'i.can_upload_receipt',
]) {
  if (!supervisorJs.includes(required)) addError(`Supervisor receipt UI invariant is missing: ${required}`);
}
for (const required of [
  'sn_supervisor_can_access_invoice',
  'sn_supervisor_can_upload_receipt_for_status',
  'SN_Helpers::upload_receipt',
  "$source = $is_converter ? 'converter_upload' : 'supervisor_upload'",
]) {
  if (!supervisorReceiptUpload.includes(required)) addError(`Supervisor receipt endpoint invariant is missing: ${required}`);
}
for (const required of ["'partial_paid'", "'gateway_paid'", "'payment_archived'", 'array_intersect($statuses, $terminal)']) {
  if (!supervisorReceiptStatusGuard.includes(required)) addError(`Supervisor receipt status guard is incomplete: ${required}`);
}
checks.supervisorReceiptUpload = true;

// 2.0.85: conversion invoices follow the actual conversion actor; receipt
// permissions remain role-specific, and HR bulk selection is filter-safe.
const internalPaymentAccess = extractFunctions(plugin).get('sn_can_perform_internal_invoice_payment_action') || '';
const manualPaymentSubmit = extractFunctions(plugin).get('ajax_submit_manual_payment') || '';
const converterInvoices = extractFunctions(dotFlow).get('handle_converter_invoices') || '';
for (const required of [
  "i.issued_by_user_id=%d",
  "'can_submit_manual_payment'",
  "'can_upload_receipt'",
  "'receipt_url'",
]) {
  if (!converterInvoices.includes(required)) addError(`Conversion invoice actor/payment invariant is missing: ${required}`);
}
for (const required of [
  "$action === 'manual_payment'",
  'sn_converter_can_access_invoice',
  'sn_invoice_is_conversion_payment',
  "return 'seller_manual'",
  "return 'converter_manual'",
]) {
  if (!internalPaymentAccess.includes(required) && !plugin.includes(required)) addError(`Role-specific invoice payment permission is missing: ${required}`);
}
for (const required of ["! empty($access['internal'])", 'sn_internal_payment_source_for_current_user']) {
  if (!manualPaymentSubmit.includes(required)) addError(`Internal manual-payment source invariant is missing: ${required}`);
}
for (const required of ['sn-seller-manual-payment-open', 'sn_submit_manual_payment', 'sn-seller-manual-receipt', "form.append('receipt',receipt)"]) {
  if (!sellerJs.includes(required)) addError(`Seller unified manual payment UI is missing: ${required}`);
}
if (!manualPaymentSubmit.includes("$fields['receipt_url']") || !manualPaymentSubmit.includes('SN_Helpers::upload_receipt')) addError('Unified manual payment must validate and save receipt with payment details.');
const sellerFinancialResubmit = extractFunctions(plugin).get('ajax_seller_resend_financial') || '';
for (const required of ['sn_invoice_is_conversion_payment($id)', 'if (! $owner_allowed)', 'SN_Helpers::upload_receipt']) {
  if (!sellerFinancialResubmit.includes(required)) addError(`Seller scoped correction and receipt guard is missing: ${required}`);
}
for (const required of ['sn-converter-payment-open', 'sn-converter-receipt-upload', 'sn_supervisor_upload_receipt']) {
  if (!converterInvoiceJs.includes(required)) addError(`Converter invoice payment UI is missing: ${required}`);
}
for (const required of ['sn-supervisor-payment-open', 'sn_submit_manual_payment']) {
  if (!supervisorJs.includes(required)) addError(`Supervisor manual-payment UI is missing: ${required}`);
}

// 2.0.102: destination last-four stays supported but is optional for internal
// sales-team manual payment / receipt entry. Public customer payment remains strict.
const activatorPaymentSchema = source('includes/class-sn-activator.php');
const manualPaymentSave = extractFunctions(plugin).get('sn_save_invoice_manual_payment') || '';
const destinationInput = extractFunctions(plugin).get('sn_manual_destination_card_input') || '';
const destinationDisplay = extractFunctions(plugin).get('sn_manual_destination_card_display') || '';
for (const text of [activatorPaymentSchema, plugin]) {
  if (!text.includes('manual_card_to_number')) addError('Full manual destination-card schema is missing.');
}
for (const required of [
  "$submitted_to = ! empty($access['internal'])",
  "sn_manual_destination_card_input($submitted_to, $invoice, ! empty($access['internal']))",
  "'manual_card_to_number' => $to_number !== '' ? $to_number : null",
  "$to !== ''",
  '۴ رقم آخر کارت مقصد اختیاری است',
]) {
  if (!manualPaymentSubmit.includes(required)) addError(`Internal optional destination-card validation is missing: ${required}`);
}
for (const required of ["$allow_empty && $digits === ''", "'valid' => true, 'last4' => '', 'number' => ''", "strlen($digits) !== 4", "'last4' => $digits", "'number' => $full_number", 'sn_resolve_invoice_card($invoice)']) {
  if (!destinationInput.includes(required)) addError(`Destination last-four normalization is missing: ${required}`);
}
for (const required of ['manual_card_to_number', 'sn_filter_existing_columns_fresh', "$data['manual_card_to_number'] = null"]) {
  if (!manualPaymentSave.includes(required)) addError(`Manual destination-card persistence is missing: ${required}`);
}
for (const [label, script, required] of [
  ['seller', sellerJs, 'sn-seller-manual-card-to'],
  ['converter', converterInvoiceJs, 'sn-converter-payment-card-to'],
  ['supervisor', supervisorJs, 'sn-supervisor-payment-card-to'],
]) {
  if (!script.includes(required) || !script.includes("form.append('card_to',to)") || !script.includes('۴ رقم آخر کارت مقصد (اختیاری)') || !script.includes("to!=='' && !/^\\d{4}$/")) addError(`${label} optional manual-payment destination-card UI is incomplete.`);
}
for (const [label, script, required, optionalGuard] of [
  ['converter receipt', converterInvoiceJs, "fd.append('card_to',to)", "to!=='' && !/^\\d{4}$/"],
  ['supervisor receipt', supervisorJs, "fd.append('card_to',cardTo)", "cardTo!=='' && !/^\\d{4}$/"],
]) {
  if (!script.includes(required) || !script.includes(optionalGuard)) addError(`${label} optional destination-card upload wiring is missing.`);
}
for (const required of [
  "sn_manual_destination_card_input(sanitize_text_field(wp_unslash($_POST['card_to'] ?? '')), $invoice, true)",
  '۴ رقم آخر کارت مقصد اختیاری است',
]) {
  if (!supervisorReceiptUpload.includes(required)) addError(`Optional receipt destination-card server guard is missing: ${required}`);
}
for (const [label, script, required] of [
  ['converter receipt modal', converterInvoiceJs, 'sn-converter-receipt-open'],
  ['supervisor receipt modal', supervisorJs, 'sn-supervisor-receipt-open'],
]) {
  if (!script.includes(required) || !script.includes('sn-payment-entry-modal')) addError(`${label} polished UI is missing.`);
}
for (const script of [sellerJs, converterInvoiceJs, supervisorJs]) {
  if (script.includes('شماره ۱۶ رقمی کارت مقصد') || script.includes('۱۶ رقم کارت مقصد')) addError('A sales-team form still requests the full destination card number.');
}
for (const required of ['manual_card_to_number', 'payment_card_number', 'return substr($number_digits, -4)']) {
  if (!destinationDisplay.includes(required)) addError(`Legacy destination-card display fallback is missing: ${required}`);
}
for (const required of ['۴ رقم آخر کارت مقصد', 'manual_card_to_display']) {
  if (!managerScript.includes(required)) addError(`Finance destination-card display is missing: ${required}`);
}
for (const required of ['sn-payment-entry-card', 'sn-payment-entry-grid', 'sn-payment-proof-open']) {
  if (!publicCss.includes(required)) addError(`Payment-entry UI style is missing: ${required}`);
}
checks.salesTeamDestinationCard = true;

for (const required of [
  '$visible = $targets.filter(\':visible\')',
  "fd.delete('visible_user_ids[]')",
  'sortTableRows(table, idx, direction)',
  'sn_hr_workforce_rows_cache',
  'sn_hr_manager_candidates_cache',
]) {
  if (!source('assets/js/public.js').includes(required) && !plugin.includes(required)) addError(`HR filter/select/performance invariant is missing: ${required}`);
}
checks.roleInvoicePaymentAndHrFilters = true;

// 2.0.78 gateway/card routing/customer behavior hardening.
if (zibalReportFetch.includes("'timeout' => 45") || zibalReportFetch.includes('sn_zibal_report_request_bodies')) {
  addError('Zibal export still performs the old long/retried synchronous report request.');
}
for (const required of ["min(12", "'limit_response_size' => 8 * MB_IN_BYTES", "'reject_unsafe_urls' => true"]) {
  if (!zibalReportFetch.includes(required)) addError(`Zibal 502 guard is missing: ${required}`);
}
if (zibalExportHandler.includes("['response' => 502]") || zibalExportHandler.includes("['response'=>502]")) {
  addError('Zibal export still emits a provider failure as HTTP 502.');
}
for (const required of ['remote_report_fallback', 'sales-network-gateway-finance-', 'X-SN-Export-Request-ID', "fflush($out)"]) {
  if (!zibalExportHandler.includes(required)) addError(`Zibal local fallback invariant is missing: ${required}`);
}

const dotFlow2078 = source('includes/class-sn-dot-flow.php');
const invoiceJs2078 = source('assets/js/public-invoice.js');
const shellJs2078 = source('assets/js/public-shell.js');
const managerCardsResolver = extractFunctions(plugin).get('sn_resolve_sales_manager_card_for_seller') || '';
for (const required of ['sn_sales_manager_cards', 'sales_manager_id', 'legacy_sales_deputy_fallback']) {
  if (!managerCardsResolver.includes(required) && !plugin.includes(required)) addError(`Sales-manager card routing invariant is missing: ${required}`);
}
for (const required of ['payment_card_sales_manager_user_id', "'sn_card_to_card_enabled' => '1'", "'sn_sales_manager_cards' => []"]) {
  if (!activator.includes(required)) addError(`2.0.78 additive schema/default invariant is missing: ${required}`);
}
if (!dotFlow2078.includes('payment_card_sales_manager_user_id')) addError('Dot follow-up invoices do not retain the sales-manager card snapshot.');

const customerActionsScope = extractFunctions(plugin).get('sn_customer_actions_scope_sql') || '';
for (const required of ['seller','converter','supervisor','senior_supervisor','sales_manager','sn_dot_cases','sn_sales_manager_scope_seller_ids']) {
  if (!customerActionsScope.includes(required)) addError(`Staff customer-action scope invariant is missing: ${required}`);
}
for (const required of ['data-tab="customer-actions"', 'manager-customer-actions']) {
  if (!plugin.includes(required) && !dotFlow2078.includes(required)) addError(`Staff customer-action tab invariant is missing: ${required}`);
}
for (const required of ['sn-staff-customer-actions', 'sn_seller_customer_actions', 'sn-staff-action-detail']) {
  if (!shellJs2078.includes(required)) addError(`Shared customer-action UI invariant is missing: ${required}`);
}

const manualPaymentSave2078 = extractFunctions(plugin).get('sn_save_invoice_manual_payment') || '';
const supervisorReceiptGuard2078 = extractFunctions(plugin).get('sn_supervisor_can_upload_receipt_for_status') || '';
for (const required of ['sn_card_to_card_enabled', 'پرداخت کارت‌به‌کارت غیرفعال است']) {
  if (!manualPaymentSave2078.includes(required)) addError(`Card-to-card server toggle invariant is missing: ${required}`);
}
if (!supervisorReceiptGuard2078.includes('sn_card_to_card_enabled')) addError('Supervisor receipt UI does not honor the card-to-card toggle.');
for (const required of ['card_to_card_enabled', 'cardToCardEnabled', "toggle(cardToCardEnabled)"]) {
  if (!invoiceJs2078.includes(required) && !plugin.includes(required)) addError(`Card-to-card invoice UI invariant is missing: ${required}`);
}
checks.gatewayManagerCardsCustomerBehavior = true;

// 2.0.58 customer profile accordions and authoritative WooCommerce card media.
for (const required of [
  "wc_get_product( $content_product_id )",
  "wp_get_attachment_image_url( $image_id, 'large' )",
  'get_description()',
  "do_blocks( $description )",
  "wpautop( $description )",
  "'description_html'",
  'sn-customer-card-description',
  'class="sn-customer-accordion',
  'id="sn-customer-account"',
  'id="sn-customer-cards" open',
]) {
  if (!releaseCustomerPortal.includes(required)) addError(`Customer accordion/product-content invariant is missing: ${required}`);
}
for (const required of ['Customer profile v9', '.sn-customer-accordion > summary', 'aspect-ratio:3/2', '.sn-customer-card-description']) {
  if (!customerCss.includes(required)) addError(`Customer accordion/product-content CSS is missing: ${required}`);
}
for (const required of ['profileAccordions', "other.open = false", 'accordion.scrollIntoView']) {
  if (!customerPortalJs.includes(required)) addError(`Customer accordion interaction is missing: ${required}`);
}
checks.customerProfileAccordionProductContent = true;

// Typed final execution is additive and snapshots product/form configuration
// before assignment. Wallet charging is idempotent and shipping has its own
// permission boundary instead of borrowing the execution-expert action.
const execution = source('includes/class-sn-operations-execution.php');
for (const required of [
  'sn_operations_execution_cases',
  'sn_operations_form_definitions',
  'sn_operations_form_submissions',
  "'wallet_charge' => 'شارژ کیف پول'",
  "'physical_invoice' => 'فاکتور کالا'",
  "'form' => 'فرم'",
  '_sn_execution_fulfillment_type',
  '_sn_execution_form_id',
  'form_schema_snapshot_json',
  'sn_operations_execution_expert',
  'operations_execution_expert',
  'sn_operations_execution_shipping_update',
  "wallet_status='charging'",
  "'request_id'=>'sn-execution-'",
  "apply_filters('sn_operations_wallet_request'",
  'validated_https_url(',
  'operations_shipping_status_changed',
  'operations_form_executed',
  'operations_form_not_executed',
]) {
  if (!execution.includes(required)) addError(`Typed execution invariant is missing: ${required}`);
}
if (execution.includes('wp_set_auth_cookie') || execution.includes('user_pass')) {
  addError('Execution workflow contains a customer-login bypass instead of a destination-issued one-time URL.');
}
if (!source('includes/class-sn-product-flow.php').includes('render_shipping_handoffs')) {
  addError('Physical card handoff is not rendered in the existing Shipping panel.');
}
for (const required of [
  'operations_execution_assigned',
  'operations_wallet_charged',
  'operations_physical_sent_to_shipping',
  'operations_form_completed',
  'operations_shipping_status_changed',
]) {
  if (!releaseCustomerPortal.includes(required)) addError(`Customer card timeline is missing: ${required}`);
}
checks.typedOperationsExecution = true;

// 2.0.90: the public Marketing Dot Flow request copy stays neutral while the
// configured WooCommerce product name remains visible in the confirmation modal.
// Static customer-facing labels must not introduce the words «اعتبار» or «اشتراک».
const dotFlowSource = source('includes/class-sn-dot-flow.php');
const marketingRenderMatch = dotFlowSource.match(/public function render_marketing_form\( array \$atts = \[\] \): string \{([\s\S]*?)\n\tpublic function handle_marketing_start_payment\(\): void \{/);
if (!marketingRenderMatch) {
  addError('Marketing Dot Flow render_marketing_form() could not be audited.');
} else {
  const marketingRender = marketingRenderMatch[1];
  if (/اعتبار|اشتراک/.test(marketingRender)) addError('Marketing Dot Flow customer form exposes forbidden credit/subscription wording.');
  if (!marketingRender.includes('$product->get_name()')) addError('Marketing Dot Flow confirmation modal does not show the configured product name.');
  if (!marketingRender.includes('>ادامه درخواست</button>')) addError('Marketing Dot Flow primary CTA is not the neutral «ادامه درخواست».');
}
const marketingCustomerStart = dotFlowSource.indexOf('public function handle_marketing_save_lead');
const marketingCustomerEnd = dotFlowSource.indexOf('public function render_customer_flow');
const marketingCustomerSegment = marketingCustomerStart >= 0 && marketingCustomerEnd > marketingCustomerStart
  ? dotFlowSource.slice(marketingCustomerStart, marketingCustomerEnd)
  : '';
if (!marketingCustomerSegment) addError('Marketing Dot Flow customer request handlers could not be audited.');
if (/اعتبارسنجی|اشتراک/.test(marketingCustomerSegment)) addError('Marketing Dot Flow customer request handlers expose forbidden assessment/subscription wording.');
if (!marketingCustomerSegment.includes("'پرداخت درخواست دات فلو '")) addError('Marketing Dot Flow gateway description is not neutral request wording.');
checks.marketingNeutralRequestCopy = true;

// 2.0.91: dedicated successful-payment report must remain scoped, product-aware,
// and inclusive of legacy/manual successful invoices without double-counting them.
const reportsRender2091 = extractFunctions(plugin).get('render_reports_panel') || '';
const successSql2091 = extractFunctions(plugin).get('sn_successful_payment_report_sql') || '';
const successRows2091 = extractFunctions(plugin).get('sn_successful_payment_report_rows') || '';
const exportRows2091 = extractFunctions(plugin).get('sn_export_report_rows') || '';
for (const required of [
  "'reports-successful-payments' => 'گزارش'",
  'sn_render_successful_payments_report($filters)',
  "'successful_payments' => 'پرداخت‌های موفق'",
]) {
  if (!reportsRender2091.includes(required)) addError(`2.0.91 successful-payment report tab invariant is missing: ${required}`);
}
for (const required of [
  "sn_report_seller_sql('i.seller_id'",
  "p.status IN ({$success_values})",
  'NOT EXISTS (SELECT 1 FROM',
  'sn_invoice_items',
  'product_type=%s',
  "'payment' row_source",
  "'invoice_fallback' row_source",
]) {
  if (!successSql2091.includes(required)) addError(`2.0.91 successful-payment query invariant is missing: ${required}`);
}
if (!successRows2091.includes('ORDER BY payment_at DESC')) addError('2.0.91 successful-payment rows are not ordered newest-first.');
for (const required of ["$type === 'successful_payments'", 'sn_successful_payment_report_product_type_label', 'sn_mask_phone']) {
  if (!exportRows2091.includes(required)) addError(`2.0.91 successful-payment export invariant is missing: ${required}`);
}
checks.successfulPaymentReport = true;

// 2.0.92: status routing must survive legacy/blank state rows, all Operations
// contact outcomes must be filterable, and count allocation must cap every
// selected recipient independently instead of only capping the grand total.
const sellerFlow2092 = source('includes/class-sn-seller-flow.php');
const operationsFlow2092 = source('includes/class-sn-operations-flow.php');
const operationsJs2092 = source('assets/js/operations-flow.js');
for (const required of [
  "if ( $current && $new_status === '' && $inferred_status !== '' )",
  "if ( (string) ( $lead['seller_flow_status'] ?? '' ) === '' )",
  "'follow up' => 'callback'",
]) {
  if (!sellerFlow2092.includes(required)) addError(`2.0.92 seller status routing invariant is missing: ${required}`);
}
for (const required of [
  'contact_status VARCHAR(30)',
  "SET contact_status='no_answer'",
  "SET contact_status='follow_up'",
  'data-sn-ops-contact-status',
  'data-contact-status=',
]) {
  if (!operationsFlow2092.includes(required)) addError(`2.0.92 Operations contact routing invariant is missing: ${required}`);
}
for (const required of ['selectedContactStatus', "getAttribute('data-contact-status')"]) {
  if (!operationsJs2092.includes(required)) addError(`2.0.92 Operations contact filter invariant is missing: ${required}`);
}
if (!dotFlowSource.includes('data-sn-dot-filter="contact"')) addError('2.0.92 converter contact-result filter is missing.');
for (const required of ['requested_recipient_counts', 'recipient_quotas', 'recipient_shortfalls']) {
  if (!plugin.includes(required)) addError(`2.0.92 exact recipient allocation invariant is missing: ${required}`);
}
checks.statusRoutingAndExactAllocation = true;

// 2.0.93: a successful allocation clears the selected recipients/items but
// keeps the entered per-recipient count available for the next allocation.
const allocationUi2093 = {
  'assets/js/public.js': source('assets/js/public.js'),
  'assets/js/public-manager.js': source('assets/js/public-manager.js'),
  'assets/js/public-supervisor.js': source('assets/js/public-supervisor.js'),
};
for (const [file, text] of Object.entries(allocationUi2093)) {
  if (text.includes("$form.find('input[name=\"per_seller_count\"]').val('');")) {
    addError(`2.0.93 allocation count is still cleared after success in ${file}.`);
  }
}
for (const file of ['assets/js/public.js', 'assets/js/public-manager.js']) {
  for (const required of ['retainedPerSellerCount', ".val(retainedPerSellerCount)"]) {
    if (!allocationUi2093[file].includes(required)) addError(`2.0.93 allocation count refresh retention is missing in ${file}: ${required}`);
  }
}
for (const required of ['persistSupervisorCount($form)', 'restoreSupervisorCount($form)']) {
  if (!allocationUi2093['assets/js/public-supervisor.js'].includes(required)) addError(`2.0.93 supervisor allocation count retention is missing: ${required}`);
}
checks.allocationCountRetention = true;

// 2.0.94: every staff invoice workspace exposes one shared, permission-checked
// SMS resend action. It must stay authenticated, rate-limited and auditable.
for (const required of [
  "add_action('wp_ajax_sn_resend_invoice_sms'",
  'public function ajax_resend_invoice_sms()',
  'private function sn_can_resend_invoice_sms(',
  "'sn_resend_invoice_sms_' . $actor_id . '_' . $invoice_id",
  "'invoice_sms_resent'",
  "'invoice_sms_resend_failed'",
]) {
  if (!plugin.includes(required)) addError(`2.0.94 invoice SMS resend server invariant is missing: ${required}`);
}
if (plugin.includes("wp_ajax_nopriv_sn_resend_invoice_sms")) addError('2.0.94 invoice SMS resend must never be public.');
const invoiceSmsShell2094 = source('assets/js/public-shell.js');
for (const required of ['.sn-resend-invoice-sms', "form.append('action', 'sn_resend_invoice_sms')", 'در حال ارسال...']) {
  if (!invoiceSmsShell2094.includes(required)) addError(`2.0.94 invoice SMS resend shared UI handler is missing: ${required}`);
}
for (const file of ['assets/js/public-seller.js', 'assets/js/public-supervisor.js', 'assets/js/public-manager.js', 'assets/js/public-converter.js']) {
  if (!source(file).includes('sn-resend-invoice-sms')) addError(`2.0.94 invoice SMS resend button is missing in ${file}.`);
}
checks.invoiceSmsResend = true;

// 2.0.95: conversion-payment SMS must have a bounded recovery route and test
// previews must never be recorded as real sends or trigger a live fallback.
for (const required of [
  'payment_sms_previewed',
  'payment_sms_fallback_sent',
  "'delivery_route' => $fallback_sent ? 'canonical_invoice' : 'all_routes_failed'",
  "if ( $sent && ! $is_test_preview )",
  'Never turn an explicit test/preview run into a real SMS fallback.',
  '(new SN_SMS())->send_invoice_link(',
]) {
  if (!dotFlowSource.includes(required)) addError(`2.0.95 conversion payment SMS recovery invariant is missing: ${required}`);
}
if (dotFlowSource.includes("if ( trim( $template ) === '' ) { return false; }")) {
  addError('2.0.95 Dot SMS still suppresses the built-in template after a failed Faraz pattern.');
}
const smsService2095 = source('includes/class-sn-sms.php');
for (const required of [
  "get_option( 'sn_card_to_card_enabled', '1' ) !== '1'",
  'build_invoice_link_preview( $phone, $invoice_code, $invoice_url, $customer_name, $amount, $card_number )',
  "in_array( $body['status'], [ 'success', 'OK', true ], true )",
]) {
  if (!smsService2095.includes(required)) addError(`2.0.95 canonical invoice SMS safety invariant is missing: ${required}`);
}
checks.conversionPaymentSmsRecovery = true;

// All statically referenced CRM actions must be registered. Dynamic action maps
// are read only from register_hooks() methods that concatenate the action name.
const registeredActions = new Set();
const usedActions = new Set();
for (const file of phpFiles) {
  const text = source(file);
  for (const match of text.matchAll(/add_action\(\s*['"](?:wp_ajax(?:_nopriv)?|admin_post(?:_nopriv)?)_([^'"]+)['"]/g)) {
    registeredActions.add(match[1]);
  }
  for (const match of text.matchAll(/name=['"]action['"]\s+value=['"](sn_[a-z0-9_]+)['"]/g)) usedActions.add(match[1]);
  for (const match of text.matchAll(/['"]action['"]\s*=>\s*['"](sn_[a-z0-9_]+)['"]/g)) usedActions.add(match[1]);
  for (const match of text.matchAll(/action\s*:\s*['"](sn_[a-z0-9_]+)['"]/g)) usedActions.add(match[1]);
  const registerHooks = extractFunctions(text).get('register_hooks') || '';
  if (/['"]wp_ajax_['"]\s*\.\s*\$action|['"]admin_post_['"]\s*\.\s*\$action/.test(registerHooks)) {
    for (const match of registerHooks.matchAll(/['"](sn_[a-z0-9_]+)['"]\s*=>\s*['"][A-Za-z_][A-Za-z0-9_]*['"]/g)) {
      registeredActions.add(match[1]);
    }
  }
}
for (const file of jsFiles) {
  const text = source(file);
  for (const match of text.matchAll(/action\s*:\s*['"](sn_[a-z0-9_]+)['"]/g)) usedActions.add(match[1]);
  for (const match of text.matchAll(/(?:append|set)\(\s*['"]action['"]\s*,\s*['"](sn_[a-z0-9_]+)['"]/g)) usedActions.add(match[1]);
}
for (const action of usedActions) {
  if (!registeredActions.has(action)) addError(`Action is used but not registered: ${action}`);
}
checks.registeredActions = registeredActions.size;
checks.usedActions = usedActions.size;

// Destructive schema statements are allowed only inside the explicitly guarded
// administrator reset. Install, upgrade and ordinary request paths stay additive.
for (const file of phpFiles) {
  let runtimeText = source(file);
  if (file === 'includes/class-sn-plugin.php') {
    const resetBody = extractFunctions(runtimeText).get('ajax_full_data_reset') || '';
    for (const required of ["current_user_can('manage_options')", "check_ajax_referer('sn_full_data_reset'", "confirmation !== 'حذف کامل داده‌ها'", "$_POST['confirmed']"]) {
      if (!resetBody.includes(required)) addError(`Full reset destructive guard is missing: ${required}`);
    }
    runtimeText = runtimeText.replace(resetBody, '');
  }
  if (/\b(?:DROP|TRUNCATE|RENAME)\s+TABLE\b/i.test(runtimeText)) {
		addError(`Destructive schema statement found in ${file}`);
	}
}

// Use the real PHP parser when the environment provides it; otherwise emit a
// visible warning rather than representing this audit as runtime validation.
if (commandExists('php')) {
  let linted = 0;
  for (const file of phpFiles) {
    const result = childProcess.spawnSync('php', ['-l', path.join(root, file)], { encoding: 'utf8' });
    if (result.status !== 0) addError(`PHP lint failed: ${file}: ${(result.stderr || result.stdout).trim()}`);
    linted++;
  }
  checks.phpLintedFiles = linted;
} else {
  checks.phpLintedFiles = 0;
  addWarning('PHP CLI is unavailable; run php -l and WordPress smoke tests on Staging.');
}

const report = {
  ok: errors.length === 0,
  checks,
  errors,
  warnings,
};

if (process.argv.includes('--json')) {
  process.stdout.write(JSON.stringify(report, null, 2) + '\n');
} else {
  process.stdout.write(`${report.ok ? 'PASS' : 'FAIL'} Sales Network static audit\n`);
  for (const [name, value] of Object.entries(checks)) process.stdout.write(`- ${name}: ${value}\n`);
  for (const warning of warnings) process.stdout.write(`- WARNING: ${warning}\n`);
  for (const error of errors) process.stdout.write(`- ERROR: ${error}\n`);
}

process.exitCode = report.ok ? 0 : 1;
