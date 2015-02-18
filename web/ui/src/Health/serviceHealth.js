/* jshint multistr: true */
(function() {
    'use strict';

    angular.module('serviceHealth', []).
    factory("$serviceHealth", ["$rootScope", "$q", "resourcesFactory", "$translate",
    function($rootScope, $q, resourcesFactory, $translate){

        var statuses = {};

        // updates health check data for all services
        function update(serviceList) {

            // TODO - these methods should return promises, but they
            // don't so use our own promises
            var servicesDeferred = $q.defer();
            var healthCheckDeferred = $q.defer();

            // TODO - deal with serviceList in a better way
            servicesDeferred.resolve(serviceList);

            resourcesFactory.get_service_health(function(healthChecks){
                healthCheckDeferred.resolve(healthChecks);
            });

            return $q.all({
                services: servicesDeferred.promise,
                health: healthCheckDeferred.promise
            }).then(function(results){
                var serviceHealthCheck, instanceHealthCheck,
                    serviceStatus, instanceStatus, instanceUniqueId,
                    statuses = {};

                // iterate services healthchecks
                for(var serviceId in results.services){
                    serviceHealthCheck = results.health.Statuses[serviceId];
                    serviceStatus = new Status(serviceId, results.services[serviceId].name, results.services[serviceId].model.DesiredState, results.services[serviceId].model.Instances);

                    // if no healthcheck for this service mark as down
                    if(!serviceHealthCheck){
                        serviceStatus.statusRollup.incDown();
                        serviceStatus.evaluateStatus();

                    // otherwise, look for instances
                    } else {

                        // iterate instances healthchecks
                        for(var instanceId in serviceHealthCheck){
                            instanceHealthCheck = serviceHealthCheck[instanceId];
                            instanceUniqueId = serviceId +"."+ instanceId;
                            // evaluate the status of this instance
                            instanceStatus = new Status(instanceUniqueId, results.services[serviceId].name +" "+ instanceId, results.services[serviceId].model.DesiredState, results.services[serviceId].model.Instances);
                            instanceStatus.evaluateHealthChecks(instanceHealthCheck, results.health.Timestamp);
                            
                            // add this guy's statuses to hash map for easy lookup
                            statuses[instanceUniqueId] = instanceStatus;
                            // add this guy's status to his parent
                            serviceStatus.children.push(instanceStatus);
                        }
                        
                        // now that this services instances have been evaluated,
                        // evaluate the status of this service
                        serviceStatus.evaluateChildren();
                    }

                    statuses[serviceId] = serviceStatus;
                }

                // NOTE: resolves returned promise with statuses object
                return statuses;

            }).catch(function(err){
                // something went awry
                console.log("Promise err", err);
            });
        }

        // used by Status to examine children and figure
        // out what the parent's status is
        function StatusRollup(){
            this.good = 0;
            this.bad = 0;
            this.down = 0;
            this.unknown = 0;
            this.total = 0;
        }
        StatusRollup.prototype = {
            constructor: StatusRollup,

            incGood: function(){
                this.incStatus("good");
            },
            incBad: function(){
                this.incStatus("bad");
            },
            incDown: function(){
                this.incStatus("down");
            },
            incUnknown: function(){
                this.incStatus("unknown");
            },
            incStatus: function(status){
                if(this[status] !== undefined){
                    this[status]++;
                    this.total++;
                }
            },

            // TODO - use assertion style ie: status.is.good() or status.any.good()
            anyBad: function(){
                return !!this.bad;
            },
            allBad: function(){
                return this.total && this.bad === this.total;
            },
            anyGood: function(){
                return !!this.good;
            },
            allGood: function(){
                return this.total && this.good === this.total;
            },
            anyDown: function(){
                return !!this.down;
            },
            allDown: function(){
                return this.total && this.down === this.total;
            },
            anyUnknown: function(){
                return !!this.unknown;
            },
            allUnknown: function(){
                return this.total && this.unknown === this.total;
            }
        };

        function Status(id, name, desiredState, numInstances){
            this.id = id;
            this.name = name;
            this.desiredState = desiredState;
            this.numInstances = numInstances;

            this.statusRollup = new StatusRollup();
            this.children = [];

            // bad, good, unknown, down
            // TODO - use enum or constants for statuses
            this.status = null;
            this.description = null;
        }

        Status.prototype = {
            constructor: Status,

            // distill this service's statusRollup into a single value
            evaluateStatus: function(){
                if(this.desiredState === 1){
                    // if any failing, bad!
                    if(this.statusRollup.anyBad()){
                        this.status = "bad";
                        this.description = $translate.instant("failing_health_checks");

                    // if any down, oh no!
                    } else if(this.statusRollup.anyDown()){
                        this.status = "unknown";
                        this.description = $translate.instant("starting_service");

                    // if all are good, yay! good!
                    } else if(this.statusRollup.allGood()){
                        this.status = "good";
                        this.description = $translate.instant("passing_health_checks");
                    
                    // some health checks are late
                    } else {
                        this.status = "unknown";
                        this.description = $translate.instant("missing_health_checks");
                    }

                } else if(this.desiredState === 0){
                    // should be down, but is still passing... weird
                    if(this.statusRollup.anyGood()){
                        this.status = "unknown";
                        this.description = $translate.instant("stopping_service");

                    // stuff is down as expected
                    } else {
                        this.status = "down";
                        this.description = $translate.instant("container_down");
                    }
                }
            },

            // roll up child status into this status
            evaluateChildren: function(){

                this.statusRollup = this.children.reduce(function(acc, childStatus){
                    acc.incStatus(childStatus.status);
                    return acc;
                }.bind(this), new StatusRollup());

                // if total doesn't match numInstances, then some
                // stuff is missing! mark unknown!
                if(this.numInstances !== undefined && this.numInstances >= this.statusRollup.total){
                    this.statusRollup.unknown += this.numInstances - this.statusRollup.total;
                    this.statusRollup.total = this.numInstances; 
                }

                this.evaluateStatus();
            },

            // set this status's statusRollup based on healthchecks
            evaluateHealthChecks: function(healthChecks, timestamp){
                var status;

                this.statusRollup = new StatusRollup();

                for(var healthCheck in healthChecks){
                    status = evaluateHealthCheck(healthChecks[healthCheck], timestamp);

                    // this is a healthcheck status object... kinda weird...
                    this.children.push({
                        name: healthCheck,
                        status: status
                    });
                    
                    // add this guy's status to the total
                    this.statusRollup.incStatus(status);
                }

                this.evaluateStatus();
            },

        };

        // determine the health of a healthCheck based on start time, 
        // up time and healthcheck
        function evaluateHealthCheck(hc, timestamp){
            var status = {};

            // calculates the number of missed healthchecks since last start time
            var missedIntervals = (timestamp - Math.max(hc.Timestamp, hc.StartedAt)) / hc.Interval;

            // if service hasn't started yet
            if(hc.StartedAt === undefined){
                status = "down";
            
            // if healthCheck has missed 2 updates, mark unknown
            } else if (missedIntervals > 2 && missedIntervals < 60) {
                status = "unknown";

            // if healthCheck has missed 60 updates, mark failed
            } else if (missedIntervals > 60) {
                status = "bad";

            // if Status is passed, then good!
            } else if(hc.Status === "passed") {
                status = "good";

            // if Status is failed, then bad!
            } else if(hc.Status === "failed") {
                status = "bad";

            // otherwise I have no idea
            } else {
                status = "unknown";
            }

            return status;
        }

        return {
            update: update,
            get: function(id){
                return statuses[id];
            }
        };
    }]);

})();
