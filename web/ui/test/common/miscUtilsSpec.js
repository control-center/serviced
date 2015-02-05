

describe('miscUtils', function() {
    var $scope = null;
    var miscUtils = null;

    beforeEach(module('controlplaneTest'));
    beforeEach(module('miscUtils'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        miscUtils = $injector.get('miscUtils')

        // FIXME: would it be better to remove the 'if (DEBUG)' code in miscUtils.js instead?
        DEBUG = null
    }));

    afterEach(function(){
        delete DEBUG
    });

    describe('buildTable', function() {
        it('Returns object with sort_icons', function() {
            var headers = [ {id: 'foo'}, {id: 'bar'}, {id: 'baz'}];

            var table = miscUtils.buildTable('foo', headers);

            expect(table.sort).toBe('foo');
            expect(table.sort_icons).not.toBeUndefined();
            expect(table.set_order).not.toBeUndefined();
            expect(table.get_order_class).not.toBeUndefined();
        });
    });

    describe('getFullPath', function() {
        it('Returns pool.Id when there is no parent', function() {
            var pool = { ID: 'Test' };
            expect(miscUtils.getFullPath({}, pool)).toBe(pool.ID);

            pool = { ID: 'Test', ParentId: 'Nonexistent' };
            expect(miscUtils.getFullPath({}, pool)).toBe(pool.ID);

            // Should handle null
            expect(miscUtils.getFullPath(null, pool)).toBe(pool.ID);
        });

        it('Returns hierarchical path', function() {
            var allPools = {
                'Test3': { ID: 'Test3', ParentId: 'Test2' },
                'Test2': { ID: 'Test2', ParentId: 'Test1' },
                'Test1': { ID: 'Test1', ParentId: '' }
            }
            var pool = allPools['Test3'];
            expect(miscUtils.getFullPath(allPools, pool)).toBe('Test1 > Test2 > Test3');
        });
    });

    describe('get_order_class', function() {
        it('Includes \'active\' for value or -value of table.sort', function() {
            var table = {
                sort: 'foo',
                sort_icons: { foo: 'bar', baz: 'wibble' }
            };
            expect(miscUtils.get_order_class('foo', table)).toMatch(/ active$/);

            table.sort = '-foo';
            expect(miscUtils.get_order_class('foo', table)).toMatch(/ active$/);
            expect(miscUtils.get_order_class('baz', table)).toMatch(/ wibble$/);
        });

        it('Returns not \'active\' when no match on table.sort', function() {
            var table = {
                sort: 'foo',
                sort_icons: { foo: 'bar', baz: 'wibble' }
            };

            var result = miscUtils.get_order_class('no-sort-match', table);
            expect(result).not.toMatch(/ active$/);
            expect(result).toMatch(/ undefined$/);
        });
    });

    describe('map_to_array', function() {
        it('Transforms map to a new array', function() {
            var dummy_data = { test1: 'abc', test2: { foo: 'bar' }};
            var dummy_data_array = miscUtils.map_to_array(dummy_data);
            expect(dummy_data_array).toEqual(['abc', {foo: 'bar'}]);
        });

        it('Transforms empty map to empty array', function() {
            var dummy_data = {};
            var dummy_data_array = miscUtils.map_to_array(dummy_data);
            expect(dummy_data_array).toEqual([]);
        });
    });

    describe('refreshPools', function() {
        it('Transforms mapped pools into array in scope', function() {
            var dummy_data = fake_pools();
            var scope = {};

             miscUtils.refreshPools(scope, fake_resources_service(), false);

            // refreshPools now adds data above and beyond just transforming into an array
            for (var i=0; i < scope.pools.data.length; i++) {
                // Expect the basic fields to be consistent
                var dummyPool = dummy_data[scope.pools.data[i].Id];
                expect(dummyPool.Name).toEqual(scope.pools.data[i].Name);
                expect(dummyPool.ParentId).toEqual(scope.pools.data[i].ParentId);
                expect(dummyPool.CoreLimit).toEqual(scope.pools.data[i].CoreLimit);
                expect(dummyPool.MemoryLimit).toEqual(scope.pools.data[i].MemoryLimit);
                expect(dummyPool.Priority).toEqual(scope.pools.data[i].Priority);
            }
        });

        it('Puts pool data in scope', function() {
            var dummy_data = fake_pools();
            var scope = {};

            miscUtils.refreshPools(scope, fake_resources_service(), false);

            for (key in dummy_data) {
                var scopedPool = scope.pools.mapped[key];
                var dummyPool = dummy_data[key];
                expect(dummyPool.Name).toEqual(scopedPool.Name);
                expect(dummyPool.ParentId).toEqual(scopedPool.ParentId);
                expect(dummyPool.CoreLimit).toEqual(scopedPool.CoreLimit);
                expect(dummyPool.MemoryLimit).toEqual(scopedPool.MemoryLimit);
                expect(dummyPool.Priority).toEqual(scopedPool.Priority);
            }
        });

        it('Sets the current pool based on scope.params', function() {
            var dummy_data = fake_pools();
            var dummy_data_array = miscUtils.map_to_array(dummy_data);
            var scope = { params: { poolID: dummy_data_array[0].Id }};

            miscUtils.refreshPools(scope, fake_resources_service(), false);

            expect(scope.pools.current).not.toBeUndefined();
            expect(scope.pools.current.Name).toBe(dummy_data_array[0].Name);
        });
    });

    describe('set_order', function() {
        it('Updates table.sort', function() {
            var table = {
                sort: 'foo',
                sort_icons: { foo: 'bar', baz: 'wibble' }
            };

            miscUtils.set_order('foo', table);
            expect(table.sort).toBe('-foo');

            miscUtils.set_order('foo', table);
            expect(table.sort).toBe('foo');

            miscUtils.set_order('bar', table);
            expect(table.sort).toBe('bar');
        });

        it('Updates table.sort_icons', function() {
            var table = {
                sort: 'foo',
                sort_icons: { foo: 'bar', baz: 'wibble' }
            };

            miscUtils.set_order('bar', table);
            expect(table.sort_icons['foo']).toBe('glyphicon-chevron-down');
            expect(table.sort_icons['bar']).toBe('glyphicon-chevron-up');
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
    }

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
           get_pools: function(cacheOk, callback) {
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
