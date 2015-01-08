/* global angular, console, $ */
/* jshint multistr: true */
(function() {
    'use strict';

    angular.module('serviceHealth', []).
    factory("$serviceHealth", ["$rootScope", "$q", "resourcesFactory", "$interval", "$translate",
    function($rootScope, $q, resourcesFactory, $interval, $translate){

        var statuses = {};

        var STATUS_STYLES = {
            "bad": "glyphicon glyphicon-exclamation bad",
            "good": "glyphicon glyphicon-ok good",
            "unknown": "glyphicon glyphicon-question unknown",
            "down": "glyphicon glyphicon-minus disabled"
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

            $q.all({
                services: servicesDeferred.promise,
                health: healthCheckDeferred.promise
            }).then(function(results){
                var serviceHealthCheck, instanceHealthCheck,
                    serviceStatus, instanceStatus, instanceUniqueId,
                    statuses = {};

                // iterate services healthchecks
                for(var serviceId in results.services){
                    serviceHealthCheck = results.health.Statuses[serviceId];
                    serviceStatus = new Status(serviceId, results.services[serviceId].name, results.services[serviceId].service.DesiredState, results.services[serviceId].service.Instances);

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
                            instanceStatus = new Status(instanceUniqueId, results.services[serviceId].name +" "+ instanceId, results.services[serviceId].service.DesiredState, results.services[serviceId].service.Instances);
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

                updateHealthCheckUI(statuses);

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

        var healthcheckTemplate = '<i class="healthIcon glyphicon"></i><div class="healthIconBadge"></div>';

        function updateHealthCheckUI(statuses){
            // select all healthchecks DOM elements and look
            // up their respective class thing
            $(".healthCheck").each(function(i, el){
                var $el = $(el),
                    id = $el.attr("data-id"),
                    lastStatus = $el.attr("data-lastStatus"),
                    $healthIcon, $badge,
                    statusObj, popoverHTML,
                    hideHealthChecks,
                    placement = "right",
                    popoverObj, template;

                // if this is an unintialized healthcheck html element,
                // put template stuff inside
                if(!$el.children().length){
                    $el.html(healthcheckTemplate);
                }

                $healthIcon = $el.find(".healthIcon");
                $badge = $el.find(".healthIconBadge");

                // for some reason this healthcheck has no id,
                // so no icon for you!
                if(!id){
                    return;
                }

                statusObj = statuses[id];

                // this instance is on its way up, so create an "unknown" status for it
                if(!statusObj){
                    statusObj = new Status(id, "", 1);
                    statusObj.statusRollup.incDown();
                    statusObj.evaluateStatus();
                }

                // determine if we should hide healthchecks
                hideHealthChecks = statusObj.statusRollup.allGood() ||
                    statusObj.statusRollup.allDown() ||
                    statusObj.desiredState === 0;

                // if service should be up and there is more than 1 instance, show number of instances
                if(statusObj.desiredState === 1 && statusObj.statusRollup.total > 1){
                    $el.addClass("wide");
                    $badge.text(statusObj.statusRollup.good +"/"+ statusObj.statusRollup.total).show();

                // else, hide the badge
                } else {
                    $el.removeClass("wide");
                    $badge.hide();
                }
               
                // setup popover

                // if this $el is inside a .serviceTitle, make the popover point down
                if($el.parent().hasClass("serviceTitle")){
                    placement = "bottom";
                }

                // if this statusObj has children, we wanna show
                // them in the healtcheck tooltip, so generate
                // some yummy html
                if(statusObj.children.length){
                    popoverHTML = [];

                    var isHealthCheckStatus = function(status){
                       return !status.id;
                    };

                    // if this status's children are healthchecks,
                    // no need for instance rows, go straight to healthcheck rows
                    if(statusObj.children.length && isHealthCheckStatus(statusObj.children[0])){
                        // if these are JUST healthchecks, then don't allow them
                        // to be hidden. this ensures that healthchecks show up for
                        // running instances.
                        hideHealthChecks = false;
                        // don't show count badge for healthchecks either
                        $badge.hide();
                        $el.removeClass("wide");

                        statusObj.children.forEach(function(hc){
                            popoverHTML.push(bindHealthCheckRowTemplate(hc));
                        });
                         
                    // else these are instances, so create instance rows
                    // AND healthcheck rows
                    } else {
                        statusObj.children.forEach(function(instanceStatus){
                            // if this is becoming too long, stop adding rows
                            if(popoverHTML.length >= 15){
                                // add an overflow indicator if not already there
                                if(popoverHTML[popoverHTML.length-1] !== "..."){
                                    popoverHTML.push("..."); 
                                }
                                return;
                            }

                            // only create an instance row for this instance if
                            // it's in a bad or unknown state
                            if(instanceStatus.status === "bad" || instanceStatus.status === "unknown"){
                                popoverHTML.push("<div class='healthTooltipDetailRow'>");
                                popoverHTML.push("<div style='font-weight: bold; font-size: .9em; padding: 5px 0 3px 0;'>"+ instanceStatus.name +"</div>");
                                instanceStatus.children.forEach(function(hc){
                                    popoverHTML.push(bindHealthCheckRowTemplate(hc));
                                });
                                popoverHTML.push("</div>");
                            }
                        });
                    }

                    popoverHTML = popoverHTML.join("");
                }

                // choose a popover template which is just a title,
                // or a title and content
                template = hideHealthChecks || !popoverHTML ?
                    '<div class="popover" role="tooltip"><div class="arrow"></div><h3 class="popover-title"></h3></div>' :
                    '<div class="popover" role="tooltip"><div class="arrow"></div><h3 class="popover-title"></h3><div class="popover-content"></div></div>';
                
                // NOTE: directly accessing the bootstrap popover
                // data object here.
                popoverObj = $el.data("bs.popover");
                
                // if popover element already exists, update it
                if(popoverObj){
                    // update title, content, and template
                    popoverObj.options.title = statusObj.description;
                    popoverObj.options.content = popoverHTML;
                    popoverObj.options.template = template;
                    
                    // if the tooltip already exists, change the contents
                    // to the new template
                    if(popoverObj.$tip){
                        popoverObj.$tip.html($(template).html());
                    }

                    // force popover to update using the new options
                    popoverObj.setContent();

                    // if the popover is currently visible, update
                    // it immediately, but turn off animation to
                    // prevent it fading in
                    if(popoverObj.$tip.is(":visible")){
                        popoverObj.options.animation = false;
                        popoverObj.show();
                        popoverObj.options.animation = true;
                    }
            
                // if popover element doesn't exist, create it
                } else {
                    $el.popover({
                        trigger: "hover",
                        placement: placement,
                        delay: 0,
                        title: statusObj.description,
                        html: true,
                        content: popoverHTML,
                        template: template
                    });
                }

                $el.removeClass(Object.keys(STATUS_STYLES).join(" "))
                    .addClass(statusObj.status);

                // if the status has changed since last tick, or
                // it was and is still unknown, notify user
                if(lastStatus !== statusObj.status || lastStatus === "unknown" && statusObj.status === "unknown"){
                    bounceStatus($el);
                }
                // store the status for comparison later
                $el.attr("data-lastStatus", statusObj.status);
            });
        }
        function bindHealthCheckRowTemplate(hc){
            return "<div class='healthTooltipDetailRow "+ hc.status +"'>\
                    <i class='healthIcon glyphicon'></i>\
                <div class='healthTooltipDetailName'>"+ hc.name +"</div>\
            </div>";
        }

        function bounceStatus($el){
            $el.addClass("zoom");

            $el.on("webkitAnimationEnd animationend", function(){
                $el.removeClass("zoom");
                // clean up animation end listener
                $el.off("webkitAnimationEnd animationend");
            });
        }

        return {
            update: update
        };
    }]);

})();
