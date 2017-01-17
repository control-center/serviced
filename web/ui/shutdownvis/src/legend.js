// build legend html
function buildLegendRowEl(config){
    let html = `
        <td class="series_color">
            <div class="line-piece top actual" style="border-bottom-color: ${config.color};"></div>
            <div class="line-piece bottom actual" style="border-top-color: ${config.color};"></div>
        </td>
        <td style="padding-right: 8px;">Actual</td>
        <td class="series_color">
            <div class="line-piece top derived" style="border-bottom-color: ${config.color2};"></div>
            <div class="line-piece bottom derived" style="border-top-color: ${config.color2};"></div>
        </td>
        <td style="padding-right: 8px;">Derived</td>
        <td class="metric_name">${config.name}</td>
    `;
    let el = document.createElement("tr");
    el.classList.add("legend_row");
    el.dataset.id = config.name;
    el.innerHTML = html;
    return el;
}

// build legend
export function buildLegend(data, el, events){
    let legendRowEls = data.map((d, i) => {
        return buildLegendRowEl({
            name: d.metric,
            color: d.color,
            color2: d.color2
        });
    });

    // TODO - unbind event listeners
    el.innerHTML = "";

    legendRowEls.forEach(row => {
        el.appendChild(row);
        for(let event in events){
            let fn = events[event];
            row.addEventListener(event, e => {
                fn(e.currentTarget.dataset.id);
            });
        }
    });
    
}
