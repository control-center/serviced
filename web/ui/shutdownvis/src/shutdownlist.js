function buildShutdownRowEl(config){
    let html = `
        <span class="shutdown_label">${config.label}</span> -
        <span class="shutdown_time">${new Date(config.dp[0])}</span>
    `;
    let el = document.createElement("li");
    el.innerHTML = html;
    return el;
}

export function buildShutdownList(shutdowns, el){
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
    });

    shutdownCountEl.innerHTML = shutdownRowEls.length;
    if(shutdownRowEls.length){
        el.style.display = "block";
    } else {
        el.style.display = "hidden";
    }
}

