// instancesFactory
// - maintains a list of instances and keeps it in sync with the backend.
(function() {
    'use strict';

    var instanceMap = {},
        instanceList = [],
        // make angular share with everybody!
        resourcesFactory, $q, serviceHealth;

    var UPDATE_FREQUENCY = 3000,
        updatePromise;

    angular.module('instancesFactory', []).
    factory("instancesFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "$serviceHealth",
    function($rootScope, q, _resourcesFactory, $interval, _serviceHealth){

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        $q = q;
        serviceHealth = _serviceHealth;

        // public interface for instancesFactory
        // TODO - evaluate what should be exposed
        return {
            // returns an instance by id
            get: function(id){
                return instanceMap[id];
            },

            getByServiceId: (id) => {
                let results = [];
                for(let i in instanceMap){
                    if(instanceMap[i].model.ServiceID === id){
                        results.push(instanceMap[i]);
                    }
                }
                return results;
            },

            getByHostId: (id) => {
                let results = [];
                for(let i in instanceMap){
                    if(instanceMap[i].model.HostID === id){
                        results.push(instanceMap[i]);
                    }
                }
                return results;
            },

            update: update,
            init: init,

            instanceMap: instanceMap,
            instanceList: instanceList
        };

        // TODO - this can most likely be removed entirely
        function init(){
            if(!updatePromise){
                updatePromise = $interval(update, UPDATE_FREQUENCY);
            }
        }

        function update(){
            var deferred = $q.defer();

            resourcesFactory.get_running_services()
                .success((data, status) => {
                    var included = [];

                    for(let id in data){
                        let instance = data[id];

                        // update
                        if(instanceMap[instance.ID]){
                            instanceMap[instance.ID].update(instance);

                        // new
                        } else {
                            instanceMap[instance.ID] = new Instance(instance);
                            instanceList.push(instanceMap[instance.ID]);
                        }

                        included.push(instance.ID);
                    }

                    // delete
                    if(included.length !== Object.keys(instanceMap).length){
                        // iterate instanceMap and find keys
                        // not present in included list
                        for(let id in instanceMap){
                            if(included.indexOf(id) === -1){
                                instanceList.splice(instanceList.indexOf(instanceMap[id], 1));
                                delete instanceMap[id];
                            }
                        }
                    }

                    deferred.resolve();
                });

            return deferred.promise;
        }

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
