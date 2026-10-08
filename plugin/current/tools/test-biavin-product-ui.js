/* Generate fixtures with test-biavin-live-product.php DIR; run this with DIR and jsdom on NODE_PATH. */
const fs = require('fs');
const path = require('path');
const assert = require('assert');
const {JSDOM} = require('jsdom');
const fixtureDir = path.resolve(process.argv[2]);
let checks = 0;
for (const initial of ['wallet_charge','form','physical_invoice','none']) {
  const dom = new JSDOM(fs.readFileSync(path.join(fixtureDir, `metabox-${initial}.html`), 'utf8'), {runScripts:'dangerously'});
  const doc = dom.window.document;
  const select = doc.querySelector('[data-sn-execution-type]');
  for (const type of ['wallet_charge','form','physical_invoice','']) {
    select.value = type; select.dispatchEvent(new dom.window.Event('change'));
    const input = doc.querySelector('[name="sn_execution_wallet_destination_url"]');
    assert.equal(input.closest('[hidden]'), null, 'destination remains visible for every execution type');
    assert.equal(doc.querySelectorAll('[name="sn_execution_wallet_destination_url"]').length, 1);
    assert.equal(doc.querySelector('[data-sn-execution-field="wallet_charge"]').hidden, type !== 'wallet_charge');
    assert.equal(doc.querySelector('[data-sn-execution-field="form"]').hidden, type !== 'form');
    checks += 4;
  }
  dom.window.close();
}
const card = new JSDOM(fs.readFileSync(path.join(fixtureDir, 'card-destination.html'), 'utf8'));
const anchors = card.window.document.querySelectorAll('.sn-customer-card-destination a');
assert.equal(anchors.length, 2); checks++;
for (const a of anchors) {
  assert.equal(a.href, 'https://updated.example/102?a=1&b=2');
  assert.equal(a.target, '_blank');
  assert(a.relList.contains('noopener') && a.relList.contains('noreferrer'));
  checks += 3;
}
card.window.close();
console.log(`PASS ${checks} Biavin product DOM/UI checks`);
