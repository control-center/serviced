// hostssFactory
// - maintains a list of hosts and keeps it in sync with the backend.
(function() {
    'use strict';

    // make angular share with everybody!
    var resourcesFactory, $q, instancesFactory;

    angular.module('hostsFactory', []).
    factory("hostsFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "instancesFactory", "baseFactory",
    function($rootScope, q, _resourcesFactory, $interval, _instancesFactory, baseFactory){
        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        instancesFactory = _instancesFactory;
        $q = q;

        var newFactory = baseFactory(Host, "get_hosts");

        return {
            get: newFactory.get,
            update: newFactory.update,
            hostMap: newFactory.objMap,
            hostList: newFactory.objArr,
            activate: newFactory.activate,
            deactivate: newFactory.deactivate,
        };

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
