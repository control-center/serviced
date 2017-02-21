(function () {
    'use strict';

    // share angular services outside of angular context
    let $notification, serviceHealth, $q, resourcesFactory, utils, Instance;

    const MAX_REQUEST_IDS = 15;

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
        service.subservices.forEach( function(svc) {
            getDescendents(descendents, svc);
            descendents[svc.id] = svc;
        });
        return descendents;
    }

    class Service {

        constructor(model) {
            this.subservices = [];
            this.instances = [];
            this.publicEndpoints = [];
            this.update(model);
        }

        update(model) {

            // basically new-up with exisiting children and instances
            this.name = model.Name;
            this.id = model.ID;
            this.currentState = model.CurrentState;
            this.desiredState = model.DesiredState;
            this.emergencyShutdown = model.EmergencyShutdown;
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
            let deferred = $q.defer();
            let statuses;
            // kick off request for statuses for public endpoint
            // service statuses
            this.getEndpointStatuses()
                .then(s => {
                    s = s || [];
                    // convert array of endpoint service statuses
                    // to a map of serviceid -> endpoint status
                    statuses = s.reduce((acc, s) => {
                        acc[s.ServiceID] = s;
                        return acc;
                    },{});
                })
                // kick off request for public endpoints
                .then(() => {
                    return resourcesFactory.v2.getServicePublicEndpoints(this.id);
                })
                .then(response => {
                    this.publicEndpoints = response.map(ept => {
                        // TODO - dont modify model data :(
                        ept.Value = `${ept.ServiceName} - ${ept.Application}`;
                        // if this endpoint has a service status,
                        // add that service's desiredState so that
                        // the endpoint knows if its accessible or not
                        if(statuses[ept.ServiceID]){
                            ept.desiredState = statuses[ept.ServiceID].DesiredState;
                        }
                        return ept;
                    });
                    deferred.resolve();
                })
                .catch(error => {
                    console.warn(error);
                    deferred.reject();
                });
            return deferred.promise;
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

        getEndpointStatuses(){
            let eps = this.publicEndpoints.reduce((acc, ep) => {
                acc[ep.ServiceID] = ep;
                return acc;
            }, {});
            let ids = Object.keys(eps);
            return this.getServiceStatuses(ids);
        }

        // fetch and update service statuses for all
        // descendents of this service
        updateDescendentStatuses() {
            let deferred = $q.defer();
            let descendents = getDescendents({}, this);
            let ids = Object.keys(descendents);
            this.getServiceStatuses(ids)
                .then(results => {
                    if (results && results.length) {
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

        // the list of ids can get extrememly long, which
        // can push the acceptable limits of URL length,
        // so split the request up
        getServiceStatuses(ids){
            let promises = [];

            let count = Math.ceil(ids.length / MAX_REQUEST_IDS);
            for(let i = 0; i < count; i++){
                let start = i * MAX_REQUEST_IDS;
                // either slice to the end of this array, or
                // the next MAX_REQUEST_IDS elements
                let end = Math.min(start + MAX_REQUEST_IDS, ids.length);
                let idsSlice = ids.slice(start, end);
                promises.push(resourcesFactory.v2.getServiceStatuses(idsSlice));
            }

            // NOTE - this will cancel all on the first failure
            // which may not be the desired behavior
            let all = $q.all(promises).then(results => {
                // results will be an array of results, one
                // for each promise. the caller is expecting
                // a single array of results, so lets squish
                // em all together
                let statuses = results.reduce((arr, r) => arr.concat(r), []);
                return $q.when(statuses);
            })
            .catch(err => {
                console.warn("failure getting service statuses", err);
            });

            // TODO - optionally return [all, promises]
            return all;
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

        isContainer() {
            return this.hasChildren() && !this.model.Startup;
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
                    let detail = data.Detail || data || "";
                    $notification.create("Stop Instance failed", detail).error();
                });
        }

        cancelPending() {
            if (this.currentState === "pending_start") {
                return this.stop();
            } else if (this.currentState === "pending_stop") {
                return this.start();
            } else if (this.currentState === "pending_restart") {
                return this.start();
            }
        }


        // mark services updated to trigger render via $watch
        touch() {
            this.lastUpdate = new Date().getTime();
        }

        // kicks off request to update fast-moving instances and service state
        fetchAllStates() {
            return $q.all([
                this.fetchInstances(),
                this.getStatus(),
                this.getEndpointStatuses()])
                .then(results => {
                    let myStatus = results[1],
                        otherStatuses = results[2];
                    this.updateState(myStatus, otherStatuses);
                }, error => {
                    console.warn("Unable to fetch instance states");
                });
        }

        // update fast-moving service and instance state
        updateState(myStatus, otherStatuses) {

            // update service status
            this.desiredState = myStatus.DesiredState;
            this.currentState = myStatus.CurrentState;
            this.emergencyShutdown = myStatus.EmergencyShutdown;

            // update public endpoints
            if(otherStatuses){
                otherStatuses = otherStatuses.reduce((acc, s) => {
                    acc[s.ServiceID] = s;
                    return acc;
                },{});
                this.publicEndpoints.forEach(ept => {
                    ept.desiredState = otherStatuses[ept.ServiceID].DesiredState;
                });
            }

            let instanceMap = this.instances.reduce((map, i) => {
                map[i.model.InstanceID] = i;
                return map;
            }, {});

            // iterate instance health and either update the service
            // instance with the health, or create a temporary instace
            // object to hold the health for evaluation
            let instances = [];
            myStatus.Status.forEach(instanceStatus => {
                let instance = instanceMap[instanceStatus.InstanceID];
                if(!instance){
                    // create a temporary instance object to
                    // use the status object
                    instance = new Instance({
                        InstanceID: instanceStatus.InstanceID,
                        ServiceID: this.id,
                        HealthStatus: instanceStatus.HealthStatus,
                        MemoryUsage: instanceStatus.MemoryUsage,
                        // NOTE - fake values!
                        HostID: 0,
                        RAMCommitment: myStatus.RAMCommitment
                    });
                }
                instance.updateState(instanceStatus);
                instances.push(instance);
            });

            // get health for this service and its instances
            this.status = serviceHealth.evaluate(this, instances);
            this.resourcesGood = true;
            for (var i = 0; i < instances.length; i++) {
                if (!instances[i].resourcesGood()) {
                    this.resourcesGood = false;
		    break;
                }
            }

	    this.touch();
        }
    }

    // class methods
    Service.countAffectedDescendants = function(service, state){
        let deferred = $q.defer();
        resourcesFactory.v2.getDescendantCounts(service.id)
            .then((data, status) => {
                var count = 0;
                switch (state) {
                    case "start":
                        // When starting, we only care about autostart
                        // services that are currently stopped
                        if (data.auto) {
                            count += data.auto["0"] || 0;
                        }
                        break;
                    case "restart":
                    case "stop":
                        // When stopping or restarting, we care about
                        // running services that are either manual or
                        // autostart
                        if (data.auto) {
                            count += data.auto["1"] || 0;
                        }
                        if (data.manual) {
                            count += data.manual["1"] || 0;
                        }
                        break;
                }
                deferred.resolve(count);
            })
            .catch((data, status) => {
                deferred.reject(data);
            });
        return deferred.promise;
    };


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
