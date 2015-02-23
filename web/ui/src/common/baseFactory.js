// baseFactory

(function() {
    'use strict';

    var UPDATE_FREQUENCY = 3000;

    var $q, $interval;

    angular.module('baseFactory', []).
    factory("baseFactory", ["$rootScope", "$q", "resourcesFactory", "$interval",
    function($rootScope, _$q, resourcesFactory, _$interval){

        $q = _$q;
        $interval = _$interval;

        return BaseFactory;
    }]);

    // TODO make update frequency configurable
    // TODO - default ObjConstructor
    function BaseFactory(ObjConstructor, updateFn){
        this.objMap = {};
        this.objArr = [];
        this.updateFn = updateFn;
        this.ObjConstructor = ObjConstructor;
    }

    BaseFactory.prototype = {
        constructor: BaseFactory,

        // TODO - debounce
        update: function(){
            var deferred = $q.defer();
            this.updateFn()
                .success((data, status) => {
                    var included = [];

                    for(let id in data){
                        let obj = data[id];

                        // update
                        if(this.objMap[obj.ID]){
                            this.objMap[obj.ID].update(obj);

                        // new
                        } else {
                            this.objMap[obj.ID] = new this.ObjConstructor(obj);
                            this.objArr.push(this.objMap[obj.ID]);
                        }

                        included.push(obj.ID);
                    }

                    // delete
                    if(included.length !== Object.keys(this.objMap).length){
                        // iterate objMap and find keys
                        // not present in included list
                        for(let id in this.objMap){
                            if(included.indexOf(id) === -1){
                                this.objArr.splice(this.objArr.indexOf(this.objMap[id], 1));
                                delete this.objMap[id];
                            }
                        }
                    }

                    deferred.resolve();
                });
            return deferred.promise;
        },

        activate: function(){
            this.deactivate();
            this.updatePromise = $interval(() => {
                this.update.call(this);
            }, UPDATE_FREQUENCY);
            this.update();
        },

        deactivate: function(){
            if(this.updatePromise){
                $interval.cancel(this.updatePromise);
                this.updatePromise = null;
            }
        },

        get: function(id){
            return this.objMap[id];
        }
    };


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
