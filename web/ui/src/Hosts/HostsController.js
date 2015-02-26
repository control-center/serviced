/* HostsController
 * Displays details for a specific host
 */
(function(){
    "use strict";

    controlplane.controller("HostsController", ["$scope", "$routeParams", "$location", "$filter", "resourcesFactory", "authService", "$modalService", "$interval", "$translate", "$notification", "miscUtils", "hostsFactory", "poolsFactory",
    function($scope, $routeParams, $location, $filter, resourcesFactory, authService, $modalService, $interval, $translate, $notification, utils, hostsFactory, poolsFactory){
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.name = "hosts";
        $scope.params = $routeParams;

        $scope.breadcrumbs = [
            { label: 'breadcrumb_hosts', itemClass: 'active' }
        ];

        $scope.indent = utils.indentClass;
        $scope.newHost = {};

        $scope.modalAddHost = function() {
            $modalService.create({
                templateUrl: "add-host.html",
                model: $scope,
                title: "add_host",
                actions: [
                    {
                        role: "cancel",
                        action: function(){
                            $scope.newHost = {};
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: "add_host",
                        action: function(){
                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                resourcesFactory.add_host($scope.newHost)
                                    .success(function(data, status){
                                        $notification.create("", data.Detail).success();
                                        this.close();
                                        $scope.newHost = {
                                            poolID: $scope.params.poolID
                                        };
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
                ]
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

                            resourcesFactory.remove_host(hostId)
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

        $scope.dropped = [];

        // Build metadata for displaying a list of hosts
        $scope.hosts = utils.buildTable('Name', [
            { id: 'Name', name: 'Name'}
        ]);

        // update hosts
        update();

        hostsFactory.activate();
        poolsFactory.activate();

        $scope.$on("$destroy", function(){
            hostsFactory.deactivate();
        });

        function update(){
            hostsFactory.update()
                .then(() => {
                    $scope.hosts.all = hostsFactory.hostList;
                });

            poolsFactory.update()
                .then(() => {
                    $scope.pools = poolsFactory.poolList;
                });
        }

    }]);
})();
