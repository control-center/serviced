/* InternalServiceDetailsController
 * Displays details for an internal service
 */
(function(){
    "use strict";

    let $q, resourcesFactory, authService, $interval, servicedConfig, InternalService, log, params;

    class InternalServiceDetailsController {

        constructor($scope) {

            authService.checkLogin(this);

            this.touch();

            this.fetchInternalService(params.id)
                .then(() => this.service.fetchInstances())
                .then(() => this.refresh())
                .then(() => {
                    if (this.service.model.Parent) {
                        $scope.breadcrumbs = [
                            { label: 'applications', url: '/apps' },
                            { label: this.service.model.Parent.Name, url: '/internalservices' },
                            { label: this.service.model.Name, itemClass: "active", id: this.service.id }
                        ];
                    } else {
                        $scope.breadcrumbs = [
                            { label: 'applications', url: '/apps' },
                            { label: this.service.Name, itemClass: "active", id: this.service.id }
                        ];

                    }

                    $scope.$emit("ready");
                });

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
        }

        touch() {
            this.lastUpdate = new Date().getTime();
        }

        fetchInternalService(id){
            let deferred = $q.defer();
            resourcesFactory.v2.getInternalService(id)
                .then(data => {
                    this.service = new InternalService(data);
                    this.touch();
                    deferred.resolve();
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });
            return deferred.promise;
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

        refresh() {
            return resourcesFactory.v2.getInternalServiceStatuses([this.service.id])
                .then(data => {
                    let statusMap = data.reduce((map, s) => {
                        map[s.ServiceID] = s;
                        return map;
                    }, {});

                    this.service.updateStatus(statusMap[this.service.id]);

                    this.touch();
                });
        }
    }

    controlplane.controller("InternalServiceDetailsController", [
    "$scope", "$q", "resourcesFactory", "authService", "$interval",
    "servicedConfig", "InternalService", "log", "$routeParams",
    function($scope, _$q, _resourcesFactory, _authService,
    _$interval, _servicedConfig, _InternalService, _log, _$routeParams) {

        $q = _$q;
        resourcesFactory = _resourcesFactory;
        authService = _authService;
        $interval = _$interval;
        servicedConfig = _servicedConfig;
        InternalService = _InternalService;
        log = _log;
        params = _$routeParams;

        return new InternalServiceDetailsController($scope);
    }]);


})();
