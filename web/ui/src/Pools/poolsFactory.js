// poolsFactory
// - maintains a list of pools and keeps it in sync with the backend.
(function() {
    'use strict';

    var resourcesFactory, $q;

    angular.module('poolsFactory', []).
    factory("poolsFactory", ["$rootScope", "$q", "resourcesFactory", "$interval", "baseFactory",
    function ($rootScope, q, _resourcesFactory, $interval, BaseFactory) {

        // share resourcesFactory throughout
        resourcesFactory = _resourcesFactory;
        $q = q;

        var newFactory = new BaseFactory(Pool, resourcesFactory.getPools);

        newFactory.totalCount = 0;

        newFactory.update = function () {
            var deferred = $q.defer();
            this.updateFn()
                .success((data, status) => {
                    this.totalCount = data.total;

                    var included = [];

                    for (let idx in data.results) {
                        let obj = data.results[idx];
                        let id = obj.ID;

                        // update
                        if (this.objMap[id]) {
                            this.objMap[id].update(obj);

                            // new
                        } else {
                            this.objMap[id] = new this.ObjConstructor(obj);
                            this.objArr.push(this.objMap[id]);
                        }

                        included.push(id);
                    }

                    // delete
                    if (included.length !== Object.keys(this.objMap).length) {
                        // iterate objMap and find keys
                        // not present in included list
                        for (let id in this.objMap) {
                            if (included.indexOf(id) === -1) {
                                this.objArr.splice(this.objArr.indexOf(this.objMap[id]), 1);
                                delete this.objMap[id];
                            }
                        }
                    }

                    deferred.resolve();
                })
                .error((data, status) => {
                    deferred.reject(data);
                })
                .finally(() => {
                    // notify the first request is complete
                    if (!this.lastUpdate) {
                        $rootScope.$emit("ready");
                    }

                    this.lastUpdate = new Date().getTime();
                });
            return deferred.promise;
        };

        return newFactory;
    }]);

    // Pool object constructor
    // takes a pool object (backend pool object)
    // and wraps it with extra functionality and info
    function Pool(pool){
        this.update(pool);
        this.updateFn = resourcesFactory.getPools;
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
