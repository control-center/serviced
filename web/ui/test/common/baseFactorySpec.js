/* global jasmine: true, beforeEach: true, expect: true, inject: true, module: true */

describe('baseFactory', function() {
    // load up actual services
    beforeEach(function(){
        module('servicedConfig');
        module('baseFactory');
    });

    // load up mock services
    beforeEach(function(){
        module(resourcesFactoryMock);
    });

    var $q, $interval, scope;
    beforeEach(inject(function($injector){
        $q = $injector.get("$q");
        $interval = $injector.get("$interval");
        scope = $injector.get("$rootScope").$new();
    }));

    // object constructor for use with baseFactory
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

    // mock objects to be returned from API
    var bubbles = {
            ID: 12346,
            Name: "Bubbles",
            ingredient: "sugar"
        },
        buttercup = {
            ID: 12347,
            Name: "Buttercup",
            ingredient: "spice"
        },
        bud = {
            ID: 12347,
            Name: "Bud",
            ingredient: "spice"
        },
        blossom = {
            ID: 12345,
            Name: "Blossom",
            ingredient: "everything nice"
        },
        professor = {
            ID: 12348,
            Name: "Professor"
        };

    // mock responses from API
    var mockData = {};
    mockData[bubbles.ID] = bubbles;
    mockData[buttercup.ID] = buttercup;
    mockData[blossom.ID] = blossom;

    var mockData2 = {};
    mockData2[bubbles.ID] = bubbles;
    mockData2[buttercup.ID] = buttercup;

    var mockData3 = {};
    mockData3[bubbles.ID] = bubbles;
    mockData3[bud.ID] = bud;
    mockData3[blossom.ID] = blossom;

    var mockData4 = {};
    mockData4[bubbles.ID] = bubbles;
    mockData4[buttercup.ID] = buttercup;
    mockData4[blossom.ID] = blossom;
    mockData4[professor.ID] = professor;




    it("Calls provided update function", inject(function(baseFactory){
        var updateFnSpy = jasmine.createSpy("updateFn").and.callFake(function(){
            return httpify($q.defer()).promise;
        });

        var newFactory = new baseFactory(function(){}, updateFnSpy);
        newFactory.update();

        expect(updateFnSpy).toHaveBeenCalled();
    }));

    it("Uses provided ObjConstructor", inject(function(baseFactory){
        var deferred = httpify($q.defer());

        var updateFnSpy = jasmine.createSpy("updateFn").and.returnValue(deferred.promise);
        var objConstructorSpy = jasmine.createSpy("objConstructor");

        var newFactory = new baseFactory(objConstructorSpy, updateFnSpy);
        newFactory.update();

        deferred.resolve(mockData);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(objConstructorSpy).toHaveBeenCalled();
    }));

    it("Updates both objArr and objMap", inject(function(baseFactory){
        var deferred = httpify($q.defer());
        var updateFn = function(){
            return deferred.promise;
        };
        var newFactory = new baseFactory(Obj, updateFn);
        newFactory.update();
        deferred.resolve(mockData);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(newFactory.objArr.length).toBe(Object.keys(newFactory.objMap).length);
    }));

    it("Adds objects", inject(function(baseFactory){
        var deferred = httpify($q.defer());
        var updateFn = function(){
            return deferred.promise;
        };
        var newFactory = new baseFactory(Obj, updateFn);
        newFactory.update();
        deferred.resolve(mockData);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(newFactory.objArr.length).toEqual(3);

        // generate a new deferred for next update call
        deferred = httpify($q.defer());
        newFactory.update();
        deferred.resolve(mockData4);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(newFactory.objArr.length).toEqual(4);
    }));

    it("Updates existing objects", inject(function(baseFactory){
        var deferred = httpify($q.defer());
        var updateFn = function(){
            return deferred.promise;
        };
        var newFactory = new baseFactory(Obj, updateFn);
        newFactory.update();
        deferred.resolve(mockData);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(newFactory.get(12347).name).toBe(mockData[12347].Name);

        // generate a new deferred for next update call
        deferred = httpify($q.defer());
        newFactory.update();

        deferred.resolve(mockData3);
        // force a tick so promise can resolve
        scope.$root.$digest();

        // mockData3 replaces buttercup with bud
        expect(newFactory.get(bud.ID).name).toBe(bud.Name);
    }));

    it("Deletes objects", inject(function(baseFactory){
        var deferred = httpify($q.defer());
        var updateFn = function(){
            return deferred.promise;
        };
        var newFactory = new baseFactory(Obj, updateFn);
        newFactory.update();
        deferred.resolve(mockData);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(newFactory.objArr.length).toEqual(3);

        // generate a new deferred for next update call
        deferred = httpify($q.defer());
        newFactory.update();
        deferred.resolve(mockData2);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(newFactory.objArr.length).toEqual(2);
    }));

    it("Starts polling updateFn", inject(function(baseFactory){
        var deferred = httpify($q.defer());
        var updateFn = function(){
            return deferred.promise;
        };
        var newFactory = new baseFactory(Obj, updateFn);
        newFactory.activate();

        $interval.flush(3001);
        deferred.resolve(mockData);
        // force a tick so promise can resolve
        scope.$root.$digest();
        expect(newFactory.objArr.length).toEqual(3);

        deferred = httpify($q.defer());
        $interval.flush(3001);
        deferred.resolve(mockData4);
        // force a tick so promise can resolve
        scope.$root.$digest();
        expect(newFactory.objArr.length).toEqual(4);

        deferred = httpify($q.defer());
        $interval.flush(3001);
        deferred.resolve(mockData2);
        // force a tick so promise can resolve
        scope.$root.$digest();
        expect(newFactory.objArr.length).toEqual(2);
    }));


    it("Doesn't start polling if already polling", inject(function(baseFactory){
        var deferred = httpify($q.defer());
        var updateFn = function(){
            return deferred.promise;
        };
        var newFactory = new baseFactory(Obj, updateFn);
        var pollPromise;

        newFactory.activate();

        pollPromise = newFactory.updatePromise;

        newFactory.activate();

        expect(pollPromise).toBe(newFactory.updatePromise);
    }));


    it("Stops polling", inject(function(baseFactory){
        var deferred = httpify($q.defer());
        var updateFn = function(){
            return deferred.promise;
        };
        var newFactory = new baseFactory(Obj, updateFn);

        newFactory.activate();
        $interval.flush(3001);
        newFactory.deactivate();

        expect(newFactory.updatePromise).toBe(null);
    }));
});
