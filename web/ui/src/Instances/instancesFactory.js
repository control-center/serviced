// instancesFactory
// - maintains a list of instances and keeps it in sync with the backend.
(function() {
    'use strict';

    var resourcesFactory, $q, serviceHealth;

    angular.module('instancesFactory', []).
    factory("instancesFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "$serviceHealth", "baseFactory",
    function($rootScope, q, _resourcesFactory, $interval, _serviceHealth, BaseFactory){

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        $q = q;
        serviceHealth = _serviceHealth;

        var newFactory = new BaseFactory(Instance, resourcesFactory.get_running_services);

        // alias some stuff for ease of use
        newFactory.instanceArr = newFactory.objArr;
        newFactory.instanceMap = newFactory.objMap;

        angular.extend(newFactory, {
            getByServiceId: function(id){
                let results = [];
                for(let i in this.objMap){
                    if(this.objMap[i].model.ServiceID === id){
                        results.push(this.objMap[i]);
                    }
                }
                return results;
            },

            getByHostId: function(id){
                let results = [];
                for(let i in this.objMap){
                    if(this.objMap[i].model.HostID === id){
                        results.push(this.objMap[i]);
                    }
                }
                return results;
            },
        });

        return newFactory;
    }]);

    // Instance object constructor
    // takes a instance object (backend instance object)
    // and wraps it with extra functionality and info
    function Instance(instance){
        this.active = false;
        this.update(instance);
    }

    Instance.prototype = {
        constructor: Instance,

        update: function(instance){
            if(instance){
               this.updateInstanceDef(instance);
            }

            // update service health
            // TODO - should service update itself, its controller
            // update the service, or serviceHealth update all services?
            this.status = serviceHealth.get(this.healthId);

        },

        updateInstanceDef: function(instance){
            this.name = instance.Name;
            this.id = instance.ID;
            this.model = Object.freeze(instance);
            // TODO - formally define health id
            this.healthId = this.id +"."+ instance.InstanceID;
            this.desiredState = instance.DesiredState;
        },

        stop: function(){
            resourcesFactory.kill_running(this.model.HostID, this.id)
                .then(() => {
                    this.update();
                });
            // desired state 0 is stop
            this.desiredState = 0;
        }
    };

})();
