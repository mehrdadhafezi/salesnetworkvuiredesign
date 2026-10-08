/* Saved product settings -> existing card snapshots, in resumable batches. */
(function () {
    'use strict';
    function init(root) {
        var preview = root.querySelector('[data-sync-preview]');
        var apply = root.querySelector('[data-sync-apply]');
        var pause = root.querySelector('[data-sync-pause]');
        var report = root.querySelector('[data-sync-report]');
        var scope = root.querySelector('[data-sync-scope]');
        var fields = Array.from(root.querySelectorAll('[data-sync-field]'));
        var state = null, busy = false, stopping = false;
        var reasons = {eligible:'قابل اعمال', closed_card:'کارت بسته یا اقدام‌شده', upgrade_or_payment:'ارتقا یا فاکتور ثبت‌شده', execution:'کد صادرشده یا وارد اجرا', recorded_action:'اقدام ثبت‌شده', custom_credit:'اعتبار اختصاصی', inactive:'اشتراک غیرفعال', invalid_snapshot:'اطلاعات اشتراک نامعتبر', missing_card:'کارت حذف‌شده'};
        var watched = 'input[name="sn_product_credit_amount"],#sn-product-upgrade-list input,#sn-product-upgrade-list select';
        function signature() {
            return JSON.stringify(Array.from(document.querySelectorAll(watched)).map(function(el){return [el.name, el.type === 'checkbox' ? el.checked : el.value];}));
        }
        var initial = signature();
        function dirty() { return signature() !== initial; }
        function controls() {
            preview.disabled = busy || dirty();
            apply.disabled = busy || dirty() || !state || state.done;
            fields.forEach(function(el){el.disabled = busy || (state && state.started && !state.done);});
            scope.disabled = busy || !!(state && state.started && !state.done);
            pause.hidden = !busy || !state || !state.started;
            pause.disabled = stopping;
            apply.textContent = state && state.started && !state.done ? 'ادامه اعمال تغییرات' : 'اعمال روی کارت‌های قبلی';
        }
        function invalidate() { if (!busy) {state = null; controls(); report.textContent = dirty() ? 'محصول تغییر کرده است. ابتدا محصول را ذخیره کنید و سپس پیش‌نمایش بگیرید.' : 'برای تنظیمات انتخاب‌شده پیش‌نمایش تازه بگیرید.';} }
        fields.forEach(function(el){el.addEventListener('change',invalidate);});
        scope.addEventListener('change',invalidate);
        ['input','change'].forEach(function(event){document.addEventListener(event,function(e){if(e.target.matches(watched)){invalidate();}});});
        async function request(action, data) {
            var body = new URLSearchParams({action:action, product_id:root.dataset.productId, nonce:root.dataset.nonce});
            Object.keys(data).forEach(function(key){
                if (Array.isArray(data[key])) {data[key].forEach(function(value){body.append(key+'[]',value);});}
                else {body.set(key,String(data[key]));}
            });
            var controller = new AbortController();
            var timer = setTimeout(function(){controller.abort();},45000);
            try {
                var response = await fetch(root.dataset.url,{method:'POST',credentials:'same-origin',body:body,signal:controller.signal});
                var json = await response.json();
                if (!response.ok || !json.success) {throw new Error(json.data && json.data.message || 'درخواست پذیرفته نشد؛ صفحه را تازه کنید.');}
                return json.data;
            } catch (e) {
                if (e.name === 'AbortError') {throw new Error('پاسخ این مرحله نرسید. با «ادامه اعمال تغییرات» وضعیت را بررسی کنید؛ درخواست تکراری دوباره اعمال نمی‌شود.');}
                throw e;
            } finally {clearTimeout(timer);}
        }
        function lines(counts) {
            return counts.map(function(row){return (reasons[row.reason] || row.reason)+': '+Number(row.total).toLocaleString('fa-IR');}).join('\n');
        }
        function progress(data) {
            var text = (data.done ? 'انجام شد.' : 'در حال اعمال تغییرات…')+'\nبررسی‌شده: '+data.scanned+' از '+data.total+'\nبه‌روزرسانی‌شده: '+data.changed+' | از قبل یکسان: '+data.unchanged+' | اعمال‌نشده: '+data.skipped;
            var skipped = Object.keys(data.reasons || {}).map(function(key){return {reason:key,total:data.reasons[key]};});
            report.textContent = text+(skipped.length ? '\n'+lines(skipped) : '');
        }
        preview.addEventListener('click',async function(){
            if (busy || dirty()) {return;}
            var chosen = fields.filter(function(el){return el.checked;}).map(function(el){return el.value;});
            if (!chosen.length) {report.textContent = 'حداقل یک فیلد را انتخاب کنید.';return;}
            state = null;busy = true;controls();report.textContent = 'در حال بررسی کارت‌های قبلی…';
            try {
                var data = await request('sn_product_card_sync_preview',{fields:chosen,scope:scope.value});
                state = {token:data.token,cursor:0,done:data.total===0,started:false};
                report.textContent = 'تنظیمات ذخیره‌شده محصول — اعتبار: '+Number(data.credit).toLocaleString('fa-IR')+' تومان\nکارت‌های موجود: '+data.total+'\n'+lines(data.counts)+'\n'+(chosen.indexOf('upgrade_options')>=0 ? 'حالت‌های افزایشی نیز جایگزین می‌شوند.\n' : '')+'نمونه‌ها:\n'+data.samples.map(function(row){return 'کارت #'+row.id+'، اشتراک #'+row.membership_id+': '+Number(row.base_credit_snapshot || 0).toLocaleString('fa-IR')+' تومان — '+(reasons[row.reason] || row.reason);}).join('\n');
            } catch (e) {report.textContent = e.message;}
            finally {busy = false;controls();}
        });
        apply.addEventListener('click',async function(){
            if (busy || dirty() || !state || state.done) {return;}
            state.started = true;busy = true;stopping = false;controls();
            try {
                do {
                    var data = await request('sn_product_card_sync_apply',{token:state.token,cursor:state.cursor});
                    state.cursor = data.cursor;state.done = data.done;progress(data);
                } while (!state.done && !stopping && !dirty());
                if (!state.done) {report.textContent += '\nمتوقف شد؛ می‌توانید ادامه دهید.';}
            } catch (e) {report.textContent += '\n'+e.message;}
            finally {busy = false;controls();}
        });
        pause.addEventListener('click',function(){stopping = true;controls();});
        controls();
    }
    function boot(){document.querySelectorAll('[data-sn-product-card-sync]').forEach(init);}
    if(document.readyState === 'loading'){document.addEventListener('DOMContentLoaded',boot);}else{boot();}
})();
