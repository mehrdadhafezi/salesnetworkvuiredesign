(function () {
    'use strict';
    document.addEventListener('click', function (event) {
        if (!event.target.closest('[data-ir-print]')) return;
        document.querySelectorAll('.sn-ir details').forEach(function (item) { item.open = true; });
        window.print();
    });
}());
