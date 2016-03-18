/* global controlplane: true */

/* PoolsControl
 * Displays list of pools
 */
(function() {
    'use strict';

    controlplane.controller("PoolsController", ["$scope", "$routeParams", "$location", "$filter", "$timeout", "resourcesFactory", "authService", "$modalService", "$translate", "$notification", "miscUtils", "poolsFactory",
    function($scope, $routeParams, $location, $filter, $timeout, resourcesFactory, authService, $modalService, $translate, $notification, utils, poolsFactory){
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.click_pool = function(id) {
            resourcesFactory.routeToPool(id);
        };

        // Function to remove a pool
        $scope.clickRemovePool = function(poolID) {
            if ($scope.isDefaultPool(poolID)) {
              return;
            }
            $modalService.create({
                template: $translate.instant("confirm_remove_pool") + "<strong>"+ poolID +"</strong>",
                model: $scope,
                title: "remove_pool",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "remove_pool",
                        classes: "btn-danger",
                        action: function(){
                            resourcesFactory.removePool(poolID)
                                .success(function(data) {
                                    $notification.create("Removed Pool", poolID).success();
                                    poolsFactory.update();
                                })
                                .error(data => {
                                    $notification.create("Remove Pool failed", data.Detail).error();
                                });

                            this.close();
                        }
                    }
                ]
            });
        };

        // Function for opening add pool modal
        $scope.modalAddPool = function() {
            $scope.newPool = {};
            $modalService.create({
                templateUrl: "add-pool.html",
                model: $scope,
                title: "add_pool",
                actions: [
                    {
                        role: "cancel",
                        action: function(){
                            $scope.newPool = {};
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: "add_pool",
                        action: function(){
                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                $scope.add_pool()
                                    .success(function(data, status){
                                        $notification.create("Added new Pool", data.Detail).success();
                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Adding pool failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ]
            });
        };

        // Function for adding new pools - through modal
        $scope.add_pool = function() {
            return resourcesFactory.addPool($scope.newPool)
                .success(function(data){
                    poolsFactory.update();
                    // Reset for another add
                    $scope.newPool = {};
                });
        };

        $scope.isDefaultPool = function(poolID) {
          return poolID === "default";
        };

        function init(){
            $scope.name = "pools";
            $scope.params = $routeParams;
            $scope.newPool = {};

            $scope.breadcrumbs = [
                { label: 'breadcrumb_pools', itemClass: 'active' }
            ];

            // start polling
            poolsFactory.activate();

            $scope.pools = {};
            poolsFactory.update()
                .then(() => {
                    $scope.pools = poolsFactory.poolMap;
                });

            $scope.poolsTable = {
                sorting: {
                    id: "asc"
                },
                watchExpression: function(){
                    // if poolsFactory updates, update view
                    return poolsFactory.lastUpdate;
                }
            };
        }

        // kick off controller
        init();

        $scope.$on("$destroy", function(){
            poolsFactory.deactivate();
        });

    }]);
})();
