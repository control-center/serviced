import {get, debounce} from "./utils";
import {buildControls} from "./controls";
import {buildLegend} from "./legend";
import {buildShutdownList} from "./shutdownlist";
import {buildGraph} from "./graph";

// colors come in pairs, one for actual data, 
// the other for calculated data
const COLORS = [
    ["#a6cee3", "#1f78b4"],
    ["#b2df8a", "#33a02c"],
    ["#cab2d6", "#6a3d9a"]
];

// try to recalculate prediction values
// as the slider is adjusted.
const CALCULATE_ON_INPUT = false;
const PREDICTION_DEBOUNCE = 200;

const TARGET_DP_COUNT = 10000;

let state = {
    model: {
        margin: 15e10,
        lookAhead: 20 * 60 * 1000,
        theWindow: 30 * 60 * 1000 
    },
    tsdb: {
        url: "http://uiboss:4242",
        tenantIDs: ["8noa8p8e84m3stayz1zqr3egx"],
        start: new Date(new Date().getTime() - 1000 * 60 * 60 * 24 * 7),
        end: new Date()
    },
    actual: undefined,
    calculated: undefined,
    shutdowns: undefined,
    graph: undefined
};

function getDiskUsageMetrics(tenants){
    let metrics = [
        "storage.pool.data.available",
        "storage.pool.metadata.available",
    ];
    metrics = metrics.concat(tenants.map(t => `storage.filesystem.available.${t}`));
    return metrics;
}

function buildTSDBQuery(url, metrics, start, end){
    let base = `${url}/api/query?`,
        params = [];
    params.push(`start=${start.getTime()}`);
    params.push(`end=${end.getTime()}`);
    params = params.concat(metrics.map(m => `m=sum:${m}`));
    return base + params.join("&");
}

function fetchDiskUsage(config){
    let {url, tenantIDs, start, end} = config;
    let metrics = getDiskUsageMetrics(tenantIDs),
        tsdbURL = buildTSDBQuery(url, metrics, start, end);

    // load up mock data
    //tsdbURL = "available.json";

    return get(tsdbURL)
        .catch(e => {
            console.error("probalo fetching data", e);
        });
}

// given data from tsdb, reformat it
// for our purposes
function reformatData(data){
    return new Promise((resolve, reject) => {
        let actual = data.map((m, i) => {
            // transform data to an array of [timestamp, value],
            // and sort by timestamp
            let data = Object.keys(m.dps)
                .sort((a,b) => (+a) - (+b))
                .map(ts => [ts*1000, m.dps[ts]])
                // NOTE - this is about 7 days worth
                .slice(0,50000);

            return {
                metric: m.metric,
                color: COLORS[i][0],
                color2: COLORS[i][1],
                tags: m.tags,
                data: data
            };
        });
        resolve(actual);
    });
}

// give actual series data, generate prediction
// series by apply a prediction algorithm
function generatePredictions(data){
    let calcPromises = data.map((m, i) => {
        return new Promise((resolve, reject) => {
            let worker = new Worker("workers/predict.js");
            worker.postMessage({ data: m.data, theWindow: state.model.theWindow, lookAhead: state.model.lookAhead });
            worker.onmessage = function(e){
                resolve({
                    metric: m.metric,
                    derived: true,
                    color: COLORS[i][1],
                    tags: m.tags,
                    data: e.data
                });
                worker.terminate();
            };
            worker.onerror = e => {
                console.error("predict worker error", e);
                reject(e);
                worker.terminate();
            };
        });
    });
    return Promise.all(calcPromises);
}

// given series data, determine when and where
// shutdowns occur
function generateShutdowns(data){
    let shutdownPromises = data.map(s => {
        return new Promise((resolve, reject) => {
            let worker = new Worker("workers/detectshutdowns.js");
            worker.postMessage({ dpsByTime: s.data, metric: s.metric, margin: state.model.margin });
            worker.onmessage = function(e){
                let shutups = e.data.map(d => {
                    return {metric: s.metric, dp: d};
                });
                resolve(shutups);
                worker.terminate();
            };
            worker.onerror = e => {
                console.error("shutdown worker error", e);
                reject(e);
                worker.terminate();
            };
        });
    });

    return Promise.all(shutdownPromises);
}

function updatePredictions(actual){
    console.time("generate predictions");
    return generatePredictions(state.actual)
        .then(results => {
            console.timeEnd("generate predictions");
            console.time("generate shutdowns");
            state.calculated = results;
            return generateShutdowns(results);
        })
        .then(results => {
            console.timeEnd("generate shutdowns");
            updateShutdowns(results);
            console.log("DONE");
            console.log("");
        });
}

