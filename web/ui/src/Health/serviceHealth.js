/* jshint multistr: true */
(function() {
    'use strict';

	// OK means health check is passing
	const OK = "passed";
	// Failed means health check is responsive, but failing
	const FAILED = "failed";
	// Timeout means health check is non-responsive in the given time
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

    serviceHealthModule.factory("$serviceHealth", ["$translate",
    function($translate){

        var statuses = {};

        // updates health check data for all services
        function update(serviceList) {

            var serviceStatus, instanceStatus, instanceUniqueId, service;

            statuses = {};

            // iterate services healthchecks
            for(var serviceId in serviceList){
                service = serviceList[serviceId];
                serviceStatus = new Status(
                    serviceId,
                    service.name,
                    service.model.DesiredState);

                // refresh list of instances
                service.getServiceInstances();

                // if this service has instances, evaluate their health
                service.instances.forEach(instance => {

                    // create a new status rollup for this instance
                    instanceUniqueId = serviceId +"."+ instance.id;
                    instanceStatus = new Status(
                        instanceUniqueId,
                        service.name +" "+ instance.id,
                        service.model.DesiredState);

                    // evalute instance healthchecks and roll em up
                    instanceStatus.evaluateHealthChecks(instance.healthChecks);
                    // store resulting status on instance
                    instance.status = instanceStatus;

                    // add this guy's statuses to hash map for easy lookup
                    statuses[instanceUniqueId] = instanceStatus;
                    // add this guy's status to his parent
                    serviceStatus.children.push(instanceStatus);
                });

                // now that this services instances have been evaluated,
                // evaluate the status of this service
                serviceStatus.evaluateChildren();

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

        function Status(id, name, desiredState){
            this.id = id;
            this.name = name;
            this.desiredState = desiredState;

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
                    // shouldnt be running, but still getting health checks,
                    // so probably stopping
                    if(this.statusRollup.anyOK() || this.statusRollup.anyFailed() ||
                            this.statusRollup.anyUnknown()){
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
                this.evaluateStatus();
            },

            // set this status's statusRollup based on healthchecks
            // NOTE - subtly different than evaluateChildren
            evaluateHealthChecks: function(healthChecks){
                for(var name in healthChecks){
                    this.statusRollup.incStatus(healthChecks[name]);
                    this.children.push({
                        name: name,
                        status: healthChecks[name]
                    });
                }
                this.evaluateStatus();
            },

        };

        return {
            update: update,
            get: function(id){
                var status = statuses[id];

                // if no status found, return unknown
                if(!status){
                    status = new Status(id, UNKNOWN, 0);
                    status.evaluateStatus();
                }

                return status;
            }
        };
    }]);

})();
