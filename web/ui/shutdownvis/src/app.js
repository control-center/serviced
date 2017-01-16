import {get, post, debounce} from "./utils";
import {buildControls} from "./controls";
import {buildLegend} from "./legend";
import {buildShutdownList} from "./shutdownlist";
import {buildGraph} from "./graph";

// colors come in pairs, one for actual data, 
// the other for calculated data
const COLORS = [
    ["#a6cee3", "#1f78b4"],
    ["#b2df8a", "#33a02c"],
    ["#fb9a99", "#e31a1c"]
];

const SHUTDOWN_DEBOUNCE = 100;
const PREDICTION_DEBOUNCE = 200;

let state = {
    model: {
        margin: 15e10,
        lookAhead: 20 * 60 * 1000,
        theWindow: 5 * 60 * 1000 
    },
    actual: undefined,
    calculated: undefined,
    shutdowns: undefined,
    graph: undefined
};

function fetchDiskUsage(url, metrics, start, end){
    // TODO - use url, start, end to hit tsdb
    // for the data instead of mock data
    return get("available.json")
        .catch(e => {
            console.error("probalo fetching data", e);
        });
}

// given data from tsdb, reformat it
// for our purposes
function reformatData(data){
    return new Promise((resolve, reject) => {
        let worker = new Worker("workers/reformat.js");
        worker.postMessage({ data: data, colors: COLORS });
        worker.onmessage = function(e){
            resolve(e.data);
            worker.terminate();
        };
        worker.onerror = e => {
            console.error("reformat worker error", e);
            reject(e);
            worker.terminate();
        };
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
                console.log("got calculated for", m.metric);
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
    // TOOD - get from control
    let shutdownPromises = data.map(s => {
        return new Promise((resolve, reject) => {
            let worker = new Worker("workers/detectshutdowns.js");
            worker.postMessage({ dpsByTime: s.data, metric: s.metric, margin: state.model.margin });
            worker.onmessage = function(e){
                console.log("got shutdown for", s.metric);
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
    return generatePredictions(state.actual)
        .then(results => {
            state.calculated = results;
            return generateShutdowns(results);
        })
        .then(results => {
            updateShutdowns(results);
            console.log("DONE");
        });
}

// NOTE - as a side effect this updates the graph. probably
// shouldnt rely on that 
function updateShutdowns(shutdowns){
    shutdowns = shutdowns.reduce((arr,s) => arr.concat(s), []);
    let shutdownsEl = document.getElementById("shutdowns");
    buildShutdownList(shutdowns, shutdownsEl);
    let series = state.actual.concat(state.calculated);
    // NOTE - assumes graph exists
    state.graph.update(series, shutdowns, state.model.margin);
    state.shutdowns = shutdowns;
}

function init(data){
    let controlsEl = document.getElementById("controls");
    buildControls([
        {
            label: "Margin",
            min: 0,
            max: state.model.margin * 2,
            step: 1,
            val: state.model.margin,
            units: "GB",
            toUnit: val => Math.floor(val / 1000000000),
            events: {
                input: debounce(newVal => {
                    state.model.margin = +newVal;
                    generateShutdowns(state.calculated)
                        .then(updateShutdowns);
                    console.log("update margin");
                }, SHUTDOWN_DEBOUNCE)
            }
        },{
            label: "Lookahead",
            min: 0,
            max: state.model.lookAhead * 2,
            step: 1,
            val: state.model.lookAhead,
            units: "Minutes",
            toUnit: val => Math.floor(val / 1000 / 60),
            events: {
                input: debounce(newVal => {
                    state.model.lookAhead = +newVal;
                    updatePredictions(state.actual)
                        .catch(err => console.error(err));
                    console.log("update lookahead");
                }, PREDICTION_DEBOUNCE)
            }
        },{
            label: "Window",
            min: 0,
            max: state.model.theWindow * 2,
            step: 1,
            val: state.model.theWindow,
            units: "Seconds",
            toUnit: val => Math.floor(val / 1000),
            events: {
                input: debounce(newVal => {
                    state.model.theWindow = +newVal;
                    updatePredictions(state.actual)
                        .catch(err => console.error(err));
                    console.log("update window");
                }, PREDICTION_DEBOUNCE)
            }
        }
    
    ], controlsEl);

    // create graph
    state.graph = buildGraph(document.getElementById("graph"));

    // sort-globals up above
    fetchDiskUsage()
        .then(reformatData)
        .then(results => {
            state.actual = results;
            return generatePredictions(results);
        })
        .then(results => {
            state.calculated = results;
            return generateShutdowns(results);
        })
        .then(results => {
            updateShutdowns(results);

            let legendEl = document.querySelector(".legend");
            let series = state.actual.concat(state.calculated);
            buildLegend(state.actual, legendEl, {
                mouseenter: name => state.graph.focusSeries(name),
                mouseleave: name => state.graph.focusSeries(null)
            });
        })
        .catch(err => console.error(err));
}

/* TODO
 * - shutdown list hover nav to shutdown in graph
 * - wire opentsdb config
 * - start/end datetime picker
 * - clean up global state object
 */

init();
