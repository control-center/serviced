function buildSliderEl(config){
    let html = `
        <label>${config.label}</label>
        <input type="range" step="${config.step}" min="${config.min}" max="${config.max}" value="${config.val}">
        <div class="value">
            <span class="value_num">${config.toUnit(config.val)}</span><span class="value_units">${config.units}</span>
        </div>
    `;
    let el = document.createElement("div");
    el.classList.add("control", "slider");
    el.innerHTML = html;

    let inputEl = el.querySelector("input"),
        valueEl = el.querySelector(".value_num");

    inputEl.addEventListener("input", e => {
        config.value = inputEl.value;
        valueEl.innerText = config.toUnit(config.value);
    });

    for(let name in config.events){
        inputEl.addEventListener(name, e => {
            config.events[name](e.currentTarget.value);
        });
    }

    return el;
}

// bind controls to a model
export function buildControls(params, el){
    let shutdownControlsEl = el.querySelector(".shutdown_controls");

    let controlEls = params.map(param => {
        return buildSliderEl(param);
    });

    controlEls.forEach(control => {
        // TODO - change listener to bind changes
        // to model
        shutdownControlsEl.appendChild(control);
    });
}
