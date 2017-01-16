onmessage = function(e){
    let {dpsByTime, metric, margin} = e.data;
    let shutdowns = detectShutdowns(dpsByTime, metric, margin);
    postMessage(shutdowns);
};

// NOTE - expects dps sorted by time
function detectShutdowns(dpsByTime, metric, margin){
    // TODO - cache sorted result
    let dpsByValue = dpsByTime.slice().sort((a,b) => a[1] - b[1]),
        shutdowns = [],
        isShutdown = false;

    // TODO - bisect instead?
    for(let i = 0; i < dpsByValue.length; i++){
        let val = dpsByValue[i][1];
        if(isNaN(val)){
            continue;
        }

        // the rest of the values are above the margin,
        // so we good yo
        if(val > margin){
            break;
        }

        let index = dpsByTime.indexOf(dpsByValue[i]),
            prev = dpsByTime[--index];

        // find the first non NaN previous value
        while(prev && isNaN(prev[1])){
            prev = dpsByTime[--index];
        }

        // if the previous value is below the margin,
        // then this is not a new shutdown
        if(prev && prev[1] < margin){
            continue;
        }
        /*
        shutdowns.push({
            label: metric,
            dp: dpsByValue[i]
        });
        */
        shutdowns.push(dpsByValue[i]);
    }

    // sort chronologically
    shutdowns = shutdowns.sort((a,b) => (+a[0]) - (+b[0]));

    return shutdowns;
}

