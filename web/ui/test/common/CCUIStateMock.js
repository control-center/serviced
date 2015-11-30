// An Angular factory that returns a mock implementation of the CCUIState
//
// Call 'beforeEach(module(CCUIStateMock))' to inject this factory into a test case, and
// Angular will then inject an instance of the spy created by this factory.
var CCUIStateMock = function($provide) {
    $provide.factory('CCUIState', function() {
        var mock = jasmine.createSpyObj('CCUIState', [
            "get"
        ]);

        return mock;
    });
};
