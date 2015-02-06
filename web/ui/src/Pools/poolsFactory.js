// poolsFactory
// - maintains a list of pools and keeps it in sync with the backend.
(function() {
    'use strict';

    var poolMap = {},
        poolList = [],
        // make angular share with everybody!
        resourcesFactory, $q;

    var UPDATE_FREQUENCY = 3000,
        updatePromise;

    angular.module('poolsFactory', []).
    factory("poolsFactory", ["$rootScope", "$q", "resourcesFactory", "$interval",
    function($rootScope, q, _resourcesFactory, $interval){

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        $q = q;

        // public interface for poolsFactory
        // TODO - evaluate what should be exposed
        return {
            // returns a pool by id
            get: function(id){
                return poolMap[id];
            },

            update: update,
            init: init,

            poolMap: poolMap,
            poolList: poolList
        };

        // TODO - this can most likely be removed entirely
        function init(){
            if(!updatePromise){
                updatePromise = $interval(update, UPDATE_FREQUENCY);
            }
        }

        function update(){
            var deferred = $q.defer();

            resourcesFactory.get_pools()
                .success((data, status) => {
                    // TODO - this seems like a nice reusable pattern
                    var included = [];

                    for(let id in data){
                        let pool = data[id];

                        // update
                        if(poolMap[pool.ID]){
                            poolMap[pool.ID].update(pool);

                        // new
                        } else {
                            poolMap[pool.ID] = new Pool(pool);
                            poolList.push(poolMap[pool.ID]);
                        }

                        included.push(pool.ID);
                    }

                    // delete
                    if(included.length !== Object.keys(poolMap).length){
                        // iterate poolMap and find keys
                        // not present in included list
                        for(let id in poolMap){
                            if(included.indexOf(id) === -1){
                                poolList.splice(poolList.indexOf(poolMap[id], 1));
                                delete poolMap[id];
                            }
                        }
                    }

                    deferred.resolve();
                });

            return deferred.promise;
        }

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
