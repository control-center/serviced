// instancesFactory
// - maintains a list of instances and keeps it in sync with the backend.
(function() {
    'use strict';

    var resourcesFactory, $q, serviceHealth, $notification, utils;

    angular.module('instancesFactory', []).
    factory("instancesFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "$serviceHealth", "baseFactory", "$notification", "miscUtils",
    function($rootScope, q, _resourcesFactory, $interval, _serviceHealth, BaseFactory, _notification, _utils){

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        $q = q;
        serviceHealth = _serviceHealth;
        $notification = _notification;
        utils = _utils;

        var newFactory = new BaseFactory(Instance, resourcesFactory.getServiceInstances);

        // alias some stuff for ease of use
        newFactory.instanceArr = newFactory.objArr;
        newFactory.instanceMap = newFactory.objMap;

        angular.extend(newFactory, {
            getByServiceId: function(id){
                let results = [];
                for(let i in this.instanceMap){
                    if(this.instanceMap[i].model.ServiceID === id){
                        results.push(this.instanceMap[i]);
                    }
                }
                return results;
            },

            getByHostId: function(id){
                let results = [];
                for(let i in this.instanceMap){
                    if(this.instanceMap[i].model.HostID === id){
                        results.push(this.instanceMap[i]);
                    }
                }
                return results;
            },
        });

        newFactory.update = utils.after(newFactory.update, function(){
            // call update on all children
            newFactory.instanceArr.forEach(instance => instance.update());
        }, newFactory);

        return newFactory;
    }]);

    // Instance object constructor
    // takes a instance object (backend instance object)
    // and wraps it with extra functionality and info
    function Instance(instance) {
        this.active = false;

        this.resources = {
            RAMCommitment: 0,
            RAMLast: 0,
            RAMMax: 0,
            RAMAverage: 0
        };

        this.healthChecks = {};
        this.update(instance);
    }

    Instance.prototype = {
        constructor: Instance,

        update: function(instance) {
            if(instance){
               this.updateInstanceDef(instance);
            }
        },

        updateInstanceDef: function(instance) {
            this.name = instance.Name;
            this.id = instance.ID;
            this.model = Object.freeze(instance);
            this.resources.RAMAverage = Math.max(0, instance.RAMAverage);
            this.resources.RAMLast = Math.max(0, instance.RAMLast);
            this.resources.RAMMax = Math.max(0, instance.RAMMax);
            this.resources.RAMCommitment = utils.parseEngineeringNotation(instance.RAMCommitment);

            var hc = {};
            for(var name in instance.HealthChecks){
                hc[name] = instance.HealthChecks[name].Status;
            }
            this.healthChecks = hc;
        },

        stop: function(){
            resourcesFactory.killRunning(this.model.HostID, this.id)
                .success(() => {
                    this.update();
                })
                .error((data, status) => {
                    $notification.create("Stop Instance failed", data.Detail).error();
                });
        },

        resourcesGood: function() {
            return this.resources.RAMLast < this.resources.RAMCommitment;
        }
    };

})();
