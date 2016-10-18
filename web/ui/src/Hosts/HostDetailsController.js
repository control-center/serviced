/* HostDetailsController
 * Displays list of hosts
 */
(function() {
    'use strict';

    class HostDetailsController {

        constructor($scope, $routeParams, $location, resourcesFactory, authService,
                    $modalService, $translate, $notification, $interval, servicedConfig,
                    log, miscUtils, Host, Instance, $q) {

            authService.checkLogin(this);

            this.resourcesFactory = resourcesFactory;
            this.$modalService = $modalService;
            this.$translate = $translate;
            this.$notification = $notification;
            this.utils = miscUtils;
            this.Host = Host;
            this.params = $routeParams;
            this.Instance = Instance;
            this.$interval = $interval;
            this.$q = $q;

            this.name = "hostdetails";

            $scope.breadcrumbs = [
                { label: 'breadcrumb_hosts', url: '/hosts' }
            ];


            this.hostInstances = [];

            this.touch();
            this.touchInstances();

            $scope.ipsTable = {
                sorting: {
                    InterfaceName: "asc"
                },
                watchExpression: () => this.lastUpdate
            };

            $scope.instancesTable = {
                sorting: {
                    name: "asc"
                },
                watchExpression: () => this.lastInstanceUpdate
            };

            this.refreshHost()
                .then(this.refresh())
                .then(() => {
                    $scope.breadcrumbs.push({
                        label: this.currentHost.name,
                        itemClass: 'active' }
                    );

                    $scope.$emit("ready");
                    }
                );

            this.updateFrequency = 3000;
            servicedConfig.getConfig()
                .then(config => {
                    this.updateFrequency = config.PollFrequency * 1000;
                }).catch(err => {
                    let errMessage = err.data ? err.data.Detail : err.statusText;
                    log.error("could not load serviced config:", errMessage);
                });


            this.startPolling();

            $scope.$on("$destroy", () => this.stopPolling());

            this.newScope = () => $scope.$new(true);
            this.getHostStatus = this.getHostStatus.bind(this);
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        touchInstances() {
            this.lastInstanceUpdate = new Date().getTime();
        }

        refreshHost() {
            return this.resourcesFactory.getHost(this.params.hostId)
                .then(data => {
                    this.currentHost = new this.Host(data);
                    this.touch();
                });
        }

        refreshInstances() {
            return this.resourcesFactory.v2.getHostInstances(this.params.hostId)
                .then(data => {
                    this.hostInstances = data.map(i => new this.Instance(i));
                    this.touchInstances();
                });
        }

        refreshHostStatus() {
            var id = this.params.hostId;
            return this.resourcesFactory.v2.getHostStatuses([id])
                .then(data => {
                    var statusHash = data.reduce(function(hash, status) {
                        hash[status.HostID] = status; return hash;
                    }, {});

                    if (id in statusHash) {
                        this.currentHost.status = statusHash[id];
                    }
                });
        }

        refresh() {
            return this.$q.all([ 
                this.refreshInstances(),
                this.refreshHostStatus()]
            );
        }

        getHostStatus() {
            return this.currentHost.status;
        }

        startPolling() {
            if (!this.updatePromise) {
                this.updatePromise = this.$interval(
                    () => this.refresh(), this.updateFrequency
                );
            }
        }

        stopPolling() {
            if (this.updatePromise) {
                this.$interval.cancel(this.updatePromise);
                this.updatePromise = null;
            }
        }

        viewLog(instance) {
            let modalScope = this.newScope();
            modalScope.resourcesFactory = this.resourcesFactory;
            modalScope.utils = this.utils;
            modalScope.editService = angular.copy(instance);
            modalScope.$modalService = this.$modalService;

            this.resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id)
                .success(function(log) {
                    modalScope.log = log.Detail;
                    modalScope.$modalService.create({
                        templateUrl: "view-log.html",
                        model: modalScope,
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
                                    modalScope.resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id).success(function(log) {
                                        modalScope.log = log.Detail;
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
                                    modalScope.utils.downloadFile('/services/' + instance.model.ServiceID + '/' + instance.model.ID + '/logs/download');
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
        }

        click_app(instance) {
            this.$location.path('/services/' + instance.model.ServiceID);
        }

        editCurrentHost() {
            let modalScope = this.newScope();
            modalScope.resourcesFactory = this.resourcesFactory;
            modalScope.utils = this.utils;
            modalScope.$modalService = this.$modalService;
            modalScope.$notification = this.$notification;
            modalScope.refreshHost = () => this.refreshHost();
            modalScope.currentHost = this.currentHost;

            modalScope.editableHost = {
                Name: this.currentHost.name,
                RAMLimit: this.currentHost.RAMLimit
            };

            this.$modalService.create({
                templateUrl: "edit-host.html",
                model: modalScope,
                title: "title_edit_host",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "btn_save_changes",
                        action: function(){
                            var hostModel = angular.copy(modalScope.currentHost.model);
                            angular.extend(hostModel, modalScope.editableHost);

                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();

                                // update host with recently edited host
                                modalScope.resourcesFactory.updateHost(modalScope.currentHost.id, hostModel)
                                    .success(function(data, status){
                                        modalScope.$notification.create("Updated host", hostModel.Name).success();
                                        modalScope.refreshHost();
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
                    var err = modalScope.utils.validateRAMLimit(modalScope.editableHost.RAMLimit, modalScope.currentHost.model.Memory);
                    if(err){
                        this.createNotification("Error", err).error();
                        return false;
                    }
                    return true;
                }
            });
        }

        resetKeys() {
            this.modal_confirmResetKeys();
        }

        modal_confirmResetKeys() {
            let scope = this.newScope();
            scope.host = this.currentHost;
            scope.$modalService = this.$modalService;
            scope.resourcesFactory = this.resourcesFactory;

            this.$modalService.create({
                template: this.$translate.instant("reset_host_keys", {name: this.currentHost.name}),
                model: scope,
                title: this.$translate.instant("title_reset_host_keys"),
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        classes: "submit btn-primary",
                        label: this.$translate.instant("btn_reset_keys"),
                        action: function(){
                            // disable ok button, and store the re-enable function
                            let enableSubmit = this.disableSubmitButton();

                            scope.resourcesFactory.resetHostKeys(scope.host.id)
                                .success((data, status) => {
                                    scope.$modalService.modals.displayHostKeys(data.PrivateKey, scope.host.name);
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
        }

        restart(hostId, instanceId) {
            this.resourcesFactory.killRunning(hostId, instanceId)
                .then(this.refresh());
        }
    }

    HostDetailsController.$inject = ["$scope", "$routeParams", "$location", "resourcesFactory",
        "authService", "$modalService", "$translate", "$notification",
        "$interval", "servicedConfig", "log", "miscUtils", "Host", "Instance", "$q"];
    controlplane.controller("HostDetailsController", HostDetailsController);

})();
