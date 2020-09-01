"use strict";

function tabsWithContent(container) {
    let myID = document.getElementById(container);

    let tabs = myID.querySelectorAll('.tabs li');
    let tabsContent = myID.querySelectorAll('.tab-content');
  
    let deactvateAllTabs = function () {
      tabs.forEach(function (tab) {
        tab.classList.remove('is-active');
      });
    };
  
    let hideTabsContent = function () {
      tabsContent.forEach(function (tabContent) {
        tabContent.classList.remove('is-active');
      });
    };
  
    let activateTabsContent = function (tab) {
      tabsContent[getIndex(tab)].classList.add('is-active');
    };
  
    let getIndex = function (el) {
      return [...el.parentElement.children].indexOf(el);
    };
  
    tabs.forEach(function (tab) {
      tab.addEventListener('click', function () {
        deactvateAllTabs();
        hideTabsContent();
        tab.classList.add('is-active');
        activateTabsContent(tab);
      });
    })
  
    tabs[0].click();
  };

window.Kaepora = {
    bindSortableTables(tableQuery) {
        const getCellValue = (tr, idx) => tr.children[idx].innerText || tr.children[idx].textContent;
        const comparer = (idx, asc) => (a, b) => ((v1, v2) =>
                (v1 !== '' && v2 !== '' && !isNaN(v1) && !isNaN(v2))
                ? (v1 - v2)
                : ( v1.toString().localeCompare(v2))
        )(
            getCellValue(asc ? a : b, idx),
            getCellValue(asc ? b : a, idx)
        );

        document.querySelectorAll(tableQuery).forEach(table => {
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
    },

    bindTabs(container, tabPrefix) {
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

        function getFirstTab() {
            return document.querySelector(container).getElementsByTagName("li")[0];
        }

        function onHistoryChange() {
            const fragment = window.location.hash.substr(1);
            const tab = fragment == ""
                ? getFirstTab()
                :  document.querySelector(`[data-target="${tabPrefix}${fragment}"]`)
            ;
            if (!tab) {
                return;
            }

            deselectAll(container);
            selectOne(tab);
        }

        document.querySelector(container).addEventListener('click', e => {
            deselectAll(container);
            selectOne(e.target.parentNode); // target is <a>, the <li> carries the data.
        });

        window.onpopstate = onHistoryChange;
        onHistoryChange();
    },

    updateLocalDatetimes() {
        const els = document.querySelectorAll('.js-local-datetime');
        for (let el of els) {
            const date = new Date(el.dataset.timestamp * 1000);
            el.innerHTML = date.toLocaleTimeString(undefined, {
                year: 'numeric',
                month: 'numeric',
                day: 'numeric',
                hour: 'numeric',
                minute: 'numeric',
            });
        }
    },
};


(function (){
    document.addEventListener('DOMContentLoaded', () => {
        window.Kaepora.updateLocalDatetimes();
    });
})();
