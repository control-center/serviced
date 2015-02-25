/* globals controlplane: true */

/* HostDetailsController
 * Displays list of hosts
 */
(function() {
    'use strict';

    controlplane.controller("HostDetailsController", ["$scope", "$routeParams", "$location", "resourcesFactory", "authService", "$modalService", "$translate", "miscUtils", "hostsFactory", "$notification",
    function($scope, $routeParams, $location, resourcesFactory, authService, $modalService, $translate, utils, hostsFactory, $notification){
        // Ensure logged in
        authService.checkLogin($scope);

        $scope.name = "hostdetails";
        $scope.params = $routeParams;

        $scope.breadcrumbs = [
            { label: 'breadcrumb_hosts', url: '#/hosts' }
        ];

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

        $scope.viewLog = function(instance) {
            $scope.editService = angular.copy(instance);
            resourcesFactory.get_service_state_logs(instance.model.ServiceID, instance.id).success(function(log) {
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
                            action: () => {
                                var textarea = this.$el.find("textarea");
                                resourcesFactory.get_service_state_logs(instance.model.ServiceID, instance.id).success(function(log) {
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

        // update hosts
        update();

        function update(){
            // start polling
            hostsFactory.activate();

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
        });
    }]);
})();
