/* HostsController
 * Displays details for a specific host
 */
(function(){
    "use strict";

    let resourcesFactory, authService, $modalService, $translate, $notification,
        areUIReady, $interval, servicedConfig, utils, Host, log, Pool;

    class HostsController {

        constructor($scope) {

            authService.checkLogin(this);

            this.touch();

            this.pools = [];

            this.name = "hosts";
            this.indent = utils.indentClass;
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
                watchExpression: () => this.lastUpdate,
                searchColumns: ['name', 'model.PoolID', 'model.Cores', 'model.KernelVersion']
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

            // These methods will be called by directives, so when they are executed
            // 'this' will be the directive and not the contoller.  To solve this we can
            // bind 'this' to the controller.
            this.updateHostsInView = this.updateHostsInView.bind(this);
            this.getHostStatus = this.getHostStatus.bind(this);

            // New scopes are created to use as models for the modals dialogs.
            // They require some additional methods that are on the global
            // scope.  Since we want to keep $scope limited to just the constructor,
            // this method can be used to create new scopes for modals.
            this.newScope = () => $scope.$new(true);
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        refreshHosts() {
            return resourcesFactory.v2.getHosts().then(data => {
                this.hosts = data.map(result => new Host(result));
                this.touch();
            });
        }

        refreshPoolIds() {
            resourcesFactory.v2.getPools()
                .success(data => {
                    this.pools = data.map(result => new Pool(result)).sort((first, second) => first.id.localeCompare(second.id));
                })
                .error(data => {
                    $notification.create("Unable to load pools.", data.Detail).error();
                });
        }

        refreshHostStatuses() {
            if (!this.hostsInView || this.hostsInView.length < 1) { return; }

            let ids = this.hostsInView.map(h => h.id);
            return resourcesFactory.v2.getHostStatuses(ids).then(data => {
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
                this.updatePromise = $interval(
                    () => this.refreshHostStatuses(),
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

        clickAddHost() {
            let modalScope = this.newScope();
            modalScope.refreshHosts = () => this.refreshHosts();
            modalScope.poolIds = this.pools.map(p => p.id);

            modalScope.newHost = {
                port: $translate.instant('placeholder_port'),
                PoolID: modalScope.poolIds[0] || "",
            };

            areUIReady.lock();
            $modalService.create({
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

                                resourcesFactory.addHost(modalScope.newHost)
                                    .success(function(data, status){
                                        $modalService.modals.displayHostKeys(data.PrivateKey, data.Registered, modalScope.newHost.host);
                                        modalScope.refreshHosts();
                                    }.bind(this))
                                    .error(function(data, status){
                                        // TODO - form error highlighting
                                        this.createNotification("", data.Detail).error();
                                        // reenable button
                                        enableSubmit();
                                    }.bind(this));
                            }
                        }
                    }
                ],
                validate: function(){
                    var err = utils.validateHostName(modalScope.newHost.host, $translate) ||
                        utils.validatePortNumber(modalScope.newHost.port, $translate) ||
                        utils.validateRAMLimit(modalScope.newHost.RAMLimit);
                    if(err){
                        this.createNotification("Error", err).error();
                        return false;
                    }
                    return true;
                },
                onShow: () => areUIReady.unlock()
            });
        }

        clickRemoveHost(id) {
            let modalScope = this.newScope();
            modalScope.refreshHosts = () => this.refreshHosts();

            let hostToRemove = this.hosts.find(h => h.id === id);
            $modalService.create({
                template: $translate.instant("confirm_remove_host") + " <strong>" + hostToRemove.name + "</strong>",
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
                            resourcesFactory.removeHost(id)
                                .success(function(data, status) {
                                    $notification.create("Removed host", id).success();
                                    modalScope.refreshHosts();
                                    this.close();
                                }.bind(this))
                                .error(function(data, status){
                                    $notification.create("Removing host failed", data.Detail).error();
                                    this.close();
                                }.bind(this));
                        }
                    }
                ]
            });
        }

        clickHost(id) {
            resourcesFactory.routeToHost(id);
        }

        clickPool(id) {
            resourcesFactory.routeToPool(id);
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
    }

    controlplane.controller("HostsController", ["$scope", "resourcesFactory", "authService",
        "$modalService", "$translate", "$notification", "areUIReady",
        "$interval", "servicedConfig", "log", "miscUtils", "Host", "Pool",
        function($scope, _resourcesFactory, _authService, _$modalService, _$translate,
        _$notification, _areUIReady, _$interval, _servicedConfig, _log, _miscUtils, _Host, _Pool) {

            resourcesFactory = _resourcesFactory;
            authService = _authService;
            $modalService = _$modalService;
            $translate = _$translate;
            $notification = _$notification;
            areUIReady = _areUIReady;
            $interval = _$interval;
            servicedConfig = _servicedConfig;
            utils = _miscUtils;
            Host = _Host;
            log = _log;
            Pool = _Pool;

        return new HostsController($scope);

    }]);

})();


