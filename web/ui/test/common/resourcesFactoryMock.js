/* global jasmine: true, httpify: true */
// An Angular factory that returns a mock implementation of resourcesFactory.js
//
// Call 'beforeEach(module(authServiceMock))' to inject this factory into a test case, and
// Angular will then inject an instance of the spy created by this factory.
var resourcesFactoryMock = function($provide) {
    $provide.factory('resourcesFactory', function($q) {
        var mock = jasmine.createSpyObj('resourcesFactory', [
            "assignIP", "getPools", "getPoolIP",
            "getService", "startService", "stopService", "restartService",
            "getServices", "getUIConfig"
        ]);

        var currDeferred;

        mock.getServices = mock.getServices.and.callFake(function(){
            currDeferred = httpify($q.defer());
            return currDeferred.promise;
        });

        // expose the last used deferred promise so that it
        // can be manually fulfilled with mock data
        mock._getCurrDeferred = function(){
            return currDeferred;
        };

        mock.startService = mock.startService.and.callFake(function(){
            currDeferred = httpify($q.defer());
            return currDeferred.promise;
        });
        mock.startService = mock.stopService.and.callFake(function(){
            currDeferred = httpify($q.defer());
            return currDeferred.promise;
        });
        mock.startService = mock.restartService.and.callFake(function(){
            currDeferred = httpify($q.defer());
            return currDeferred.promise;
        });

        mock.getUIConfig = mock.getUIConfig.and.callFake(function(){
            currDeferred = httpify($q.defer());
            return currDeferred.promise;
        });

        return mock;
    });
};
