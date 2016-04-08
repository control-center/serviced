// servicesFactory
// - maintains a list of services and keeps it in sync with the backend.
// - links services with their parents and children
// - aggregates and caches service information (such as descendents)
(function() {
    'use strict';

    var resourcesFactory, $q, serviceHealth, instancesFactory, utils;

    angular.module('servicesFactory', []).
    factory("servicesFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "$serviceHealth", "instancesFactory", "baseFactory", "miscUtils",
    function($rootScope, q, _resourcesFactory, $interval, _serviceHealth, _instancesFactory, BaseFactory, _utils){

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        instancesFactory = _instancesFactory;
        serviceHealth = _serviceHealth;
        utils = _utils;
        $q = q;

        var UPDATE_PADDING = 1000;

        var newFactory = new BaseFactory(Service, resourcesFactory.getServices);

        // alias some stuff for ease of use
        newFactory.serviceTree = newFactory.objArr;
        newFactory.serviceMap = newFactory.objMap;

        angular.extend(newFactory, {
            // TODO - update list by application instead
            // of all services ever?
            update: function(force, skipUpdateInstances){
                var deferred = $q.defer(),
                    now = new Date().getTime(),
                    requestTime = now,
                    since;

                // if this is the first update, request
                // all services
                if(this.lastRequest === undefined || force){
                    since = 0;

                // request all data since the last request
                // was made to ensure any new data that came
                // in DURING the request is filled
                } else {
                    since = (now - this.lastRequest) + UPDATE_PADDING;
                }

                resourcesFactory.getServices(since)
                    .success((data, status) => {
                        // TODO - change backend to send
                        // updated, created, and deleted
                        // separately from each other
                        data.forEach((serviceDef) => {
                            var currentParent, service;

                                // update
                                if(this.serviceMap[serviceDef.ID]){
                                    service = this.serviceMap[serviceDef.ID];
                                    currentParent = service.parent;

                                    // if the service parent has changed,
                                    // update its tree stuff (parent, depth, etc)
                                    if(currentParent && serviceDef.ParentServiceID !== service.parent.id){
                                        this.serviceMap[serviceDef.ID].update(serviceDef);
                                        this.addServiceToTree(service);

                                    // otherwise, just update the service
                                    } else {
                                        this.serviceMap[serviceDef.ID].update(serviceDef);
                                    }

                            // new
                            } else {
                                this.serviceMap[serviceDef.ID] = new Service(serviceDef);
                                this.addServiceToTree(this.serviceMap[serviceDef.ID]);
                            }

                            // TODO - deleted service

                        });

                        // check to see if orphans found parents
                        for(let id in this.serviceMap){
                            if(this.serviceMap[id].isOrphan){
                                this.addServiceToTree(this.serviceMap[id]);
                            }
                        }

                        // HACK - services should update themselves?
                        this.updateHealth();

                        // notify the first services request is done
                        $rootScope.$emit("ready");

                        // time last SUCCESSFUL request began
                        this.lastRequest = requestTime;
                        this.lastUpdate = new Date().getTime();

                        deferred.resolve();
                    });

                // keep instances up to date
                if(!skipUpdateInstances){
                    instancesFactory.update();
                }

                return deferred.promise;
            },

            // adds a service object to the service tree
            // in the appropriate place
            addServiceToTree: function(service){
                var parent;
                // if this is not a top level service
                if(service.model.ParentServiceID){
                    parent = this.serviceMap[service.model.ParentServiceID];

                    // if the parent isn't available, mark
                    // as an orphaned service and early return
                    if(!parent){
                        service.isOrphan = true;
                        return;
                    }

                    service.isOrphan = false;

                    parent.addChild(service);

                // if this is a top level service
                } else {
                    this.serviceTree.push(service);
                    //this.serviceTree.sort(sortServicesByName);
                }

                // ICKY GROSS HACK!
                // iterate tree and store tree depth on
                // individual services
                // TODO - find a more elegant way to keep track of depth
                // TODO - remove apps from service tree if they get a parent
                this.serviceTree.forEach(function(topService){
                    topService.depth = 0;
                    topService.children.forEach(function recurse(service){
                        service.depth = service.parent.depth + 1;
                        service.children.forEach(recurse);
                    });
                });
            },

            // TODO - debounce this guy
            updateHealth: function(){
                let statuses = serviceHealth.update(this.serviceMap);
                for(var id in statuses){
                    // attach status to associated service
                    if(this.serviceMap[id]){
                        this.serviceMap[id].status = statuses[id];
                    }
                }
            }
        });

        return newFactory;
    }]);

    function sortServicesByName(a, b){
        return a.name.toLowerCase() < b.name.toLowerCase() ? -1 : 1;
    }

    // Service object constructor
    // takes a service object (backend service object)
    // and wraps it with extra functionality and info
    function Service(service, parent){
        this.parent = parent;
        this.children = [];
        this.instances = [];

        // tree depth
        this.depth = 0;

        // cache for computed values
        this.cache = new Cache(["addresses", "descendents", "publicEndpoints", "exportedServiceEndpoints"]);

        this.resources = {
            RAMCommitment: 0,
            RAMAverage: 0
        };

        this.update(service);

        // this newly created child should be
        // registered with its parent
        // TODO - this makes parent update twice...
        if(this.parent){
            this.parent.addChild(this);
        }
    }


    // service types
    // internal service
    var ISVC = "isvc",
        // a service with no parent
        APP = "app",
        // a service with children but no
        // startup command
        META = "meta",
        // a service who's parent is still
        // being deployed
        DEPLOYING = "deploying";

    // DesiredState enum
    var START = 1,
        STOP = 0,
        RESTART = -1;

    Service.prototype = {
        constructor: Service,

        // updates the immutable service object
        // and marks any computed properties dirty
        update: function(service){
            if(service){
                this.updateServiceDef(service);
            }

            // update service health
            // TODO - should service update itself, its controller
            // update the service, or serviceHealth update all services?
            this.status = serviceHealth.get(this.id);

            this.evaluateServiceType();

            // invalidate caches
            this.markDirty();

            // notify parent that this is now dirty
            if(this.parent){
                this.parent.update();
            }
        },

        updateServiceDef: function(service){
            // these properties are for convenience
            this.name = service.Name;
            this.id = service.ID;
            // NOTE: desiredState is mutable to improve UX
            this.desiredState = service.DesiredState;

            // make service immutable
            this.model = Object.freeze(service);

        },

        // invalidate all caches. This is needed
        // when descendents update
        markDirty: function(){
            this.cache.markAllDirty();
        },

        evaluateServiceType: function(){
            // infer service type
            this.type = [];
            if(this.model.ID.indexOf("isvc-") !== -1){
                this.type.push(ISVC);
            }

            if(!this.model.ParentServiceID){
                this.type.push(APP);
            }

            if(this.children.length && !this.model.Startup){
                this.type.push(META);
            }

            if(this.parent && this.parent.isDeploying()){
                this.type.push(DEPLOYING);
            }
        },

        addChild: function(service){
            // if this service is not already in the list
            if(this.children.indexOf(service) === -1){
                this.children.push(service);

                // make sure this child knows who
                // its parent is
                service.setParent(this);

                // alpha sort children
                this.children.sort(sortServicesByName);

                this.update();
            }
        },

        removeChild: function(service){
            var childIndex = this.children.indexOf(service);

            if(childIndex !== -1){
                this.children.splice(childIndex, 1);
            }
            this.update();
        },

        setParent: function(service){
            // if this is already the parent, IM OUT!
            if(this.parent === service){
                return;
            }

            // if a parent is already set, remove
            // this service from its childship
            if(this.parent){
                this.parent.removeChild(this);
            }

            this.parent = service;
            this.parent.addChild(this);
            this.update();
        },

        // start, stop, or restart this service
        start: function(skipChildren){
            var promise = resourcesFactory.startService(this.id, skipChildren),
                oldDesiredState = this.desiredState;

            this.desiredState = START;

            // if something breaks, return desired
            // state to its previous value
            return promise.error(() => {
                this.desiredState = oldDesiredState;
            });
        },
        stop: function(skipChildren){
            var promise = resourcesFactory.stopService(this.id, skipChildren),
                oldDesiredState = this.desiredState;

            this.desiredState = STOP;

            // if something breaks, return desired
            // state to its previous value
            return promise.error(() => {
                this.desiredState = oldDesiredState;
            });
        },
        restart: function(skipChildren){
            var promise = resourcesFactory.restartService(this.id, skipChildren),
                oldDesiredState = this.desiredState;

            this.desiredState = RESTART;

            // if something breaks, return desired
            // state to its previous value
            return promise.error(() => {
                this.desiredState = oldDesiredState;
            });
        },

        // gets a list of running instances of this service.
        // NOTE: this isn't using a cache because this can be
        // invalidated at any time, so it should always be checked
        getServiceInstances: function(){
            this.instances = instancesFactory.getByServiceId(this.id);
            this.instances.sort(function(a,b) {
                return a.model.InstanceID > b.model.InstanceID;
            });
            return this.instances;
        },

        resourcesGood: function() {
            var instances = this.getServiceInstances();
            for (var i = 0; i < instances.length; i++) {
                if (!instances[i].resourcesGood()) {
                    return false;
                }
            }
            return true;
        },

        // some convenience methods
        isIsvc: function(){
            return !!~this.type.indexOf(ISVC);
        },

        isApp: function(){
            return !!~this.type.indexOf(APP);
        },

        isDeploying: function(){
            return !!~this.type.indexOf(DEPLOYING);
        },

        // HACK: this is a temporary fix to mark
        // services deploying.
        markDeploying: function(){
            this.type.push(DEPLOYING);
        },

        // if any cache is dirty, this whole object
        // is dirty
        isDirty: function(){
            return this.cache.anyDirty();
        },

        hasInstances: function(){
            return !!this.instances.length;
        }
    };

    Object.defineProperty(Service.prototype, "descendents", {
        get: function(){
            var descendents = this.cache.getIfClean("descendents");

            if(descendents){
                return descendents;
            }

            descendents = this.children.reduce(function(acc, child){
                acc.push(child);
                return acc.concat(child.descendents);
            }, []);

            Object.freeze(descendents);
            this.cache.cache("descendents", descendents);
            return descendents;
        }
    });

    Object.defineProperty(Service.prototype, "addresses", {
        get: function(){
            var addresses = this.cache.getIfClean("addresses");

            // if valid cache, early return it
            if(addresses){
                return addresses;
            }

            // otherwise, get some new data
            var services = this.descendents.slice();

            // we also want to see the Endpoints for this
            // service, so add it to the list
            services.push(this);

            // iterate services
            addresses = services.reduce(function(acc, service){

                var result = [];

                // if Endpoints, iterate Endpoints
                if(service.model.Endpoints){
                    result = service.model.Endpoints.reduce(function(acc, endpoint){
                        if (endpoint.AddressConfig.Port > 0 && endpoint.AddressConfig.Protocol) {
                            acc.push({
                                ID: endpoint.AddressAssignment.ID,
                                AssignmentType: endpoint.AddressAssignment.AssignmentType,
                                EndpointName: endpoint.AddressAssignment.EndpointName,
                                IPAddr: endpoint.AddressAssignment.IPAddr,
                                Port: endpoint.AddressConfig.Port,
                                HostID: endpoint.AddressAssignment.HostID,
                                PoolID: service.model.PoolID,
                                ServiceID: service.id,
                                ServiceName: service.name
                            });
                        }
                        return acc;
                    }, []);
                }

                return acc.concat(result);
            }, []);

            Object.freeze(addresses);
            this.cache.cache("addresses", addresses);
            return addresses;
        }
    });

    // fetch public endpoints for service and all descendents
    Object.defineProperty(Service.prototype, "publicEndpoints", {
        get: function(){
            var publicEndpoints = this.cache.getIfClean("publicEndpoints");

            // if valid cache, early return it
            if(publicEndpoints){
                return publicEndpoints;
            }

            // otherwise, get some data
            var services = this.descendents.slice();

            // we also want to see the Endpoints for this
            // service, so add it to the list
            services.push(this);

            // iterate services
            publicEndpoints = services.reduce(function(acc, service){

                var result = [];

                // if Endpoints, iterate Endpoints
                if(service.model.Endpoints){
                    result = service.model.Endpoints.reduce(function(acc, endpoint){
                        // if VHosts, iterate VHosts
                        if(endpoint.VHostList){
                            endpoint.VHostList.forEach(function(VHost){
                                acc.push({
                                    Name: VHost.Name,
                                    Enabled: VHost.Enabled,
                                    Application: service.name,
                                    ServiceEndpoint: endpoint.Application,
                                    ApplicationId: service.id,
                                    Value: service.name +" - "+ endpoint.Application,
                                    type: "vhost",
                                });
                            });
                        }
                        // if ports, iterate ports
                        if(endpoint.PortList){
                            endpoint.PortList.forEach(function(port){
                                acc.push({
                                    PortAddr: port.PortAddr,
                                    Enabled: port.Enabled,
                                    Application: service.name,
                                    ServiceEndpoint: endpoint.Application,
                                    ApplicationId: service.id,
                                    Value: service.name +" - "+ endpoint.Application,
                                    type: "port",
                                });
                            });
                        }

                        return acc;
                    }, []);
                }

                return acc.concat(result);
            }, []);

            Object.freeze(publicEndpoints);
            this.cache.cache("publicEndpoints", publicEndpoints);
            return publicEndpoints;
        }
    });

    // fetch public endpoints for service and all descendents
    Object.defineProperty(Service.prototype, "exportedServiceEndpoints", {
        get: function(){
            var exportedServiceEndpoints = this.cache.getIfClean("exportedServiceEndpoints");

            // if valid cache, early return it
            if(exportedServiceEndpoints){
                return exportedServiceEndpoints;
            }

            // otherwise, get some data
            var services = this.descendents.slice();

            // we also want to see the Endpoints for this
            // service, so add it to the list
            services.push(this);

            // iterate services
            exportedServiceEndpoints = services.reduce(function(acc, service){
                var result = [];

                // if Endpoints, iterate Endpoints
                if(service.model.Endpoints){
                    result = service.model.Endpoints.reduce(function(acc, endpoint){

                        // if this exports tcp, add it to our list.
                        if(endpoint.Purpose === "export" && endpoint.Protocol === "tcp"){
                            acc.push({
                                Application: service.name,
                                ServiceEndpoint: endpoint.Application,
                                ApplicationId: service.id,
                                Value: service.name +" - "+ endpoint.Application,
                            });
                        }

                        return acc;
                    }, []);
                }

                return acc.concat(result);
            }, []);

            Object.freeze(exportedServiceEndpoints);
            this.cache.cache("exportedServiceEndpoints", exportedServiceEndpoints);
            return exportedServiceEndpoints;
        }
    });

    // simple cache object
    // TODO - angular has this sorta stuff built in
    function Cache(caches){
        this.caches = {};
        if(caches){
            caches.forEach(function(name){
                this.addCache(name);
            }.bind(this));
        }
    }
    Cache.prototype = {
        constructor: Cache,
        addCache: function(name){
            this.caches[name] = {
                data: null,
                dirty: false
            };
        },
        markDirty: function(name){
            this.mark(name, true);
        },
        markAllDirty: function(){
            for(var key in this.caches){
                this.markDirty(key);
            }
        },
        markClean: function(name){
            this.mark(name, false);
        },
        markAllClean: function(){
            for(var key in this.caches){
                this.markClean(key);
            }
        },
        cache: function(name, data){
            this.caches[name].data = data;
            this.caches[name].dirty = false;
        },
        get: function(name){
            return this.caches[name].data;
        },
        getIfClean: function(name){
            if(!this.caches[name].dirty){
                return this.caches[name].data;
            }
        },
        mark: function(name, flag){
            this.caches[name].dirty = flag;
        },
        isDirty: function(name){
            return this.caches[name].dirty;
        },
        anyDirty: function(){
            for(var i in this.caches){
                if(this.caches[i].dirty){
                    return true;
                }
            }
            return false;
        }
    };


})();
