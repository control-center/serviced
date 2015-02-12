// baseFactory

(function() {
    'use strict';

    var UPDATE_FREQUENCY = 3000;

    angular.module('baseFactory', []).
    factory("baseFactory", ["$rootScope", "$q", "resourcesFactory", "$interval",
    function($rootScope, $q, resourcesFactory, $interval){

        // return a function that can be called to unwrap
        // a shiny new factory
        return function(FactoryObj, resourceMethodName){
            var objMap = {},
                objArr = [],
                update, updatePromise,
                activate, deactivate;

            update = function(){
                var deferred = $q.defer();
                resourcesFactory[resourceMethodName]()
                    .success((data, status) => {
                        var included = [];

                        for(let id in data){
                            let obj = data[id];

                            // update
                            if(objMap[obj.ID]){
                                objMap[obj.ID].update(obj);

                            // new
                            } else {
                                objMap[obj.ID] = new FactoryObj(obj);
                                objArr.push(objMap[obj.ID]);
                            }

                            included.push(obj.ID);
                        }

                        // delete
                        if(included.length !== Object.keys(objMap).length){
                            // iterate objMap and find keys
                            // not present in included list
                            for(let id in objMap){
                                if(included.indexOf(id) === -1){
                                    objArr.splice(objArr.indexOf(objMap[id], 1));
                                    delete objMap[id];
                                }
                            }
                        }

                        deferred.resolve();
                    });
                return deferred.promise;
            };


            activate = function(){
                deactivate();
                updatePromise = $interval(update, UPDATE_FREQUENCY);
            };

            deactivate = function(){
                if(updatePromise){
                    $interval.cancel(updatePromise);
                }
            };

            return {
                update: update,
                get: function(id){
                    return objMap[id];
                },
                activate: activate,
                deactivate: deactivate,
                objMap: objMap,
                objArr: objArr
            };
        };

    }]);


    function Obj(obj){
        this.update(obj);
    }

    Obj.prototype = {
        constructor: Obj,
        update: function(obj){
            if(obj){
                this.updateObjDef(obj);
            }
        },
        updateObjDef: function(obj){
            this.name = obj.Name;
            this.id = obj.ID;
            this.model = Object.freeze(obj);
        },
    };

})();
