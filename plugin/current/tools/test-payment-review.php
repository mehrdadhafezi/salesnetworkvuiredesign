<?php
// Run with PHP CLI; no WordPress connection or real gateway requests.
define('ABSPATH', __DIR__.'/');
function wp_timezone() { return new DateTimeZone($GLOBALS['test_site_zone'] ?? 'UTC'); }
function apply_filters($name,$value,...$args) { return $value; }
require __DIR__.'/../includes/class-sn-helpers.php';
require __DIR__.'/../includes/class-sn-payment-transactions.php';
require __DIR__.'/../includes/class-sn-plugin.php';
$plugin=(new ReflectionClass('SN_Plugin'))->newInstanceWithoutConstructor();
$format=new ReflectionMethod('SN_Plugin','sn_zibal_export_paid_at');
function same($actual,$expected,$name) { if($actual!==$expected) { throw new RuntimeException($name.': '.var_export($actual,true).' != '.var_export($expected,true)); } echo "PASS $name\n"; }
foreach(['UTC','Asia/Tehran','America/New_York'] as $site) {
 $GLOBALS['test_site_zone']=$site;
 same($format->invoke($plugin,'2026-10-06 10:20:37','zibal_report'),'1405/07/14-13:50:37',"UTC provider time on $site site");
 same($format->invoke($plugin,'2026-10-06T10:20:37.123Z','zibal_report'),'1405/07/14-13:50:37',"ISO Z on $site site");
 same($format->invoke($plugin,'2026-10-06T13:50:37+03:30','zibal_report'),'1405/07/14-13:50:37',"explicit Tehran offset on $site site");
 same($format->invoke($plugin,'1405/07/14-22:45:09','zibal_report'),'1405/07/15-02:15:09',"Jalali UTC crosses midnight on $site site");
}
$GLOBALS['test_site_zone']='Asia/Tehran';
same($format->invoke($plugin,'2026-10-06 13:50:37','local'),'1405/07/14-13:50:37','local Tehran is not shifted twice');
$GLOBALS['test_site_zone']='UTC';
same($format->invoke($plugin,'2026-10-06 10:20:37','local'),'1405/07/14-13:50:37','local UTC converted once');
same($format->invoke($plugin,'2026-10-06','zibal_report'),'1405/07/14','date-only has no invented time');
same($format->invoke($plugin,'2026-02-30 10:00:00','zibal_report'),'','invalid date rejected');
same($format->invoke($plugin,'not a date','zibal_report'),'','garbage date rejected');
$epoch=(new DateTimeImmutable('2026-10-06T10:20:37Z'))->getTimestamp();
same($format->invoke($plugin,(string)$epoch,'zibal_report'),'1405/07/14-13:50:37','Unix seconds');
same($format->invoke($plugin,(string)($epoch*1000),'zibal_report'),'1405/07/14-13:50:37','Unix milliseconds');
$rows=[['amount'=>400000,'status'=>'pending','payment_stage_no'=>1,'proof_data'=>'{}'],['amount'=>600000,'status'=>'pending','payment_stage_no'=>1,'proof_data'=>'{}'],['amount'=>1000000,'status'=>'pending','payment_stage_no'=>1,'authority'=>'zibal:11'],['amount'=>900000,'status'=>'rejected','payment_stage_no'=>1,'proof_data'=>'{}'],['amount'=>500000,'status'=>'approved','payment_stage_no'=>2,'proof_data'=>'{}']];
$s=SN_Payment_Transactions::summary($rows,1,1000000);
same($s['pending_amount'],1000000.0,'mixed manual transfers sum');
same($s['recorded_amount'],1000000.0,'failed, rejected, other-stage and unfinished gateway excluded');
same($s['ready'],true,'full registered sum can be approved');
$s=SN_Payment_Transactions::summary([$rows[0]],1,1000000);
same($s['available_amount'],600000.0,'partial transfer leaves balance');
same($s['ready'],false,'partial sum cannot approve full stage');
foreach(['-1','100junk',[],INF,'1.001'] as $bad) { same(SN_Payment_Transactions::parse_amount($bad),-1.0,'invalid amount '.json_encode($bad)); }
same(SN_Payment_Transactions::parse_amount('۴۰۰٬۰۰۰'),400000.0,'Persian separated amount');
same(SN_Payment_Transactions::amount_error(600000,1000000,400000),'','exact remaining amount accepted');
same(SN_Payment_Transactions::amount_error(600001,1000000,400000)!=='',true,'excess amount rejected');
echo "Payment review checks passed\n";
