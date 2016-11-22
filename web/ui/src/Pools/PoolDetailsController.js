/* global controlplane: true */

/* PoolDetailsController
 * Displays details of a specific pool
 */
(function() {
    'use strict';

    controlplane.controller("PoolDetailsController", ["$scope", "$routeParams", "$location",
    "resourcesFactory", "authService", "$modalService", "$translate", "$notification",
    "miscUtils", "areUIReady", "POOL_PERMISSIONS", "Pool", "$rootScope",
    function($scope, $routeParams, $location, resourcesFactory,
    authService, $modalService, $translate, $notification, utils,
    areUIReady, POOL_PERMISSIONS, Pool, $rootScope){
        // Ensure logged in
        authService.checkLogin($scope);

        // allow templates to get the list
        // of permissions
        $scope.permissions = POOL_PERMISSIONS;

        $scope.clickRemoveVirtualIp = function(ip) {
            $modalService.create({
                template: $translate.instant("confirm_remove_virtual_ip") + " <strong>"+ ip.IP +"</strong>",
                model: $scope,
                title: "remove_virtual_ip",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "remove_virtual_ip",
                        classes: "btn-danger",
                        action: function(){
                            resourcesFactory.removePoolVirtualIP(ip.PoolID, ip.IP)
                                .success(function(data) {
                                    $notification.create("Removed Virtual IP", ip.IP).success();
                                    update();
                                })
                                .error(data => {
                                    $notification.create("Remove Virtual IP failed", data.Detail).error();
                                });
                            this.close();
                        }
                    }
                ]
            });
        };

        // Add Virtual Ip Modal - Add button action
        $scope.addVirtualIp = function(pool) {
            var ip = $scope.add_virtual_ip;

            return resourcesFactory.addPoolVirtualIP(ip.PoolID, ip.IP, ip.Netmask, ip.BindInterface)
                .success(function(data, status){
                    $scope.add_virtual_ip = {};
                    $notification.create("Added new pool virtual ip", ip).success();
                    update();
                })
                .error((data, status) => {
                    $notification.create("Add Virtual IP failed", data.Detail).error();
                });

        };

        // Open the virtual ip modal
        $scope.modalAddVirtualIp = function(pool) {
            areUIReady.lock();
            $scope.add_virtual_ip = {'PoolID': pool.id, 'IP':"", 'Netmask':"", 'BindInterface':""};
            $modalService.create({
                templateUrl: "pool-add-virtualip.html",
                model: $scope,
                title: "add_virtual_ip",
                actions: [
                    {
                        role: "cancel",
                        action: function(){
                            $scope.add_virtual_ip = {};
                            this.close();
                        }
                    },{
                        role: "ok",
                        label: "add_virtual_ip",
                        action: function(){
                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                $scope.addVirtualIp($scope.add_virtual_ip)
                                    .success(function(data, status){
                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                       this.createNotification("Adding pool virtual ip failed", data.Detail).error();
                                       enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                onShow: function(){
                    areUIReady.unlock();
                }
            });
        };

        // route host clicks to host page
        $scope.clickHost = function(hostId) {
            resourcesFactory.routeToHost(hostId);
        };

        $scope.editCurrentPool = function(){
            $scope.editablePool = {
                ID: $scope.currentPool.model.ID,
                ConnectionTimeout: utils.humanizeDuration($scope.currentPool.model.ConnectionTimeout),
                permissions: new utils.NgBitset(POOL_PERMISSIONS.length, $scope.currentPool.model.Permissions)
            };

            $modalService.create({
                templateUrl: "edit-pool.html",
                model: $scope,
                title: "title_edit_pool",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "btn_save_changes",
                        action: function(){
                            var poolModel = angular.copy($scope.currentPool.model);
                            angular.extend(poolModel, $scope.editablePool);

                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();
                                // convert validated human input into ms for rest call
                                poolModel.ConnectionTimeout = utils.parseDuration($scope.editablePool.ConnectionTimeout);
                                // add the Permissions field and remove the NgBitset field
                                poolModel.Permissions = poolModel.permissions.val;
                                delete poolModel.permissions;
                                // update pool with recently edited pool
                                resourcesFactory.updatePool($scope.currentPool.model.ID, poolModel)
                                    .success(function(data, status){
                                        $notification.create("Updated pool", poolModel.ID).success();
                                        update();
                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Update pool failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                validate: function(){
                    var err = utils.validateDuration($scope.editablePool.ConnectionTimeout);
                    if(err){
                        this.createNotification("Error", err).error();
                        return false;
                    }
                    return true;
                }
            });
        };

        function setCurrentPool(pool){
            $scope.currentPool = new Pool(pool);
            $scope.currentPool.fetchHosts();
        }

        // update/refresh current pools data
        function update(){
            return resourcesFactory.getPool($scope.params.poolID)
                .success(pool => setCurrentPool(pool));
        }

        function init(){

            $scope.params = $routeParams;

            $scope.add_virtual_ip = {};

            $scope.breadcrumbs = [
                { label: 'breadcrumb_pools', url: '/pools' }
            ];

            $scope.virtualIPsTable = {
                sorting: {
                    IP: "asc"
                },
                searchColumns: ['IP', 'BindInterfaces']
            };

            $scope.hostsTable = {
                sorting: {
                    name: "asc"
                },
                searchColumns: ['name', 'model.Cores', 'model.KernelVersion']
            };

            // get the pool the do some first time setup
            update().then(() => {
                $scope.breadcrumbs.push({label: $scope.currentPool.id, itemClass: 'active'});
                $rootScope.$emit("ready");
            });
        }

        init();
    }]);
})();
