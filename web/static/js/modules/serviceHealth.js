/* global angular, console, $ */
(function() {
    'use strict';

    angular.module('serviceHealth', []).
    factory("$serviceHealth", ["$rootScope", "$q", "$http", "resourcesService", "$interval", "$translate", function($rootScope, $q, $http, resourcesService, $interval, $translate){

        var servicesService = resourcesService;

        var STATUS_STYLES = {
            "bad": "glyphicon-exclamation-sign bad",
            "good": "glyphicon-ok-sign good",
            "unknown": "glyphicon-question-sign unknown",
            // "disabled": "glyphicon-minus-sign disabled",
            "disabled": ""
        };

        // auto update all service health statuses
        var updateInterval = $interval(function(){
            // NOTE: can't use update directly because $interval
            // passes an argument to the function it calls,
            // which breaks the update function
            update();
        }, 3000);

        // simple array search util
        function findInArray(key, arr, val){
            for(var i = 0; i < arr.length; i++){
                if(arr[i][key] === val){
                    return arr[i];
                }
            }
        }

        function getRunningServiceById(serviceId){
            // subservices isn't defined if we're on a single
            // service page, so just skip this service alltogether
            if(!running) return;
            
            return findInArray("ServiceID", running, serviceId);
        }

        // updates health check data for all services
        // `appId` is the id of the specific service being clicked
        function update(appId) {

            // TODO - these methods should return promises, but they
            // don't so use our own promises
            var servicesDeferred = $q.defer();
            var runningServicesDeferred = $q.defer();
            var healthCheckDeferred = $http.get("/servicehealth");

            servicesService.get_services(true, function(top, mapped){
                servicesDeferred.resolve(mapped);
            });

            servicesService.get_running_services(function(runningServices){
                runningServicesDeferred.resolve(runningServices);
            });

            $q.all({
                services: servicesDeferred.promise,
                health: healthCheckDeferred,
                running: runningServicesDeferred.promise
            }).then(function(results){
                evaluateServiceStatus(results.running, results.services, results.health.data, appId);
            }).catch(function(err){
                // something went awry
                console.log("Promise err", err);
            });
        }

        function evaluateServiceStatus(running, services, healthCheckData, appId) {

            var healths = healthCheckData.Statuses,
                serverTimestamp = healthCheckData.Timestamp;

            var service, healthCheck, runningService, startTime,
                passingAny, failingAny, unknownAny, downAny, status,
                missedIntervals, tooltipMessage;

            for (var ServiceId in healths) {

                service = services[ServiceId];
                runningService = findInArray("ServiceID", running, ServiceId);

                if(!service){
                    return;
                }

                // get the time this service was started
                if(runningService){
                    startTime = new Date(runningService.StartedAt).getTime();

                // otherwise service hasn't been started
                } else {
                    startTime = 0;
                }

                healthCheck = healths[ServiceId];

                service.healthTooltipTitle = "";

                passingAny = false;
                failingAny = false;
                unknownAny = false;
                downAny = false;
                status = null;
                missedIntervals = 0;
                tooltipMessage = "";

                for (var name in healthCheck) {

                    // calculates the number of missed healthchecks since last start time
                    missedIntervals = (serverTimestamp - Math.max(healthCheck[name].Timestamp, startTime)) / healthCheck[name].Interval;

                    // if service hasn't started yet
                    if(!startTime){
                        healthCheck[name].Status = "down";
                    
                    // if service has missed 2 updates, mark unknown
                    } else if (missedIntervals > 2 && missedIntervals < 60) {
                        healthCheck[name].Status = "unknown";

                    // if service has missed 60 updates, mark failed
                    } else if (missedIntervals > 60) {
                        healthCheck[name].Status = "failed";
                    }

                    switch(healthCheck[name].Status){
                        case "passed":
                            passingAny = true;
                            break;
                        case "failed":
                            failingAny = true;
                            break;
                        case "unknown":
                            unknownAny = true;
                            break;
                        case "down":
                            downAny = true;
                            break;
                        default:
                            break;
                    }

                    // TODO - do something with `name` so the user has a
                    // more detailed explanation of which checks are failing
                }

                // the following conditions are relevant when the service
                // *should* be started
                if(service.DesiredState === 1){

                    // service should be up, but is failing. bad!
                    if(failingAny){
                        status = "bad";
                        tooltipMessage = $translate("failing_health_checks");

                    // service should be up, but container has not
                    // yet loaded
                    } else if(downAny){
                        status = "unknown";
                        tooltipMessage = $translate("container_unavailable");

                    // service should be up, but seems unresponsive
                    // It could be just starting, or on its way down
                    } else if(!passingAny && unknownAny){
                        status = "unknown";
                        tooltipMessage = $translate("missing_health_checks");

                    // service is up and healthy
                    } else if(passingAny && !unknownAny){
                        status = "good";
                        tooltipMessage = $translate("passing_health_checks");
                    }

                // the following conditions are relevant when the service
                // *should* be off
                } else if(service.DesiredState === 0){

                    // it should be off, but its still on... weird.
                    if(passingAny){
                        status = "unknown";
                        tooltipMessage = $translate("stopping_service");
                        // TODO - enable stop control?

                    // service is off, as expected
                    } else {
                        status = "disabled";
                    }
                }

                updateServiceStatus(service, status, tooltipMessage);
            }

            // if a specific appId was provided, its status may not
            // yet be part of health checks, so give it unknown status
            if(appId && !findInArray("ServiceID", running, appId)){

                // if this appId doesn't exist in the services list, then
                // something must be pretty messed up
                if(!services[appId]){
                    throw new Error("Service with id", appId, "does not exist");
                }
                
                updateServiceStatus(services[appId], "unknown", $translate("container_unavailable"));
            }
        }

        function updateServiceStatus(service, status, tooltipMessage){
            setStatus(service, status);

            // if the status has changed since last tick, or
            // it was and is still unknown, notify user
            if(service.healthStatus !== status ||
                service.healthStatus === "unknown" && status === "unknown"){
                bounceStatus(service);
            }

            service.healthTooltipTitle = tooltipMessage;

            // store the status for comparison later
            service.healthStatus = status;
        }

        function setStatus(service, status){
            service.healthIconClass = ["glyphicon", STATUS_STYLES[status]];
        }
        function bounceStatus(service){
            service.healthIconClass.push("zoom");

            // TODO - dont touch dom!
            var $el = $("tr[data-id='"+ service.ID +"'] .healthIcon");
            $el.on("webkitAnimationEnd", function(){
                // if zoom is in the class list, remove it
                if(~service.healthIconClass.indexOf("zoom")){
                    service.healthIconClass.splice(service.healthIconClass.indexOf("zoom"), 1);
                }
                // clean up animation end listener
                $el.off("webkitAnimationEnd");
            });
        }

        return {
            update: update
        };
    }]);

})();
