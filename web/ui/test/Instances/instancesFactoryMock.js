/* global jasmine: true */
var instancesFactoryMock = function($provide) {
    $provide.factory('instancesFactory', function() {
        var mock = jasmine.createSpyObj('instancesFactory', [
            "update",
            "activate",
            "deactivate"
        ]);
        return mock;
    });
};
