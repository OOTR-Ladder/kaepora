"use strict";
(function (){
    bindTabs(".js-stats-tabs");

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
