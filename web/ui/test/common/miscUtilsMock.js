// An Angular factory that returns a mock implementation of the miscUtils
//
// Call 'beforeEach(module(miscUtilsMock))' to inject this factory into a test case, and
// Angular will then inject an instance of the spy created by this factory.
var miscUtilsMock = function($provide) {
    $provide.factory('miscUtils', function() {
        var mock = jasmine.createSpyObj('miscUtils', [
            'unauthorized',
            'indentClass',
            'downloadFile',
            'getModeFromFilename',
            'updateLanguage',
            'capitalizeFirst',
            'after',
            'mapToArr',
            'memoize',
            'needsHostAlias',
            'parseEngineeringNotation',
            'validateRAMLimit'
        ]);

        return mock;
    });
};
