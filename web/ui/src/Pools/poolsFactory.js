// poolsFactory
// - maintains a list of pools and keeps it in sync with the backend.
(function() {
    'use strict';

    var resourcesFactory, $q;

    angular.module('poolsFactory', []).
    factory("poolsFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "baseFactory",
    function($rootScope, q, _resourcesFactory, $interval, BaseFactory){

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        $q = q;

        var newFactory = new BaseFactory(Pool, resourcesFactory.getPools);

        // alias some stuff for ease of use
        newFactory.poolList = newFactory.objArr;
        newFactory.poolMap = newFactory.objMap;

        return newFactory;
    }]);

    // Pool object constructor
    // takes a pool object (backend pool object)
    // and wraps it with extra functionality and info
    function Pool(pool){
        this.update(pool);
    }

    Pool.prototype = {
        constructor: Pool,

        update: function(pool){
            if(pool){
               this.updatePoolDef(pool);
            }
        },

        updatePoolDef: function(pool){
            this.name = pool.Name;
            this.id = pool.ID;
            this.model = Object.freeze(pool);
        }
    };

})();
