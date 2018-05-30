var angularAuth0Mock = function($provide) {
    $provide.factory('angularAuth0', function() {
        var mock = jasmine.createSpyObj('angularAuth0', [
            "checkSession:," +
            "authorize"
        ]);

        return mock;
    });
};
