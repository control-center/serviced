/* globals jstz: true */

/* graphpanel
 * creates graphs from graph configs, and provides
 * controls for displayed range and update frequency
 */

(function() {
    'use strict';

    angular.module('graphPanel', [])
    .directive("graphPanel", ["$interval", "$location",
    function($interval, $location){
        return {
            restrict: "E",
            scope: {
                serviceId: "=",
                graphConfigs: "="
            },
            templateUrl: "/static/partials/graphpanel.html",
            link: function($scope, element){
            
                // configure viz library
                zenoss.visualization.url = $location.protocol() + "://" + $location.host() + ':' + $location.port();
                zenoss.visualization.urlPath = '/metrics/static/performance/query/';
                zenoss.visualization.urlPerformance = '/metrics/api/performance/query/';
                zenoss.visualization.debug = false;

                $scope.graphs = {};
                $scope.showStartEnd = false;
                $scope.showGraphControls = false;
                $scope.refreshInterval = 300000;

                var momentFormat = "MM/DD/YYYY  HH:mm:ss";

                // graph configuration used to generate
                // query service requests
                $scope.graphConfig = {
                    aggregator: "sum",
                    start: zenoss.utils.createDate("1h-ago").format(momentFormat),
                    end: zenoss.utils.createDate("0s-ago").format(momentFormat),
                    range: "1h-ago",
                    now: true
                };

                $scope.viz = function(graph) {
                    var id = $scope.getUniqueGraphId(graph),
                        graphCopy;

                    if (!$scope.graphs[id]) {
                        if (window.zenoss === undefined) {
                            return "Not collecting stats, graphs unavailable";
                        } else {
                            // create a copy of graph so that range changes
                            // do not affect the original service def
                            graphCopy = angular.copy(graph);

                            // set graphs to local browser time
                            graphCopy.timezone = jstz.determine().name();

                            updateGraphRequest(graphCopy);
                            zenoss.visualization.chart.create(id, graphCopy);

                            // store graph def for later use
                            $scope.graphs[id] = graphCopy;
                        }
                    }
                };

                $scope.datetimePickerOptions = {
                    maxDate: new Date(),
                    mask:true,
                    closeOnDateSelect: true,
                    format: "m/d/Y  H:i:s",
                    onChangeDateTime: function(){
                        // let angular finish current digest cycle
                        // before updating the graphs
                        setTimeout(function(){
                            $scope.refreshGraphs();
                        }, 0);
                    }
                };

                // select options for graph aggregation
                $scope.aggregators = [
                    {
                        name: "Average",
                        val: "avg"
                    },{
                        name: "Sum",
                        val: "sum"
                    }
                ];
                // refresh intervals
                $scope.intervals= [
                    {
                        name: "1 Second",
                        val: 1000
                    },{
                        name: "5 Seconds",
                        val: 5000
                    },{
                        name: "1 Minute",
                        val: 60000
                    },{
                        name: "5 Minutes",
                        val: 300000
                    },{
                        name: "15 Minutes",
                        val: 900000
                    },{
                        name: "Never",
                        val: 0
                    }
                ];
                // select options for graph ranges
                var CUSTOM_RANGE = "custom";
                $scope.ranges = [
                    {
                        name: "Last hour",
                        val: "1h-ago"
                    },{
                        name: "Last 4 hours",
                        val: "4h-ago"
                    },{
                        name: "Last 12 hours",
                        val: "12h-ago"
                    },{
                        name: "Last 24 hours",
                        val: "1d-ago"
                    },{
                        name: "Last 48 hours",
                        val: "2d-ago"
                    },{
                        name: "[Custom]",
                        val: CUSTOM_RANGE
                    }
                ];

                // on range select change, update start/end
                // values to reflect the selected range
                $scope.rangeChange = function(){
                    var range = $scope.graphConfig.range;

                    if(range === CUSTOM_RANGE){
                        // show start/end options
                        $scope.showStartEnd = true;
                    } else {
                        // hide start/end opts
                        $scope.showStartEnd = false;

                        // parse graph range into something the date picker likes
                        $scope.graphConfig.start = zenoss.utils.createDate($scope.graphConfig.range).format(momentFormat);
                        
                        // when using a range, always use "now" for the end time
                        $scope.graphConfig.end = zenoss.utils.createDate("0s-ago").format(momentFormat);
                    }

                    $scope.refreshGraphs();
                };


                // on refresh change, update refresh interval
                $scope.setupAutoRefresh = function(){
                    // cancel existing refresh
                    $interval.cancel($scope.refreshPromise);

                    // if refreshInterval is zero, don't setup
                    // a refresh interval
                    if($scope.refreshInterval){
                        // start auto-refresh
                        $scope.refreshPromise = $interval(function(){
                            $scope.refreshGraphs();
                        }, $scope.refreshInterval);
                    }
                };
                
                // kick off inital graph request
                $scope.setupAutoRefresh();

                $scope.refreshGraphs = function(){
                    // don't refresh graph if tab is not visible
                    if(document.hidden){
                        return;
                    }

                    var graph;

                    // iterate and update all graphs
                    for(var i in $scope.graphs){
                        graph = $scope.graphs[i];
                        updateGraphRequest(graph);
                        zenoss.visualization.chart.update(i, graph);
                    }
                };

                $scope.getUniqueGraphId = function(graph){
                    return ($scope.serviceId +"-graph-"+ graph.id).replace(/\./g, "_");
                };

                $scope.cleanup = function(){
                    var chart;

                    // remove graph from cache
                    for(var id in $scope.graphs){
                        // TODO - expose removeChart() and use it
                        chart = zenoss.visualization.chart.getChart(id);
                        chart.onDestroyed();
                    }

                    $scope.graphs = {};
                };

                $scope.graphControlsPopover = function(){
                    $scope.showGraphControls = !$scope.showGraphControls;

                };

                // make clicking anywhere outside of graph
                // control hide it
                var hideGraphControls = function(){
                    $scope.showGraphControls = false;
                    // force angular to apply the visibility change
                    $scope.$apply();
                };
                angular.element("html").on("click", hideGraphControls);

                $scope.$watch("serviceId", $scope.cleanup);

                $scope.$on("$destroy", function(){
                    $scope.cleanup();
                    $interval.cancel($scope.refreshPromise);
                    angular.element("html").off("click", hideGraphControls);
                });

                function updateGraphRequest(graph){
                    // update aggregator
                    graph.datapoints.forEach(function(dp){
                        dp.aggregator = $scope.graphConfig.aggregator;
                    });

                    // if end should always be "now", use current time
                    if($scope.graphConfig.now){
                        $scope.graphConfig.end = zenoss.utils.createDate("0s-ago").format(momentFormat);
                        graph.range.end = zenoss.utils.createDate($scope.graphConfig.end).valueOf();

                    // else, use specified end time
                    } else {
                        graph.range.end = zenoss.utils.createDate($scope.graphConfig.end).valueOf();
                    }

                    // if range, update start time
                    if($scope.graphConfig.range !== CUSTOM_RANGE){
                        $scope.graphConfig.start = zenoss.utils.createDate($scope.graphConfig.range).format(momentFormat);
                    }
                    // update start/end
                    graph.range.start = zenoss.utils.createDate($scope.graphConfig.start).valueOf();
                }
            }
        };
    }]);
})();
