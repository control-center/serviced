export function get(url){
    return doXHR("GET", url);
}

export function post(url, body){
    return doXHR("POST", url, body);
}

function doXHR(method, url, body){
    if(body !== null && (typeof body !== "string")){
        body = JSON.stringify(body);
    }
    return new Promise((resolve, reject) => {
        var xhr = new XMLHttpRequest();
        let handleErr = function(msg){
            return function(){
                console.warn("xhr", msg, xhr);
                reject(xhr.status);
            };
        };
        xhr.open(method, url);
        xhr.onload = function() {
            if (xhr.status === 200) {
                // NOTE - assume valid json
                resolve(JSON.parse(xhr.response));
            } else {
                handleErr();
            }
        };
        xhr.onerror = handleErr("error");
        xhr.onabort = handleErr("abort");
        xhr.ontimeout = handleErr("timeout");
        xhr.send(body);
    });
}

export function debounce(fn, time){
    let last = new Date().getTime();
    return function(){
        let now = new Date().getTime();
        if(now - last > time){
            fn.apply(null, arguments);
            last = now;
        }
    };
}

// NOTE - setting width 0 will wrap on every newline
export function wrap(text, width) {
    text.each(function() {
        var text = d3.select(this),
            words = text.text().split(/\n/).reverse(),
            word,
            line = [],
            lineNumber = 0,
            lineHeight = 1.1, // ems
            y = text.attr("y"),
            dy = parseFloat(text.attr("dy")),
            tspan = text.text(null).append("tspan").attr("x", 0).attr("y", y).attr("dy", dy + "em");

        while (word = words.pop()) {
            line.push(word);
            tspan.text(line.join(" "));
            if (tspan.node().getComputedTextLength() > width) {
                line.pop();
                tspan.text(line.join(" "));
                line = [word];
                tspan = text
                    .append("tspan")
                    .attr("x", 0)
                    .attr("y", y)
                    .attr("dy", ++lineNumber * lineHeight + dy + "em")
                    .text(word);
            }

        }
    });
}
