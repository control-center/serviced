/* HostsController
 * Displays details for a specific host
 */
(function(){
    "use strict";

    class HostsController {

        constructor($scope, $routeParams, resourcesFactory, authService,
                    $modalService, $translate, $notification, areUIReady,
                    $interval, servicedConfig, log, miscUtils, Host) {

            authService.checkLogin(this);

            this.resourcesFactory = resourcesFactory;
            this.$modalService = $modalService;
            this.$translate = $translate;
            this.$notification = $notification;
            this.areUIReady = areUIReady;
            this.$interval = $interval;
            this.utils = miscUtils;
            this.params = $routeParams;

            this.touch();

            this.name = "hosts";
            this.indent = this.utils.indentClass;
            this.hostsInView = [];

            this.updateFrequency = 3000;
            servicedConfig.getConfig()
                .then(config => {
                    this.updateFrequency = config.PollFrequency * 1000;
                }).catch(err => {
                    let errMessage = err.data ? err.data.Detail : err.statusText;
                    log.error("could not load serviced config:", errMessage);
                });

            $scope.breadcrumbs = [
                { label: 'breadcrumb_hosts', itemClass: 'active' }
            ];

            $scope.hostsTable = {
                sorting: {
                    name: "asc"
                },
                watchExpression: () => this.lastUpdate
            };

            $scope.dropped = [];

            this.refreshHosts().then(() => {
                $scope.$emit("ready");
            },
                error => $notification.create("Unable to load hosts.", error.Detail).error()
            );

            this.refreshPoolIds();

            this.startPolling();

            $scope.$on("$destroy", () => {
                this.stopPolling();
            });

            this.updateHostsInView = this.updateHostsInView.bind(this);
            this.getHostStatus = this.getHostStatus.bind(this);

            this.newScope = () => $scope.$new(true);
            this.newHost = data => new Host(data);
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        refreshHosts() {
            return this.resourcesFactory.v2.getHosts().then(data => {
                this.hosts = data.map(result => this.newHost(result));
                this.touch();
            });
        }

        refreshPoolIds() {
            this.resourcesFactory.v2.getPools()
                .success(data => {
                    this.poolIds = data.map(result => result.ID).sort();
                })
                .error(data => {
                    this.$notification.create("Unable to load pools.", data.Detail).error();
                });
        }

        refreshHostStatuses() {
            if (!this.hostsInView || this.hostsInView.length < 1) { return; }

            let ids = this.hostsInView.map(h => h.id);
            return this.resourcesFactory.v2.getHostStatuses(ids).then(data => {
                    let statusHash = data.reduce(function(hash, status) {
                        hash[status.HostID] = status; return hash;
                    }, {});

                    this.hosts.forEach(h => {
                        if (h.id in statusHash) {
                            h.status = statusHash[h.id];
                        }
                    });
                });
        }

        startPolling() {
            if (!this.updatePromise) {
                this.updatePromise = this.$interval(
                    () => this.refreshHostStatuses(),
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

        clickAddHost() {
            let modalScope = this.newScope();
            modalScope.resourcesFactory = this.resourcesFactory;
            modalScope.$modalService = this.$modalService;
            modalScope.refreshHosts = () => this.refreshHosts();
            modalScope.utils = this.utils;
            modalScope.$translate = this.$translate;
            modalScope.$notification = this.$notification;
            modalScope.poolIds = this.poolIds;

            modalScope.newHost = {
                port: this.$translate.instant('placeholder_port'),
                PoolID: this.arrayEmpty(this.poolsIds) ? "" : this.poolIds[0]
            };

            this.areUIReady.lock();
            this.$modalService.create({
                templateUrl: "add-host.html",
                model: modalScope,
                title: "add_host",
                actions: [
                    {
                        role: "cancel",
                    },{
                        role: "ok",
                        label: "btn_next",
                        icon: "glyphicon-chevron-right",
                        action: function(){
                            if(this.validate()){
                                // disable ok button, and store the re-enable function
                                var enableSubmit = this.disableSubmitButton();
                                if (modalScope.newHost.RAMLimit === undefined || modalScope.newHost.RAMLimit === '') {
                                    modalScope.newHost.RAMLimit = "100%";
                                }

                                modalScope.newHost.IPAddr = modalScope.newHost.host + ':' + modalScope.newHost.port;

                                modalScope.resourcesFactory.addHost(modalScope.newHost)
                                    .success(function(data, status){
                                        modalScope.$modalService.modals.displayHostKeys(data.PrivateKey, data.Registered, modalScope.newHost.host);
                                        modalScope.refreshHosts();
                                    }.bind(this))
                                    .error(function(data, status){
                                        // TODO - form error highlighting
                                        modalScope.$notification.create("", data.Detail).error();
                                        // reenable button
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                validate: function(){
                    var err = modalScope.utils.validateHostName(modalScope.newHost.host, modalScope.$translate) ||
                        modalScope.utils.validatePortNumber(modalScope.newHost.port, modalScope.$translate) ||
                        modalScope.utils.validateRAMLimit(modalScope.newHost.RAMLimit);
                    if(err){
                        modalScope.$notification.create("Error", err).error();
                        return false;
                    }
                    return true;
                },
                onShow: () => {
                    this.areUIReady.unlock();
                }
            });
        }

        clickRemoveHost(id) {
            let modalScope = this.newScope();
            modalScope.resourcesFactory = this.resourcesFactory;
            modalScope.refreshHosts = () => this.refreshHosts();
            modalScope.update = this.update;
            modalScope.$notification = this.$notification;

            let hostToRemove = this.hosts.find(h => h.id === id);
            this.$modalService.create({
                template: this.$translate.instant("confirm_remove_host") + " <strong>" + hostToRemove.name + "</strong>",
                model: modalScope,
                title: "remove_host",
                actions: [
                    {
                        role: "cancel"
                    },{
                        role: "ok",
                        label: "remove_host",
                        classes: "btn-danger",
                        action: function(){
                            modalScope.resourcesFactory.removeHost(id)
                                .success(function(data, status) {
                                    modalScope.$notification.create("Removed host", id).success();
                                    modalScope.refreshHosts();
                                    this.close();
                                }.bind(this))
                                .error(function(data, status){
                                    modalScope.$notification.create("Removing host failed", data.Detail).error();
                                    this.close();
                                }.bind(this));
                        }
                    }
                ]
            });
        }

        clickHost(id) {
            this.resourcesFactory.routeToHost(id);
        }

        clickPool(id) {
            this.resourcesFactory.routeToPool(id);
        }

        getHostStatus(id) {
            let index = this.hostsInView.findIndex(h => h.id === id);
            if (index > -1) {
                return this.hostsInView[index].status;
            }
        }

        updateHostsInView(data) {
            this.hostsInView = data;
            this.refreshHostStatuses();
        }

        arrayEmpty(array) {
            return typeof array !== "undefined" && array !== null && array.length > 0;
        }
    }

    HostsController.$inject = ["$scope", "$routeParams", "resourcesFactory",
        "authService", "$modalService", "$translate", "$notification", "areUIReady",
        "$interval", "servicedConfig", "log", "miscUtils", "Host"];
    controlplane.controller("HostsController", HostsController);

})();


