function HostDetailsControl($scope, $routeParams, $location, resourcesService, authService, statsService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "hostdetails";
    $scope.params = $routeParams;

    $scope.visualization = zenoss.visualization;
    $scope.visualization.url = $location.protocol() + "://" + $location.host() + ':' + $location.port();
    $scope.visualization.urlPath = '/metrics/static/performance/query/';
    $scope.visualization.urlPerformance = '/metrics/api/performance/query/';
    $scope.visualization.debug = false;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_hosts', url: '#/hosts' }
    ];

    // Also ensure we have a list of hosts
    refreshHosts($scope, resourcesService, true);

    $scope.running = buildTable('Name', [
        { id: 'Name', name: 'running_tbl_running' },
        { id: 'StartedAt', name: 'running_tbl_start' },
        { id: 'View', name: 'running_tbl_actions' }
    ]);

    $scope.ip_addresses = buildTable('Interface', [
        { id: 'Interface', name: 'ip_addresses_interface' },
        { id: 'Ip', name: 'ip_addresses_ip' }
    ]);

    $scope.graph = buildTable('Name', [
        { id: 'CPU', name: 'graph_tbl_cpu'},
        { id: 'Memory', name: 'graph_tbl_mem'}
    ]);

    $scope.viewConfig = function(running) {
        $scope.editService = $.extend({}, running);
        $scope.editService.config = 'TODO: Implement';
        $('#editConfig').modal('show');
    };

    $scope.viewLog = function(running) {
        $scope.editService = $.extend({}, running);
        resourcesService.get_service_state_logs(running.ServiceId, running.Id, function(log) {
            $scope.editService.log = log.Detail;
            $('#viewLog').modal('show');
        });
    };

    $scope.click_app = function(instance) {
        $location.path('/services/' + instance.ServiceId);
    };

    $scope.killRunning = function(running) {
        resourcesService.kill_running(running.HostId, running.Id, function() {
            refreshRunningForHost($scope, resourcesService, $scope.params.hostId);
        });
    };

    refreshRunningForHost($scope, resourcesService, $scope.params.hostId);
    refreshHosts($scope, resourcesService, true, function() {
        if ($scope.hosts.current) {
            $scope.breadcrumbs.push({ label: $scope.hosts.current.Name, itemClass: 'active' });
        }
    });

    statsService.is_collecting(function(status) {
        if (status == 200) {
            $scope.collectingStats = true;
        } else {
            $scope.collectingStats = false;
        }
    });

    $scope.cpuconfig = {
        "datapoints": [
            {
                "aggregator": "avg",
                "color": "#aec7e8",
                "expression": null,
                "fill": false,
                "format": "%6.2f",
                "id": "system",
                "legend": "CPU (System)",
                "metric": "CpuacctStat.system",
                "name": "CPU (System)",
                "rate": true,
                "rateOptions": {},
                "type": "line"
            },
            {
                "aggregator": "avg",
                "color": "#98df8a",
                "expression": null,
                "fill": false,
                "format": "%6.2f",
                "id": "user",
                "legend": "CPU (User)",
                "metric": "CpuacctStat.user",
                "name": "CPU (User)",
                "rate": true,
                "rateOptions": {},
                "type": "line"
            }
        ],
        "footer": false,
        "format": "%6.2f",
        "maxy": null,
        "miny": 0,
        "range": {
            "end": "0s-ago",
            "start": "1h-ago"
        },
        "returnset": "EXACT",
        "tags": {},
        "type": "line",
        "downsample": "1m-avg"
    };

    $scope.ofdconfig = {
        "datapoints": [
            {
                "aggregator": "avg",
                "color": "#aec7e8",
                "expression": null,
                "fill": false,
                "format": "%6.2f",
                "id": "ofd",
                "legend": "Serviced Open File Descriptors",
                "metric": "Serviced.OpenFileDescriptors",
                "name": "Serviced Open File Descriptors",
                "rate": false,
                "rateOptions": {},
                "type": "line"
            },
        ],
        "footer": false,
        "format": "%d",
        "maxy": null,
        "miny": 0,
        "range": {
            "end": "0s-ago",
            "start": "1h-ago"
        },
        "returnset": "EXACT",
        "tags": {},
        "type": "line",
        "downsample": "1m-avg"
    };

    $scope.memconfig = {
        "datapoints": [
            {
                "aggregator": "avg",
                "color": "#aec7e8",
                "expression": null,
                "expression": null,
                "fill": false,
                "format": "%d",
                "id": "pgfault",
                "legend": "Page Faults",
                "metric": "MemoryStat.pgfault",
                "name": "Page Faults",
                "rate": true,
                "rateOptions": {},
                "type": "line"
            }
        ],
        "footer": false,
        "format": "%6.2f",
        "maxy": null,
        "miny": 0,
        "range": {
            "end": "0s-ago",
            "start": "1h-ago"
        },
        "returnset": "EXACT",
        "tags": {},
        "type": "line",
        "downsample": "1m-avg"
    };

    $scope.rssconfig = {
        "datapoints": [
            {
                "aggregator": "avg",
                "expression": "rpn:1024,/,1024,/",
                "fill": false,
                "format": "%6.2f",
                "id": "rssmemory",
                "legend": "RSS Memory",
                "metric": "MemoryStat.rss",
                "name": "RSS Memory",
                "rateOptions": {},
                "type": "line",
                "fill": true
            }
        ],
        "footer": false,
        "format": "%6.2f",
        "maxy": null,
        "miny": 0,
        "range": {
            "end": "0s-ago",
            "start": "1h-ago"
        },
        "yAxisLabel": "MB",
        "returnset": "EXACT",
        height: 300,
        width: 300,
        "tags": {},
        "type": "line",
        "downsample": "1m-avg"
    };

    $scope.drawn = {};

    $scope.viz = function(id, config) {
        if (!$scope.drawn[id]) {
            if (window.zenoss === undefined) {
                return "Not collecting stats, graphs unavailable";
            } else {
                zenoss.visualization.chart.create(id, config);
                $scope.drawn[id] = true;
            }
        }
    };
}
