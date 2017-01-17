function buildShutdownRowEl(config){
    let html = `
        <span class="shutdown_label">${config.label}</span> -
        <span class="shutdown_time">${new Date(config.dp[0])}</span>
    `;
    let el = document.createElement("li");
    el.dataset.metric = config.label;
    el.dataset.timestamp = config.dp[0];
    el.innerHTML = html;
    return el;
}

export function buildShutdownList(shutdowns, el, events){
    let shutdownListEl = el.querySelector(".shutdown_list"),
        shutdownCountEl = el.querySelector(".shutdown_count");

    let shutdownRowEls = shutdowns.map(shutdown => {
        return buildShutdownRowEl({
            label: shutdown.metric,
            dp: shutdown.dp
        });
    });

    shutdownListEl.innerHTML = "";

    shutdownRowEls.forEach(shutdown => {
        shutdownListEl.appendChild(shutdown);

        for(let event in events){
            let fn = events[event];
            shutdown.addEventListener(event, e => {
                fn(e.currentTarget.dataset.metric, e.currentTarget.dataset.timestamp);
            });
        }
    });

    shutdownCountEl.innerHTML = shutdownRowEls.length;
    if(shutdownRowEls.length){
        el.style.display = "block";
    } else {
        el.style.display = "hidden";
    }
}

