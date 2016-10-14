/* globals controlplane: true */

/* HostDetailsController
 * Displays list of hosts
 */
(function() {
    'use strict';

    controlplane.controller("HostDetailsController", [
    "$scope", "$routeParams", "$location", "resourcesFactory", "authService",
    "$modalService", "$translate", "miscUtils", "hostsFactory", "$notification",
    "instancesFactory", "servicesFactory", "$interval",
    function($scope, $routeParams, $location, resourcesFactory, authService,
    $modalService, $translate, utils, hostsFactory, $notification,
    instancesFactory, servicesFactory, $interval){
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

        $scope.resetKeys = function() {
            $scope.modal_confirmResetKeys();
        };

        $scope.modal_confirmResetKeys = function(){
            let scope = $scope.$new(true);
            scope.host = $scope.currentHost;

            $modalService.create({
                template: $translate.instant("reset_host_keys", {name: $scope.currentHost.name}),
                model: scope,
                title: $translate.instant("title_reset_host_keys"),
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        classes: "submit btn-primary",
                        label: $translate.instant("btn_reset_keys"),
                        action: function(){
                            // disable ok button, and store the re-enable function
                            let enableSubmit = this.disableSubmitButton();

                            resourcesFactory.resetHostKeys($scope.currentHost.id)
                                .success((data, status) => {
                                    $modalService.modals.displayHostKeys(data.PrivateKey, $scope.currentHost.name);
                                })
                                .error((data, status) => {
                                    // TODO - form error highlighting
                                    this.createNotification("", data.Detail).error();
                                    // reenable button
                                    enableSubmit();
                                });
                        }
                    }
                ]
            });
        };

        // TODO - move this to Host.js when v2 stuff drops
        // provide a way for hostIcon to get statuses
        $scope.getHostStatus = function(id){
            if($scope.hostStatuses){
                return $scope.hostStatuses[id];
            }
        };
        $scope.getHostStatusClass = function(host){
            if(!host){
                return "unknown";
            }

            let status = $scope.getHostStatus(host.id);

            // stuff hasnt loaded, so unknown
            if(!status){
                return "unknown";
            }

            let active = status.Active,
                authed = status.Authenticated;

            // connected and authenticated
            if(active && authed){
                return "passed";

            // connected but not yet authenticated
            } else if(active && !authed){
                // TODO - something more clearly related to auth
                return "unknown";

            // not connected
            } else {
                return "failed";
            }
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

            // TODO - remove this and consolidate with v2
            // status polling
            $scope.hostStatusInterval = $interval(() => {
                resourcesFactory.getHostStatuses()
                    .then(data => {
                        let statuses = {};
                        data.forEach(s => statuses[s.HostID] = s);
                        $scope.hostStatuses = statuses;
                    }, err => {
                        console.log("err", err);
                    });
            }, 3000);

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
            $interval.cancel($scope.hostStatusInterval);
        });
    }]);
})();
