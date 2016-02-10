// hostssFactory
// - maintains a list of hosts and keeps it in sync with the backend.
(function() {
    'use strict';

    // make angular share with everybody!
    var resourcesFactory, $q, instancesFactory, utils;

    angular.module('hostsFactory', []).
    factory("hostsFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "instancesFactory", "baseFactory", "miscUtils",
    function($rootScope, q, _resourcesFactory, $interval, _instancesFactory, BaseFactory, _utils){
        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        instancesFactory = _instancesFactory;
        $q = q;
        utils = _utils;

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
            return this.RAMAverage <= this.RAMLimitBytes;
        },

        RAMIsPercent: function(){
            return this.RAMLimit.endsWith("%");
        }
    };

    Object.defineProperty(Host.prototype, "instances", {
        get: function(){
            return instancesFactory.getByHostId(this.id);
        }
    });

    // RAMLimit may not be set yet, so use RAMCommitment
    // NOTE: RAMCommitment is deprecated
    Object.defineProperty(Host.prototype, "RAMLimit", {
        get: function() {
            if(!this.model){
                return undefined;
            }

            if(this.model.RAMLimit){
                return this.model.RAMLimit;
            } else {
                // RAMCommitment of 0 means 100% commitment
                return this.model.RAMCommitment || "100%";
            }
        }
    });

    // get the RAMLimit in bytes
    Object.defineProperty(Host.prototype, "RAMLimitBytes", {
        get: function() {
            // if percentange
            if(this.RAMIsPercent()){
                return +this.RAMLimit.slice(0,-1) * this.model.Memory * 0.01;
            // if stringy value
            } else {
                return utils.parseEngineeringNotation(this.RAMLimit);
            }
        }
    });

    Object.defineProperty(Host.prototype, "RAMLast", {
        get: function() {
            var instances = this.instances;
            var sum = 0;
            for (var i = 0; i < instances.length; i++) {
                sum += instances[i].resources.RAMLast;
            }
            return sum;
        }
    });

    Object.defineProperty(Host.prototype, "RAMMax", {
        get: function() {
            var instances = this.instances;
            var sum = 0;
            for (var i = 0; i < instances.length; i++) {
                sum += instances[i].resources.RAMMax;
            }
            return sum;
        }
    });

    Object.defineProperty(Host.prototype, "RAMAverage", {
        get: function() {
            var instances = this.instances;
            var sum = 0;
            for (var i = 0; i < instances.length; i++) {
                sum += instances[i].resources.RAMAverage;
            }
            return sum;
        }
    });

})();
