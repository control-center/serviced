/* globals controlplane: true */

/* HostDetailsController
 * Displays list of hosts
 */
(function() {
    'use strict';

    controlplane.controller("HostDetailsController", ["$scope", "$routeParams", "$location", "resourcesFactory", "authService", "$modalService", "$translate", "miscUtils",
    function($scope, $routeParams, $location, resourcesFactory, authService, $modalService, $translate, utils) {
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.name = "hostdetails";
        $scope.params = $routeParams;

        $scope.breadcrumbs = [
            { label: 'breadcrumb_hosts', url: '#/hosts' }
        ];

        $scope.resourcesFactory = resourcesFactory;

        // Also ensure we have a list of hosts
        utils.refreshHosts($scope, resourcesFactory, true);

        $scope.running = utils.buildTable('Name', [
            { id: 'Name', name: 'label_service' },
            { id: 'StartedAt', name: 'running_tbl_start' },
            { id: 'View', name: 'running_tbl_actions' }
        ]);

        $scope.ip_addresses = utils.buildTable('Interface', [
            { id: 'Interface', name: 'ip_addresses_interface' },
            { id: 'Ip', name: 'ip_addresses_ip' },
            { id: 'MAC Address', name: 'ip_addresses_mac' }
        ]);

        $scope.viewLog = function(running) {
            $scope.editService = $.extend({}, running);
            resourcesFactory.get_service_state_logs(running.ServiceID, running.ID, function(log) {
                $scope.editService.log = log.Detail;
                $modalService.create({
                    templateUrl: "view-log.html",
                    model: $scope,
                    title: "title_log",
                    bigModal: true,
                    actions: [
                        {
                            classes: "btn-default",
                            label: "download",
                            action: function(){
                                utils.downloadFile('/services/' + running.ServiceID + '/' + running.ID + '/logs/download');
                            },
                            icon: "glyphicon-download"
                        },
                        {
                            role: "cancel",
                            classes: "btn-default",
                            label: "close"
                        }
                    ],
                    onShow: function(){
                        var textarea = this.$el.find("textarea");
                        textarea.scrollTop(textarea[0].scrollHeight - textarea.height());
                    }
                });
            });
        };

        $scope.toggleRunning = function(app, status, resourcesFactory, skipChildren) {
            var serviceId,
                newState;

            // if app is an instance, use ServiceId
            if("InstanceID" in app){
                serviceId = app.ServiceID;

            // else, app is a service, so use ID
            } else {
                serviceId = app.ID;
            }

            switch(status) {
                case 'start':
                    newState = 1;
                    resourcesFactory.start_service(serviceId, function(){}, skipChildren);
                    break;

                case 'stop':
                    newState = 0;
                    resourcesFactory.stop_service(serviceId, function(){}, skipChildren);
                    break;

                case 'restart':
                    newState = -1;
                    resourcesFactory.restart_service(serviceId, function(){}, skipChildren);
                    break;
            }

            app.DesiredState = newState;
        };

        $scope.click_app = function(instance) {
            $location.path('/services/' + instance.ServiceID);
        };

        $scope.updateHost = function(){
            var modifiedHost = $.extend({}, $scope.hosts.current);
            resourcesFactory.update_host(modifiedHost.ID, modifiedHost, function() {
                utils.refreshHosts($scope, resourcesFactory, false);
            });
        };

        utils.refreshRunningForHost($scope, resourcesFactory, $scope.params.hostId);
        utils.refreshHosts($scope, resourcesFactory, true, function() {
            if ($scope.hosts.current) {
                $scope.breadcrumbs.push({ label: $scope.hosts.current.Name, itemClass: 'active' });
            }
        });

        // Ensure we have a list of pools
        utils.refreshPools($scope, resourcesFactory, false);

        resourcesFactory.get_stats(function(status) {
            if (status === 200) {
                $scope.collectingStats = true;
            } else {
                $scope.collectingStats = false;
            }
        });
    }]);
})();
