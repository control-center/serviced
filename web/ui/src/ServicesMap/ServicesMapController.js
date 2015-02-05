/* globals controlplane: true, dagreD3: true */

/* ServicesMapController
 * Displays dagre graph of services to hosts
 */
(function(){
    "use strict";

    controlplane.controller("ServicesMapController", ["$scope", "$location", "$routeParams", "authService", "resourcesFactory", "servicesFactory", "miscUtils", "hostsFactory",
    function($scope, $location, $routeParams, authService, resourcesFactory, servicesFactory, utils, hostsFactory) {
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
                    id: service.id,
                    value: { label: service.name}
                };

                if(!nodeClasses[service.id]){
                    nodeClasses[service.id] = 'service notrunning';
                }

                if (service.service.ParentServiceID !== '') {
                    nodeClasses[service.service.ParentServiceID] = 'service meta';
                    edges[edges.length] = {
                        u: service.service.ParentServiceID,
                        v: key
                    };
                }
            }

            var addedHosts = {};

            for (var i=0; i < runningServices.length; i++) {
                var running = runningServices[i];
                if (running.HostID) {
                    if (!addedHosts[running.HostID]) {
                        states[states.length] = {
                            id: running.HostID,
                            value: { label: hostsFactory.hostMap[running.HostID].name }
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
            }

            var layout = dagreD3.layout().nodeSep(5).rankSep(100).rankDir("LR");
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

        // TODO - replace the data_received stuff with promise
        // aggregation
        hostsFactory.update()
            .then(() => {
                data_received.hosts = true;
                draw();
            });
        servicesFactory.update().then(function() {
            data_received.services = true;
            $scope.services = {
                mapped: servicesFactory.serviceMap
            };
            draw();
        });
        resourcesFactory.get_running_services(function(data) {
            data_received.running = true;
            runningServices = data;
            draw();
        });
    }]);
})();
