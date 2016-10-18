/* global controlplane: true */

/* PoolsControl
 * Displays list of pools
 */
(function() {
    'use strict';

    class PoolsController {

        constructor ($scope, resourcesFactory, authService, $modalService, 
                     $translate, $notification, areUIReady, $interval, servicedConfig, 
                     log, POOL_PERMISSIONS, miscUtils, Pool, $q)
        {
            authService.checkLogin(this);

            this.resourcesFactory = resourcesFactory;
            this.$modalService = $modalService;
            this.$translate = $translate;
            this.$notification = $notification;
            this.areUIReady = areUIReady;
            this.$interval = $interval;
            this.utils = miscUtils;
            this.$q = $q;
            this.permissions = POOL_PERMISSIONS;
            this.name = "pools";

            this.refreshPools().then(() => $scope.$emit("ready"));

            this.updateFrequency = 3000;
            servicedConfig.getConfig()
                .then(config => {
                    this.updateFrequency = config.PollFrequency * 1000;
                }).catch(err => {
                    let errMessage = err.data ? err.data.Detail : err.statusText;
                    log.error("could not load serviced config:", errMessage);
                });

            $scope.poolsTable = {
                sorting: { id: "asc" },
                watchExpression: () => this.lastUpdate
            };

            $scope.breadcrumbs = [
                { label: 'breadcrumb_pools', itemClass: 'active' }
            ];

            this.startPolling();

            $scope.$on("$destroy", () => this.stopPolling());

            this.newScope = () => $scope.$new(true);
            this.newPool = (data) => new Pool(data);
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        isDefaultPool(id) {
            return id  === "default";
        }

        refreshPools() {
            let deferred = this.$q.defer();
            this.resourcesFactory.v2.getPools()
                .success(data => {
                    this.pools = data.map(result => this.newPool(result));
                    this.totalPoolCount = data.length;
                    this.touch();
                    deferred.resolve();
                })
                .error(data => {
                    this.$notification.create("Unable to load pools.", data.Detail).error();
                    deferred.reject();
                });
            return deferred.promise;
        }

        startPolling() {
            if (!this.updatePromise) {
                this.updatePromise = this.$interval(
                    () => this.refreshPools(),
                    this.updateFrequency
                );
            }
        }

        stopPolling() {
            if (this.updatePromise) {
                this.$interval.cancel(this.updatePromise);
                this.updatePromise = null;
            }
        }

        clickPool(id) {
            this.resourcesFactory.routeToPool(id);
        }

        clickRemovePool(id) {
            if (this.isDefaultPool(id)) {
                return;
            }

            let modalScope = this.newScope();
            modalScope.resourcesFactory = this.resourcesFactory;
            modalScope.$notification = this.$notification;
            modalScope.refreshPools = () => this.refreshPools();

            this.$modalService.create({
                template: this.$translate.instant("confirm_remove_pool") + "<strong>"+ id +"</strong>",
                model: modalScope,
                title: "remove_pool",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "remove_pool",
                        classes: "btn-danger",
                        action: function(){
                            modalScope.resourcesFactory.removePool(id)
                                .success(function(data) {
                                    modalScope.$notification.create("Removed Pool", id).success();
                                    modalScope.refreshPools();
                                })
                                .error(data => {
                                    modalScope.$notification.create("Remove Pool failed", data.Detail).error();
                                });

                            this.close();
                        }
                    }
                ]
            });
        }

        clickAddPool() {
            let modalScope = this.newScope();
            modalScope.resourcesFactory = this.resourcesFactory;
            modalScope.$notification = this.$notification;
            modalScope.refreshPools = () => this.refreshPools();
            modalScope.permissions = this.permissions;
            modalScope.newPool = {
                permissions: new this.utils.NgBitset(this.permissions.length, 3)
            };

            this.areUIReady.lock();
            this.$modalService.create({
                templateUrl: "add-pool.html",
                model: modalScope,
                title: "add_pool",
                actions: [
                    {
                        role: "cancel",
                        action: function() {
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: "add_pool",
                        action: function() {
                            if(this.validate()) {
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                // add the Permissions field and remove the NgBitset field
                                modalScope.newPool.Permissions = modalScope.newPool.permissions.val;
                                delete modalScope.newPool.permissions;

                                modalScope.resourcesFactory.addPool(modalScope.newPool)
                                    .success(function(data, status){
                                        this.close();
                                        modalScope.$notification.create("Added new Pool", data.Detail).success();
                                        modalScope.refreshPools();
                                    }.bind(this))
                                    .error(function(data, status){
                                        modalScope.$notification.create("Adding pool failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                onShow: () => this.areUIReady.unlock()
            });
        }
    }

    PoolsController.$inject = ["$scope", "resourcesFactory", "authService", 
        "$modalService", "$translate", "$notification", "areUIReady", "$interval", 
        "servicedConfig", "log","POOL_PERMISSIONS", "miscUtils", "Pool", "$q"];
    controlplane.controller("PoolsController", PoolsController);

})();
