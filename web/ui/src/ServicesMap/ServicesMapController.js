/* globals controlplane: true, dagreD3: true */

/* ServicesMapController
 * Displays dagre graph of services to hosts
 */
(function(){
    "use strict";

    controlplane.controller("ServicesMapController", ["$scope", "$location", "$routeParams", "authService", "resourcesFactory", "servicesFactory", "miscUtils", "hostsFactory", "instancesFactory", "$q",
    function($scope, $location, $routeParams, authService, resourcesFactory, servicesFactory, utils, hostsFactory, instancesFactory, $q) {
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.name = "servicesmap";
        $scope.params = $routeParams;

        $scope.breadcrumbs = [
            { label: 'breadcrumb_deployed', url: '#/apps' },
            { label: 'breadcrumb_services_map', itemClass: 'active' }
        ];

        var g = new dagreD3.graphlib.Graph();
        g.setGraph({
            nodesep: 10,
            ranksep: 75,
            rankdir: "LR"
        });
        var svg = d3.select(".service_map");
        var inner = svg.select("g");
        var render = new dagreD3.render();

        svg.height = $(".service_map").height();

        var draw = function(services, instances) {

            var nodes = [];
            var edges = [];
            var nodeClasses = {};

            // create service nodes and links
            for (var serviceId in services) {
                var service = services[serviceId];

                // if this is an isvc, dont add it
                if(service.isIsvc()){
                    continue;
                }

                // add this service to the list of service nodes
                nodes.push({
                    id: service.id,
                    config: {
                        label: service.name,
                        class: "service",
                        paddingTop: 6, paddingBottom: 6,
                        paddingLeft: 8, paddingRight: 8
                    }
                });

                // if this service has a parent, add it to the
                // list of edges
                if (service.model.ParentServiceID !== '') {
                    // if this service has a parent, mark its
                    // parent as meta
                    nodeClasses[service.model.ParentServiceID] = 'service meta';

                    // link this service to its parent
                    edges.push({
                        source: service.model.ParentServiceID,
                        target: serviceId,
                        config: {
                            lineInterpolate: "basis"
                        }
                    });
                }

            }

            // link services to hosts
            for (var i=0; i < instances.length; i++) {
                var running = instances[i];
                // if this running service has a HostID
                if (running.model.HostID) {
                    // if this host isnt in the list of hosts
                    if (!nodeClasses[running.model.HostID]) {

                        // add the host the the graph
                        nodes.push({
                            id: running.model.HostID,
                            config: {
                                label: hostsFactory.get(running.model.HostID).name,
                                class: "host",
                                paddingTop: 6, paddingBottom: 6,
                                paddingLeft: 8, paddingRight: 8,
                                // round corners to distinguish
                                // from services
                                rx: 10,
                                ry: 10
                            }
                        });

                        // mark this node as a host
                        nodeClasses[running.model.HostID] = 'host';
                    }

                    // mark running service
                    nodeClasses[running.model.ServiceID] = 'service running '+ running.status.status;

                    // create a link from this service to the host
                    // link this service to its parent
                    edges.push({
                        source: running.model.ServiceID,
                        target: running.model.HostID,
                        config: {
                            lineInterpolate: "basis"
                        }
                    });
                }
            }

            // attach all the cool stuff we just made
            // to the graph
            edges.forEach(function(edge){
                g.setEdge(edge.source, edge.target, edge.config);
            });
            nodes.forEach(function(node){
                if(nodeClasses[node.id]){
                    node.config.class = nodeClasses[node.id];
                }
                g.setNode(node.id, node.config);
            });

            render(inner, g);
            $(".service_map_loader").fadeOut(150);

            // Add zoom behavior
            var svg = d3.select(".service_map");
            svg.call(d3.behavior.zoom().on("zoom", function() {
                var ev = d3.event;
                svg.select("g")
                    .attr("transform", "translate(" + ev.translate + ") scale(" + ev.scale + ")");
            }));
        };

        console.log("Fetching services, instances, and hosts");
        // TODO - loading indicator
        $q.all([hostsFactory.update(), servicesFactory.update(), instancesFactory.update()]).then(function(){
            draw(servicesFactory.serviceMap, instancesFactory.instanceArr);
        });
    }]);
})();
