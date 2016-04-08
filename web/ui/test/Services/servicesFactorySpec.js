/* global spyOn: true, beforeEach: true, expect: true, inject: true, module: true */

describe('servicesFactory', function() {

    // load up actual services
    beforeEach(function(){
        module('servicedConfig');
        module('baseFactory');
        module('servicesFactory');
        module('miscUtils');
        module('serviceHealth');
    });

    // load up mock services
    beforeEach(function(){
        module(resourcesFactoryMock);
        module(instancesFactoryMock);
        module(translateMock);
    });

    var resourcesFactory, scope, serviceHealth, servicesFactory, hcStatus, instancesFactory;
    beforeEach(inject(function($injector){
        resourcesFactory = $injector.get("resourcesFactory");
        scope = $injector.get("$rootScope").$new();
        serviceHealth = $injector.get("$serviceHealth");
        servicesFactory = $injector.get("servicesFactory");
        hcStatus = $injector.get("hcStatus");
        instancesFactory = $injector.get("instancesFactory");
    }));

    var serviceDefA = {
            ID: "123456",
            Name: "Service A",
            RAMCommitment: "1024M",
            DesiredState: 1
        },
        serviceDefB = {
            ID: "123457",
            Name: "Service B",
            RAMCommitment: "1024M",
            DesiredState: 1
        },
        serviceDefC = {
            ID: "123458",
            Name: "Service C",
            ParentServiceID: "123457",
            RAMCommitment: "1024M",
            DesiredState: 1
        };


    it("Passes a `since` value of 0 on the first update request", function(){
        servicesFactory.update();
        expect(resourcesFactory.getServices).toHaveBeenCalled();
        expect(resourcesFactory.getServices.calls.mostRecent().args[0]).toEqual(0);
    });

    it("Passes an expected `since` for subsquent updates", function(done){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        setTimeout(function(){
            servicesFactory.update();
            // 1199 is 1000ms UPDATE_PADDING plus 200ms for this timeout
            // minus 1ms so we can expect the value to be greater than
            // or equal to 1200
            expect(resourcesFactory.getServices.calls.mostRecent().args[0]).toBeGreaterThan(1199);
            var deferred = resourcesFactory._getCurrDeferred();
            deferred.resolve([serviceDefA, serviceDefB]);
            // force a tick so promise can resolve
            scope.$root.$digest();

            done();
        }, 200);
    });

    it("Creates services", function(){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(Object.keys(servicesFactory.serviceMap).length).toBe(2);
    });

    it("Adds children to a service", function(){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefB.ID).children.length).toBe(1);
    });

    it("Updates services", function(){
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
    });

    it("Updates a service's children", function(){
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
    });

    it("Adds top level services to serviceTree", function(){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.serviceTree.length).toBe(2);
    });

    it("Adds depth property to services in serviceTree", function(){
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
    });

    it("Identifies services whose parent cannot be found", function(){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        expect(servicesFactory.get(serviceDefC.ID).isOrphan).toBe(true);
    });

    it("Updates orphaned service whose parent is found", function(){
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
    });

    it("Attaches service health to a service", function(){
        // setup some services
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        // add instances to the services
        var mockService = servicesFactory.get(serviceDefA.ID);
        var mockInstance = {
            id: "67890",
            model: { ServiceID: serviceDefA.ID },
            healthChecks: { "check1": hcStatus.OK }
        };
        mockService.instances.push(mockInstance);

        // push mock instance into instances factory
        // so that servicesFactory can retrieve it later
        instancesFactory._pushInstance(mockInstance);

        // update services to evaluate instance status
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([]);
        // force a tick so promise can resolve
        scope.$root.$digest();

        var status = servicesFactory.get(serviceDefA.ID).status;
        expect(status.status).toBe(hcStatus.OK);
    });
});

