#!/usr/bin/env node
'use strict';
const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const root = path.resolve(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'assets/js/public-shell.js'), 'utf8');
const helper = source.slice(source.indexOf('/* Multi-receipt transport'));
const alerts = [];
const context = {window: {alert: message => alerts.push(message)}, document: {addEventListener() {}}};
vm.runInNewContext(helper, context);
const api = context.window.snReceiptFiles;
const normalize = x => JSON.parse(JSON.stringify(x));
let tests = 0;
function test(name, fn) { fn(); tests++; console.log('PASS '+name); }
const file = (name, size = 20) => ({name, size});
const form = () => ({entries: [], append(k, v) {this.entries.push([k, v]);}});
test('all selected files are sent with an exact batch count', () => {
 const fd = form(), files = [file('اول.jpg'), file('second.png'), file('third.pdf')];
 assert.strictEqual(api.append(fd, {files}), true);
 assert.deepStrictEqual(fd.entries.map(x => x[0]), ['receipt[]','receipt[]','receipt[]','receipt_count']);
 assert.strictEqual(fd.entries[3][1], '3');
 assert.strictEqual(fd.entries[2][1], files[2]);
});
test('the second invalid file prevents the entire request', () => {
 const fd = form(); assert.strictEqual(api.append(fd, {files:[file('ok.jpg'),file('bad.php')]}), false);
 assert.strictEqual(fd.entries.length, 0);
});
test('per-file, batch byte and file-count limits', () => {
 assert(api.error([file('a.pdf',5*1024*1024+1)]));
 assert(api.error(Array.from({length:5},()=>file('a.pdf',5*1024*1024))));
 assert(api.error(Array.from({length:11},()=>file('a.pdf'))));
 assert.strictEqual(api.error(Array.from({length:10},()=>file('a.pdf',2*1024*1024))), '');
});
test('legacy invoice and JSON collection URLs are deduplicated', () => {
 assert.deepStrictEqual(normalize(api.urls({receipt_url:'https://local/old.pdf',receipt_file:'https://local/old.pdf'})), ['https://local/old.pdf']);
 assert.deepStrictEqual(normalize(api.urls({receipt_urls:'["https://local/a.jpg","https://local/b.pdf"]',receipt_url:'https://local/a.jpg'})), ['https://local/a.jpg','https://local/b.pdf']);
 assert.deepStrictEqual(normalize(api.urls({receipt_urls:'broken'})), []);
});
test('non-HTTP links are excluded and attributes escaped', () => {
 const html = api.links({receipt_urls:['javascript:alert(1)','https://local/a.pdf?x="<bad>']});
 assert(!html.includes('javascript:')); assert(html.includes('&quot;&lt;bad&gt;')); assert(html.includes('rel="noopener"'));
});
test('every receipt picker supports multiple files and every handler uses batch transport', () => {
 for(const dir of ['assets/js','includes']) {
  for(const name of fs.readdirSync(path.join(root,dir))) {
   if(!/\.(js|php)$/.test(name))continue;
   const text=fs.readFileSync(path.join(root,dir,name),'utf8');
   for(const tag of text.match(/<input\b[^>]*>/g)||[]) {
    if(tag.includes('type="file"')&&tag.includes('receipt'))assert(tag.includes(' multiple'),name+': '+tag);
   }
   if(dir==='assets/js')assert(!/\.append\('receipt',/.test(text), name+' still sends only one file');
  }
 }
});
console.log(tests+' multi-receipt checks passed');
