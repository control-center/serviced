/* global jasmine: true, spyOn: true, beforeEach: true, DEBUG: true, expect: true, inject: true, module: true */

describe('miscUtils', function() {
    var $scope = null;
    var miscUtils = null;

    beforeEach(module('controlplaneTest'));
    beforeEach(module('miscUtils'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        miscUtils = $injector.get('miscUtils');

        // FIXME: would it be better to remove the 'if (DEBUG)' code in miscUtils.js instead?
        DEBUG = null;
    }));

    describe('mapToArr', function() {
        it('Transforms map to a new array', function() {
            var dummy_data = { test1: 'abc', test2: { foo: 'bar' }};
            var dummy_data_array = miscUtils.mapToArr(dummy_data);
            expect(dummy_data_array).toEqual(['abc', {foo: 'bar'}]);
        });

        it('Transforms empty map to empty array', function() {
            var dummy_data = {};
            var dummy_data_array = miscUtils.mapToArr(dummy_data);
            expect(dummy_data_array).toEqual([]);
        });
    });

    describe("memoize", function(){
        it("Calls the memoized function", function(){
            var fn = jasmine.createSpy("memoized"),
                hash = jasmine.createSpy("hash");

            var memoized = miscUtils.memoize(fn, hash);
            memoized();

            expect(fn).toHaveBeenCalled();
        });

        it("Doesn't call the memoized function twice", function(){
            var fn = jasmine.createSpy("memoized"),
                hash = jasmine.createSpy("hash");

            var memoized = miscUtils.memoize(fn, hash);
            memoized();
            memoized();

            expect(fn.calls.count()).toEqual(1);
        });

        it("Passes args to the memoized function", function(){
            var fn = jasmine.createSpy("memoized"),
                hash = jasmine.createSpy("hash"),
                args = [1,2,3];

            var memoized = miscUtils.memoize(fn, hash);
            memoized.apply(undefined, args);

            expect(fn).toHaveBeenCalled();
            expect(fn.calls.argsFor(0)).toEqual(args);
        });

        it("Passes args to the hash function", function(){
            var fn = jasmine.createSpy("memoized"),
                hash = jasmine.createSpy("hash"),
                args = [1,2,3];

            var memoized = miscUtils.memoize(fn, hash);
            memoized.apply(undefined, args);

            expect(hash.calls.argsFor(0)).toEqual(args);
        });

        it("Calls the memoized function again when the hash result changes", function(){
            var fn = jasmine.createSpy("memoized"),
                hashVal = 0,
                hash = jasmine.createSpy("hash"),
                hashFn = hash.and.callFake(function(){ return hashVal; });

            var memoized = miscUtils.memoize(fn, hashFn);
            memoized();
            memoized();

            expect(fn.calls.count()).toEqual(1);

            // fake hash returning a new/different value
            hashVal = 1;

            memoized();
            expect(fn.calls.count()).toEqual(2);

        });
    });

    describe("capitalizeFirst", function(){
        it("Capitalizes the first character in a string", function(){
            expect(miscUtils.capitalizeFirst("hello")).toEqual("Hello");
        });

        it("Capitalizes the first character in a single character string", function(){
            expect(miscUtils.capitalizeFirst("h")).toEqual("H");
        });

        it("Handles an empty string", function(){
            expect(miscUtils.capitalizeFirst("")).toEqual("");
        });
    });

    // FIXME: Need to refactor unauthorized so that location can be injected as illustrated below.
    // describe('unauthorized', function() {
    //     it('Sets the path to /login', function() {
    //         var loc = { path: function(){} };
    //         spyOn(loc, 'path');

    //         miscUtils.unauthorized(loc);

    //         expect(loc.path).toHaveBeenCalled();
    //         expect(loc.path.mostRecentCall.args[0]).toBe('/login');
    //     });
    // });


    ///////////////////////////////////////////////////////////////////////////
    // FIXME: These 'fakes' were copied from the deleted file
    //          serviced/web/static/tests/spec.js.
    //        Using a Jasmine spy might be better
    ///////////////////////////////////////////////////////////////////////////

    var fake_hosts = function() {
        return {
            "abc": {
                Id: "abc",
                PoolID: "default",
                Name: "abchost",
                IpAddr: "192.168.33.12",
                Cores: 2,
                Memory: 3061190144,
                PrivateNetwork: "255.255.255.0"
            },
            "def": {
                Id: "def",
                PoolID: "default",
                Name: "defhost",
                IpAddr: "192.168.33.13",
                Cores: 1,
                Memory: 12345,
                PrivateNetwork: "255.255.255.0"
            },
            "host123": {
                Id: "host123",
                PoolID: "pool123",
                Name: "some fake host",
                IpAddr: "192.168.33.14",
                Cores: 2,
                Memory: 2048
            },
            "fakeHost1": {
                Id: "fakeHost1",
                PoolID: "pool123",
                Name: "some fake host",
                IpAddr: "192.168.33.15",
                Cores: 2,
                Memory: 2048
            }
        };
    };

    var fake_hosts_for_pool = function(poolId) {
        var mappedHosts = {
            "pool123": [{HostID: "host123", PoolID: "pool123"}],
            "default": [{HostID: "abc", PoolID: "default"}, {HostID: "def", PoolID: "default"}]
        };
        return mappedHosts[poolId];
    };

    var fake_pools = function() {
        return {
            "default": {
                Id: "default",
                ParentId: "",
                CoreLimit: 0,
                MemoryLimit: 0,
                Priority: 0
            },
            "foo": {
                Id: "foo",
                ParentId: "default",
                CoreLimit: 2,
                MemoryLimit: 1024,
                Priority: 1
            },
            "bar": {
                Id: "bar",
                ParentId: "default",
                CoreLimit: 8,
                MemoryLimit: 8192,
                Priority: 2
            },
            "pool123": {
                Id: "pool123",
                ParentId: "foo",
                CoreLimit: 1,
                MemoryLimit: 512,
                Priority: 2
            }
        };
    };

    var fake_services = function() {
        return [
            fake1, service234
        ];
    };

    var fake_service_logs = function() {
        return { Detail: "foo bar" };
    };

    var fake_services_tree = function() {
        fake1.children = [ fake1Child ];
        var tree = {};
        fake_services().map(function(e) {
            tree[e.Id] = e;
        });
        return tree;
    };

    var fake_snapshot_service = function() {
        return { Detail: "once upon a time" };
    };

    var fake_resources_service =  function() {
       return {
           getPools: function(cacheOk, callback) {
               callback(fake_pools());
           },
           get_hosts: function(cacheOk, callback) {
               callback(fake_hosts());
           },
           get_hosts_for_pool: function(cacheOk, poolId, callback) {
               callback(fake_hosts_for_pool(poolId));
           },
           add_pool: function(pool, callback) {
               callback({});
           },
           add_host: function(host, callback) {
               callback({});
           },
           remove_pool: function(poolId, callback) {
               callback({});
           },
           remove_host: function(hostId, callback) {
               callback({});
           },
           update_pool: function(poolId, pool, callback) {
               callback({});
           },
           update_host: function(hostId, host, callback) {
               callback({});
           },
           get_app_templates: function(cacheOk, callback) {
               callback(fake_templates());
           },
           get_services: function(cacheOk, callback) {
               callback(fake_services(), fake_services_tree());
           },
           get_service_logs: function(serviceId, callback) {
               callback(fake_service_logs());
           },
           get_service_state_logs: function(serviceId, serviceStateId, callback) {
               callback(fake_service_logs());
           },
           get_running_services_for_service: function(serviceId, callback) {
               callback(fake_running_for_host());
           },
           get_running_services_for_host: function(hostId, callback) {
               callback(fake_running_for_host());
           },
           add_service: function(service, callback) {
               callback({});
           },
           snapshot_service: function(serviceId, callback) {
               callback(fake_snapshot_service());
           },
           update_service: function(serviceId, service, callback) {
               callback({});
           },
           remove_service: function(serviceId, callback) {
               callback({});
           },
           start_service: function(serviceId, callback) {
               callback({});
           },
           stop_service: function(serviceId, callback) {
               callback({});
           }
       };
    };

 });
