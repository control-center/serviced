function HostDetailsControl($scope, $routeParams, $location, resourcesService, authService, $modalService, $translate) {
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

    $scope.resourcesService = resourcesService;

    // Also ensure we have a list of hosts
    refreshHosts($scope, resourcesService, true);

    $scope.running = buildTable('Name', [
        { id: 'Name', name: 'label_service' },
        { id: 'StartedAt', name: 'running_tbl_start' },
        { id: 'View', name: 'running_tbl_actions' }
    ]);

    $scope.ip_addresses = buildTable('Interface', [
        { id: 'Interface', name: 'ip_addresses_interface' },
        { id: 'Ip', name: 'ip_addresses_ip' },
        { id: 'MAC Address', name: 'ip_addresses_mac' }
    ]);

    $scope.viewLog = function(running) {
        $scope.editService = $.extend({}, running);
        resourcesService.get_service_state_logs(running.ServiceID, running.ID, function(log) {
            $scope.editService.log = log.Detail;
            $modalService.create({
                templateUrl: "view-log.html",
                model: $scope,
                title: "title_log",
                bigModal: true,
                actions: [
                    {
                        classes: "btn-default",
                        label: "download",
                        action: function(){
                            createDownloadLinkAndClick($scope.editService.Name + "_" + Date.now() + ".log", $scope.editService.log);
                        },
                        icon: "glyphicon-download"
                    },
                    {
                        role: "cancel",
                        classes: "btn-default",
                        label: "close"
                    }
                ],
                onShow: function(){
                    var textarea = this.$el.find("textarea");
                    textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                }
            });
        });
    };

    $scope.toggleRunning = toggleRunning;

    $scope.click_app = function(instance) {
        $location.path('/services/' + instance.ServiceID);
    };

    $scope.updateHost = function(){
        var modifiedHost = $.extend({}, $scope.hosts.current);
        resourcesService.update_host(modifiedHost.ID, modifiedHost, function() {
            refreshHosts($scope, resourcesService, false);
        });
    };

    refreshRunningForHost($scope, resourcesService, $scope.params.hostId);
    refreshHosts($scope, resourcesService, true, function() {
        if ($scope.hosts.current) {
            $scope.breadcrumbs.push({ label: $scope.hosts.current.Name, itemClass: 'active' });
        }
    });

    // Ensure we have a list of pools
    refreshPools($scope, resourcesService, false);

    resourcesService.get_stats(function(status) {
        if (status == 200) {
            $scope.collectingStats = true;
        } else {
            $scope.collectingStats = false;
        }
    });

    // XXX prevent the graphs from being drawn multiple times
    //     by angular's processing engine
    $scope.drawn = {};


    //index: graph index for div id selection
    //graph: the graph to display
    $scope.viz = function(index, graph) {
        var id = $scope.hosts.current.ID+'-graph-'+index;
        if (!$scope.drawn[id]) {
            if (window.zenoss === undefined) {
                return "Not collecting stats, graphs unavailable";
            } else {
                graph.timezone = jstz.determine().name();
                zenoss.visualization.chart.create(id, graph);
                $scope.drawn[id] = graph;
                $scope.aggregator = graph.datapoints[0].aggregator;
            }
        }
    };

    $scope.options = {
        maxTime: new Date(),
        maxDate: new Date(),
        mask:true
    };
    var now = new Date(),
        end = moment(now),
        start = moment().subtract(1, "hours");
    $scope.timeRange = {
        time_start: start.format("YYYY/MM/DD HH:mm"),
        time_end: end.format("YYYY/MM/DD HH:mm")
    };

    $scope.updateGraphs = function(){
        for(var i in $scope.drawn){
            $scope.updateGraph(i, $scope.drawn[i]);
        }
    };

    $scope.updateGraph = function(id, config){
        config.range.start = moment($scope.timeRange.time_start)._d.getTime();
        config.range.end = moment($scope.timeRange.time_end)._d.getTime();
        zenoss.visualization.chart.update(id, config);
    };

    // TODO - make this more generic to handle updating any
    // graph config propery
    $scope.aggregators = [
        {
            name: "Average",
            val: "avg"
        },{
            name: "Sum",
            val: "sum"
        }
    ];
    $scope.updateGraphsAggregator = function(){
        // iterate each graphDef
        for(var i in $scope.drawn){
            // then iterate each graphDef's datapoints
            $scope.drawn[i].datapoints.forEach(function(dp){
                dp.aggregator = $scope.aggregator;
            });
        }
        $scope.updateGraphs();
    };
}
