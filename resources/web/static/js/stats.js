"use strict";
(function (){
    bindTabs(".js-stats-tabs");
    bindSortableTables(".js-table-sortable");

    function bindSortableTables(tableQuery) {
        const getCellValue = (tr, idx) => tr.children[idx].innerText || tr.children[idx].textContent;
        const comparer = (idx, asc) => (a, b) => (
            (v1, v2) =>
                (v1 !== '' && v2 !== '' && !isNaN(v1) && !isNaN(v2))
                ? (v1 - v2)
                : ( v1.toString().localeCompare(v2))
        )(
            getCellValue(asc ? a : b, idx),
            getCellValue(asc ? b : a, idx)
        );

        const table = document.querySelectorAll(tableQuery).forEach(table => {
            const body = table.querySelector("tbody");
            let dir = true;
            let lastIndex = 0;

            table.querySelectorAll(`thead tr th`).forEach(th => th.addEventListener('click', e => {
                const idx = Array.from(th.parentNode.children).indexOf(th);
                dir = (idx == lastIndex) ? !dir : true;
                lastIndex = idx;

                table.querySelectorAll('thead th span').forEach(span => span.innerHTML = "");
                th.querySelector('span').innerHTML = dir ? "↓" : "↑";

                Array.from(body.querySelectorAll('tr')).
                    sort(comparer(idx, dir)).
                    forEach(tr => body.appendChild(tr));
            }));
        });
    }

    function bindTabs(container) {
        function deselectAll(container) {
            for (let tab of document.querySelector(container).getElementsByTagName("li")) {
                tab.classList.remove("is-active");

                const target = document.querySelector(tab.dataset.target);
                if (!target.classList.contains("is-hidden")) {
                    target.classList.add("is-hidden");
                    target.setAttribute("aria-selected", false);
                }
            }
        };

        function selectOne(tab) {
            tab.classList.add("is-active");
            const target = document.querySelector(tab.dataset.target);
            target.classList.remove("is-hidden");
            target.setAttribute("aria-selected", true);
        }

        document.querySelector(container).addEventListener('click', e => {
            deselectAll(container);
            selectOne(e.target.parentNode); // target is <a>, the <li> carries the data.
        });

        const fragment = window.location.hash.substr(1);
        const tab = document.querySelector(`[data-target=".js-stats-tab-${fragment}"]`);
        if (tab) {
            deselectAll(container);
            selectOne(tab);
        }
    }
})();
