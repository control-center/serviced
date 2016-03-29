/* global spyOn: true, beforeEach: true, expect: true, inject: true, module: true, translateMock: true */

describe('serviceHealth', function() {
    beforeEach(module('controlplaneTest'));

    // load up actual services
    beforeEach(function(){
        module('serviceHealth');
    });

    // load up mock services
    beforeEach(function(){
        module(translateMock);
    });

    var serviceHealth, hcStatus;
    beforeEach(inject(function($injector) {
        serviceHealth = $injector.get('$serviceHealth');
        hcStatus = $injector.get('hcStatus');
    }));

    // service desired states
    var STARTED = 1;
    var STOPPED = 0;

    function createMockService(serviceID){
        var id = serviceID || Math.floor(Math.random() * 10000);
        return {
            id: id,
            name: "mockservice" + id,
            model: {
                DesiredState: STOPPED
            },
            instances: [],
            getServiceInstances: function(){return this.instances;}
        };
    }
    function createMockInstance(serviceID){
        return {
            id: Math.floor(Math.random() * 10000),
            model: {
                ServiceID: serviceID,
                HealthChecks: {}
            }
        };
    }

    function createServiceWithHealthCheck(desiredState, healthCheckStatus){
        var id = Math.floor(Math.random() * 10000);

        // create instance
        var mockInstance = createMockInstance(id);
        // set healthcheck status
        mockInstance.healthChecks = {
            "check1": healthCheckStatus
        };

        // create service
        var mockService = createMockService(id);
        mockService.model.DesiredState = desiredState;

        // attach instance
        mockService.instances = [mockInstance];
        var mockServiceList = {};
        mockServiceList[id] = mockService;

        // update service
        serviceHealth.update(mockServiceList);

        return id;
    }

    // shouldnt running, no healthchecks, good!
    it("marks a stopped service with no health check as not running", function(){
        var mockService = createMockService();
        var mockServiceList = {};
        mockServiceList[mockService.id] = mockService;
        var statuses = serviceHealth.update(mockServiceList);
        expect(statuses[mockService.id].status).toBe(hcStatus.NOT_RUNNING);
    });

    // should be running, but no healthchecks yet. Probably starting up
    it("marks a started service with no health check as unknown", function(){
        var mockService = createMockService();
        mockService.model.DesiredState = STARTED;
        var mockServiceList = {};
        mockServiceList[mockService.id] = mockService;
        var statuses = serviceHealth.update(mockServiceList);
        expect(statuses[mockService.id].status).toBe(hcStatus.UNKNOWN);
    });

    // shouldnt be running, but still passing healthchecks. maybe shutting down?
    it("marks a stopped service with a passing health check as unknown", function(){
        var id = createServiceWithHealthCheck(STOPPED, hcStatus.OK);
        var status = serviceHealth.get(id);
        expect(status.status).toBe(hcStatus.UNKNOWN);
    });

    // should be running, health checks passing, good!
    it("marks a started service with a passing health check as passing", function(){
        var id = createServiceWithHealthCheck(STARTED, hcStatus.OK);
        var status = serviceHealth.get(id);
        expect(status.status).toBe(hcStatus.OK);
    });

    // shouldnt be running, health checks failing, maybe shutting down?
    it("marks a stopped service with a failed health check as unknown", function(){
        var id = createServiceWithHealthCheck(STOPPED, hcStatus.FAILED);
        var status = serviceHealth.get(id);
        expect(status.status).toBe(hcStatus.UNKNOWN);
    });

    // should be running, but health checks failing, definitely bad!
    it("marks a started service with a failed health check as failed", function(){
        var id = createServiceWithHealthCheck(STARTED, hcStatus.FAILED);
        var status = serviceHealth.get(id);
        expect(status.status).toBe(hcStatus.FAILED);
    });

    // shouldnt be running, health checks marked not running, good!
    it("marks a stopped service with a not running health check as not running", function(){
        var id = createServiceWithHealthCheck(STOPPED, hcStatus.NOT_RUNNING);
        var status = serviceHealth.get(id);
        expect(status.status).toBe(hcStatus.NOT_RUNNING);
    });

    // should be running, health checks marked not running, maybe starting up?
    it("marks a started service with a not running health check as unknown", function(){
        var id = createServiceWithHealthCheck(STARTED, hcStatus.NOT_RUNNING);
        var status = serviceHealth.get(id);
        expect(status.status).toBe(hcStatus.UNKNOWN);
    });

    // shouldnt be running, health checks unknown, not really sure here.
    it("marks a stopped service with an unknown health check as unknown", function(){
        var id = createServiceWithHealthCheck(STOPPED, hcStatus.UNKNOWN);
        var status = serviceHealth.get(id);
        expect(status.status).toBe(hcStatus.UNKNOWN);
    });

    // should be running, health checks unknown, maybe in the process of failing?
    it("marks a started service with an unknown health check as unknown", function(){
        var id = createServiceWithHealthCheck(STARTED, hcStatus.UNKNOWN);
        var status = serviceHealth.get(id);
        expect(status.status).toBe(hcStatus.UNKNOWN);
    });

});
