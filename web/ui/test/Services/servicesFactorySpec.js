/* global jasmine: true, beforeEach: true, expect: true, inject: true, module: true */

describe('servicesFactory', function() {

    // load up actual services
    beforeEach(function(){
        module('baseFactory');
        module('servicesFactory');
        module('miscUtils');
    });

    // load up mock services
    beforeEach(function(){
        module(resourcesFactoryMock);
        module(serviceHealthMock);
        module(instancesFactoryMock);
    });

    var $q, $interval, resourcesFactory, scope, instancesFactory, serviceHealth;
    beforeEach(inject(function($injector){
        $q = $injector.get("$q");
        $interval = $injector.get("$interval");
        resourcesFactory = $injector.get("resourcesFactory");
        scope = $injector.get("$rootScope").$new();
        instancesFactory = $injector.get("instancesFactory");
        serviceHealth = $injector.get("$serviceHealth");
    }));

    var serviceDefA = {
            ID: "123456",
            Name: "Service A"
        },
        serviceDefB = {
            ID: "123457",
            Name: "Service B"
        },
        serviceDefC = {
            ID: "123458",
            Name: "Service C",
            ParentServiceID: "123457"
        };


    it("Passes a `since` value of 0 on the first update request", inject(function(servicesFactory){
        servicesFactory.update();
        expect(resourcesFactory.getServices).toHaveBeenCalled();
        expect(resourcesFactory.getServices.calls.mostRecent().args[0]).toEqual(0);
    }));

    it("Passes an expected `since` for subsquent updates", function(done){
        inject(function(servicesFactory){

            servicesFactory.update();

            setTimeout(function(){
                servicesFactory.update();
                // 1199 is 1000ms UPDATE_PADDING plus 200ms for this timeout
                // minus 1ms so we can expect the value to be greater than
                // or equal to 1200
                expect(resourcesFactory.getServices.calls.mostRecent().args[0]).toBeGreaterThan(1199);
                done();
            }, 200);
        });
    });

    it("Creates services", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(Object.keys(servicesFactory.serviceMap).length).toBe(2);
    }));

    it("Adds children to a service", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefB.ID).children.length).toBe(1);
    }));

    it("Updates services", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(Object.keys(servicesFactory.serviceMap).length).toBe(2);

        servicesFactory.update();
        deferred = resourcesFactory._getCurrDeferred();
        var serviceDefB2 = {
            ID: serviceDefB.ID,
            Name: "Service B2"
        };
        deferred.resolve([serviceDefA, serviceDefB2]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefB.ID).name).toBe(serviceDefB2.Name);
    }));

    it("Updates a service's children", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        servicesFactory.update();
        deferred = resourcesFactory._getCurrDeferred();
        // change serviceDefC's parent from B to A
        deferred.resolve([
            {
                ID: serviceDefC.ID,
                Name: serviceDefC.Name,
                ParentServiceID: serviceDefA.ID
            }
        ]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        // child was removed from B
        expect(servicesFactory.get(serviceDefB.ID).children.length).toBe(0);
        // child was added to A
        expect(servicesFactory.get(serviceDefA.ID).children.length).toBe(1);
    }));

    it("Adds top level services to serviceTree", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.serviceTree.length).toBe(2);
    }));

    it("Adds depth property to services in serviceTree", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        var serviceDefD = {
            ID: "123459",
            Name: "Service D",
            ParentServiceID: serviceDefC.ID
        };
        deferred.resolve([serviceDefA, serviceDefB, serviceDefC, serviceDefD]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefA.ID).depth).toBe(0);
        expect(servicesFactory.get(serviceDefC.ID).depth).toBe(1);
        expect(servicesFactory.get(serviceDefD.ID).depth).toBe(2);
    }));

    it("Identifies services whose parent cannot be found", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefC.ID).isOrphan).toBe(true);
    }));

    it("Updates orphaned service whose parent is found", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefC.ID).isOrphan).toBe(true);

        servicesFactory.update();
        deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefB]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefC.ID).isOrphan).toBe(false);
        expect(servicesFactory.get(serviceDefB.ID).children.length).toBe(1);
    }));

    it("Activates instancesFactory when activate is called", inject(function(servicesFactory){
        servicesFactory.activate();
        expect(instancesFactory.activate).toHaveBeenCalled();
    }));

    it("Deactivates instancesFactory when deactivate is called", inject(function(servicesFactory){
        servicesFactory.activate();
        servicesFactory.deactivate();
        expect(instancesFactory.deactivate).toHaveBeenCalled();
    }));

    it("Attaches service health to a service", inject(function(servicesFactory){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        var serviceHealthDeferred = serviceHealth._getCurrDeferred(),
            serviceHealthData = {};

        serviceHealthData[serviceDefA.ID] = "hi";
        serviceHealthDeferred.resolve(serviceHealthData);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefA.ID).status).toBe("hi");
    }));
});

