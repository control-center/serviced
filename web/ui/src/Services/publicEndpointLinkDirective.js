/* publicEndpointLink
 * directive for displaying the link for a public
 * endpoint.  May be a link, label, with or without
 * hover text.
 */
(function() {
    'use strict';

    angular.module('publicEndpointLink', [])
    .directive("publicEndpointLink", ["$compile", "$location",
        "servicesFactory", "$translate",
    function($compile, $location, servicesFactory, $translate){
        return {
            restrict: "E",
            scope: {
                publicEndpoint: "=",
                state: "@",
                hostAlias: "=",
            },
            link: function ($scope, element, attrs){
                var publicEndpoint = $scope.publicEndpoint;

                // A method to return the displayed URL for an endpoint.
                var getUrl = function(publicEndpoint){
                    // Form the url based on the endpoint properties.
                    var url = "";
                    if ("Name" in publicEndpoint){
                        var port = $location.port() === "" ? "" : ":" + $location.port();
                        var host = publicEndpoint.Name.indexOf('.') === -1 ?
                            publicEndpoint.Name + ".{{hostAlias}}" : publicEndpoint.Name;
                        url = $location.protocol() + "://" + host + port;
                    } else if ("PortAddr" in publicEndpoint){
                        // Port public endpoint
                        var portAddr = publicEndpoint.PortAddr;
                        var protocol = publicEndpoint.Protocol.toLowerCase();
                        if(portAddr.startsWith(":")){
                            portAddr = "{{hostAlias}}" + portAddr;
                        }
                        // Remove the port for standard http/https ports.
                        if(protocol !== "") {
                            var parts = portAddr.split(":");
                            if (protocol === "http" && parts[1] === "80") {
                                portAddr = parts[0];
                            } else if (protocol === "https" && parts[1] === "443") {
                                portAddr = parts[0];
                            }
                            url = protocol + "://" + portAddr;
                        } else {
                            url = portAddr;
                        }                
                    }
                    return url;
                };
                                                                                
                var isServiceRunning = function(id){
                    var service = servicesFactory.get(id);
                    return service.desiredState === 1;
                };
                
                var addPopover = function(element, translation){
                    // Set the popup with the content data.
                    element.popover({
                        trigger: "hover",
                        placement: "top",
                        delay: 0,
                        content: $translate.instant(translation),
                    });                    
                };
                
                var url = getUrl(publicEndpoint);
                var html = "";
                var popover = false;
                
                // If we have an appid, this is a subservice.
                if ("ApplicationId" in publicEndpoint){
                    // Check the service and endpoint and..
                    if (!isServiceRunning(publicEndpoint.ApplicationId) || !publicEndpoint.Enabled) {
                        // .. show the url as a url label (not clickable) with a bootstrap popover..
                        html = '<span><b>' + url + '</b></span>';
                        popover = true;
                    } else if (publicEndpoint.Protocol !== '') {
                        // ..or show the url as a clickable link.
                        html = '<a target="_blank" class="link" href="' + url + '">' + url + '</a>';
                    } else {
                        // ..or just show the host:port for the port endpoint.
                        html = '<span>' + url + '</span>';
                    }
                } else {
                    // This is a top level application.  Check the state and..
                    if (+$scope.state !== 1 || !publicEndpoint.Enabled){
                        // .. show the url as a label with a bootstrap popover..
                        html = '<span>' + url + '</span>';
                        popover = true;
                    } else {
                        // ..or show the url as a clickable link.
                        html = '<a target="_blank" class="link" href="' + url + '">' + url + '</a>';
                    }
                }
                
                // Compile the element.
                var el =$compile(html)($scope);
                if (popover){
                    addPopover(el, "vhost_unavailable");
                }
                element.replaceWith(el);
            }
        };
    }]);
})();
