/* global jasmine: true, httpify: true */
var translateMock = function($provide) {
    $provide.factory('$translate', function($q) {
        var mock = jasmine.createSpyObj('$translate', [
            "instant",
            "storageKey",
            "storage",
            "preferredLanguage"
        ]);

        mock.instant = mock.instant.and.callFake(function(str){
            return str + " translated";
        });

        return mock;
    });
};
