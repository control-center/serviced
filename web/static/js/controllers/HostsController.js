/* HostsController
 * Displays details for a specific host
 */
(function(){
    "use strict";

    controlplane.controller("HostsController", ["$scope", "$routeParams", "$location", "$filter", "resourcesFactory", "authService", "$modalService", "$interval", "$translate", "$notification", "miscUtils",
    function($scope, $routeParams, $location, $filter, resourcesFactory, authService, $modalService, $interval, $translate, $notification, utils){
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

                                $scope.add_host()
                                    .success(function(data, status){
                                        $notification.create("", data.Detail).success();
                                        this.close();
                                        $scope.newHost = {};
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
        
        $scope.add_host = function() {
            return resourcesFactory.add_host($scope.newHost)
            .success(function(data) {
                // After adding, refresh our list
                utils.refreshHosts($scope, resourcesFactory, false, hostCallback);
                
                // Reset for another add
                $scope.newHost = {
                    poolID: $scope.params.poolID
                };
            });
        };
        
        $scope.remove_host = function(hostId) {
            $modalService.create({
                template: $translate.instant("confirm_remove_host") + " <strong>"+ $scope.hosts.mapped[hostId].Name +"</strong>",
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
                                    utils.refreshHosts($scope, resourcesFactory, false, hostCallback);
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
            $location.path('/hosts/' + hostId);
        };

        $scope.clickPool = function(poolID) {
            $location.path('/pools/' + poolID);
        };

        $scope.dropped = [];

        $scope.filterHosts = function() {
            if (!$scope.hosts.filtered) {
                $scope.hosts.filtered = [];
            }
            // Run ordering filter, built in
            var ordered = $filter('orderBy')($scope.hosts.all, $scope.hosts.sort);
            // Run search filter, built in
            var filtered = $filter('filter')(ordered, $scope.hosts.search);
            // Run filter for pool and child pools, custom
            var treeFiltered = $filter('treeFilter')(filtered, 'poolID', $scope.subPools);

            if(treeFiltered){
                $scope.hosts.filtered = treeFiltered;
            }

            return $scope.hosts.filtered;
        };

        // Build metadata for displaying a list of hosts
        $scope.hosts = utils.buildTable('Name', [
            { id: 'Name', name: 'Name'},
            { id: 'fullPath', name: 'Assigned Resource Pool'},
        ]);

        $scope.$on("$destroy", function(){
            resourcesFactory.unregisterAllPolls();
        });

        var hostCallback = function() {
            $scope.filterHosts();
            updateActiveHosts();
        };

        function updateActiveHosts() {
            if ($scope.hosts) {
                resourcesFactory.get_running_hosts(function(data){
                    for (var i in $scope.hosts.filtered) {
                        var host = $scope.hosts.filtered[i];
                        host.active = 'no';
                        for (var j in data) {
                            if (data[j] === host.ID) {
                                host.active = 'yes';
                            }
                        }
                    }
                });
            }
        }

        resourcesFactory.registerPoll("activeHosts", updateActiveHosts, 3000);

        // Ensure we have a list of pools
        utils.refreshPools($scope, resourcesFactory, false);

        // Also ensure we have a list of hosts
        utils.refreshHosts($scope, resourcesFactory, false, hostCallback);
    }]);
})();
