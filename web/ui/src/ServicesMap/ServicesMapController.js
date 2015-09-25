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
            { label: 'breadcrumb_deployed', url: '/apps' },
            { label: 'breadcrumb_services_map', itemClass: 'active' }
        ];

        // flag if this is the first time the service
        // map has been updated
        var isFirstTime = true;
        $scope.refreshFrequency = 30000;

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

        // Add zoom behavior
        var zoom = d3.behavior.zoom().on("zoom", function() {
            var ev = d3.event;
            inner.attr("transform", "translate(" + ev.translate + ") scale(" + ev.scale + ")");
        });
        svg.call(zoom);

        var draw = function(services, instances, isUpdate) {

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

            if(edges.length && nodes.length){

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

                if(isFirstTime){
                    isFirstTime = false;
                    // Zoom and scale to fit
                    var zoomScale = zoom.scale();
                    var padding = 200;
                    var graphWidth = g.graph().width + padding;
                    var graphHeight = g.graph().height + padding;
                    var width = parseInt(svg.style("width").replace(/px/, ""));
                    var height = parseInt(svg.style("height").replace(/px/, ""));
                    zoomScale = Math.min(width / graphWidth, height / graphHeight);
                    var translate = [
                        (width/2) - ((graphWidth*zoomScale)/2) + (padding*zoomScale/2),
                        (height/2) - ((graphHeight*zoomScale)/2) + (padding*zoomScale/2)
                    ];

                    zoom.translate(translate);
                    zoom.scale(zoomScale);
                    zoom.event(isUpdate ? svg.transition().duration(500) : d3.select("svg"));
                }

                // hide messages
                $(".service_map_loader").fadeOut(150);
            } else {
                // show "no services" message
                $(".service_map_loader.loading").hide();
                $(".service_map_loader.no_services").show();
            }

        };

        $scope.update = function(){
            return $q.all([hostsFactory.update(), servicesFactory.update(true, true), instancesFactory.update()]).then(function(){
                draw(servicesFactory.serviceMap, instancesFactory.instanceArr);
                $scope.lastUpdate = new Date();
            });
        };

        $scope.pollUpdate = function(){
            $scope.update().then(function(){
                setTimeout(function(){
                    $scope.pollUpdate();
                }, $scope.refreshFrequency);
            });
        };

        $scope.pollUpdate();
    }]);
})();
