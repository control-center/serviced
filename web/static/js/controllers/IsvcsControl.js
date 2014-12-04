function IsvcsControl($scope, $routeParams, $location, resourcesService, authService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "isvcs";
    $scope.params = $routeParams;

    $scope.visualization = zenoss.visualization;
    $scope.visualization.url = $location.protocol() + "://" + $location.host() + ':' + $location.port();
    $scope.visualization.urlPath = '/metrics/static/performance/query/';
    $scope.visualization.urlPerformance = '/metrics/api/performance/query/';
    $scope.visualization.debug = false;

    $scope.breadcrumbs = [{
        label: 'Internal Services',
        url: '#/isvcs'
    }];


    $scope.getCPUGraph = function(isvcname) {
        return {
            "datapoints": [{
                "aggregator": "avg",
                "color": "#aec7e8",
                "fill": false,
                "format": "%4.2f",
                "id": "system",
                "legend": "CPU (System)",
                "metric": "cgroup.cpuacct.system",
                "name": "CPU (System)",
                "rate": true,
                "rateOptions": {},
                "type": "line"
            }, {
                "aggregator": "avg",
                "color": "#98df8a",
                "fill": false,
                "format": "%4.2f",
                "id": "user",
                "legend": "CPU (User)",
                "metric": "cgroup.cpuacct.user",
                "name": "CPU (User)",
                "rate": true,
                "rateOptions": {},
                "type": "line"
            }],
            "footer": false,
            "format": "%4.2f",
            "maxy": 100,
            "miny": 0,
            "range": {
                "end": "0s-ago",
                "start": "1h-ago"
            },
            "yAxisLabel": "% Used",
            "returnset": "EXACT",
            "tags": {
                "isvcname": [isvcname]
            },
            "type": "line",
            "timezone": jstz.determine().name()
        };
    };

    $scope.getRSSGraph = function(isvcname) {
        return {
            "datapoints": [{
                "aggregator": "avg",
                "fill": false,
                "format": "%4.2f",
                "id": "rssmemory",
                "legend": "Memory Usage",
                "metric": "cgroup.memory.totalrss",
                "name": "Memory Usage",
                "rateOptions": {},
                "type": "line",
            }],
            "footer": false,
            "format": "%4.2f",
            "maxy": null,
            "miny": 0,
            "range": {
                "end": "0s-ago",
                "start": "1h-ago"
            },
            "yAxisLabel": "bytes",
            "returnset": "EXACT",
            height: 300,
            width: 300,
            "tags": {
                "isvcname": [isvcname]
            },
            "type": "line",
            "timezone": jstz.determine().name()
        };
    };


    // XXX prevent the graphs from being drawn multiple times
    //     by angular's processing engine
    $scope.drawn = {};

    //index: graph index for div id selection
    //graph: the graph to display
    $scope.viz = function(id, graph) {
        if (!$scope.drawn[id]) {
            if (window.zenoss === undefined) {
                return "Not collecting stats, graphs unavailable";
            } else {
                graph.timezone = jstz.determine().name();
                console.log(id, graph);
                zenoss.visualization.chart.create(id, graph);
                $scope.drawn[id] = true;
            }
        }
    };
}
