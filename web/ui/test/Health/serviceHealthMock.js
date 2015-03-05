/* global jasmine: true, httpify: true */
var serviceHealthMock = function($provide) {
    $provide.factory('$serviceHealth', function($q) {
        var mock = jasmine.createSpyObj('serviceHealth', [
            "update",
            "get"
        ]);

        var currDeferred;

        mock.update = mock.update.and.callFake(function(){
            currDeferred = httpify($q.defer());
            return currDeferred.promise;
        });

        mock.get = mock.get.and.callFake(function(id){
            return {};
        });

        // expose the last used deferred promise so that it
        // can be manually fulfilled with mock data
        mock._getCurrDeferred = function(){
            return currDeferred;
        };

        return mock;
    });
};
