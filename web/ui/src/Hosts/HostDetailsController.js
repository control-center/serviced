/* HostDetailsController
 * Displays list of hosts
 */
(function() {
    'use strict';

    let params, $location, resourcesFactory, authService, $modalService,
        $translate, $notification, $interval, servicedConfig, log, utils,
        Host, Instance, $q;

    class HostDetailsController {

        constructor($scope) {

            authService.checkLogin(this);

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
                watchExpression: () => this.lastUpdate,
                searchColumns: ['InterfaceName', 'IPAddress', 'MACAddress']
            };

            $scope.instancesTable = {
                sorting: {
                    name: "asc"
                },
                watchExpression: () => this.lastInstanceUpdate,
                searchColumns: ['model.ServiceName']
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

            // New scopes are created to use as models for the modals dialogs.
            // They require some additional methods that are on the global
            // scope.  Since we want to keep $scope limited to just the constructor,
            // this method can be used to create new scopes for modals.
            this.newScope = () => $scope.$new(true);

            // This method will be called by a directive, so when it is executed
            // 'this' will be the directive and not the contoller.  To solve this we can
            // bind 'this' to the controller.
            this.getHostStatus = this.getHostStatus.bind(this);
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        touchInstances() {
            this.lastInstanceUpdate = new Date().getTime();
        }

        refreshHost() {
            return resourcesFactory.getHost(params.hostId)
                .then(data => {
                    this.currentHost = new Host(data);
                    this.touch();
                });
        }

        refreshInstances() {
            return resourcesFactory.v2.getHostInstances(params.hostId)
                .then(data => {
                    this.hostInstances = data.map(i => new Instance(i));
                    this.touchInstances();
                });
        }

        refreshHostStatus() {
            var id = params.hostId;
            return resourcesFactory.v2.getHostStatuses([id])
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
            return $q.all([
                this.refreshInstances(),
                this.refreshHostStatus()]
            );
        }

        getHostStatus() {
            return this.currentHost.status;
        }

        startPolling() {
            if (!this.updatePromise) {
                this.updatePromise = $interval(
                    () => this.refresh(), this.updateFrequency
                );
            }
        }

        stopPolling() {
            if (this.updatePromise) {
                $interval.cancel(this.updatePromise);
                this.updatePromise = null;
            }
        }

        viewLog(instance) {
            let modalScope = this.newScope();
            modalScope.editService = angular.copy(instance);

            resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id)
                .success(function(log) {
                    modalScope.log = log.Detail;
                    $modalService.create({
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
                                    resourcesFactory.getInstanceLogs(instance.model.ServiceID, instance.id).success(function(log) {
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
                                    utils.downloadFile('/services/' + instance.model.ServiceID + '/' + instance.id + '/logs/download');
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
            $location.path('/services/' + instance.model.ServiceID);
        }

        editCurrentHost() {
            let modalScope = this.newScope();
            modalScope.refreshHost = () => this.refreshHost();
            modalScope.currentHost = this.currentHost;

            modalScope.editableHost = {
                Name: this.currentHost.name,
                RAMLimit: this.currentHost.RAMLimit
            };

            $modalService.create({
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
                                resourcesFactory.updateHost(modalScope.currentHost.id, hostModel)
                                    .success(function(data, status){
                                        $notification.create("Updated host", hostModel.Name).success();
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
                    var err = utils.validateRAMLimit(modalScope.editableHost.RAMLimit, modalScope.currentHost.model.Memory);
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

            $modalService.create({
                template: $translate.instant("reset_host_keys", {name: this.currentHost.name}),
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

                            resourcesFactory.resetHostKeys(scope.host.id)
                                .success((data, status) => {
                                    $modalService.modals.displayHostKeys(data.PrivateKey, data.Registered, scope.host.name);
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
            resourcesFactory.killRunning(hostId, instanceId)
                .then(this.refresh());
        }
    }

    controlplane.controller("HostDetailsController", ["$scope", "$routeParams", "$location",
        "resourcesFactory", "authService", "$modalService", "$translate", "$notification",
        "$interval", "servicedConfig", "log", "miscUtils", "Host", "Instance", "$q",
        function($scope, _$routeParams, _$location, _resourcesFactory, _authService, _$modalService,
        _$translate, _$notification, _$interval, _servicedConfig, _log, _miscUtils,
        _Host, _Instance, _$q) {

            params = _$routeParams;
            $location = _$location;
            resourcesFactory = _resourcesFactory;
            authService = _authService;
            $modalService = _$modalService;
            $translate = _$translate;
            $notification = _$notification;
            $interval = _$interval;
            servicedConfig = _servicedConfig;
            utils = _miscUtils;
            Host = _Host;
            Instance = _Instance;
            $q = _$q;
            log = _log;

        return new HostDetailsController($scope);

    }]);

})();
