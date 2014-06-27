/* global angular, console, $ */
'use strict';

(function() {

    angular.module('serviceHealth', []).
    factory("$serviceHealth", ["$rootScope", function($rootScope){

        var services;

        var STATUS_STYLES = {
            "bad": "glyphicon-exclamation-sign bad",
            "good": "glyphicon-ok-sign good",
            "unknown": "glyphicon-question-sign unknown",
            "disabled": "glyphicon-minus-sign disabled",
        };

        function getServiceById(serviceId){
            for(var i = 0; i < services.subservices.length; i++){
                if(services.subservices[i].Id === serviceId){
                    return services.subservices[i];
                }
            }
        }

        function setStatus(service, status){
            service.healthIconClass = ["glyphicon", STATUS_STYLES[status]];
        }
        function bounceStatus(service){
            service.healthIconClass.push("zoom");

            // TODO - dont touch dom!
            var $el = $("tr[data-id='"+ service.Id +"'] .healthIcon");
            $el.on("webkitAnimationEnd", function(){
                // if zoom is in the class list, remove it
                if(~service.healthIconClass.indexOf("zoom")){
                    service.healthIconClass.splice(service.healthIconClass.indexOf("zoom"), 1);
                }
                // clean up animation end listener
                $el.off("webkitAnimationEnd");
            });
        }

        function update(id) {
            if(!services){
                console.error("Health check failed. No services to check.");
            }

            // TODO - if id is provided, update just that id
            
            $.get("/servicehealth", function(packet) {
                var healths = packet.Statuses;
                var timestamp = packet.Timestamp;

                var service, data,
                    passingAny, failingAny, unknownAny, status, missedIntervals;

                for (var ServiceId in healths) {

                    service = getServiceById(ServiceId);

                    if(!service){
                        throw new Error("Could not find service with id" + ServiceId);
                    }

                    data = healths[ServiceId];

                    service.healthTooltipTitle = "";

                    passingAny = false;
                    failingAny = false;
                    unknownAny = false;
                    status = null;
                    missedIntervals = 0;

                    for (var name in data) {

                        missedIntervals = (timestamp - data[name].Timestamp) / data[name].Interval;

                        // if service has missed 2 updates, mark unknown
                        if (missedIntervals > 2 && missedIntervals < 30) {
                            data[name].Status = "unknown";

                        // if service has missed 30 updates, mark failed
                        } else if (missedIntervals > 30) {
                            data[name].Status = "failed";
                        }

                        switch(data[name].Status){
                            case "passed":
                                passingAny = true;
                                break;
                            case "failed":
                                failingAny = true;
                                break;
                            case "unknown":
                                unknownAny = true;
                                break;
                            default:
                                break;
                        }

                        // TODO - does `data` ever contain more than just one key?
                        // need to make this tooltip a little more useful
                        service.healthTooltipTitle = name + ":" + data[name].Status;
                    }

                    // the following conditions are relevant when the service
                    // *should* be started
                    if(service.DesiredState === 1){

                        // service should be up, but is failing. bad!
                        if(failingAny){
                            status = "bad";

                        // service should be up, but seems unresponsive
                        // It could be just starting, or on its way down
                        } else if(!passingAny && unknownAny){
                            status = "unknown";

                        // service is up and healthy
                        } else if(passingAny && !unknownAny){
                            status = "good";
                        }

                    // the following conditions are relevant when the service
                    // *should* be off
                    } else if(service.DesiredState === 0){

                        // it should be off, but its still on... weird.
                        if(passingAny){
                            status = "unknown";

                        // service is off, as expected
                        } else {
                            status = "disabled";
                        }
                    }

                    setStatus(service, status);

                    // if the status has changed since last tick, or
                    // it was and is still unknown, notify user
                    if(service.healthStatus !== status ||
                        service.healthStatus === "unknown" && status === "unknown"){
                        bounceStatus(service);
                    }

                    // store the status for comparison later
                    service.healthStatus = status;
                }
            });
        }

        function setServices($services){
            services = $services;
        }

        // expose serviceHealth to everyone
        // HACK - this really is terrible :/
        $rootScope.serviceHealth = {
            update: update,
            setServices: setServices
        };

        return {
            update: update,
            setServices: setServices
        };
    }]);

})();