/* global controlplane: true */

/* PoolsControl
 * Displays list of pools
 */
(function() {
    'use strict';

    let resourcesFactory, authService, $modalService, $translate, $notification,
        areUIReady, $interval, servicedConfig, log, permissions, utils,
        Pool, $q;

    class PoolsController {

        constructor ($scope)
        {
            authService.checkLogin(this);

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
                watchExpression: () => this.lastUpdate,
                searchColumns: ['id','model.CoreCapacity']
            };

            $scope.breadcrumbs = [
                { label: 'breadcrumb_pools', itemClass: 'active' }
            ];

            this.startPolling();

            $scope.$on("$destroy", () => this.stopPolling());

            // New scopes are created to use as models for the modals dialogs.
            // They require some additional methods that are on the global
            // scope.  Since we want to keep $scope limited to just the constructor,
            // this method can be used to create new scopes for modals.
            this.newScope = () => $scope.$new(true);
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        isDefaultPool(id) {
            return id  === "default";
        }

        refreshPools() {
            let deferred = $q.defer();
            resourcesFactory.v2.getPools()
                .success(data => {
                    this.pools = data.map(result => new Pool(result));
                    this.totalPoolCount = data.length;
                    this.touch();
                    deferred.resolve();
                })
                .error(data => {
                    let message = (data && data.Detail) || "";
                    console.warn("Unable to load pools.", message);
                    deferred.reject();
                });
            return deferred.promise;
        }

        startPolling() {
            if (!this.updatePromise) {
                this.updatePromise = $interval(
                    () => this.refreshPools(),
                    this.updateFrequency
                );
            }
        }

        stopPolling() {
            if (this.updatePromise) {
                $interval.cancel(this.updatePromise);
                this.updatePromise = null;
            }
        }

        clickPool(id) {
            resourcesFactory.routeToPool(id);
        }

        clickRemovePool(id) {
            if (this.isDefaultPool(id)) {
                return;
            }

            let modalScope = this.newScope();
            modalScope.refreshPools = () => this.refreshPools();

            $modalService.create({
                template: $translate.instant("confirm_remove_pool") + "<strong>"+ utils.escapeHTML(id) +"</strong>",
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
                            resourcesFactory.removePool(id)
                                .success(function(data) {
                                    $notification.create("Removed Pool", utils.escapeHTML(id)).success();
                                    modalScope.refreshPools();
                                })
                                .error(data => {
                                    let message = (data && data.Detail) || "";
                                    $notification.create("Remove Pool failed", utils.escapeHTML(message)).error();
                                });

                            this.close();
                        }
                    }
                ]
            });
        }

        clickAddPool() {
            let modalScope = this.newScope();
            modalScope.refreshPools = () => this.refreshPools();
            modalScope.permissions = permissions;
            modalScope.newPool = {
                permissions: new utils.NgBitset(permissions.length, 3)
            };

            areUIReady.lock();
            $modalService.create({
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

                                // make a copy of the payload, add the permissions field, and remove the NgBitset
                                var payload = angular.copy(modalScope.newPool);
                                payload.Permissions = modalScope.newPool.permissions.val;
                                delete payload.permissions;

                                resourcesFactory.addPool(payload)
                                    .success(function(data, status){
                                        this.close();
                                        $notification.create("Added new Pool", utils.escapeHTML(data.Detail)).success();
                                        modalScope.refreshPools();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Adding pool failed", utils.escapeHTML(data.Detail)).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                onShow: () => areUIReady.unlock()
            });
        }
    }

    controlplane.controller("PoolsController", ["$scope", "resourcesFactory", "authService",
        "$modalService", "$translate", "$notification", "areUIReady", "$interval",
        "servicedConfig", "log","POOL_PERMISSIONS", "miscUtils", "Pool", "$q",
        function($scope, _resourcesFactory, _authService, _$modalService, _$translate,
        _$notification, _areUIReady, _$interval, _servicedConfig, _log, _POOL_PERMISSIONS,
        _miscUtils, _Pool, _$q) {

            resourcesFactory = _resourcesFactory;
            authService = _authService;
            $modalService = _$modalService;
            $translate = _$translate;
            $notification = _$notification;
            areUIReady = _areUIReady;
            $interval = _$interval;
            servicedConfig = _servicedConfig;
            log = _log;
            permissions = _POOL_PERMISSIONS;
            utils = _miscUtils;
            Pool = _Pool;
            $q = _$q;

        return new PoolsController($scope);

    }]);
})();
