/* global jasmine: true */
var instancesFactoryMock = function($provide) {
    $provide.factory('instancesFactory', function() {
        var mock = jasmine.createSpyObj('instancesFactory', [
            "update",
            "activate",
            "deactivate",
            "getByServiceId",
        ]);

        var instances = [];
        mock._pushInstance = function(instance){
            instances.push(instance);
        };
        mock.getByServiceId = mock.getByServiceId.and.callFake(function(id){
            return instances.filter(function(i){ return i.model.ServiceID === id; });
        });

        return mock;
    });
};