// NOTE - as a side effect this updates the graph. probably
// shouldnt rely on that 
function updateShutdowns(shutdowns){
    // flatten and sort list of shutdowns
    shutdowns = shutdowns
        .reduce((arr,s) => arr.concat(s), [])
        .sort((a,b) => a.dp[0] - b.dp[0]);

    let shutdownsEl = document.getElementById("shutdowns");
    buildShutdownList(shutdowns, shutdownsEl, {
        mouseenter: (metric, ts) => state.graph.focusShutdown(metric, ts),
        mouseleave: (metric, ts) => state.graph.focusShutdown(null)
    });

    // estimate total datapoint count and try to keep
    // it under TARGET_DP_COUNT
    let totalDPs = state.actual[0].data.length * state.actual.length * 2;
    let nth = Math.floor(totalDPs / TARGET_DP_COUNT);

    console.time(`reduce series to every ${nth}th`);
    let series = state.actual
        .concat(state.calculated)
        .map(s => {
            let copy = Object.assign({}, s);
            // return every nth dp
            copy.data = copy.data.filter((d,i) => i % nth ? false : true);
            return copy;
        });
    console.timeEnd(`reduce series to every ${nth}th`);
    // NOTE - assumes graph exists
    state.graph.update(series, shutdowns, state.model.margin);
    state.shutdowns = shutdowns;
}

function updateLegend(actual){
    // update the legend
    let legendEl = document.querySelector(".legend");
    buildLegend(actual, legendEl, {
        mouseenter: name => state.graph.focusSeries(name),
        mouseleave: name => state.graph.focusSeries(null)
    });
}

function fetchAllTheThings(){
    console.time("fetch data");
    return fetchDiskUsage(state.tsdb)
        .then(results => {
            console.timeEnd("fetch data");
            console.time("reformat data");
            return reformatData(results);
        })
        .then(results => {
            console.timeEnd("reformat data");
            console.time("generate predictions");
            state.actual = results;
            return generatePredictions(results);
        })
        .then(results => {
            console.timeEnd("generate predictions");
            console.time("generate shutdowns");
            state.calculated = results;
            return generateShutdowns(results);
        })
        .then(results => {
            console.timeEnd("generate shutdowns");
            updateShutdowns(results);
            updateLegend(state.actual);
        });
}

function init(data){

    let tsdbControlsEl = document.querySelector(".tsdb_controls");
    buildControls([
        {
            type: "textInput",
            label: "URL",
            val: state.tsdb.url,
            events: {
                change: newVal => {
                    state.tsdb.url = newVal;
                }
            }
        },{
            type: "textInput",
            label: "Tenant IDs",
            val: state.tsdb.tenantIDs.join(","),
            events: {
                change: newVal => {
                    state.tsdb.tenantIDs = newVal.split(",");
                }
            }
        },{
            type: "textInput",
            label: "Start",
            val: state.tsdb.start.toString(),
            events: {
                change: newVal => {
                    state.tsdb.start = new Date(newVal);
                }
            }
        },{
            type: "textInput",
            label: "End",
            val: state.tsdb.end.toString(),
            events: {
                change: newVal => {
                    state.tsdb.end = new Date(newVal);
                }
            }
        },{
            type: "button",
            label: "Query",
            style: "float: right; margin: 10px 20px;",
            events: {
                click: () => {
                    console.log("Fetchin!");
                    fetchAllTheThings(state.tsdb)
                        .catch(err => console.error(err));
                }
            }
        }
    ], tsdbControlsEl);

    let shutdownControlsEl = document.querySelector(".shutdown_controls");
    buildControls([
        {
            type: "slider",
            label: "Margin",
            min: 0,
            max: state.model.margin * 2,
            step: 1,
            val: state.model.margin,
            units: "GB",
            toUnit: val => Math.floor(val / 1000000000),
            events: {
                input: newVal => {
                    state.model.margin = +newVal;
                    state.graph.update(null, null, state.model.margin);
                },
                change: newVal => {
                    state.model.margin = +newVal;
                    console.time("generate shutdowns");
                    generateShutdowns(state.calculated)
                        .then(results => {
                            console.timeEnd("generate shutdowns");
                            updateShutdowns(results);
                        });
                }
            }
        },{
            type: "slider",
            label: "Lookahead",
            min: 0,
            max: state.model.lookAhead * 2,
            step: 1,
            val: state.model.lookAhead,
            units: "Minutes",
            toUnit: val => Math.floor(val / 1000 / 60),
            events: {
                input: debounce(newVal => {
                    if(CALCULATE_ON_INPUT){
                        state.model.lookAhead = +newVal;
                        updatePredictions(state.actual)
                            .catch(err => console.error(err));
                    }
                }, PREDICTION_DEBOUNCE),
                change: newVal => {
                    if(!CALCULATE_ON_INPUT){
                        state.model.lookAhead = +newVal;
                        updatePredictions(state.actual)
                            .catch(err => console.error(err));
                    }
                }
            }
        },{
            type: "slider",
            label: "Window",
            min: 0,
            max: state.model.theWindow * 2,
            step: 1,
            val: state.model.theWindow,
            units: "Seconds",
            toUnit: val => Math.floor(val / 1000),
            events: {
                input: debounce(newVal => {
                    if(CALCULATE_ON_INPUT){
                        state.model.theWindow = +newVal;
                        updatePredictions(state.actual)
                            .catch(err => console.error(err));
                    }
                }, PREDICTION_DEBOUNCE),
                change: newVal => {
                    if(!CALCULATE_ON_INPUT){
                        state.model.theWindow = +newVal;
                        updatePredictions(state.actual)
                            .catch(err => console.error(err));
                    }
                }
            }
        }
    
    ], shutdownControlsEl);

    // create graph
    state.graph = buildGraph(document.getElementById("graph"));

    fetchAllTheThings()
        .catch(err => console.error(err));
}

init();
