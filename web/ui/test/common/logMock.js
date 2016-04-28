var logMock = function($provide) {
    $provide.factory('log', function() {
        var mock = jasmine.createSpyObj('log', [
            "setLogLevel",
            "debug",
            "log",
            "info",
            "warn",
            "error"
        ]);

        return mock;
    });
};
