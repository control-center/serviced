/* global controlplane: true */

/* PoolDetailsController
 * Displays details of a specific pool
 */
(function() {
    'use strict';

    controlplane.controller("PoolDetailsController", ["$scope", "$routeParams", "$location", "resourcesFactory", "authService", "$modalService", "$translate", "$notification", "miscUtils", "hostsFactory", "poolsFactory",
    function($scope, $routeParams, $location, resourcesFactory, authService, $modalService, $translate, $notification, utils, hostsFactory, poolsFactory){
        // Ensure logged in
        authService.checkLogin($scope);

        //
        // Scope methods
        //

        // Pool view action - delete
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
                                    poolsFactory.update();
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
                    poolsFactory.update();
                })
                .error((data, status) => {
                    $notification.create("Add Virtual IP failed", data.Detail).error();
                });

        };

        // Open the virtual ip modal
        $scope.modalAddVirtualIp = function(pool) {
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
                ]
            });
        };

        // route host clicks to host page
        $scope.clickHost = function(hostId) {
            resourcesFactory.routeToHost(hostId);
        };

        function init(){

            $scope.name = "pooldetails";
            $scope.params = $routeParams;

            $scope.add_virtual_ip = {};

            $scope.breadcrumbs = [
                { label: 'breadcrumb_pools', url: '/pools' }
            ];

            // start polling
            poolsFactory.activate();

            // Ensure we have a list of pools
            poolsFactory.update()
                .then(() => {
                    $scope.currentPool = poolsFactory.get($scope.params.poolID);
                    if ($scope.currentPool) {
                        $scope.breadcrumbs.push({label: $scope.currentPool.id, itemClass: 'active'});

                        // start polling
                        hostsFactory.activate();

                        hostsFactory.update()
                            .then(() => {
                               // reduce the list to hosts associated with this pool
                                $scope.hosts = hostsFactory.hostList.filter(function(host){
                                    return host.model.PoolID === $scope.currentPool.id;
                                });

                            });
                    }

                });

            $scope.virtualIPsTable = {
                sorting: {
                    IP: "asc"
                },
                watchExpression: function(){
                    // if poolsFactory updates, update view
                    return poolsFactory.lastUpdate;
                }
            };

            $scope.hostsTable = {
                sorting: {
                    name: "asc"
                },
                watchExpression: function(){
                    return hostsFactory.lastUpdate;
                }
            };
        }

        init();

        $scope.$on("$destroy", function(){
            poolsFactory.deactivate();
            hostsFactory.deactivate();
        });
    }]);
})();
