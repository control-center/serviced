/* global controlplane: true */

/* PoolsControl
 * Displays list of pools
 */
(function() {
    'use strict';

    let POOL_PERMISSIONS;

    // Pool object constructor takes a pool object (backend pool object)
    // and wraps it with extra functionality and info
    class Pool{
    
        constructor(pool){
            this.id = pool.ID;
            this.model = Object.freeze(pool);
            this.calculatePermissions();
        }

        calculatePermissions(){
            // build a list of permissions
            // this pool has
            // NOTE: permissions include description
            // and friendly label for the UI to display
            let val = this.model.Permissions;
            this.permissions = [];
            POOL_PERMISSIONS.forEach(perm => {
                if((val & perm.position) !== 0){
                    this.permissions.push(perm);
                }
            });
        }
    }

    controlplane.controller("PoolsController", ["$scope", "$routeParams",
    "resourcesFactory", "authService", "$modalService", "$translate",
    "$notification", "areUIReady", "$interval", "servicedConfig", "log",
    "POOL_PERMISSIONS", "miscUtils",
    function($scope, $routeParams, resourcesFactory, authService, $modalService,
    $translate, $notification, areUIReady, $interval, servicedConfig, log,
    _POOL_PERMISSIONS, utils){

        // Ensure logged in
        authService.checkLogin($scope);

        // share POOL_PERMISSIONS outside of
        // this function context
        POOL_PERMISSIONS = _POOL_PERMISSIONS;

        // allow templates to get the list
        // of permissions
        $scope.permissions = POOL_PERMISSIONS;

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
            $scope.newPool = {
                permissions: new utils.NgBitset(POOL_PERMISSIONS.length, 3)
            };
            $modalService.create({
                templateUrl: "add-pool.html",
                model: $scope,
                title: "add_pool",
                actions: [
                    {
                        role: "cancel",
                        action: function(){
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: "add_pool",
                        action: function(){
                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                // add the Permissions field and remove the NgBitset field
                                $scope.newPool.Permissions = $scope.newPool.permissions.val;
                                delete $scope.newPool.permissions;

                                resourcesFactory.addPool($scope.newPool)
                                    .success(function(data, status){
                                        $notification.create("Added new Pool", data.Detail).success();
                                        updatePools();
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
