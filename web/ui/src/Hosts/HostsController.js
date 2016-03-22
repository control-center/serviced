/* HostsController
 * Displays details for a specific host
 */
(function(){
    "use strict";

    controlplane.controller("HostsController", ["$scope", "$routeParams", "$location", "$filter", "resourcesFactory", "authService", "$modalService", "$interval", "$translate", "$notification", "miscUtils", "hostsFactory", "poolsFactory", "servicesFactory",
    function($scope, $routeParams, $location, $filter, resourcesFactory, authService, $modalService, $interval, $translate, $notification, utils, hostsFactory, poolsFactory, servicesFactory){
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.indent = utils.indentClass;

        $scope.resetNewHost = function(){
            $scope.newHost = {
                port: $translate.instant('placeholder_port')
            };
            if ($scope.pools.length > 0){
                $scope.newHost.PoolID = $scope.pools[0].id;
            }
        };

        $scope.modalAddHost = function() {
            $modalService.create({
                templateUrl: "add-host.html",
                model: $scope,
                title: "add_host",
                actions: [
                    {
                        role: "cancel",
                        action: function(){
                            $scope.resetNewHost();
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: "add_host",
                        action: function(){
                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();
                                if ($scope.newHost.RAMLimit === undefined || $scope.newHost.RAMLimit === '') {
                                    $scope.newHost.RAMLimit = "100%";
                                }

                                $scope.newHost.IPAddr = $scope.newHost.host + ':' + $scope.newHost.port;

                                resourcesFactory.addHost($scope.newHost)
                                    .success(function(data, status){
                                        $notification.create("", data.Detail).success();
                                        this.close();
                                        update();
                                    }.bind(this))
                                    .error(function(data, status){
                                        // TODO - form error highlighting
                                        this.createNotification("", data.Detail).error();
                                        // reenable button
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                validate: function(){
                    var err = utils.validateHostName($scope.newHost.host, $translate) ||
                        utils.validatePortNumber($scope.newHost.port, $translate) ||
                        utils.validateRAMLimit($scope.newHost.RAMLimit);
                    if(err){
                        this.createNotification("Error", err).error();
                        return false;
                    }
                    return true;
                }
            });
        };

        $scope.remove_host = function(hostId) {
            $modalService.create({
                template: $translate.instant("confirm_remove_host") + " <strong>"+ hostsFactory.get(hostId).name +"</strong>",
                model: $scope,
                title: "remove_host",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "remove_host",
                        classes: "btn-danger",
                        action: function(){

                            resourcesFactory.removeHost(hostId)
                                .success(function(data, status) {
                                    $notification.create("Removed host", hostId).success();
                                    // After removing, refresh our list
                                    update();
                                    this.close();
                                }.bind(this))
                                .error(function(data, status){
                                    $notification.create("Removing host failed", data.Detail).error();
                                    this.close();
                                }.bind(this));
                        }
                    }
                ]
            });
        };

        $scope.clickHost = function(hostId) {
            resourcesFactory.routeToHost(hostId);
        };

        $scope.clickPool = function(poolID) {
            resourcesFactory.routeToPool(poolID);
        };

        function update(){
            hostsFactory.update()
                .then(() => {
                    $scope.hosts = hostsFactory.hostList;
                });

            poolsFactory.update()
                .then(() => {
                    $scope.pools = poolsFactory.poolList;
                    $scope.resetNewHost();
                });
        }

        function init(){
            $scope.name = "hosts";
            $scope.params = $routeParams;

            $scope.breadcrumbs = [
                { label: 'breadcrumb_hosts', itemClass: 'active' }
            ];

            $scope.hostsTable = {
                sorting: {
                    name: "asc"
                },
                watchExpression: function(){
                    return hostsFactory.lastUpdate;
                }
            };

            $scope.dropped = [];

            // update hosts
            update();

            servicesFactory.activate();
            hostsFactory.activate();
            poolsFactory.activate();
        }

        init();

        $scope.$on("$destroy", function(){
            hostsFactory.deactivate();
            servicesFactory.deactivate();
            poolsFactory.deactivate();
        });
    }]);
})();
