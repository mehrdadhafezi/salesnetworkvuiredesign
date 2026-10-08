# Code Evidence & Action Registry

Static source appendix — CODE VERIFIED. HR / Finance role-specific runtime: NOT LIVE VERIFIED. No handlers were executed. Extracted hook inventories are not a claim that every endpoint has passed a security audit. Full source links support review of branches beyond the excerpts.

## HR / Finance and critical shared actions

| File / line | Function | Registered hooks | First guard / validation statements |
|---|---|---|---|
| [includes/class-sn-hr-transfer.php:242](../source/sales-network-v4/includes/class-sn-hr-transfer.php#L242) | [ajax_search_users](../evidence/registry-class-sn-hr-transfer--ajax_search_users.txt) | wp_ajax_sn_hr_transfer_search_users | if ( ! $this->can_manage_hr() ) { ; check_ajax_referer( 'sn_hr_transfer_search_users', 'nonce' ); |
| [includes/class-sn-hr-transfer.php:285](../source/sales-network-v4/includes/class-sn-hr-transfer.php#L285) | [handle_export](../evidence/registry-class-sn-hr-transfer--handle_export.txt) | admin_post_sn_hr_export_users | if ( ! $this->can_manage_hr() ) { ; wp_die( 'دسترسی غیرمجاز' ); ; check_admin_referer( 'sn_hr_export_users' ); ; wp_die( 'حداقل یک کاربر HR را برای خروجی انتخاب کنید.' ); ; wp_die( 'در خروجی انتخابی حداکثر ۵۰۰۰ کاربر قابل انتخاب است؛ برای تعداد بیشتر گزینه همه کاربران را بزنید.' ); ; wp_die( 'جداول منابع انسانی کامل نیستند؛ ابتدا مهاجرت/فعال‌سازی افزونه را اجرا کنید.' ); ; wp_die( 'خواندن اطلاعات خروجی انجام نشد. جزئیات در لاگ وردپرس ثبت شده است.' ); |
| [includes/class-sn-hr-transfer.php:480](../source/sales-network-v4/includes/class-sn-hr-transfer.php#L480) | [can_manage_hr](../evidence/registry-class-sn-hr-transfer--can_manage_hr.txt) | internal / registration not matched | private function can_manage_hr(): bool { ; if ( current_user_can( 'manage_options' ) ) { ; if ( ! is_user_logged_in() ) { |
| [includes/class-sn-plugin.php:2287](../source/sales-network-v4/includes/class-sn-plugin.php#L2287) | [handle_hr_view_staff_panel](../evidence/registry-class-sn-plugin--handle_hr_view_staff_panel.txt) | admin_post_sn_hr_view_staff_panel | if (! $this->sn_can_manage_hr_panel() \|\| $this->sn_staff_view_data()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_view_staff_panel_' . $target_id); ; wp_die('پنل این نیرو در دسترس نیست.'); ; if ($url === '' \|\| (string) ($resolved['route_type'] ?? '') === 'placeholder') { wp_die('پنل این نیرو در دسترس نیست.'); } |
| [includes/class-sn-plugin.php:2312](../source/sales-network-v4/includes/class-sn-plugin.php#L2312) | [handle_hr_return_from_staff](../evidence/registry-class-sn-plugin--handle_hr_return_from_staff.txt) | admin_post_sn_hr_return_from_staff | check_admin_referer('sn_hr_return_from_staff'); ; if (! $data) { wp_die('زمان مشاهده پنل به پایان رسیده است.'); } ; if (! $actor) { wp_die('حساب منابع انسانی در دسترس نیست.'); } |
| [includes/class-sn-plugin.php:9112](../source/sales-network-v4/includes/class-sn-plugin.php#L9112) | [sn_can_manage_hr_panel](../evidence/registry-class-sn-plugin--sn_can_manage_hr_panel.txt) | internal / registration not matched | private function sn_can_manage_hr_panel(): bool ; if (current_user_can('manage_options')) { ; if (! is_user_logged_in()) { |
| [includes/class-sn-plugin.php:9158](../source/sales-network-v4/includes/class-sn-plugin.php#L9158) | [sn_hr_apply_user_access_for_position](../evidence/registry-class-sn-plugin--sn_hr_apply_user_access_for_position.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-plugin.php:10338](../source/sales-network-v4/includes/class-sn-plugin.php#L10338) | [handle_hr_change_request_create](../evidence/registry-class-sn-plugin--handle_hr_change_request_create.txt) | admin_post_sn_hr_change_request_create | if (! is_user_logged_in()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_change_request_create'); |
| [includes/class-sn-plugin.php:10439](../source/sales-network-v4/includes/class-sn-plugin.php#L10439) | [handle_hr_change_request_review](../evidence/registry-class-sn-plugin--handle_hr_change_request_review.txt) | admin_post_sn_hr_change_request_review | if (! is_user_logged_in()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_change_request_review'); |
| [includes/class-sn-plugin.php:11728](../source/sales-network-v4/includes/class-sn-plugin.php#L11728) | [handle_hr_bulk_assign_hierarchy](../evidence/registry-class-sn-plugin--handle_hr_bulk_assign_hierarchy.txt) | admin_post_sn_hr_bulk_assign_hierarchy | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_bulk_assign_hierarchy'); |
| [includes/class-sn-plugin.php:12967](../source/sales-network-v4/includes/class-sn-plugin.php#L12967) | [sn_senior_supervisor_seller_rows](../evidence/registry-class-sn-plugin--sn_senior_supervisor_seller_rows.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-plugin.php:13403](../source/sales-network-v4/includes/class-sn-plugin.php#L13403) | [sn_current_user_can_sales_manager_panel](../evidence/registry-class-sn-plugin--sn_current_user_can_sales_manager_panel.txt) | internal / registration not matched | private function sn_current_user_can_sales_manager_panel(): bool ; if (! is_user_logged_in()) { return false; } ; if (current_user_can('manage_options') \|\| current_user_can('sn_view_sales_reports')) { return true; } |
| [includes/class-sn-plugin.php:16495](../source/sales-network-v4/includes/class-sn-plugin.php#L16495) | [handle_hr_save_position](../evidence/registry-class-sn-plugin--handle_hr_save_position.txt) | admin_post_sn_hr_save_position, wp_ajax_sn_hr_save_position | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_save_position'); |
| [includes/class-sn-plugin.php:16547](../source/sales-network-v4/includes/class-sn-plugin.php#L16547) | [handle_hr_save_level](../evidence/registry-class-sn-plugin--handle_hr_save_level.txt) | admin_post_sn_hr_save_level, wp_ajax_sn_hr_save_level | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز برای تغییر وضعیت سطح'); } ; check_admin_referer('sn_hr_save_level'); |
| [includes/class-sn-plugin.php:16729](../source/sales-network-v4/includes/class-sn-plugin.php#L16729) | [handle_hr_save_structure](../evidence/registry-class-sn-plugin--handle_hr_save_structure.txt) | admin_post_sn_hr_save_structure | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_save_structure'); |
| [includes/class-sn-plugin.php:17740](../source/sales-network-v4/includes/class-sn-plugin.php#L17740) | [handle_hr_csv_dry_run](../evidence/registry-class-sn-plugin--handle_hr_csv_dry_run.txt) | admin_post_sn_hr_csv_dry_run | See full function and caller; not an endpoint guard |
| [includes/class-sn-plugin.php:17745](../source/sales-network-v4/includes/class-sn-plugin.php#L17745) | [handle_hr_people_import](../evidence/registry-class-sn-plugin--handle_hr_people_import.txt) | admin_post_sn_hr_people_import | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; $has_people_import_nonce = $posted_nonce !== '' && (bool) wp_verify_nonce($posted_nonce, 'sn_hr_people_import'); ; check_admin_referer($nonce_action); |
| [includes/class-sn-plugin.php:18287](../source/sales-network-v4/includes/class-sn-plugin.php#L18287) | [handle_hr_simple_add_person](../evidence/registry-class-sn-plugin--handle_hr_simple_add_person.txt) | admin_post_sn_hr_simple_add_person, wp_ajax_sn_hr_simple_add_person | if (! $this->sn_can_manage_hr_panel()) { ; wp_die('دسترسی غیرمجاز'); ; if (! check_admin_referer('sn_hr_simple_add_person', '_wpnonce', false)) { ; wp_die('نشست منقضی شده است'); |
| [includes/class-sn-plugin.php:18451](../source/sales-network-v4/includes/class-sn-plugin.php#L18451) | [handle_hr_profile_action](../evidence/registry-class-sn-plugin--handle_hr_profile_action.txt) | admin_post_sn_hr_profile_action, wp_ajax_sn_hr_profile_action | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_profile_action'); |
| [includes/class-sn-plugin.php:18669](../source/sales-network-v4/includes/class-sn-plugin.php#L18669) | [handle_hr_inline_profile_update](../evidence/registry-class-sn-plugin--handle_hr_inline_profile_update.txt) | admin_post_sn_hr_inline_profile_update | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_inline_profile_update'); |
| [includes/class-sn-plugin.php:18744](../source/sales-network-v4/includes/class-sn-plugin.php#L18744) | [handle_hr_bulk_inline_profile_update](../evidence/registry-class-sn-plugin--handle_hr_bulk_inline_profile_update.txt) | admin_post_sn_hr_bulk_inline_profile_update, wp_ajax_sn_hr_bulk_inline_profile_update | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_bulk_inline_profile_update'); |
| [includes/class-sn-plugin.php:18796](../source/sales-network-v4/includes/class-sn-plugin.php#L18796) | [handle_hr_bulk_workforce_action](../evidence/registry-class-sn-plugin--handle_hr_bulk_workforce_action.txt) | admin_post_sn_hr_bulk_workforce_action, wp_ajax_sn_hr_bulk_workforce_action | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_bulk_workforce_action'); |
| [includes/class-sn-plugin.php:18875](../source/sales-network-v4/includes/class-sn-plugin.php#L18875) | [handle_hr_replace_supervisor](../evidence/registry-class-sn-plugin--handle_hr_replace_supervisor.txt) | admin_post_sn_hr_replace_supervisor | if (! $this->sn_can_manage_hr_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_hr_replace_supervisor'); |
| [includes/class-sn-plugin.php:18976](../source/sales-network-v4/includes/class-sn-plugin.php#L18976) | [handle_hr_send_login_credentials_sms](../evidence/registry-class-sn-plugin--handle_hr_send_login_credentials_sms.txt) | admin_post_sn_hr_send_login_credentials_sms | if (! $this->sn_can_manage_hr_panel()) { ; wp_die('دسترسی غیرمجاز'); ; check_admin_referer('sn_hr_send_login_credentials_sms'); |
| [includes/class-sn-plugin.php:19060](../source/sales-network-v4/includes/class-sn-plugin.php#L19060) | [handle_hr_set_user_password](../evidence/registry-class-sn-plugin--handle_hr_set_user_password.txt) | admin_post_sn_hr_set_user_password, wp_ajax_sn_hr_set_user_password | if (! $this->sn_can_manage_hr_panel()) { ; wp_die('دسترسی غیرمجاز'); ; if (! check_ajax_referer('sn_hr_set_user_password', '_wpnonce', false)) { ; check_admin_referer('sn_hr_set_user_password'); |
| [includes/class-sn-plugin.php:19175](../source/sales-network-v4/includes/class-sn-plugin.php#L19175) | [handle_hr_bulk_set_password](../evidence/registry-class-sn-plugin--handle_hr_bulk_set_password.txt) | admin_post_sn_hr_bulk_set_password, wp_ajax_sn_hr_bulk_set_password | if (! $this->sn_can_manage_hr_panel()) { ; wp_die('دسترسی غیرمجاز'); ; if (! check_ajax_referer('sn_hr_bulk_set_password', '_wpnonce', false)) { ; check_admin_referer('sn_hr_bulk_set_password'); |
| [includes/class-sn-plugin.php:19614](../source/sales-network-v4/includes/class-sn-plugin.php#L19614) | [handle_hr_compensation_timeline_action](../evidence/registry-class-sn-plugin--handle_hr_compensation_timeline_action.txt) | admin_post_sn_hr_compensation_timeline_action | if (! $this->sn_can_manage_hr_panel()) { ; wp_die('دسترسی غیرمجاز'); ; check_admin_referer('sn_hr_compensation_timeline_action'); |
| [includes/class-sn-plugin.php:20758](../source/sales-network-v4/includes/class-sn-plugin.php#L20758) | [handle_distribution_return_items](../evidence/registry-class-sn-plugin--handle_distribution_return_items.txt) | admin_post_sn_distribution_return_items, wp_ajax_sn_distribution_return_items | if (! is_user_logged_in()) { wp_die('دسترسی غیرمجاز'); } ; $form_nonce_status = $nonce_value !== '' && (bool) wp_verify_nonce($nonce_value, 'sn_distribution_return_items'); ; check_admin_referer('sn_distribution_return_items'); ; if (! current_user_can('manage_options') && ! in_array($actor_position, ['sales_deputy','sales_manager','senior_supervisor','supervisor'], true)) { ; wp_die('دسترسی غیرمجاز'); |
| [includes/class-sn-plugin.php:22033](../source/sales-network-v4/includes/class-sn-plugin.php#L22033) | [sn_distribution_item_has_invoice_lock](../evidence/registry-class-sn-plugin--sn_distribution_item_has_invoice_lock.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-plugin.php:22053](../source/sales-network-v4/includes/class-sn-plugin.php#L22053) | [handle_mis_return_assignments](../evidence/registry-class-sn-plugin--handle_mis_return_assignments.txt) | admin_post_sn_mis_return_assignments | if (! $this->sn_can_access_mis_panel()) { wp_die('دسترسی غیرمجاز'); } ; check_admin_referer('sn_mis_return_assignments'); |
| [includes/class-sn-plugin.php:29228](../source/sales-network-v4/includes/class-sn-plugin.php#L29228) | [sn_can_view_finance](../evidence/registry-class-sn-plugin--sn_can_view_finance.txt) | internal / registration not matched | private function sn_can_view_finance(): bool ; return current_user_can('manage_options') ; \|\| current_user_can('sn_view_finance') |
| [includes/class-sn-plugin.php:29235](../source/sales-network-v4/includes/class-sn-plugin.php#L29235) | [sn_can_view_wallet_commission](../evidence/registry-class-sn-plugin--sn_can_view_wallet_commission.txt) | internal / registration not matched | private function sn_can_view_wallet_commission(): bool ; return current_user_can('manage_options') \|\| current_user_can('sn_view_wallet_commission'); |
| [includes/class-sn-plugin.php:29240](../source/sales-network-v4/includes/class-sn-plugin.php#L29240) | [sn_can_approve_payment](../evidence/registry-class-sn-plugin--sn_can_approve_payment.txt) | internal / registration not matched | private function sn_can_approve_payment(): bool ; return current_user_can('manage_options') \|\| current_user_can('sn_approve_payment'); |
| [includes/class-sn-plugin.php:29245](../source/sales-network-v4/includes/class-sn-plugin.php#L29245) | [sn_can_reject_payment](../evidence/registry-class-sn-plugin--sn_can_reject_payment.txt) | internal / registration not matched | private function sn_can_reject_payment(): bool ; return current_user_can('manage_options') \|\| current_user_can('sn_reject_payment'); |
| [includes/class-sn-plugin.php:29275](../source/sales-network-v4/includes/class-sn-plugin.php#L29275) | [sn_internal_invoice_info_access](../evidence/registry-class-sn-plugin--sn_internal_invoice_info_access.txt) | internal / registration not matched | if (! $invoice \|\| ! is_user_logged_in() \|\| $viewer_user_id <= 0) { ; if ($this->sn_can_view_finance()) { |
| [includes/class-sn-plugin.php:29392](../source/sales-network-v4/includes/class-sn-plugin.php#L29392) | [sn_can_perform_internal_invoice_payment_action](../evidence/registry-class-sn-plugin--sn_can_perform_internal_invoice_payment_action.txt) | internal / registration not matched | private function sn_can_perform_internal_invoice_payment_action($invoice, string $action = ''): bool ; if (! $invoice \|\| ! is_user_logged_in()) { ; if (current_user_can('manage_options') \|\| $this->sn_can_view_finance()) { |
| [includes/class-sn-plugin.php:30430](../source/sales-network-v4/includes/class-sn-plugin.php#L30430) | [ajax_financial_approve_payment](../evidence/registry-class-sn-plugin--ajax_financial_approve_payment.txt) | wp_ajax_sn_financial_approve_payment | if (! $this->sn_can_approve_payment()) { if (function_exists('status_header')) { status_header(403); } SN_Helpers::send_json(false,'دسترسی غیرمجاز'); return; } ; $valid=check_ajax_referer('sn_admin','nonce',false)\|\|check_ajax_referer('sn_public','nonce',false); |
| [includes/class-sn-plugin.php:30445](../source/sales-network-v4/includes/class-sn-plugin.php#L30445) | [ajax_financial_bulk_approve](../evidence/registry-class-sn-plugin--ajax_financial_bulk_approve.txt) | wp_ajax_sn_financial_bulk_approve | if (! $this->sn_can_approve_payment()) { if (function_exists('status_header')) { status_header(403); } SN_Helpers::send_json(false,'دسترسی غیرمجاز'); return; } ; $valid=check_ajax_referer('sn_admin','nonce',false)\|\|check_ajax_referer('sn_public','nonce',false); |
| [includes/class-sn-plugin.php:30465](../source/sales-network-v4/includes/class-sn-plugin.php#L30465) | [ajax_financial_reject_payment](../evidence/registry-class-sn-plugin--ajax_financial_reject_payment.txt) | wp_ajax_sn_financial_reject_payment | if (! $this->sn_can_reject_payment()) { if (function_exists('status_header')) { status_header(403); } SN_Helpers::send_json(false, 'دسترسی غیرمجاز'); return; } ; $valid = check_ajax_referer('sn_admin', 'nonce', false) \|\| check_ajax_referer('sn_public', 'nonce', false); |
| [includes/class-sn-plugin.php:30678](../source/sales-network-v4/includes/class-sn-plugin.php#L30678) | [ajax_financial_reopen_invoice](../evidence/registry-class-sn-plugin--ajax_financial_reopen_invoice.txt) | wp_ajax_sn_financial_reopen_invoice | if (! $this->sn_can_approve_payment()) { if (function_exists('status_header')) { status_header(403); } SN_Helpers::send_json(false,'دسترسی غیرمجاز'); return; } ; $valid=check_ajax_referer('sn_admin','nonce',false)\|\|check_ajax_referer('sn_public','nonce',false); |
| [includes/class-sn-plugin.php:30690](../source/sales-network-v4/includes/class-sn-plugin.php#L30690) | [ajax_financial_cancel_invoice](../evidence/registry-class-sn-plugin--ajax_financial_cancel_invoice.txt) | wp_ajax_sn_financial_cancel_invoice | if (! $this->sn_can_reject_payment()) { if (function_exists('status_header')) { status_header(403); } SN_Helpers::send_json(false,'دسترسی غیرمجاز'); return; } ; $valid=check_ajax_referer('sn_admin','nonce',false)\|\|check_ajax_referer('sn_public','nonce',false); |
| [includes/class-sn-plugin.php:31009](../source/sales-network-v4/includes/class-sn-plugin.php#L31009) | [handle_financial_gateway_export](../evidence/registry-class-sn-plugin--handle_financial_gateway_export.txt) | admin_post_sn_financial_gateway_export | if (! is_user_logged_in() \|\| ! $this->sn_can_view_finance()) { ; wp_die('دسترسی به خروجی تراکنش‌های درگاهی مجاز نیست.', 'دسترسی غیرمجاز', ['response' => 403]); ; check_admin_referer('sn_financial_gateway_export'); ; wp_die('حداقل یک ستون برای خروجی انتخاب کنید.', 'ستون خروجی انتخاب نشده', ['response' => 400]); ; if ($out === false) { throw new RuntimeException('zibal_export_output_unavailable'); } |
| [includes/class-sn-plugin.php:33041](../source/sales-network-v4/includes/class-sn-plugin.php#L33041) | [handle_commission_save_rule](../evidence/registry-class-sn-plugin--handle_commission_save_rule.txt) | admin_post_sn_commission_save_rule | if (! $this->sn_can_view_finance()) { ; wp_die('دسترسی غیرمجاز'); ; check_admin_referer('sn_commission_save_rule'); |
| [includes/class-sn-plugin.php:33099](../source/sales-network-v4/includes/class-sn-plugin.php#L33099) | [handle_commission_run_dry_run](../evidence/registry-class-sn-plugin--handle_commission_run_dry_run.txt) | admin_post_sn_commission_run_dry_run | if (! $this->sn_can_view_finance()) { ; wp_die('دسترسی غیرمجاز'); ; check_admin_referer('sn_commission_run_dry_run'); |
| [includes/class-sn-plugin.php:33353](../source/sales-network-v4/includes/class-sn-plugin.php#L33353) | [handle_commission_review_run](../evidence/registry-class-sn-plugin--handle_commission_review_run.txt) | admin_post_sn_commission_review_run | if (! $this->sn_can_view_finance()) { ; wp_die('دسترسی غیرمجاز'); ; check_admin_referer('sn_commission_review_run'); |
| [includes/class-sn-plugin.php:33388](../source/sales-network-v4/includes/class-sn-plugin.php#L33388) | [handle_commission_save_posting_flag](../evidence/registry-class-sn-plugin--handle_commission_save_posting_flag.txt) | admin_post_sn_commission_save_posting_flag | if (! $this->sn_can_view_finance()) { ; wp_die('دسترسی غیرمجاز'); ; check_admin_referer('sn_commission_save_posting_flag'); |
| [includes/class-sn-plugin.php:33400](../source/sales-network-v4/includes/class-sn-plugin.php#L33400) | [handle_commission_post_wallet](../evidence/registry-class-sn-plugin--handle_commission_post_wallet.txt) | admin_post_sn_commission_post_wallet | if (! $this->sn_can_view_finance()) { ; wp_die('دسترسی غیرمجاز'); ; check_admin_referer('sn_commission_post_wallet'); |
| [includes/class-sn-plugin.php:33802](../source/sales-network-v4/includes/class-sn-plugin.php#L33802) | [ensure_finance_role](../evidence/registry-class-sn-plugin--ensure_finance_role.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-plugin.php:35652](../source/sales-network-v4/includes/class-sn-plugin.php#L35652) | [sn_credit_wallet_for_invoice](../evidence/registry-class-sn-plugin--sn_credit_wallet_for_invoice.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-plugin.php:35743](../source/sales-network-v4/includes/class-sn-plugin.php#L35743) | [sn_wallet_autopost_recent_commissions_for_seller](../evidence/registry-class-sn-plugin--sn_wallet_autopost_recent_commissions_for_seller.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-plugin.php:35850](../source/sales-network-v4/includes/class-sn-plugin.php#L35850) | [render_wallet_box_for_user](../evidence/registry-class-sn-plugin--render_wallet_box_for_user.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-plugin.php:36003](../source/sales-network-v4/includes/class-sn-plugin.php#L36003) | [ajax_wallet_manual_adjust](../evidence/registry-class-sn-plugin--ajax_wallet_manual_adjust.txt) | wp_ajax_sn_wallet_manual_adjust | if (! current_user_can('manage_options') \|\| ! check_ajax_referer('sn_admin', 'nonce', false)) { |
| [includes/class-sn-plugin.php:36023](../source/sales-network-v4/includes/class-sn-plugin.php#L36023) | [ajax_wallet_recalculate](../evidence/registry-class-sn-plugin--ajax_wallet_recalculate.txt) | wp_ajax_sn_wallet_recalculate | $valid_nonce = check_ajax_referer('sn_admin', 'nonce', false) \|\| check_ajax_referer('sn_public', 'nonce', false); ; if (! $valid_nonce \|\| (! current_user_can('manage_options') && ! $this->sn_can_view_finance())) { |
| [includes/class-sn-plugin.php:36598](../source/sales-network-v4/includes/class-sn-plugin.php#L36598) | [ajax_financial_invoices](../evidence/registry-class-sn-plugin--ajax_financial_invoices.txt) | wp_ajax_sn_financial_invoices | if (! $this->sn_can_view_finance()) { if (function_exists('status_header')) { status_header(403); } SN_Helpers::send_json(false, 'دسترسی غیرمجاز'); return; } ; $valid = check_ajax_referer('sn_admin', 'nonce', false) \|\| check_ajax_referer('sn_public', 'nonce', false); ; $can_approve = $this->sn_can_approve_payment(); ; $can_reject = $this->sn_can_reject_payment(); |
| [includes/class-sn-plugin.php:36811](../source/sales-network-v4/includes/class-sn-plugin.php#L36811) | [ajax_financial_invoice_details](../evidence/registry-class-sn-plugin--ajax_financial_invoice_details.txt) | wp_ajax_sn_financial_invoice_details | if (! is_user_logged_in()) { if (function_exists('status_header')) { status_header(403); } SN_Helpers::send_json(false, 'دسترسی غیرمجاز'); return; } ; $valid = check_ajax_referer('sn_admin', 'nonce', false) \|\| check_ajax_referer('sn_public', 'nonce', false); ; if (! $this->sn_can_view_finance() && ! $this->sn_can_resend_invoice_sms($invoice)) { if (function_exists('status_header')) { status_header(403); } SN_Helpers::send_json(false, 'این فاکتور در محدوده دسترسی شما نیست'); return; } |
| [includes/class-sn-purpose-commission.php:26](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L26) | [is_matrix_engine_enabled](../evidence/registry-class-sn-purpose-commission--is_matrix_engine_enabled.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-purpose-commission.php:169](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L169) | [register_admin_page](../evidence/registry-class-sn-purpose-commission--register_admin_page.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-purpose-commission.php:185](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L185) | [render_admin_page](../evidence/registry-class-sn-purpose-commission--render_admin_page.txt) | internal / registration not matched | if ( ! current_user_can( 'sn_view_wallet_commission' ) && ! current_user_can( 'manage_options' ) ) { wp_die( 'دسترسی غیرمجاز' ); } |
| [includes/class-sn-purpose-commission.php:303](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L303) | [handle_backfill](../evidence/registry-class-sn-purpose-commission--handle_backfill.txt) | admin_post_sn_backfill_purpose_commissions | if ( ! current_user_can( 'manage_options' ) && ! current_user_can( 'sn_manage_wallets' ) ) { wp_die( 'دسترسی غیرمجاز' ); } ; check_admin_referer( 'sn_backfill_purpose_commissions' ); |
| [includes/class-sn-purpose-commission.php:316](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L316) | [handle_save_matrix](../evidence/registry-class-sn-purpose-commission--handle_save_matrix.txt) | admin_post_sn_save_purpose_commission_matrix | if ( ! current_user_can( 'manage_options' ) && ! current_user_can( 'sn_manage_wallets' ) ) { wp_die( 'دسترسی غیرمجاز' ); } ; check_admin_referer( 'sn_save_purpose_commission_matrix' ); ; if ( is_wp_error( $result ) ) { wp_die( esc_html( $result->get_error_message() ) ); } |
| [includes/class-sn-purpose-commission.php:584](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L584) | [on_payment_stage_approved](../evidence/registry-class-sn-purpose-commission--on_payment_stage_approved.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-purpose-commission.php:593](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L593) | [post_commissions_for_stage](../evidence/registry-class-sn-purpose-commission--post_commissions_for_stage.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-purpose-commission.php:692](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L692) | [on_invoice_financial_rejected](../evidence/registry-class-sn-purpose-commission--on_invoice_financial_rejected.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/class-sn-purpose-commission.php:703](../source/sales-network-v4/includes/class-sn-purpose-commission.php#L703) | [on_invoice_financial_reopened](../evidence/registry-class-sn-purpose-commission--on_invoice_financial_reopened.txt) | internal / registration not matched | See full function and caller; not an endpoint guard |
| [includes/reports/class-sn-mis-report-source.php:6](../source/sales-network-v4/includes/reports/class-sn-mis-report-source.php#L6) | [__construct](../evidence/registry-class-sn-mis-report-source--__construct.txt) | internal / registration not matched | if (!is_user_logged_in()) { throw new RuntimeException('ورود به حساب لازم است.'); } ; if (!current_user_can('manage_options') && !in_array($position, array_merge(['mis'],array_keys(SN_MIS_Report_Model::ROLES)), true)) { throw new RuntimeException('مجوز مشاهده گزارش MIS ندارید.'); } ; if (!current_user_can('manage_options') && isset($this->people[$uid]) && !$this->people[$uid]['active']) { throw new RuntimeException('حساب سازمانی غیرفعال است.'); } ; $this->allowed = current_user_can('manage_options') \|\| $position === 'mis' ? null : $this->descendants($uid); ; $this->scope_hash = hash('sha256', wp_json_encode([$uid,$position,$this->allowed,$this->people,current_user_can('manage_options')])); |

## HR / Finance source-defined sections

The following nested headings/forms supplement the classified feature tables in the main audit. Every parent module and meaningful action is classified there; repeated form labels are not independent product features.

### render_hr_panel

- `includes/class-sn-plugin.php:10622` — پنل منابع انسانی
- `includes/class-sn-plugin.php:10646` — جزئیات فنی آخرین عملیات سطح HR
- `includes/class-sn-plugin.php:10656` — آخرین گزارش عملیات HR
- `includes/class-sn-plugin.php:10672` — جزئیات فنی گزارش [dynamic PHP]
- `includes/class-sn-plugin.php:10696` — نمای کلی و آمادگی HR
- `includes/class-sn-plugin.php:10698` — پروفایل‌ها بر اساس سمت [dynamic PHP]
- `includes/class-sn-plugin.php:10699` — پروفایل‌ها بر اساس سطح [dynamic PHP]
- `includes/class-sn-plugin.php:10700` — آمادگی HR [dynamic PHP]
- `includes/class-sn-plugin.php:10713` — سمت‌ها
- `includes/class-sn-plugin.php:10728` — [dynamic PHP] ذخیره حذف
- `includes/class-sn-plugin.php:10739` — [dynamic PHP] افزودن
- `includes/class-sn-plugin.php:10745` — سطح‌ها
- `includes/class-sn-plugin.php:10761` — [dynamic PHP]
- `includes/class-sn-plugin.php:10768` — <input type="hidden" name="action" value="sn_hr_save_level">
- `includes/class-sn-plugin.php:10787` — [dynamic PHP] افزودن
- `includes/class-sn-plugin.php:10804` — افزودن دستی نیرو
- `includes/class-sn-plugin.php:10807` — <input type="hidden" name="action" value="sn_hr_simple_add_person">
- `includes/class-sn-plugin.php:10826` — پروفایل‌های منابع انسانی - نمای قدیمی
- `includes/class-sn-plugin.php:10859` — ساختار سازمانی: دپارتمان، واحد، تیم
- `includes/class-sn-plugin.php:10867` — سلسله‌مراتب و مدیر مستقیم
- `includes/class-sn-plugin.php:10869` — پروفایل‌های سطح بالا [dynamic PHP]
- `includes/class-sn-plugin.php:10870` — ارتباط‌های مستقیم فعلی
- `includes/class-sn-plugin.php:10879` — لاگ منابع انسانی
- `includes/class-sn-plugin.php:10891` — کاربران بدون پروفایل HR
- `includes/class-sn-plugin.php:10898` — نگاشت نقش قدیمی به سمت HR

### sn_hr_render_workforce_management_section

- `includes/class-sn-plugin.php:9996` — مدیریت نیروها
- `includes/class-sn-plugin.php:10021` — <input type="hidden" name="action" value="sn_hr_bulk_inline_profile_update">
- `includes/class-sn-plugin.php:10086` — تنظیم رمز ورود
- `includes/class-sn-plugin.php:10130` — [dynamic PHP]
- `includes/class-sn-plugin.php:10132` — تنظیم رمز نیروهای انتخاب‌شده
- `includes/class-sn-plugin.php:10134` — <input type="hidden" name="action" value="sn_hr_bulk_set_password">

### sn_hr_render_bulk_workforce_section

- `includes/class-sn-plugin.php:10155` — تغییرات گروهی
- `includes/class-sn-plugin.php:10160` — [dynamic PHP] جزئیات گزارش [dynamic PHP] [dynamic PHP]
- `includes/class-sn-plugin.php:10163` — <input type="hidden" name="action" value="sn_hr_bulk_workforce_action">
- `includes/class-sn-plugin.php:10168` — انتخاب نیروها نیرویی منتقل نشده است
- `includes/class-sn-plugin.php:10176` — نوع عملیات
- `includes/class-sn-plugin.php:10186` — جزئیات عملیات

### sn_hr_render_compensation_section

- `includes/class-sn-plugin.php:11298` — حقوق و پورسانت نیروها
- `includes/class-sn-plugin.php:11302` — آمادگی حقوق و پورسانت [dynamic PHP]
- `includes/class-sn-plugin.php:11303` — استفاده از قوانین پورسانت [dynamic PHP]
- `includes/class-sn-plugin.php:11304` — قوانین فعال/غیرفعال [dynamic PHP]
- `includes/class-sn-plugin.php:11307` — ثبت بازه تاریخی حقوق و پورسانت
- `includes/class-sn-plugin.php:11310` — <input type="hidden" name="action" value="sn_hr_compensation_timeline_action">
- `includes/class-sn-plugin.php:11350` — تاریخچه موثر حقوق و پورسانت

### sn_hr_render_hierarchy_assignment_management

- `includes/class-sn-plugin.php:11597` — مدیریت زیرمجموعه‌ها
- `includes/class-sn-plugin.php:11612` — جزئیات آخرین Dry-run/Apply [dynamic PHP]
- `includes/class-sn-plugin.php:11617` — <input type="hidden" name="action" value="sn_hr_bulk_assign_hierarchy">

### render_financial_panel

- `includes/class-sn-plugin.php:30836` — پنل مالی
- `includes/class-sn-plugin.php:30845` — راهنمای جریان مالی
- `includes/class-sn-plugin.php:30858` — آمادگی فاز مالی
- `includes/class-sn-plugin.php:30875` — صف بررسی پرداخت
- `includes/class-sn-plugin.php:30885` — <input type="hidden" name="action" value="sn_financial_gateway_export">

### sn_render_wallet_readiness_section

- `includes/class-sn-plugin.php:31898` — آمادگی کیف پول و پورسانت
- `includes/class-sn-plugin.php:31912` — محاسبه پورسانت پرداخت‌های قبلی
- `includes/class-sn-plugin.php:31917` — آخرین گزارش محاسبه مجدد [dynamic PHP]
- `includes/class-sn-plugin.php:31923` — دفتر کل کیف پول
- `includes/class-sn-plugin.php:31957` — بررسی آمادگی کیف پول
- `includes/class-sn-plugin.php:31964` — آماده‌سازی اصلاح دستی کیف پول
- `includes/class-sn-plugin.php:31986` — آمادگی تسویه
- `includes/class-sn-plugin.php:31995` — لاگ کیف پول

### sn_render_commission_dry_run_section

- `includes/class-sn-plugin.php:32850` — موتور جدید پورسانت - پیش‌نمایش
- `includes/class-sn-plugin.php:32867` — عیب‌یابی آمادگی قوانین پورسانت
- `includes/class-sn-plugin.php:32880` — مدیریت قوانین پورسانت
- `includes/class-sn-plugin.php:32896` — <input type="hidden" name="action" value="sn_commission_save_rule">
- `includes/class-sn-plugin.php:32907` — <input type="hidden" name="action" value="sn_commission_save_rule">
- `includes/class-sn-plugin.php:32934` — اجرای پیش‌نمایش
- `includes/class-sn-plugin.php:32937` — <input type="hidden" name="action" value="sn_commission_run_dry_run">
- `includes/class-sn-plugin.php:32954` — نتایج پیش‌نمایش
- `includes/class-sn-plugin.php:32963` — بررسی و تایید پیش‌نمایش انتخاب‌شده
- `includes/class-sn-plugin.php:32983` — <input type="hidden" name="action" value="sn_commission_review_run">
- `includes/class-sn-plugin.php:32995` — <input type="hidden" name="action" value="sn_commission_save_posting_flag">
- `includes/class-sn-plugin.php:33001` — <input type="hidden" name="action" value="sn_commission_post_wallet">
- `includes/class-sn-plugin.php:33030` — لاگ تایید / ارسال

### render_module

- `includes/class-sn-hr-transfer.php:108` — خروجی کاربران
- `includes/class-sn-hr-transfer.php:111` — <input type="hidden" name="action" value="sn_hr_export_users">
- `includes/class-sn-hr-transfer.php:134` — ورود و بازیابی کاربران
- `includes/class-sn-hr-transfer.php:141` — <input type="hidden" name="action" value="sn_hr_people_import">

## Evidence coverage

Live inventories: seller-*-live.txt; supervisor-*-live.txt; senior-loaded-panels-live.json; manager-loaded-panels-live.json; deputy-loaded-panels-live.json; mis-loaded-panels-live.json; mis-report-details-live.txt. Some initial snapshots contain loading placeholders; loaded-panel evidence and explicit report observations take precedence. No success claim for mutation follows from button presence.

Raw registries: [endpoint-registry.json](../evidence/endpoint-registry.json), [function-index.json](../evidence/function-index.json), [status-occurrences-raw.json](../evidence/status-occurrences-raw.json). Literal-status occurrences are search candidates, not a normalized state machine.
