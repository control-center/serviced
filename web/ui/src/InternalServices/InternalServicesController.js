/* InternalServicesController
 * Displays details for internal services
 */
(function(){
    "use strict";

    let $q, resourcesFactory, authService,
        $interval, servicedConfig, InternalService, log;

    class InternalServicesController {

        constructor($scope) {

            authService.checkLogin(this);

            this.touch();

            $scope.internalServicesTable = {
                sorting: {
                    name: "asc"
                },
                watchExpression: () => this.lastUpdate
            };

            this.updateFrequency = 3000;
            servicedConfig.getConfig()
                .then(config => {
                    this.updateFrequency = config.PollFrequency * 1000;
                }).catch(err => {
                    let errMessage = err.data ? err.data.Detail : err.statusText;
                    log.error("could not load serviced config:", errMessage);
                });

            this.fetchInternalServices()
                .then(() => this.fetchInstances())
                .then(() => this.refresh())
                .then(() => {
                    $scope.breadcrumbs = [
                        { label: 'applications', url: '/apps' },
                        { label: this.parent.model.Name, itemClass: 'active' }
                    ];

                    $scope.$emit("ready");
                });

            this.startPolling();

            $scope.$on("$destroy", () => this.stopPolling());
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        startPolling() {
            if (!this.updatePromise) {
                this.updatePromise = $interval(
                    () => this.refresh(),
                    this.updateFrequency);
            }
        }

        stopPolling() {
            if (this.updatePromise) {
                $interval.cancel(this.updatePromise);
                this.updatePromise = null;
            }
        }

        fetchInternalServices() {
            let deferred = $q.defer();
            resourcesFactory.v2.getInternalServices()
                .then(data => {
                    let internalService= data.map(result => new InternalService(result));
                    let parentIndex = internalService.findIndex(i => !i.Parent);
                    if (parentIndex > -1) {
                        this.parent = internalService.splice(parentIndex, 1)[0];
                        this.children = internalService;
                    }
                    this.touch();
                    deferred.resolve();
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });
            return deferred.promise;
        }

        fetchInstances() {
            let promises = [];

            if (this.parent) {
                promises.push(this.parent.fetchInstances());
            }

            if (this.children) {
                this.children.forEach(c => promises.push(c.fetchInstances()));
            }

            return $q.all(promises);
        }

        refresh() {
            return resourcesFactory.v2.getInternalServiceStatuses()
                .then(data => {
                    let statusMap = data.reduce((map, s) => {
                        map[s.ServiceID] = s;
                        return map;
                    }, {});

                    this.parent.updateStatus(statusMap[this.parent.id]);
                    this.children.forEach(c => c.updateStatus(statusMap[c.id]));

                    this.touch();
                });
        }

        clickInternalService(id) {
            resourcesFactory.routeToInternalService(id);
        }
    }

    controlplane.controller("InternalServicesController", ["$scope", "$q", "resourcesFactory",
    "authService", "$interval", "servicedConfig", "InternalService" , "log",
    function($scope, _$q, _resourcesFactory, _authService,
    _$interval, _servicedConfig, _InternalService, _log) {

        $q = _$q;
        resourcesFactory = _resourcesFactory;
        authService = _authService;
        $interval = _$interval;
        servicedConfig = _servicedConfig;
        InternalService = _InternalService;
        log = _log;

        return new InternalServicesController($scope);
    }]);

})();
