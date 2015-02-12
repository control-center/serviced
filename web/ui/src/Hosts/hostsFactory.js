// hostssFactory
// - maintains a list of hosts and keeps it in sync with the backend.
(function() {
    'use strict';

    var hostMap = {},
        hostList = [],
        // make angular share with everybody!
        resourcesFactory, $q, instancesFactory;

    var UPDATE_FREQUENCY = 3000,
        updatePromise;

    angular.module('hostsFactory', []).
    factory("hostsFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "instancesFactory",
    function($rootScope, q, _resourcesFactory, $interval, _instancesFactory){

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        instancesFactory = _instancesFactory;
        $q = q;

        // public interface for hostsFactory
        // TODO - evaluate what should be exposed
        return {
            // returns a host by id
            get: function(id){
                return hostMap[id];
            },

            update: update,
            init: init,

            hostMap: hostMap,
            hostList: hostList
        };

        // TODO - this can most likely be removed entirely
        function init(){
            if(!updatePromise){
                updatePromise = $interval(update, UPDATE_FREQUENCY);
            }
        }

        function update(){
            var deferred = $q.defer();

            resourcesFactory.get_hosts()
                .success((data, status) => {
                    var included = [];

                    for(let id in data){
                        let host = data[id];

                        // update
                        if(hostMap[host.ID]){
                            hostMap[host.ID].update(host);

                        // new
                        } else {
                            hostMap[host.ID] = new Host(host);
                            hostList.push(hostMap[host.ID]);
                        }

                        included.push(host.ID);
                    }

                    // delete
                    if(included.length !== Object.keys(hostMap).length){
                        // iterate hostMap and find keys
                        // not present in included list
                        for(let id in hostMap){
                            if(included.indexOf(id) === -1){
                                hostList.splice(hostList.indexOf(hostMap[id], 1));
                                delete hostMap[id];
                            }
                        }
                    }

                    deferred.resolve();
                });

            // TODO - does this belong here?
            instancesFactory.init();

            return deferred.promise;
        }

    }]);

    // Host object constructor
    // takes a host object (backend host object)
    // and wraps it with extra functionality and info
    function Host(host){
        this.active = false;
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
            this.model = Object.freeze(host);
        },

        updateActive: function(){
            resourcesFactory.get_running_hosts()
                .success((activeHosts, status) => {
                    if(activeHosts[this.id]){
                        this.active = true;
                    }
                });
        }
    };

    Object.defineProperty(Host.prototype, "instances", {
        get: function(){
            return instancesFactory.getByHostId(this.id);
        }
    });

})();
