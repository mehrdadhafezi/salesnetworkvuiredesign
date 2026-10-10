"""SQL predicate regression checks using actual PHP string literals (no WordPress required).
Run: python3 tools/test-invoice-review-filters.py
This checks SQL semantics with SQLite, not PHP execution or MySQL integration.
"""
import re
import sqlite3
from pathlib import Path

source = (Path(__file__).resolve().parents[1] / 'includes/class-sn-invoice-review.php').read_text()
constants = dict(re.findall(r'private const (STATUS|ONLINE) = "([^"]+)";', source))
body = source.split('private static function status_condition', 1)[1].split('public static function register_hooks', 1)[0]
predicates = dict(re.findall(r"case '([^']+)':.*?return \[\"([^\"]+)\", \[\]\];", body, re.S))
def predicate(status):
    template = predicates.get(status, '{$current}=%s')
    return template.replace('{$current}', constants['STATUS']).replace('{$online}', constants['ONLINE']).replace('%s', '?')

db = sqlite3.connect(':memory:')
db.execute('CREATE TABLE invoices (id INTEGER, status TEXT, invoice_status TEXT, payment_status TEXT, payment_workflow_status TEXT, pay_method TEXT, current_payment_stage INTEGER, current_due_amount REAL, amount REAL)')
rows = [
    (1,'approved','approved','approved','completed','online',1,0,100),
    (2,'paid','paid','paid','completed','zibal',1,0,100),
    (3,'approved','approved','approved','completed','card',1,0,100),
    (4,'partial_paid','partial_paid','partial_paid','awaiting_assignment','online',1,0,100),
    (5,'partial_paid','partial_paid','partial_paid','awaiting_assignment','card',1,0,100),
    (6,'pre_invoice','pre_invoice','pending_payment','awaiting_payment','online',2,50,100),
    (7,'pending','pending','pending','awaiting_payment','online',1,100,100),
    (8,'pending_financial_approval','pending_financial_approval','pending_financial_approval','awaiting_financial_approval','card',1,100,100),
    (9,'receipt_uploaded','receipt_uploaded','receipt_uploaded','awaiting_financial_approval','card',1,100,100),
    (10,'rejected','approved','paid','completed','online',1,0,100),
    (11,'cancelled','approved','paid','archived','online',1,0,100),
    (12,'payment_archived','partial_paid','partial_paid','archived','online',1,0,100),
    (13,'approved','receipt_uploaded','pending_financial_approval','completed','online',1,0,100),
    (14,'','approved','approved','completed','asanpardakht',1,0,100),
    (15,None,None,'paid','completed','zarinpal',1,0,100),
    (16,'failed','failed','failed','awaiting_payment','online',1,100,100),
    (17,'pre_invoice','pre_invoice','pending_payment','awaiting_payment','card',1,100,100),
    (18,'pending_financial_approval','pre_invoice','pending_payment','awaiting_financial_approval','card',2,50,100),
    (19,'approved','pre_invoice','pending_payment','completed','online',2,0,100),
    (20,'paid','paid','paid','completed','gateway',1,0,100),
    (21,'awaiting_next_payment','','','awaiting_next_payment','card',1,0,100),
    (22,'payment_stage_issued','','','awaiting_payment','card',2,50,100),
    (23,'pending_payment','','','awaiting_payment','online',2,50,100),
]
db.executemany('INSERT INTO invoices VALUES (?,?,?,?,?,?,?,?,?)', rows)
expected = {
    'paid': [1,2,4,13,14,15,19,20],
    'approved': [1,3,13,14,19],
    'needs_review': [8,9,18],
    'receipt_uploaded': [9],
    'pending_financial_approval': [8,18],
    'pending_payment': [6,17,23],
    'awaiting_next_payment': [4,5,21],
    'payment_stage_issued': [6,22,23],
    'pre_invoice': [6,17],
    'pending': [7],
    'partial_paid': [4,5],
    'rejected': [10],
    'cancelled': [11],
    'payment_archived': [12],
}
for status, ids in expected.items():
    sql = predicate(status)
    args = [status] if '?' in sql else []
    actual = [r[0] for r in db.execute('SELECT id FROM invoices i WHERE '+sql+' ORDER BY id', args)]
    assert actual == ids, (status, actual, ids)
    count, amount = db.execute('SELECT COUNT(*), SUM(amount) FROM invoices i WHERE '+sql, args).fetchone()
    assert (count, amount) == (len(ids), len(ids)*100)
    print('PASS', status, actual)
assert db.execute('SELECT COUNT(*) FROM invoices i WHERE '+predicate('paid')+" AND i.pay_method='card'").fetchone()[0] == 0
print('PASS method intersection; 23 fixtures, 14 status filters')
