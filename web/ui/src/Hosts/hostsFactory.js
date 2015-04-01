// hostssFactory
// - maintains a list of hosts and keeps it in sync with the backend.
(function() {
    'use strict';

    // make angular share with everybody!
    var resourcesFactory, $q, instancesFactory;

    angular.module('hostsFactory', []).
    factory("hostsFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "instancesFactory", "baseFactory", "miscUtils",
    function($rootScope, q, _resourcesFactory, $interval, _instancesFactory, BaseFactory, utils){
        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        instancesFactory = _instancesFactory;
        $q = q;

        var newFactory = new BaseFactory(Host, resourcesFactory.getHosts);
        
        // alias some stuff for ease of use
        newFactory.hostList = newFactory.objArr;
        newFactory.hostMap = newFactory.objMap;

        // wrap update to do some extra work
        newFactory.update = utils.after(newFactory.update, function(){
            // check running hosts and mark them as active
            resourcesFactory.getRunningHosts()
                .success((activeHosts, status) => {
                    this.hostList.forEach(host => {
                        if(activeHosts.indexOf(host.id) !== -1){
                            host.active = true;
                        } else {
                            host.active = false;
                        }
                    });
                });

        }, newFactory);

        return newFactory;
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

        resourcesGood: function() {
            var instances = this.instances;
            for (var i = 0; i < instances.length; i++) {
                if (!instances[i].resourcesGood()) {
                    return false;
                }
            }
            return true;
        }
    };

    Object.defineProperty(Host.prototype, "instances", {
        get: function(){
            return instancesFactory.getByHostId(this.id);
        }
    });

})();
