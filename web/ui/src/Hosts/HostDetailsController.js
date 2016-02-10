/* globals controlplane: true */

/* HostDetailsController
 * Displays list of hosts
 */
(function() {
    'use strict';

    controlplane.controller("HostDetailsController", ["$scope", "$routeParams", "$location", "resourcesFactory", "authService", "$modalService", "$translate", "miscUtils", "hostsFactory", "$notification", "instancesFactory", "servicesFactory",
    function($scope, $routeParams, $location, resourcesFactory, authService, $modalService, $translate, utils, hostsFactory, $notification, instancesFactory, servicesFactory){
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.name = "hostdetails";
        $scope.params = $routeParams;

        $scope.breadcrumbs = [
            { label: 'breadcrumb_hosts', url: '/hosts' }
        ];

        $scope.viewLog = function(instance) {
            $scope.editService = angular.copy(instance);
            resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id)
                .success(function(log) {
                    $scope.editService.log = log.Detail;
                    $modalService.create({
                        templateUrl: "view-log.html",
                        model: $scope,
                        title: "title_log",
                        bigModal: true,
                        actions: [
                            {
                                role: "cancel",
                                label: "close"
                            },{
                                classes: "btn-primary",
                                label: "refresh",
                                icon: "glyphicon-repeat",
                                action: function(){
                                    var textarea = this.$el.find("textarea");
                                    resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id).success(function(log) {
                                        $scope.editService.log = log.Detail;
                                        textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                                    })
                                    .error((data, status) => {
                                        this.createNotification("Unable to fetch logs", data.Detail).error();
                                    });
                                }
                            },{
                                classes: "btn-primary",
                                label: "download",
                                action: function(){
                                    utils.downloadFile('/services/' + instance.model.ServiceID + '/' + instance.model.ID + '/logs/download');
                                },
                                icon: "glyphicon-download"
                            }
                        ],
                        onShow: function(){
                            var textarea = this.$el.find("textarea");
                            textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                        }
                    });
                })
                .error((data, status) => {
                    this.createNotification("Unable to fetch logs", data.Detail).error();
                });
        };

        $scope.click_app = function(instance) {
            $location.path('/services/' + instance.model.ServiceID);
        };

        $scope.editCurrentHost = function(){
            $scope.editableHost = {
                Name: $scope.currentHost.name,
                RAMLimit: $scope.currentHost.RAMLimit
            };

            $modalService.create({
                templateUrl: "edit-host.html",
                model: $scope,
                title: "title_edit_host",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "btn_save_changes",
                        action: function(){
                            var hostModel = angular.copy($scope.currentHost.model);
                            angular.extend(hostModel, $scope.editableHost);

                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                // update host with recently edited host
                                resourcesFactory.updateHost($scope.currentHost.id, hostModel)
                                    .success(function(data, status){
                                        $notification.create("Updated host", hostModel.Name).success();
                                        this.close();
                                    }.bind(this))
                                    .error(function(data, status){
                                        this.createNotification("Update host failed", data.Detail).error();
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                validate: function(){
                    var err = utils.validateRAMLimit($scope.editableHost.RAMLimit, $scope.currentHost.model.Memory);
                    if(err){
                        this.createNotification("Error", err).error();
                        return false;
                    }
                    return true;
                }
            });
        };

        init();

        function init(){
            // start polling
            hostsFactory.activate();
            servicesFactory.activate();
            servicesFactory.update();

            $scope.ipsTable = {
                sorting: {
                    InterfaceName: "asc"
                },
                watchExpression: function(){
                    return hostsFactory.lastUpdate;
                }
            };

            $scope.instancesTable = {
                sorting: {
                    name: "asc"
                },
                watchExpression: function(){
                    return instancesFactory.lastUpdate;
                }
            };

            // kick off hostsFactory updating
            // TODO - update loop here
            hostsFactory.update()
                .then(() => {
                    $scope.currentHost = hostsFactory.get($scope.params.hostId);
                    $scope.breadcrumbs.push({ label: $scope.currentHost.name, itemClass: 'active' });
                });

        }

        $scope.$on("$destroy", function(){
            hostsFactory.deactivate();
            servicesFactory.deactivate();
        });
    }]);
})();
