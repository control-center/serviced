/* global angular, console, $ */
/* jshint multistr: true */
(function() {
    'use strict';

    angular.module('serviceHealth', []).
    factory("$serviceHealth", ["$rootScope", "$q", "resourcesService", "$interval", "$translate",
    function($rootScope, $q, resourcesService, $interval, $translate){

        var servicesService = resourcesService;

        var STATUS_STYLES = {
            "bad": "glyphicon-exclamation-sign bad",
            "good": "glyphicon-ok-sign good",
            "unknown": "glyphicon-question-sign unknown",
            "down": "glyphicon-minus-sign disabled"
        };

        // simple array search util
        function findInArray(key, arr, val){
            for(var i = 0; i < arr.length; i++){
                if(arr[i][key] === val){
                    return arr[i];
                }
            }
        }

        // updates health check data for all services
        // `appId` is the id of the specific service being clicked
        function update(appId) {

            // TODO - these methods should return promises, but they
            // don't so use our own promises
            var servicesDeferred = $q.defer();
            var healthCheckDeferred = $q.defer();

            servicesService.update_services(function(top, mapped){
                servicesDeferred.resolve(mapped);
            });

            servicesService.get_service_health(function(healthChecks){
                healthCheckDeferred.resolve(healthChecks);
            });

            $q.all({
                services: servicesDeferred.promise,
                health: healthCheckDeferred.promise
            }).then(function(results){
                evaluateServiceStatus(results.services, results.health, appId);
            }).catch(function(err){
                // something went awry
                console.log("Promise err", err);
            });
        }

        function evaluateServiceStatus(services, healthCheckData, appId) {

            var healths = healthCheckData.Statuses,
                serverTimestamp = healthCheckData.Timestamp;

            var service, healthCheck, startTime,
                healthChecksRollup,
                tooltipDetails,
                serviceStatus, healthCheckStatus, healthCheckStatusIcon;

            for (var ServiceId in services) {

                service = services[ServiceId];
                healthCheck = healths[ServiceId];

                if(!service){
                    return;
                }

                // all the healthcheck statuses for this service
                // are rolled up into this to represent the health
                // of the entire service
                healthChecksRollup = {
                    passing: false,
                    failing: false,
                    unknown: true,
                    down: false
                };

                tooltipDetails = [];

                // determine the status of each individual healthcheck
                for (var name in healthCheck) {
                    // get the time this service was started
                    startTime = healthCheck[name].StartedAt;

                    healthCheckStatus = determineHealthCheckStatus(healthCheck[name], serverTimestamp, startTime);

                    switch(healthCheckStatus){
                        case "passed":
                            healthChecksRollup.passing = true;
                            healthCheckStatusIcon = "good";
                            break;
                        case "failed":
                            healthChecksRollup.failing = true;
                            healthCheckStatusIcon = "bad";
                            break;
                        case "unknown":
                            healthChecksRollup.unknown = true;
                            healthCheckStatusIcon = "unknown";
                            break;
                        case "down":
                            healthChecksRollup.down = true;
                            healthCheckStatusIcon = "down";
                            break;
                        default:
                            break;
                    }

                    // update tooltip details (per healthcheck)
                    tooltipDetails.push({
                        name: name,
                        status: healthCheckStatusIcon
                    });
                }

                serviceStatus = determineServiceStatus(service.DesiredState, healthChecksRollup);

                // TODO - only call this if statuses have changed since last tick
                updateServiceStatus(service, serviceStatus.status, serviceStatus.description, tooltipDetails);
            }

            // if a specific appId was provided, its status may not
            // yet be part of health checks, so give it unknown status
            if(appId && !findInArray("ServiceID", healths, appId)){

                // if this appId doesn't exist in the services list, then
                // something must be pretty messed up
                if(!services[appId]){
                    throw new Error("Service with id", appId, "does not exist");
                }

                console.log("patching in unknown status for "+ appId);
                
                updateServiceStatus(services[appId], "unknown", $translate.instant("container_unavailable"));
            }
        }

        // determines the overall health of the service by examining the status
        // of all of its healthchecks as well as the desired state of the service
        function determineServiceStatus(desiredState, healthChecksRollup){
            var status,
                description;

            // the following conditions are relevant when the service
            // *should* be started
            if(desiredState === 1){

                // service should be up, but is failing. bad!
                if(healthChecksRollup.failing){
                    status = "bad";
                    description = $translate.instant("failing_health_checks");

                // service should be up, but container has not
                // yet loaded
                } else if(healthChecksRollup.down){
                    status = "unknown";
                    description = $translate.instant("container_unavailable");

                // service should be up, but seems unresponsive
                // It could be just starting, or on its way down
                } else if(!healthChecksRollup.passing && healthChecksRollup.unknown){
                    status = "unknown";
                    description = $translate.instant("missing_health_checks");

                // service is up and healthy
                } else if(healthChecksRollup.passing && !healthChecksRollup.unknown){
                    status = "good";
                    description = $translate.instant("passing_health_checks");

                // TODO: This needs to be more representative of the health of a meta-service's children
                } else {
                    status = "good";
                    description = $translate.instant("passing_health_checks");
                }

            // the following conditions are relevant when the service
            // *should* be off
            } else if(desiredState === 0){

                // it should be off, but its still on... weird.
                if(healthChecksRollup.passing){
                    status = "unknown";
                    description = $translate.instant("stopping_service");
                    // TODO - enable stop control?

                // service is off, as expected
                } else {
                    status = "down";
                    description = $translate.instant("container_down");
                }
            }

            return {
                status: status,
                description: description
            };
        }

        // determines the status of an individual healthcheck
        function determineHealthCheckStatus(healthCheck, serverTimestamp, startTime){
            var status = healthCheck.Status;

            // calculates the number of missed healthchecks since last start time
            var missedIntervals = (serverTimestamp - Math.max(healthCheck.Timestamp, startTime)) / healthCheck.Interval;

            // if service hasn't started yet
            if(startTime === undefined){
                status = "down";
            
            // if service has missed 2 updates, mark unknown
            } else if (missedIntervals > 2 && missedIntervals < 60) {
                status = "unknown";

            // if service has missed 60 updates, mark failed
            } else if (missedIntervals > 60) {
                status = "failed";
            }

            return status;
        }

        function updateServiceStatus(service, status, description, tooltipDetails){
            tooltipDetails = tooltipDetails || [];

            var $el = $("[data-id='"+ service.ID +"'] .healthIcon"),
                tooltipsDetailsHTML;

            // remove any existing popover if not currently visible            
            if($el.popover && !$el.next('div.popover:visible').length){
                $el.popover('destroy');
            }

            tooltipsDetailsHTML = tooltipDetails.reduce(function(acc, detail){
                return acc += "<div class='healthTooltipDetailRow'>\
                    <i class='healthIcon glyphicon "+ STATUS_STYLES[detail.status] +"'></i>\
                    <div class='healthTooltipDetailName'>"+ detail.name +"</div>\
                </div>";
            }, "");

            // configure popover
            // TODO - dont touch dom!
            $el.popover({
                trigger: "hover",
                placement: "right",
                delay: 0,
                title: description,
                html: true,
                content: tooltipsDetailsHTML,

                // if DesiredState is 0 or there are no healthchecks, the
                // popover should be only a title with no content
                template: service.DesiredState === 0 || !tooltipsDetailsHTML ?
                    '<div class="popover" role="tooltip"><div class="arrow"></div><h3 class="popover-title"></h3></div>' :
                    undefined
            });
        
            // update the main health icon
            setStatus(service, status);

            // if the status has changed since last tick, or
            // it was and is still unknown, notify user
            if(service.healthStatus !== status || service.healthStatus === "unknown" && status === "unknown"){
                bounceStatus(service);
            }
            // store the status for comparison later
            service.healthStatus = status;
        }

        function setStatus(service, status){
            service.healthIconClass = ["glyphicon", STATUS_STYLES[status]];
        }

        function bounceStatus(service){
            service.healthIconClass.push("zoom");

            // TODO - dont touch dom!
            var $el = $("[data-id='"+ service.ID +"'] .healthIcon");
            if($el.length > 0){
                $el.on("webkitAnimationEnd", function(){
                    // if zoom is in the class list, remove it
                    if(~service.healthIconClass.indexOf("zoom")){
                        service.healthIconClass.splice(service.healthIconClass.indexOf("zoom"), 1);
                    }
                    // clean up animation end listener
                    $el.off("webkitAnimationEnd");
                });
            }
        }

        return {
            update: update
        };
    }]);

})();
