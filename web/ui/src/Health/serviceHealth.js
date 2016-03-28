/* jshint multistr: true */
(function() {
    'use strict';

	// OK means health check is passing
	const OK = "passed";
	// Failed means health check is responsive, but failing
	const FAILED = "failed";
	// Timeout means health check is non-responsive in the given time
    // TODO - handle timeout status
	const TIMEOUT = "timeout";
	// NotRunning means the instance is not running
	const NOT_RUNNING = "not_running";
	// Unknown means the instance hasn't checked in within the provided time
	// limit.
	const UNKNOWN = "unknown";

    let serviceHealthModule = angular.module('serviceHealth', []);

    // share constants for other packages to use
    serviceHealthModule.value("hcStatus", {
        OK: OK,
        FAILED: FAILED,
        TIMEOUT: TIMEOUT,
        NOT_RUNNING: NOT_RUNNING,
        UNKNOWN: UNKNOWN
    });

    serviceHealthModule.factory("$serviceHealth", ["$rootScope", "resourcesFactory", "$translate",
    function($rootScope, resourcesFactory, $translate){

        var statuses = {};
        var serviceHealths = {};

        // updates health check data for all services
        function update(serviceList) {

            var serviceHealthCheck, instanceHealthCheck,
                serviceStatus, instanceStatus, instanceUniqueId,
                instance, service;

            statuses = {};

            // iterate services healthchecks
            for(var serviceId in serviceList){
                service = serviceList[serviceId];
                serviceHealthCheck = serviceHealths[serviceId];
                serviceStatus = new Status(
                    serviceId,
                    service.name,
                    service.model.DesiredState,
                    service.model.Instances);

                // if no healthcheck for this service mark as not running
                if(!serviceHealthCheck){
                    serviceStatus.statusRollup.incNotRunning();
                    serviceStatus.evaluateStatus();

                // otherwise, look for instances
                } else {

                    // iterate instances healthchecks
                    for(var instanceId in serviceHealthCheck){
                        instanceHealthCheck = serviceHealthCheck[instanceId];
                        instanceUniqueId = serviceId +"."+ instanceId;
                        // evaluate the status of this instance
                        instanceStatus = new Status(
                            instanceUniqueId,
                            service.name +" "+ instanceId,
                            service.model.DesiredState,
                            service.model.Instances);

                        instanceStatus.evaluateHealthChecks(instanceHealthCheck);

                        // attach status to instance
                        instance = service.instances.find(instance => instance.id === instanceId);
                        if(instance){
                            instance.status = instanceStatus;
                        }

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

            return statuses;
        }

        // used by Status to examine children and figure
        // out what the parent's status is
        function StatusRollup(){
            this[OK] = 0;
            this[FAILED] = 0;
            this[NOT_RUNNING] = 0;
            this[UNKNOWN] = 0;
            this.total = 0;
        }
        StatusRollup.prototype = {
            constructor: StatusRollup,

            incOK: function(){
                this.incStatus(OK);
            },
            incFailed: function(){
                this.incStatus(FAILED);
            },
            incNotRunning: function(){
                this.incStatus(NOT_RUNNING);
            },
            incUnknown: function(){
                this.incStatus(UNKNOWN);
            },
            incStatus: function(status){
                if(this[status] !== undefined){
                    this[status]++;
                    this.total++;
                }
            },

            // TODO - use assertion style ie: status.is.ok() or status.any.ok()
            anyFailed: function(){
                return !!this[FAILED];
            },
            allFailed: function(){
                return this.total && this[FAILED] === this.total;
            },
            anyOK: function(){
                return !!this[OK];
            },
            allOK: function(){
                return this.total && this[OK] === this.total;
            },
            anyNotRunning: function(){
                return !!this[NOT_RUNNING];
            },
            allNotRunning: function(){
                return this.total && this[NOT_RUNNING] === this.total;
            },
            anyUnknown: function(){
                return !!this[UNKNOWN];
            },
            allUnknown: function(){
                return this.total && this[UNKNOWN] === this.total;
            }
        };

        function Status(id, name, desiredState, numInstances){
            this.id = id;
            this.name = name;
            this.desiredState = desiredState;
            this.numInstances = numInstances;

            this.statusRollup = new StatusRollup();
            this.children = [];

            this.status = null;
            this.description = null;
        }

        Status.prototype = {
            constructor: Status,

            // distill this service's statusRollup into a single value
            evaluateStatus: function(){
                if(this.desiredState === 1){
                    // if any failing, bad!
                    if(this.statusRollup.anyFailed()){
                        this.status = FAILED;
                        this.description = $translate.instant("failing_health_checks");

                    // if any notRunning, oh no!
                    } else if(this.statusRollup.anyNotRunning()){
                        this.status = UNKNOWN;
                        this.description = $translate.instant("starting_service");

                    // if all are ok, yay! ok!
                    } else if(this.statusRollup.allOK()){
                        this.status = OK;
                        this.description = $translate.instant("passing_health_checks");

                    // some health checks are late
                    } else {
                        this.status = UNKNOWN;
                        this.description = $translate.instant("missing_health_checks");
                    }

                } else if(this.desiredState === 0){
                    // should be notRunning, but is still passing... weird
                    if(this.statusRollup.anyOK()){
                        this.status = UNKNOWN;
                        this.description = $translate.instant("stopping_service");

                    // stuff is notRunning as expected
                    } else {
                        this.status = NOT_RUNNING;
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
            evaluateHealthChecks: function(healthChecks){
                var status;

                this.statusRollup = new StatusRollup();

                for(var healthCheck in healthChecks){
                    //status = evaluateHealthCheck(healthChecks[healthCheck], timestamp);
                    status = healthChecks[healthCheck];

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

        return {
            update: update,
            setInstanceHealth: function(instance, healthChecks){
                let serviceHealth, instanceHealth;

                if(!serviceHealths[instance.model.ServiceID]){
                    serviceHealths[instance.model.ServiceID] = {};
                }
                serviceHealth = serviceHealths[instance.model.ServiceID];

                instanceHealth = {};
                for(var name in healthChecks){
                    instanceHealth[name] = healthChecks[name].Status;
                }
                serviceHealth[instance.id] = instanceHealth;
            },
            get: function(id){
                var status = statuses[id];

                // if no status, return a stubbed out
                // bad status
                // HACK: this should be improved/fixed
                if(!status){
                    status = new Status(id, FAILED, 0, 1);
                    status.evaluateStatus();
                }

                return status;
            }
        };
    }]);

})();
