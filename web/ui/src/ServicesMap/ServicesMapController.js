/* globals controlplane: true, dagreD3: true */

/* ServicesMapController
 * Displays dagre graph of services to hosts
 */
(function(){
    "use strict";

    controlplane.controller("ServicesMapController", ["$scope", "$location", "$routeParams", "authService", "resourcesFactory", "servicesFactory", "miscUtils", "hostsFactory", "instancesFactory",
    function($scope, $location, $routeParams, authService, resourcesFactory, servicesFactory, utils, hostsFactory, instancesFactory) {
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.name = "servicesmap";
        $scope.params = $routeParams;

        $scope.breadcrumbs = [
            { label: 'breadcrumb_deployed', url: '#/apps' },
            { label: 'breadcrumb_services_map', itemClass: 'active' }
        ];

        var runningServices;
        var data_received = {
            hosts: false,
            running: false,
            services: false
        };
        var nodeClasses = {};

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

                if (service.model.ParentServiceID !== '') {
                    nodeClasses[service.model.ParentServiceID] = 'service meta';
                    edges[edges.length] = {
                        u: service.model.ParentServiceID,
                        v: key
                    };
                }
            }

            var addedHosts = {};

            for (var i=0; i < runningServices.length; i++) {
                var running = runningServices[i];
                if (running.model.HostID) {
                    if (!addedHosts[running.model.HostID]) {
                        states[states.length] = {
                            id: running.model.HostID,
                            value: { label: hostsFactory.get(running.model.HostID).name }
                        };
                        nodeClasses[running.model.HostID] = 'host';
                        addedHosts[running.model.HostID] = true;
                    }
                    nodeClasses[running.model.ServiceID] = 'service';
                    edges[edges.length] = {
                        u: running.model.ServiceID,
                        v: running.model.HostID
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
        instancesFactory.update().then(function(){
            data_received.running = true;
            runningServices = instancesFactory.instanceArr;
            draw();
        });
    }]);
})();
