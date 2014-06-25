function ServicesMapControl($scope, $location, $routeParams, authService, resourcesService) {
    // Ensure logged in
    authService.checkLogin($scope);

    $scope.name = "servicesmap";
    $scope.params = $routeParams;

    $scope.breadcrumbs = [
        { label: 'breadcrumb_deployed', url: '#/apps' },
        { label: 'breadcrumb_services_map', itemClass: 'active' }
    ];

    var data_received = {
        hosts: false,
        running: false,
        services: false
    };
    var nodeClasses = {};
    var runningServices = null;

    var draw = function() {
        if (!data_received.hosts) {
            console.log('Waiting for host data');
            return;
        }
        if (!data_received.running) {
            console.log('Waiting for running data');
            return;
        }
        if (!data_received.services) {
            console.log('Waiting for services data');
            return;
        }

        var states = [];
        var edges = [];

        for (var key in $scope.services.mapped) {
            var service = $scope.services.mapped[key];
            states[states.length] = {
                id: service.Id,
                value: { label: service.Name}
            };

            if(!nodeClasses[service.Id]){
                nodeClasses[service.Id] = 'service notrunning';
            }

            if (service.ParentServiceID !== '') {
                var parent = $scope.services.mapped[service.ParentServiceID];
                nodeClasses[service.ParentServiceID] = 'service meta';
                edges[edges.length] = {
                    u: service.ParentServiceID,
                    v: key
                };
            }
        }

        var addedHosts = {};

        for (var i=0; i < runningServices.length; i++) {
            var running = runningServices[i];
            if (!addedHosts[running.HostID]) {
                states[states.length] = {
                    id: running.HostID,
                    value: { label: $scope.hosts.mapped[running.HostID].Name }
                };
                nodeClasses[running.HostID] = 'host';
                addedHosts[running.HostID] = true;
            }
            nodeClasses[running.ServiceID] = 'service';
            edges[edges.length] = {
                u: running.ServiceID,
                v: running.HostID
            };

        }

        var layout = dagreD3.layout().nodeSep(5).rankDir("LR")
        var renderer = new dagreD3.Renderer().layout(layout);
        var oldDrawNode = renderer.drawNode();
        renderer.drawNode(function(graph, u, svg) {
            oldDrawNode(graph, u, svg);
            svg.attr("class", "node " + nodeClasses[u]);
        });

        renderer.run(
            dagreD3.json.decode(states, edges),
            d3.select("svg g"));

        // Add zoom behavior
        var svg = d3.select("svg");
        svg.call(d3.behavior.zoom().on("zoom", function() {
            var ev = d3.event;
            svg.select("g")
                .attr("transform", "translate(" + ev.translate + ") scale(" + ev.scale + ")");
        }));
    };

    /*
     * Each successful resourceServices call will execute draw(),
     * but draw() will do an early return unless all required
     * data is available.
     */

    resourcesService.get_running_services(function(data) {
        data_received.running = true;
        runningServices = data;
        draw();
    });

    refreshHosts($scope, resourcesService, true, function() {
        data_received.hosts = true;
        draw();
    });

    refreshServices($scope, resourcesService, true, function() {
        data_received.services = true;
        draw();
    });
}
