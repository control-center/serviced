// hostssFactory
// - maintains a list of hosts and keeps it in sync with the backend.
(function() {
    'use strict';

    var hostMap = {},
        // make angular share with everybody!
        resourcesFactory, $q;

    var UPDATE_FREQUENCY = 3000;

    angular.module('hostsFactory', []).
    factory("hostsFactory", ["$rootScope", "$q", "resourcesFactory", "$interval",
    function($rootScope, q, _resourcesFactory, $interval){

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        $q = q;

        // public interface for hostsFactory
        // TODO - evaluate what should be exposed
        return {
            // returns a host by id
            getHost: function(id){
                return hostMap[id];
            },

            update: update,

            hostMap: hostMap
        };


        function update(){
            var deferred = $q.defer();

            resourcesFactory.get_hosts(UPDATE_FREQUENCY + 1000)
                .success((data, status) => {
                    var included = [];

                    data.forEach((host) => {
                        // update
                        if(hostMap[host.ID]){
                            hostMap[host.ID].update(host);

                        // new
                        } else {
                            hostMap[host.ID] = new Host(host);
                        }

                        included.push(host.ID);
                    });

                    // delete
                    if(included.length !== Object.keys(hostMap).length){
                        // iterate hostMap and find keys
                        // not present in included list
                    }

                    deferred.resolve();
                });

            return deferred.promise;
        }

    }]);

    // Host object constructor
    // takes a host object (backend host object)
    // and wraps it with extra functionality and info
    function Host(host){
        this.update(host);
    }

    Host.prototype = {
        constructor: Host,

        update: function(host){
            if(host){
               this.updateHostDef(host);
            }
        },

        updateHostDef: function(host){
            this.name = host.Name;
            this.id = host.ID;

            // TODO - determine full pool path from PoolID
            this.fullPath = "";

            this.host = Object.freeze(host);
        },

        getRunningForHost: function(){
            var deferred = $q.defer();

            resourcesFactory.get_running_services_for_host(this.id)
                .success((instances, status) => {
                    this.instances = instances;

                    console.log("running instances for ", this.name);
                    console.log(instances);

                    deferred.resolve(instances);
                });

            return deferred.promise;
        }

    };

})();
