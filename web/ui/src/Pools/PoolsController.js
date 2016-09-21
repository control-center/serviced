/* global controlplane: true */

/* PoolsControl
 * Displays list of pools
 */
(function() {
    'use strict';

    // Pool object constructor takes a pool object (backend pool object)
    // and wraps it with extra functionality and info
    function Pool(pool){
        this.id = pool.ID;
        this.model = Object.freeze(pool);
    }

    controlplane.controller("PoolsController", ["$scope", "$routeParams",
        "resourcesFactory", "authService", "$modalService", "$translate",
        "$notification", "areUIReady", "$interval", "servicedConfig", "log",
    function($scope, $routeParams, resourcesFactory, authService, $modalService,
             $translate, $notification, areUIReady, $interval, servicedConfig, log){

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
                                    updatePools();
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
            areUIReady.lock();
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

                                resourcesFactory.addPool($scope.newPool)
                                    .success(function(data, status){
                                        $notification.create("Added new Pool", data.Detail).success();
                                        updatePools();

                                        // Reset for another add
                                        $scope.newPool = {};

                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Adding pool failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                onShow: () => {
                    areUIReady.unlock();
                }
            });
        };

        $scope.isDefaultPool = function(poolID) {
          return poolID === "default";
        };

        // Setup polling to update the pools list if it has changed.

        var lastUpdate;
        var updateFrequency = 3000;
        var updatePromise;

        servicedConfig.getConfig()
            .then(config => {
                updateFrequency = config.PollFrequency * 1000;
            }).catch(err => {
                let errMessage = err.data ? err.data.Detail : err.statusText;
                log.error("could not load serviced config:", errMessage);
            });

        function updatePools(){
            resourcesFactory.getV2Pools()
                .success(data => {
                    $scope.pools = data.map(result => new Pool(result));
                    $scope.totalPoolCount = data.length;
                })
                .error(data => {
                    $notification.create("Unable to load pools.", data.Detail).error();
                })
                .finally(() => {
                    // notify the first request is complete
                    if (!lastUpdate) {
                        $scope.$emit("ready");
                    }

                    lastUpdate = new Date().getTime();
                });
        }

        function startPolling(){
            if(!updatePromise){
                updatePromise = $interval(() => updatePools(), updateFrequency);
            }
        }

        function stopPolling(){
            if(updatePromise){
                $interval.cancel(updatePromise);
                updatePromise = null;
            }
        }

        function init(){
            $scope.name = "pools";
            $scope.params = $routeParams;
            $scope.newPool = {};

            $scope.breadcrumbs = [
                { label: 'breadcrumb_pools', itemClass: 'active' }
            ];

            startPolling();

            updatePools();

            $scope.poolsTable = {
                sorting: {
                    id: "asc"
                },
                watchExpression: function(){
                    return lastUpdate;
                }
            };
        }

        init();

        $scope.$on("$destroy", function(){
            stopPolling();
        });

    }]);


})();
