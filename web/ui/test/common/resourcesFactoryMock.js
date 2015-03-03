/* global jasmine: true */
// An Angular factory that returns a mock implementation of resourcesFactory.js
//
// Call 'beforeEach(module(authServiceMock))' to inject this factory into a test case, and
// Angular will then inject an instance of the spy created by this factory.
var resourcesFactoryMock = function($provide) {
    $provide.factory('resourcesFactory', function() {
        var mock = jasmine.createSpyObj('resourcesFactory', []);
        return mock;
    });
};