describe('servicesFactory Service object', function() {

    // load up actual services
    beforeEach(function(){
        module('servicedConfig');
        module('baseFactory');
        module('servicesFactory');
        module('miscUtils');
        module('serviceHealth');
    });

    // load up mock services
    beforeEach(function(){
        module(resourcesFactoryMock);
        module(instancesFactoryMock);
        module(translateMock);
    });

    var resourcesFactory, scope, serviceHealth, servicesFactory, hcStatus;
    beforeEach(inject(function($injector){
        resourcesFactory = $injector.get("resourcesFactory");
        scope = $injector.get("$rootScope").$new();
        serviceHealth = $injector.get("$serviceHealth");
        servicesFactory = $injector.get("servicesFactory");
        hcStatus = $injector.get("hcStatus");
    }));

    var serviceDefA = {
            ID: "isvc-123456",
            Name: "Service A",
            RAMCommitment: "1024M",
        },
        serviceDefB = {
            ID: "123457",
            Name: "Service B",
            RAMCommitment: "1024M",
        },
        serviceDefC = {
            ID: "123458",
            Name: "Service C",
            ParentServiceID: "123457",
            RAMCommitment: "1024M",
        };

    // creates a service
    var getAService = function(id){
        servicesFactory.update();
        var deferred = resourcesFactory._getCurrDeferred();
        deferred.resolve([serviceDefA, serviceDefB, serviceDefC]);
        // force a tick so promise can resolve
        scope.$root.$digest();
        return servicesFactory.get(id);
    };


    it("Freezes the service def", function(){
        var service = getAService(serviceDefA.ID);
        service.model.Name = "HORSE HOCKEY";
        expect(service.model.Name).not.toBe("HORSE HOCKEY");
    });

    it("Adds itself to its parent", function(){
        var service = getAService(serviceDefB.ID);
        expect(service.children.length).toBe(1);
    });

    it("Sets name and id properties from service def", function(){
        var service = getAService(serviceDefA.ID);
        expect(service.name).toBe(service.model.Name);
        expect(service.id).toBe(service.model.ID);
    });

    it("Evaluates its service type", function(){
        var service = getAService(serviceDefA.ID);
        expect(service.isApp()).toBe(true);
        expect(service.isIsvc()).toBe(true);

        var service2 = getAService(serviceDefC.ID);
        expect(service2.isApp()).toBe(false);
        expect(service2.isIsvc()).toBe(false);
    });

    it("Marks caches dirty on update", function(){
        var service = getAService(serviceDefA.ID);

        expect(service.cache.caches.descendents.dirty).toBe(true);
        // hitting an accessor marks the cache as clean
        var descendents = service.descendents;
        expect(service.cache.caches.descendents.dirty).toBe(false);

        service.update();
        expect(service.cache.caches.descendents.dirty).toBe(true);
    });

    it("Updates parent on update", function(){
        var serviceC = getAService(serviceDefC.ID),
            serviceB = getAService(serviceDefB.ID);

        spyOn(serviceB, "update");
        serviceC.update();
        expect(serviceB.update).toHaveBeenCalled();
    });

    it("Adds a child service only once", function(){
        var serviceA = getAService(serviceDefA.ID),
            serviceB = getAService(serviceDefB.ID);

        serviceA.addChild(serviceB);
        expect(serviceA.children[0]).toBe(serviceB);

        serviceA.addChild(serviceB);
        expect(serviceA.children.length).toBe(1);
    });

    it("Removes a child service and calls update on its parent", function(){
        var serviceA = getAService(serviceDefA.ID),
            serviceB = getAService(serviceDefB.ID),
            serviceC = getAService(serviceDefC.ID);

        // C is already a child of B. lets
        // make B a child of A
        serviceA.addChild(serviceB);

        spyOn(serviceA, "update");
        serviceB.removeChild(serviceC);
        expect(serviceB.children.length).toBe(0);
        expect(serviceA.update).toHaveBeenCalled();

    });

    it("removes itself from its old parent when changing parents", function(){
        var serviceA = getAService(serviceDefA.ID),
            serviceB = getAService(serviceDefB.ID),
            serviceC = getAService(serviceDefC.ID);

        // C is a child of B, lets move it to A
        serviceC.setParent(serviceA);
        expect(serviceC.parent).toBe(serviceA);
        expect(serviceB.children.length).toBe(0);
        expect(serviceA.children.length).toBe(1);
    });

    it("Starts, stops, and restarts services", function(){
        var serviceA = getAService(serviceDefA.ID);

        serviceA.start();
        expect(resourcesFactory.startService).toHaveBeenCalled();
        expect(serviceA.desiredState).toBe(1);

        serviceA.stop();
        expect(resourcesFactory.stopService).toHaveBeenCalled();
        expect(serviceA.desiredState).toBe(0);

        serviceA.restart();
        expect(resourcesFactory.restartService).toHaveBeenCalled();
        expect(serviceA.desiredState).toBe(-1);
    });

    it("gets immutable list of descendents", function(){
        var serviceA = getAService(serviceDefA.ID),
            serviceB = getAService(serviceDefB.ID);

        // C is already a child of B. lets
        // make B a child of A
        serviceA.addChild(serviceB);
        var descendents = serviceA.descendents;

        expect(descendents.length).toBe(2);

        // PhantomJS fails this assertion :/
        //expect(function(){ descendents.push("nope!"); }).toThrow();
    });


    it("creates a new array of descendents when cache is invalidated", function(){
        var serviceA = getAService(serviceDefA.ID),
            serviceB = getAService(serviceDefB.ID);

        // C is already a child of B. lets
        // make B a child of A
        serviceA.addChild(serviceB);
        var descendents = serviceA.descendents;

        expect(serviceA.descendents).toBe(descendents);

        // update invalidates all caches
        serviceA.update();

        expect(serviceA.descendents).not.toBe(descendents);
    });

});
