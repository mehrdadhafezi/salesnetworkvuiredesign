<?php
/** wp --user=<authorized administrator ID> eval-file this-file.php
 * Prints aggregates only. Creates a private report snapshot, not invoices or SMS.
 */
if (!defined('WP_CLI') || !WP_CLI || !current_user_can('manage_options')) { throw new RuntimeException('Run with WP-CLI as an authorized WordPress administrator.'); }
$store=new SN_MIS_Report_Store();$src=new SN_MIS_Report_Source();
$f=SN_MIS_Report_Model::filters(['campaign'=>'7','batch'=>93]);
$j=$store->start($src,$f);
do { $p=$store->step($j['id'],$src); WP_CLI::log('scanned='.$p['scanned']); } while ($p['status']==='building');
$s=$store->summary($j['id'],$src);
$flow=array_column($s['breakdown']['flow'],'total','label');$validity=array_column($s['breakdown']['validity'],'total','label');$quality=array_column($s['breakdown']['reason_quality'],'total','label');
$waiting=0;foreach($s['invoice_groups'] as $g){if($g['status']==='pre_invoice' && $g['workflow']==='awaiting_payment')$waiting+=(int)$g['total_count'];}
$actual=['rows'=>(int)$s['totals']['total'],'valid'=>(int)($validity['valid']??0),'delivered'=>(int)$s['totals']['delivered'],'sellers'=>(int)$s['totals']['seller_count'],'no_answer'=>(int)($flow['no_answer']??0),'not_purchased'=>(int)($flow['not_purchased']??0),'pre_invoice'=>(int)($flow['pre_invoice']??0),'callback'=>(int)($flow['callback']??0),'unrecorded'=>(int)($flow['unrecorded']??0),'short_reason'=>(int)($quality['short']??0),'linked_cases'=>(int)$s['totals']['linked'],'unique_invoices'=>(int)$s['invoice_totals']['total_count'],'pre_awaiting'=>$waiting,'completed_invoices'=>(int)$s['invoice_totals']['completed']];
$expected=['rows'=>413,'valid'=>367,'delivered'=>367,'sellers'=>10,'no_answer'=>148,'not_purchased'=>136,'pre_invoice'=>44,'callback'=>28,'unrecorded'=>11,'short_reason'=>42,'linked_cases'=>44,'unique_invoices'=>44,'pre_awaiting'=>43,'completed_invoices'=>1];
$compare=[];foreach($expected as $k=>$v){$compare[$k]=['baseline_2026_09_29'=>$v,'current'=>$actual[$k],'difference'=>$actual[$k]-$v];}
$ids=[];$pages=(int)ceil($actual['rows']/50);for($page=1;$page<=$pages;$page++){foreach($store->page($j['id'],$src,$page)['rows'] as $row){$ids[]=$row['row_id'];}}
$unique=count(array_unique($ids));if($unique!==$actual['rows'] || count($ids)!==$unique){WP_CLI::error('Pagination/dashboard reconciliation failed.');}
if(class_exists('ZipArchive')){do{$ej=$store->job($j['id'],$src);$e=SN_MIS_Report_XLSX::step($ej,$store,$src);}while($e['status']!=='ready');$export='PASS: XLSX internal case/invoice/review reconciliation';}else{$export='NOT RUN: PHP zip missing';}
WP_CLI::log(wp_json_encode(['notice'=>'Baseline is historical; differences require investigation, not forced correction. No raw customer data in this output.','comparison'=>$compare,'validity_all'=>$validity,'flow_all'=>$flow,'reason_quality_all'=>$quality,'invoice_groups'=>$s['invoice_groups'],'unique_case_reconciliation'=>$unique,'xlsx'=>$export,'shared_invoice_count'=>count($s['shared_invoices']),'collection'=>$s['progress']],JSON_PRETTY_PRINT|JSON_UNESCAPED_UNICODE));
$store->cancel($j['id'],$src);
