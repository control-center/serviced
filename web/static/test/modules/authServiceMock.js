// An Angular factory that returns a mock implementation of the authService
//
// Call 'beforeEach(module(authServiceMock))' to inject this factory into a test case, and
// Angular will then inject an instance of the spy created by this factory.
var authServiceMock = function($provide) {
    $provide.factory('authService', function() {
        var mock = jasmine.createSpyObj('authService', [
            'setLoggedIn',
            'login',
            'logout',
            'checkLogin'
        ]);

        return mock;
    });
};
