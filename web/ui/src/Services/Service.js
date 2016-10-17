(function () {
    'use strict';

    // share angular services outside of angular context
    let $notification, serviceHealth, $q, resourcesFactory, utils, Instance;

    controlplane.factory('Service', ServiceFactory);

    // DesiredState enum
    var START = 1,
        STOP = 0,
        RESTART = -1;

    // service types
    var ISVC = "isvc",           // internal service
        APP = "app",             // service with no parent
        META = "meta",           // service with children but no startup command
        DEPLOYING = "deploying"; // service whose parent is still being deployed

    // fetch retrieves something from the v2 api
    // endpoint and stores it on the `this` context
    function fetch(methodName, propertyName, force) {
        let deferred = $q.defer();

        if (this[propertyName] && !force) {
            deferred.resolve();
            return deferred.promise;
        }
        // NOTE: V2 API only
        // TODO: make sure methodName exists
        resourcesFactory.v2[methodName](this.id)
            .then(data => {
                this[propertyName] = data;
                this.touch();
                deferred.resolve();
            },
            error => {
                deferred.reject(error);
                console.warn(error);
            });

        return deferred.promise;
    }

    // given a service, accumulate all descendents into
    // a map keyed by service id
    function getDescendents(descendents, service) {
        service.subservices.forEach(svc => getDescendents(descendents, svc));
        descendents[service.id] = service;
        return descendents;
    }

    class Service {

        constructor(model) {
            this.subservices = [];
            this.instances = [];
            this.update(model);
        }

        update(model) {
            // basically new-up with exisiting children and instances
            this.name = model.Name;
            this.id = model.ID;
            this.desiredState = model.DesiredState;
            this.model = Object.freeze(model);
            this.evaluateServiceType();
            this.touch();
        }

        evaluateServiceType() {
            // infer service type
            this.type = [];
            if (this.model.ID.indexOf("isvc-") !== -1) {
                this.type.push(ISVC);
            }

            if (!this.model.ParentServiceID) {
                this.type.push(APP);
            }

            if (this.subservices.length && !this.model.Startup) {
                this.type.push(META);
            }

            if (this.parent && this.parent.isDeploying()) {
                this.type.push(DEPLOYING);
            }
        }

        // fills out service object
        fetchAll(force) {
            this.fetchEndpoints(force);
            this.fetchAddresses(force);
            this.fetchConfigs(force);
            // NOTE: force-fetching children will wipe
            // out the entire service tree below this service
            this.fetchServiceChildren();
            this.fetchMonitoringProfile(force);
            this.fetchExportEndpoints(force);
        }

        fetchAddresses(force) {
            fetch.call(this, "getServiceIpAssignments", "addresses", force);
        }

        fetchConfigs(force) {
            fetch.call(this, "getServiceConfigs", "configs", force);
        }

        fetchEndpoints(force) {
            // populate publicEndpoints property
            fetch.call(this, "getServicePublicEndpoints", "publicEndpoints", force);
        }

        fetchExportEndpoints(force) {
            let deferred = $q.defer();
            resourcesFactory.v2.getServiceExportEndpoints(this.id)
                .then(response => {
                    this.exportedServiceEndpoints = response
                        .filter(ept => ept.Protocol === "tcp")
                        .map(ept => {
                            // TODO - dont modify model data :(
                            ept.Value = `${ept.ServiceName} - ${ept.Application}`;
                            return ept;
                        });
                    deferred.resolve();
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });

            return deferred.promise;
        }

        fetchMonitoringProfile(force) {
            fetch.call(this, "getServiceMonitoringProfile", "monitoringProfile", force);
        }

        fetchInstances() {
            let deferred = $q.defer();
            resourcesFactory.v2.getServiceInstances(this.id)
                .then(results => {
                    results.forEach(data => {
                        // new-ing instances will cause UI bounce and force rebuilding
                        // of the popover. To minimize UI churn, update/merge status info
                        // into exisiting instance objects  
                        let iid = data.InstanceID;
                        if (this.instances[iid]) {
                            this.instances[iid].update(data);
                        } else {
                            // add into the proper instance slot here
                            this.instances[iid] = new Instance(data);
                        }
                    });
                    // chop off any extraneous instances
                    this.instances.splice(results.length);
                    deferred.resolve();
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });

            return deferred.promise;
        }

        fetchServiceChildren(force) {
            let deferred = $q.defer();
            if (this.subservices.length && !force) {
                deferred.resolve();
            }
            resourcesFactory.v2.getServiceChildren(this.id)
                .then(data => {
                    // TODO - dont blow away existing children
                    this.subservices = data.map(s => new Service(s));
                    this.touch();
                    deferred.resolve();
                },
                error => {
                    console.warn(error);
                    deferred.reject();
                });
            return deferred.promise;
        }

        // fast-moving state for this service and its instances
        // note: returns a promise that resolves with a single status object
        getStatus() {
            let deferred = $q.defer();

            resourcesFactory.v2.getServiceStatus(this.id)
                .then(results => {
                    if (results.length){
                        // getServiceStatus returns an array of results
                        // but we only want a single result
                        deferred.resolve(results[0]);
                    } else {
                        deferred.reject(`Could not get service status for id ${this.id}`);
                    }
                }, error => {
                    deferred.reject(error);
                });
            return deferred.promise;
        }

        // fetch and update service statuses for all
        // descendents of this service
        updateDescendentStatuses() {
            let deferred = $q.defer();
            let descendents = getDescendents({}, this);
            let ids = Object.keys(descendents);
            resourcesFactory.v2.getServiceStatuses(ids)
                .then(results => {
                    if (results.length) {
                        results.forEach(stat => {
                            // TODO - handle stat.NotFound
                            let svc = descendents[stat.ServiceID];
                            svc.updateState(stat);
                        });
                        deferred.resolve();
                    }
                }, error => {
                    deferred.reject(error);
                });
            return deferred.promise;
        }

        hasInstances() {
            return !!this.instances.length;
        }

        isIsvc() {
            return !!~this.type.indexOf(ISVC);
        }

        hasChildren() {
            return this.model.HasChildren;
        }

        resourcesGood() {
            for (var i = 0; i < this.instances.length; i++) {
                if (!this.instances[i].resourcesGood()) {
                    return false;
                }
            }
            return true;
        }

        // start, stop, or restart this service
        start(skipChildren) {
            var promise = resourcesFactory.startService(this.id, skipChildren),
                oldDesiredState = this.desiredState;

            this.desiredState = START;

            // if something breaks, return desired
            // state to its previous value
            return promise.error(() => {
                this.desiredState = oldDesiredState;
            });
        }

        stop(skipChildren) {
            var promise = resourcesFactory.stopService(this.id, skipChildren),
                oldDesiredState = this.desiredState;

            this.desiredState = STOP;

            // if something breaks, return desired
            // state to its previous value
            return promise.error(() => {
                this.desiredState = oldDesiredState;
            });
        }

        restart(skipChildren) {
            var promise = resourcesFactory.restartService(this.id, skipChildren),
                oldDesiredState = this.desiredState;

            this.desiredState = RESTART;

            // if something breaks, return desired
            // state to its previous value
            return promise.error(() => {
                this.desiredState = oldDesiredState;
            });
        }

        stopInstance(instance) {
            resourcesFactory.killRunning(instance.model.HostID, instance.id)
                .success(() => {
                    this.touch();
                })
                .error((data, status) => {
                    $notification.create("Stop Instance failed", data.Detail).error();
                });
        }


        // mark services updated to trigger render via $watch
        touch() {
            this.lastUpdate = new Date().getTime();
        }

        // kicks off request to update fast-moving instances and service state
        fetchAllStates() {
            return $q.all([this.fetchInstances(), this.getStatus()])
                .then(results => {
                    let statuses = results[1];
                    this.updateState(statuses);
                }, error => {
                    console.warn("Unable to fetch instance states");
                });
        }

        // update fast-moving service and instance state
        updateState(status) {

            // update service status
            this.desiredState = status.DesiredState;

            // update public endpoints
            if (this.publicEndpoints) {
                this.publicEndpoints.forEach(ept => {
                    if (ept.ServiceID === this.id) {
                        // TODO - dont modify model data
                        ept.desiredState = this.desiredState;
                    } else {
                        // TODO - deal with public endpoints which
                        // are descendents rather than children
                    }
                });
            }
            let statusMap = status.Status.reduce((map, s) => {
                map[s.InstanceID] = s;
                return map;
            }, {});

            // update instance status
            this.instances.forEach(i => {
                let s = statusMap[i.model.InstanceID];
                if (s) {
                    i.updateState(s);
                } else {
                    console.log(`Could not find status for instance ${i.id}`);
                }
            });

            // TODO: pass myself into health status and get my health status back
            serviceHealth.update({ [this.id]: this });
            this.status = serviceHealth.get(this.id);
            this.touch();
        }

    }

    ServiceFactory.$inject = ['$notification', '$serviceHealth', '$q', 'resourcesFactory', 'miscUtils', 'Instance'];
    function ServiceFactory(_$notification, _serviceHealth, _$q, _resourcesFactory, _utils, _Instance) {

        // api access via angular context
        $notification = _$notification;
        serviceHealth = _serviceHealth;
        $q = _$q;
        resourcesFactory = _resourcesFactory;
        utils = _utils;
        Instance = _Instance;

        return Service;

    }
})();
