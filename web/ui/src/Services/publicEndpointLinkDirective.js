/* publicEndpointLink
 * directive for displaying the link for a public
 * endpoint.  May be a link, label, with or without
 * hover text.
 */
(function() {
    'use strict';

    angular.module('publicEndpointLink', [])
    .directive("publicEndpointLink", ["$compile", "$location",
        "servicesFactory", "$translate","resourcesFactory",
    function($compile, $location, servicesFactory, $translate, resourcesFactory){
        return {
            restrict: "E",
            scope: {
                publicEndpoint: "=",
                state: "@",
                hostAlias: "=",
                getServiceStatus: "=",
            },
            link: function ($scope, element, attrs){
                let publicEndpoint = $scope.publicEndpoint;
                let getServiceStatus = $scope.getServiceStatus;

                // A method to return the displayed URL for an endpoint.
                var getUrl = function(publicEndpoint){
                    // Form the url based on the endpoint properties.
                    var url = "";
                    if ("ServiceName" in publicEndpoint){
                        var port = $location.port() === "" || +$location.port() === 443 ? "" : ":" + $location.port();
                        var host = publicEndpoint.ServiceName.indexOf('.') === -1 ?
                            publicEndpoint.ServiceName + ".{{hostAlias}}" : publicEndpoint.ServiceName;
                        url = $location.protocol() + "://" + host + port;
                    } else if ("PortAddress" in publicEndpoint){
                        // Port public endpoint
                        var portAddress = publicEndpoint.PortAddress;
                        var protocol = publicEndpoint.Protocol.toLowerCase();
                        if(portAddress.startsWith(":")){
                            portAddress = "{{hostAlias}}" + portAddress;
                        }
                        // Remove the port for standard http/https ports.
                        if(protocol !== "") {
                            var parts = portAddress.split(":");
                            if (protocol === "http" && parts[1] === "80") {
                                portAddress = parts[0];
                            } else if (protocol === "https" && parts[1] === "443") {
                                portAddress = parts[0];
                            }
                            url = protocol + "://" + portAddress;
                        } else {
                            url = portAddress;
                        }                
                    }
                    return url;
                };
                                                                                
                var isServiceRunning = function(id){
                    var service = getServiceStatus(id);
                    // if not found, empty service object returned
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
                if ("ServiceID" in publicEndpoint){
                    // Check the service and endpoint and..
                    if (!isServiceRunning(publicEndpoint.ServiceID) || !publicEndpoint.Enabled) {
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
