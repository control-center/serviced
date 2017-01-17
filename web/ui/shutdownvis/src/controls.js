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

function buildTextInputEl(config){
    let html = `
        <label>${config.label}</label>
        <input type="text" value="${config.val}">
    `;

    let el = document.createElement("div");
    el.classList.add("control", "text-input");
    el.innerHTML = html;

    let inputEl = el.querySelector("input");

    for(let name in config.events){
        inputEl.addEventListener(name, e => {
            config.events[name](e.currentTarget.value);
        });
    }

    return el;
}

function buildButton(config){
    let el = document.createElement("a");
    el.classList.add("control", "button", "btn");
    el.style = config.style;
    el.innerHTML = config.label;
    for(let name in config.events){
        el.addEventListener(name, e => {
            config.events[name](e);
        });
    }

    return el;
}

const CONTROL_FNS = {
    slider: buildSliderEl,
    textInput: buildTextInputEl,
    button: buildButton
};

// bind controls to a model
export function buildControls(params, el){
    let controlEls = params.map(param => {
        let build = CONTROL_FNS[param.type];
        el.appendChild(build(param));
    });
}
