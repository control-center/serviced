/* tests */
describe('EntryControl', function() {
    var $scope = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($rootScope, $controller) {
        $scope = $rootScope.$new();
        ctrl = $controller('EntryControl', { $scope: $scope });
    }));

    it('Sets 2 main links', function() {
        expect($scope.mainlinks.length).toEqual(2);
    });

    it('Creates links that contain url and label', function() {
        for(var i=0; i < $scope.mainlinks.length; i++) {
            expect($scope.mainlinks[i].url).toMatch(/^#\/.+/);
            expect($scope.mainlinks[i].label).not.toBeUndefined();
        }
    });

});

describe('LoginControl', function() {
    var $scope = null;
    var $httpBackend = null;
    var $location = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        $httpBackend = $injector.get('$httpBackend');
        ctrl = $controller('LoginControl', { $scope: $scope });
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });
 
    it('Sets some labels', function() {
        expect($scope.brand_label).not.toBeUndefined();
        expect($scope.login_button_text).not.toBeUndefined();
    });

    it('Sets path on successful login', function() {
        $httpBackend.when('POST', '/login').respond({Detail: 'SuccessfulPost'});
        $scope.login();
        $httpBackend.flush();
        expect($location.path()).toBe('/entry');
    });

    it('Does not change path on failed login', function() {
        $location.path('/login');
        $httpBackend.when('POST', '/login').respond(401, {Detail: 'Bad Login'});
        $scope.login();
        $httpBackend.flush();
        expect($location.path()).toBe('/login');
    });

});

describe('DeployedAppsControl', function() {
    var $scope = null;
    var $location = null;
    var resourcesService = null;
    var servicesService = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        resourcesService = fake_resources_service();
        servicesService = fake_services_service();
        ctrl = $controller('DeployedAppsControl', { 
            $scope: $scope, 
            resourcesService: resourcesService,
            servicesService: servicesService
        });
    }));

    it('Builds a services table', function() {
        expect_table($scope.services);
    });

    it('Provides a \'click_app\' function', function() {
        expect($scope.click_app).not.toBeUndefined();
        $scope.click_app('test');
        expect($location.path()).toBe('/services/test');
    });
});

describe('SubServiceControl', function() {
    var $scope = null;
    var $location = null;
    var resourcesService = null;
    var servicesService = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        resourcesService = fake_resources_service();
        servicesService = fake_services_service();
        ctrl = $controller('SubServiceControl', { 
            $scope: $scope, 
            resourcesService: resourcesService,
            servicesService: servicesService
        });
    }));

    it('Builds a services table', function() {
        expect_table($scope.services);
    });
});

describe('HostsControl', function() {
    var $scope = null;
    var $location = null;
    var resourcesService = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        resourcesService = fake_resources_service();
        servicesService = fake_services_service();
        ctrl = $controller('HostsControl', { 
            $scope: $scope, 
            resourcesService: resourcesService
        });
    }));

    it('Builds a pools table', function() {
        expect_table($scope.pools);
    });

    it('Builds a hosts table', function() {
        expect_table($scope.hosts);
    });

    it('Provides starter object for creating new pools and hosts', function() {
        expect($scope.newPool).not.toBeUndefined();
        expect($scope.newHost).not.toBeUndefined();
    });

    it('Provides an \'add_host\' function', function() {
        spyOn(resourcesService,'add_host');
        $scope.add_host({ IpAddr: '127.0.0.1'});
        expect(resourcesService.add_host).toHaveBeenCalled();
    });

    it('Provides a \'remove_pool\' function', function() {
        spyOn(resourcesService,'remove_pool');
        $scope.remove_pool('test');
        expect(resourcesService.remove_pool).toHaveBeenCalled();
    });

    it('Provides \'filterHosts\' function', function() {
        // By default this should produce the same as all hosts
        var filtered = $scope.filterHosts();
        expect(filtered).toEqual($scope.hosts.all);
    });

    it('Provides \'dropIt\' function for drag and drop', function() {
        // dropIt is hard to test without a browser due to jquery selector
        expect(typeof $scope.dropIt).toBe('function');
    });

});


describe('DeployWizard', function() {
    var $scope = null;
    var resourcesService = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        ctrl = $controller('DeployWizard', { 
            $scope: $scope, 
            resourcesService: fake_resources_service()
        });
    }));

    it('Defines a set of steps', function() {
        expect($scope.steps).not.toBeUndefined();
        expect($scope.steps.length).not.toBeUndefined();
        for (var i=0; i < $scope.steps.length; i++) {
            var step = $scope.steps[i];
            expect(step.content).not.toBeUndefined();
            expect(step.label).not.toBeUndefined();
        }
    });

    it('Creates an install context', function() {
        expect($scope.install).not.toBeUndefined();
        expect($scope.install.selected).not.toBeUndefined();
        expect($scope.install.templateData).not.toBeUndefined();
        expect(typeof $scope.install.templateClass).toBe('function');
        expect(typeof $scope.install.templateSelected).toBe('function');
        expect(typeof $scope.install.templateDisabled).toBe('function');
    });

    it('Provides a \'wizard_next\' function', function() {
        expect($scope.step_page).toBe($scope.steps[0].content);
        $scope.wizard_next();
        expect($scope.step_page).toBe($scope.steps[1].content);
    });


    it('Provides a \'wizard_previous\' function', function() {
        expect($scope.step_page).toBe($scope.steps[0].content);
        $scope.wizard_next();
        expect($scope.step_page).toBe($scope.steps[1].content);
        $scope.wizard_previous();
        expect($scope.step_page).toBe($scope.steps[0].content);
    });
});


describe('ActionControl', function() {
    var $scope = null;
    var $location = null;
    var resourcesService = null;
    var servicesService = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        resourcesService = fake_resources_service();
        servicesService = fake_services_service();
        ctrl = $controller('ActionControl', { 
            $scope: $scope, 
            $routeParams: { poolId: 'pool123', hostId: 'host123', serviceId: 'service234' },
            servicesService: servicesService,
            resourcesService: resourcesService,
        });
        $scope.pools = {};
        $scope.hosts = {};
        $scope.services = {};
    }));
    
    it('Creates a stub for new pools', function() {
        expect($scope.newPool).not.toBeUndefined();
        expect($scope.newPool.ParentId).toBe('pool123');
    });

    it('Crates a stub for new hosts', function() {
        expect($scope.newHost).not.toBeUndefined();
        expect($scope.newHost.PoolId).toBe('pool123');
    });

    it('Creates a stub for new services', function() {
        expect($scope.newService).not.toBeUndefined();
    });

    it('Provides an \'add_pool\' function', function() {
        // add_pool method must not be called unless $scope.pools exists
        expect($scope.add_pool).not.toBeUndefined();
        $scope.newPool.Id = 'aoeu';
        $scope.newPool.ParentId = 'fakeparent';
        spyOn(resourcesService, 'add_pool');
        $scope.add_pool();
        expect(resourcesService.add_pool).toHaveBeenCalled();
        var addedPool = resourcesService.add_pool.mostRecentCall.args[0];
        // Expect the values we set earlier
        expect(addedPool).toEqual({Id: 'aoeu', ParentId: 'fakeparent'});
        // Expect newPool to be a new object with parentId from routeParams
        expect($scope.newPool).toEqual({ParentId: 'pool123'});
    });

    it('Provides an \'add_host\' function', function() {
        // add_host method must not be called unless $scope.hosts exists
        expect($scope.add_host).not.toBeUndefined();
        $scope.newHost.Id = 'aoeu';
        $scope.newHost.PoolId = 'fakeparent';
        spyOn(resourcesService, 'add_host');
        $scope.add_host();
        expect(resourcesService.add_host).toHaveBeenCalled();
        var addedHost = resourcesService.add_host.mostRecentCall.args[0];
        // Expect the values we set earlier
        expect(addedHost).toEqual({Id: 'aoeu', PoolId: 'fakeparent'});
        // Expect addedHost to be a new object with pool ID from routeParams
        expect($scope.newHost).toEqual({PoolId: 'pool123'});
    });

    it('Provides an \'add_service\' function', function() {
        // add_service method must not be called unless $scope.pools exists
        expect($scope.add_service).not.toBeUndefined();
        $scope.newService.Id = 'aoeu';
        $scope.newService.PoolId = 'fakeparent';
        spyOn(servicesService, 'add_service');
        $scope.add_service();
        expect(servicesService.add_service).toHaveBeenCalled();
        var addedService = servicesService.add_service.mostRecentCall.args[0];
        // Expect the values we set earlier
        expect(addedService).toEqual({Id: 'aoeu', PoolId: 'fakeparent'});
        // Expect newService to be an empty object
        expect($scope.newService).toEqual({});
    });

    it('Provides a \'remove_pool\' function', function() {
        spyOn(resourcesService, 'remove_pool');
        $scope.remove_pool();
        expect(resourcesService.remove_pool).toHaveBeenCalled();
        var remPoolId = resourcesService.remove_pool.mostRecentCall.args[0];
        // Pool ID comes from routeParams
        expect(remPoolId).toBe('pool123');
        // Below should be tested, but currently depends on modal dialog being hidden
        //expect($location.path()).toBe('/resources');
    });

    it('Provides a \'remove_host\' function', function() {
        spyOn(resourcesService, 'remove_host');
        $scope.remove_host();
        expect(resourcesService.remove_host).toHaveBeenCalled();
        var remHostId = resourcesService.remove_host.mostRecentCall.args[0];
        // Host ID comes from routeParams
        expect(remHostId).toBe('host123');
        // Below should be tested, but currently depends on modal dialog being hidden
        //expect($location.path()).toBe('/pools/pool123');
    });

    it('Provides a \'remove_service\' function', function() {
        spyOn(servicesService, 'remove_service');
        $scope.remove_service();
        expect(servicesService.remove_service).toHaveBeenCalled();
        // service Id comes from routeParams
        expect(servicesService.remove_service.mostRecentCall.args[0]).toBe('service234');
        // expect($location.path()).toBe('/services');7
    });

    it('Provides an \'edit_pool\' function', function() {
        // Populate $scope.pools.current. Normally this is done by the main view controller
        refreshPools($scope, resourcesService, true);
        // edit_pool method must not be called unless $scope.pools.current exists
        expect($scope.editPool).not.toBeUndefined();
        spyOn(resourcesService, 'update_pool');
        $scope.editPool.Name = 'editedName';
        $scope.edit_pool();

        expect(resourcesService.update_pool).toHaveBeenCalled();
        expect(resourcesService.update_pool.mostRecentCall.args[0]).toBe('pool123');
        expect(resourcesService.update_pool.mostRecentCall.args[1]).toBe($scope.editPool);
    });

    it('Provides an \'edit_host\' function', function() {
        // Populate $scope.hosts.current. Normally this is done by the main view controller
        refreshHosts($scope, resourcesService, true, true);
        // edit_host method must not be called unless $scope.hosts.current exists
        expect($scope.editHost).not.toBeUndefined();
        spyOn(resourcesService, 'update_host');
        $scope.editHost.Name = 'editedName';
        $scope.edit_host();

        expect(resourcesService.update_host).toHaveBeenCalled();
        expect(resourcesService.update_host.mostRecentCall.args[0]).toBe('host123');
        expect(resourcesService.update_host.mostRecentCall.args[1]).toBe($scope.editHost);
    });

    it('Provides an \'edit_service\' function', function() {
        // Populate $scope.hosts.current. Normally this is done by the main view controller
        refreshServices($scope, servicesService, true);
        // edit_service method must not be called unless $scope.services.current exists
        expect($scope.editService).not.toBeUndefined();
        spyOn(servicesService, 'update_service');
        $scope.editService.Id = 'editedService';
        $scope.edit_service();

        expect(servicesService.update_service).toHaveBeenCalled();
        // service Id comes from routeParams
        expect(servicesService.update_service.mostRecentCall.args[0]).toBe('service234');
        expect(servicesService.update_service.mostRecentCall.args[1]).toBe($scope.editService);
    });
});

describe('NavbarControl', function() {
    var $scope = null;
    var $location = null;
    var $httpBackend = null;
    var $location = null;
    var authService = null;
    var ctrl = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        var $controller = $injector.get('$controller');
        $location = $injector.get('$location');
        $httpBackend = $injector.get('$httpBackend');
        authService = fake_auth_service();
        ctrl = $controller('NavbarControl', { 
            $scope: $scope,
            authService: authService
        });
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });

    it('Provides some navlinks', function() {
        expect($scope.navlinks).not.toBeUndefined();
        // 2 or more navlinks please.
        expect($scope.navlinks.length).toBeGreaterThan(1); 
    });

    it('Provides brand details', function() {
        expect($scope.brand).not.toBeUndefined();
        expect($scope.brand.url).not.toBeUndefined();
        expect($scope.brand.label).not.toBeUndefined();
    });

    it('Provides a \'logout\' function', function() {
        // Default for testing is to assume logged in.
        expect(authService.isLoggedIn()).toBe(true);
        $httpBackend.when('DELETE', '/login').respond({Detail: 'Logged Out'});
        $scope.logout();
        $httpBackend.flush();
        expect(authService.isLoggedIn()).toBe(false);
    });
});

describe('ServicesService', function() {
    var $scope = null;
    var $location = null;
    var $httpBackend = null;
    var $location = null;
    var servicesService = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        $location = $injector.get('$location');
        $httpBackend = $injector.get('$httpBackend');
        servicesService = $injector.get('servicesService');
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });
    
    it('Can retrieve and cache service definitions', function() {
        // The first time GET is called, we have nothing cached so the first
        // parameter is ignored.
        $httpBackend.expect('GET', '/services').respond(fake_services());
        var ser_top, ser_by_id = null;
        servicesService.get_services(false, function(top, mapped) {
            ser_top = top;
            ser_by_id = mapped;
        });
        $httpBackend.flush();
        expect(ser_top).not.toBeNull();
        expect(ser_by_id).not.toBeNull();

        // We should have some cached data this time, so do not expect any
        // HTTP traffic.
        ser_top, ser_by_id = null;
        servicesService.get_services(true, function(top, mapped) {
            ser_top = top;
            ser_by_id = mapped;
        });
        expect(ser_top).not.toBeNull();
        expect(ser_by_id).not.toBeNull();
    });

    it('Separates top level services from sub services', function() {
        // The first time GET is called, we have nothing cached so the first
        // parameter is ignored.
        $httpBackend.expect('GET', '/services').respond(fake_services());
        var ser_top, ser_by_id = null;
        servicesService.get_services(false, function(top, mapped) {
            ser_top = top;
            ser_by_id = mapped;
        });
        $httpBackend.flush();
        ser_top.map(function(ser) {
            expect(ser.ParentServiceId).toBe('');
            if (ser.children) {
                ser.children.map(function(child) {
                    expect(child.ParentServiceId).toBe(ser.Id);
                });
            }
        });
    });


    it('Can add new services', function() {
        var serv = { Id: 'test' };
        var resp = null;
        $httpBackend.expect('POST', '/services/add', serv).respond({ Detail: 'Added' });
        servicesService.add_service(serv, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Added');
    });

    it('Can update existing services', function() {
        var serv = { Id: 'test', Name: 'Test2' };
        var resp = null;
        $httpBackend.expect('POST', '/services/test', serv).respond({ Detail: 'Edited' });
        servicesService.update_service(serv.Id, serv, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Edited');
    });

    it('Can remove existing services', function() {
        var resp = null;
        $httpBackend.expect('DELETE', '/services/test').respond({ Detail: 'Deleted' });
        servicesService.remove_service('test', function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Deleted');
    });

});

describe('ResourcesService', function() {
    var $scope = null;
    var $location = null;
    var $httpBackend = null;
    var $location = null;
    var resourcesService = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $scope = $injector.get('$rootScope').$new();
        $location = $injector.get('$location');
        $httpBackend = $injector.get('$httpBackend');
        resourcesService = $injector.get('resourcesService');
    }));

    afterEach(function() {
        $httpBackend.verifyNoOutstandingExpectation();
        $httpBackend.verifyNoOutstandingRequest();
    });
    
    it('Can retrieve and cache resource pools', function() {
        // The first time GET is called, we have nothing cached so the first
        // parameter is ignored.
        $httpBackend.expect('GET', '/pools').respond(fake_pools());
        var pools = null;
        resourcesService.get_pools(false, function(data) {
            pools = data;
        });
        $httpBackend.flush();
        expect(pools).not.toBeNull();

        // We should have some cached data this time, so do not expect any
        // HTTP traffic.
        pools = null;
        resourcesService.get_pools(true, function(data) {
            pools = data;
        });
        expect(pools).not.toBeNull();
    });

    it('Can add new resource pools', function() {
        var pool = { Id: 'test' };
        var resp = null;
        $httpBackend.expect('POST', '/pools/add', pool).respond({ Detail: 'Added' });
        resourcesService.add_pool(pool, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Added');
    });

    it('Can update existing resource pools', function() {
        var pool = { Id: 'test', Name: 'Test2' };
        var resp = null;
        $httpBackend.expect('POST', '/pools/test', pool).respond({ Detail: 'Edited' });
        resourcesService.update_pool(pool.Id, pool, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Edited');
    });

    it('Can remove existing resource pools', function() {
        var resp = null;
        $httpBackend.expect('DELETE', '/pools/test').respond({ Detail: 'Deleted' });
        resourcesService.remove_pool('test', function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Deleted');
    });

    it('Can retrieve and cache hosts', function() {
        // The first time GET is called, we have nothing cached so the first
        // parameter is ignored.
        $httpBackend.expect('GET', '/hosts').respond(fake_hosts());
        var hosts = null;
        resourcesService.get_hosts(false, function(data) {
            hosts = data;
        });
        $httpBackend.flush();
        expect(hosts).not.toBeNull();

        // We should have some cached data this time, so do not expect any
        // HTTP traffic.
        hosts = null;
        resourcesService.get_hosts(true, function(data) {
            hosts = data;
        });
        expect(hosts).not.toBeNull();
    });

    it('Can add new hosts', function() {
        var host = { Id: 'test' };
        var resp = null;
        $httpBackend.expect('POST', '/hosts/add', host).respond({ Detail: 'Added' });
        resourcesService.add_host(host, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Added');
    });

    it('Can update existing hosts', function() {
        var host = { Id: 'test', Name: 'Test2' };
        var resp = null;
        $httpBackend.expect('POST', '/hosts/test', host).respond({ Detail: 'Edited' });
        resourcesService.update_host(host.Id, host, function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Edited');
    });

    it('Can remove existing hosts', function() {
        var resp = null;
        $httpBackend.expect('DELETE', '/hosts/test').respond({ Detail: 'Deleted' });
        resourcesService.remove_host('test', function(data) {
            resp = data;
        });
        $httpBackend.flush();
        expect(resp.Detail).toEqual('Deleted');
    });
});

describe('AuthService', function() {
    var $inj = null;
    var $location = null;
    var $cookies = null;
    var authService = null;

    beforeEach(module('controlplane'));

    beforeEach(inject(function($injector) {
        $inj = $injector;
        authService = $injector.get('authService');
        $location = $injector.get('$location');
        $cookies = $injector.get('$cookies');
    }));

    it('Keeps logged in flag that persists across controllers', function() {
        var scope = {};
        $cookies = {};
        // Check the base case - defaults to false
        $location.path('/before');
        authService.checkLogin(scope);
        expect($location.path()).toBe('/login');
        expect(scope.loggedIn).toBeUndefined();

        // Check basic internal state
        $location.path('/before');
        authService.login(true);
        authService.checkLogin(scope);
        expect($location.path()).toBe('/before');
        expect(scope.loggedIn).toBe(true);

        // Check across injections
        var secondAuth = $inj.get('authService');
        expect(authService).toBe(secondAuth);
        scope = {};
        secondAuth.checkLogin(scope);
        expect($location.path()).toBe('/before');
        expect(scope.loggedIn).toBe(true);
    });

    it('Checks for a cookie', function() {
        var scope = {};
        $cookies['ZCPToken'] = 'something';
        $location.path('/before');
        authService.checkLogin(scope);
        expect(scope.loggedIn).toBe(true);
        expect($location.path()).toBe('/before');
    });
});


describe('refreshServices', function() {
    it('Puts services data in scope', function() {
        var dummy_data = fake_services();
        var scope = {};
        refreshServices(scope, fake_services_service(), false);
        expect(scope.services.data).not.toBeUndefined();
        for (var i=0; i < scope.services.data.length; i++) {
            // Expect the basic fields to be consistent
            expect(dummy_data[i].Name).toEqual(scope.services.data[i].Name);
        }
    });

    it('Sets the current service based on scope.params', function() {
        var dummy_data = fake_services();
        var scope = { params: { serviceId: dummy_data[0].Id }};
        refreshServices(scope, fake_services_service(), false);
        expect(scope.services.current).not.toBeUndefined();
        expect(scope.services.current.Name).toBe(dummy_data[0].Name);
    });

    it('Maps services by ID', function() {
        var dummy_data = fake_services();
        var scope = {};
        refreshServices(scope, fake_services_service(), false);
        expect(scope.services.mapped[dummy_data[0].Id].Name).toBe(dummy_data[0].Name);
        expect(scope.services.mapped[dummy_data[1].Id].Startup).toBe(dummy_data[1].Startup);
    });
});

describe('refreshPools', function() {
    it('Transforms mapped pools into array in scope', function() {
        var dummy_data = fake_pools();
        var scope = {};
        refreshPools(scope, fake_resources_service(), false);
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
        refreshPools(scope, fake_resources_service(), false);
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
        var dummy_data_array = map_to_array(dummy_data);
        var scope = { params: { poolId: dummy_data_array[0].Id }};
        refreshPools(scope, fake_resources_service(), false);
        expect(scope.pools.current).not.toBeUndefined();
        expect(scope.pools.current.Name).toBe(dummy_data_array[0].Name);
    });
});

describe('refreshHosts', function() {
    it('Sets the current host based on scope.params', function() {
        var scope = { params: { hostId: "def" }, $watch: function() {}};
        refreshHosts(scope, fake_resources_service(), false, false);
        expect(scope.hosts.current).toEqual(fake_hosts()["def"]);
    });

    it('Puts host data into scope', function() {
        var scope = {$watch: function() {}};
        refreshHosts(scope, fake_resources_service(), false, false);
        expect(scope.hosts.mapped).toEqual(fake_hosts());
    });
});

describe('map_to_array', function() {
    it('Transforms map to a new array', function() {
        var dummy_data = { test1: 'abc', test2: { foo: 'bar' }};
        var dummy_data_array = map_to_array(dummy_data);
        expect(dummy_data_array).toEqual(['abc', {foo: 'bar'}]);
    });
});

describe('unauthorized', function() {
    it('Sets the path to /login', function() {
        var loc = { path: function(){} };
        spyOn(loc, 'path');
        unauthorized(loc);
        expect(loc.path).toHaveBeenCalled();
        expect(loc.path.mostRecentCall.args[0]).toBe('/login');
    });
});

describe('next_url', function() {
    it('Finds a link with name \'Next\'', function() {
        var result = next_url({ foo: 'bar', Links: [ 
            { Name: 'Baz', Url: '/something' }, 
            { Name: 'Next', Url: '/expected' }, 
            { Name: 'Other', Url: '/other' }
        ]});
        expect(result).toBe('/expected');
    });
});

describe('set_order', function() {
    it('Updates table.sort', function() {
        var table = {
            sort: 'foo',
            sort_icons: { foo: 'bar', baz: 'wibble' }
        };
        set_order('foo', table);
        expect(table.sort).toBe('-foo');
        set_order('foo', table);
        expect(table.sort).toBe('foo');
        set_order('bar', table);
        expect(table.sort).toBe('bar');
    });

    it('Updates table.sort_icons', function() {
        var table = {
            sort: 'foo',
            sort_icons: { foo: 'bar', baz: 'wibble' }
        };
        set_order('bar', table);
        expect(table.sort_icons['foo']).toBe('glyphicon-chevron-down');
        expect(table.sort_icons['bar']).toBe('glyphicon-chevron-up');
    });
});

describe('get_order_class', function() {
    it('Includes \'active\' for value or -value of table.sort', function() {
        var table = {
            sort: 'foo',
            sort_icons: { foo: 'bar', baz: 'wibble' }
        };
        expect(get_order_class('foo', table)).toMatch(/ active$/);
        table.sort = '-foo';
        expect(get_order_class('foo', table)).toMatch(/ active$/);
        expect(get_order_class('baz', table)).toMatch(/ wibble$/);
    });
});

describe('buildTable', function() {
    it('Returns object with sort_icons', function() {
        var headers = [ {id: 'foo'}, {id: 'bar'}, {id: 'baz'}];
        var table = buildTable('foo', headers);
        expect(table.sort).toBe('foo');
        expect(table.sort_icons).not.toBeUndefined();
        expect(table.set_order).not.toBeUndefined();
        expect(table.get_order_class).not.toBeUndefined();
    });
});

describe('updateRunning', function() {
    it('Sets text on service when state is 1', function() {
        var svc = { DesiredState: 1 };
        updateRunning(svc);
        expect(svc.runningText).toBe('started'); // started is current state
        expect(svc.notRunningText).toBe('\xA0'); // stop is action
    });

    it('Sets text on service when state is -1', function() {
        var svc = { DesiredState: -1 };
        updateRunning(svc);
        expect(svc.runningText).toBe('restarting'); // restarting is current state
        expect(svc.notRunningText).toBe('\xA0'); // stop is action
    });

    it('Sets text on service when state is 0 or other', function() {
        var svc = { DesiredState: 0 };
        updateRunning(svc);
        expect(svc.runningText).toBe('\xA0'); // start is action
        expect(svc.notRunningText).toBe('stopped'); // stopped is current state

        svc = { DesiredState: -99 };
        updateRunning(svc);
        expect(svc.runningText).toBe('\xA0'); // start is action
        expect(svc.notRunningText).toBe('stopped'); // stopped is current state

    });
});

describe('toggleRunning', function() {

    it('Sets DesiredState and updates service', function() {
        var servicesService = fake_services_service();
        var svc = {};
        spyOn(servicesService, 'update_service');

        toggleRunning(svc, 'start', servicesService);
        expect(svc.DesiredState).toBe(1);
        expect(servicesService.update_service).toHaveBeenCalled();

        toggleRunning(svc, 'stop', servicesService);
        expect(svc.DesiredState).toBe(0);
        expect(servicesService.update_service).toHaveBeenCalled();

        toggleRunning(svc, 'restart', servicesService);
        expect(svc.DesiredState).toBe(-1);
        expect(servicesService.update_service).toHaveBeenCalled();
    });
});

/*
describe('updateWorking', function() {
    it('Sets temporary text on service', function() {
        var svc = {};
        updateWorking(svc);
        expect(svc.runningText).not.toBeUndefined();
        expect(svc.notRunningText).not.toBeUndefined();
    });
});
*/

describe('getFullPath', function() {
    it('Returns pool.Id when there is no parent', function() {
        var pool = { Id: 'Test' };
        expect(getFullPath({}, pool)).toBe(pool.Id);

        pool = { Id: 'Test', ParentId: 'Nonexistent' };
        expect(getFullPath({}, pool)).toBe(pool.Id);

        // Should handle null
        expect(getFullPath(null, pool)).toBe(pool.Id);
    });

    it('Returns hierarchical path', function() {
        var allPools = {
            'Test3': { Id: 'Test3', ParentId: 'Test2' },
            'Test2': { Id: 'Test2', ParentId: 'Test1' },
            'Test1': { Id: 'Test1', ParentId: '' }
        }
        var pool = allPools['Test3'];
        expect(getFullPath(allPools, pool)).toBe('Test1 > Test2 > Test3');
    });
});

describe('flattenSubservices', function() {
    it('turns a tree structure into a flat array', function() {
        var tree = {
            id: 'top',
            children: [
                { 
                    id: 'middle1',
                    children: [
                        { id: 'leaf1' },
                        { id: 'leaf2' }
                    ]
                },
                {
                    id: 'middle2',
                    children: [ { id: 'leaf3' }, ]
                }
            ]
        }
        var result = flattenTree(0, tree);
        var expected = [ 
//            { depth: 0, id: 'top' }, // Excludes depth: 0
            { zendepth: 1, id: 'middle1' },
            { zendepth: 2, id: 'leaf1' },
            { zendepth: 2, id: 'leaf2' },
            { zendepth: 1, id: 'middle2' },
            { zendepth: 2, id: 'leaf3' }
        ];
        expect(result.length).toBe(expected.length);
        for (var i=0; i < expected.length; i++) {
            expect(result[i].depth).toBe(expected[i].depth);
            expect(result[i].id).toBe(expected[i].id);
        }
    });
});

describe('fix_pool_paths', function() {
    it('Defends against empty scope', function() {
        // just make sure you can call with an empty object
        fix_pool_paths({});
        expect(true).toBe(true);
    });

    it('Sets fullPath on hosts', function() {
        var scope = {
            pools: {
                mapped: {
                    a1: { fullPath: 'a1' },
                    a2: { fullPath: 'a1 > a2' },
                    a3: { fullPath: 'a1 > a2 > a3' }
                }
            },
            hosts: {
                all: [
                    { PoolId: 'a3' },
                    { PoolId: 'a1' },
                    { PoolId: 'a2' }
                ]
            }
        };
        fix_pool_paths(scope);
        scope.hosts.all.map(function(host) {
            expect(host.fullPath).toBe(scope.pools.mapped[host.PoolId].fullPath);
        });
    });
});

function fake_hosts_for_pool(poolId) {
    var mappedHosts = {
        "pool123": [{HostId: "host123", PoolId: "pool123"}],
        "default": [{HostId: "abc", PoolId: "default"}, {HostId: "def", PoolId: "default"}]
    };
    return mappedHosts[poolId];
}

function fake_resources_service() {
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
       }
   };
}

function fake_services_service() {
    return {
        get_services: function(cacheOk, callback) {
            callback(fake_services(), fake_services_tree());
        },
        add_service: function(service, callback) {
            callback({});
        },
        update_service: function(serviceId, service, callback) {
            callback({});
        },
        remove_service: function(serviceId, callback) {
            callback({});
        }
    };
}

function fake_auth_service() {
    var loggedIn = true;
    return {
        checkLogin: function(scope) {
            // stub does nothing
        },
        isLoggedIn: function() {
            return loggedIn;
        },
        login: function(truth) {
            loggedIn = truth;
        }
    };
}

function fake_wizard_service() {
    return {
        next_page: function(currentPath) {
            return "/fake_next";
        },
        cancel_page: function(currentPath) {
            return "/canceled";
        },
        get_context: function() {
            return {};
        },
        fix_location: function($location) {
            // do nothing
        }
    }
}

function expect_table(table) {
    expect(table).not.toBeUndefined();
    expect(table.data).not.toBeUndefined();
    expect(table.sort).not.toBeUndefined()
    expect(table.sort_icons).not.toBeUndefined();
    expect(table.get_order_class).not.toBeUndefined();
}

function expect_table_no_data(table) {
    expect(table).not.toBeUndefined();
    expect(table.sort).not.toBeUndefined()
    expect(table.sort_icons).not.toBeUndefined();
    expect(table.get_order_class).not.toBeUndefined();
}

function fake_pools() {
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
}

function fake_hosts() {
    return {
        "abc": {
            Id: "abc",
            PoolId: "default",
            Name: "abchost",
            IpAddr: "192.168.33.12",
            Cores: 2,
            Memory: 3061190144,
            PrivateNetwork: "255.255.255.0"
        },
        "def": {
            Id: "def",
            PoolId: "default",
            Name: "defhost",
            IpAddr: "192.168.33.13",
            Cores: 1,
            Memory: 12345,
            PrivateNetwork: "255.255.255.0"
        },
        "host123": {
            Id: "host123",
            PoolId: "pool123",
            Name: "some fake host",
            IpAddr: "192.168.33.14",
            Cores: 2,
            Memory: 2048
        }
    };
}

var fake1 = {
    "Id": "fakeId1",
    "Name": "mysql",
    "Startup": "/usr/libexec/mysqld",
    "Description": "Database service",
    "Instances": 0,
    "ImageId": "default",
    "PoolId": "default",
    "DesiredState": 1,
    "Endpoints": [
        {
            "Protocol": "tcp",
            "PortNumber": 3306,
            "Application": "mysql",
            "Purpose": "remote"
        }
    ],
    "ParentServiceId": ''
};
var service234 = {
    "Id": "service234",
    "Name": "zeneventd",
    "Startup": "/opt/zenoss/bin/zeneventd",
    "Description": "",
    "Instances": 0,
    "ImageId": "",
    "PoolId": "default",
    "DesiredState": 0,
    "Endpoints": null,
    "ParentServiceId": ''
};

var fake1Child = {
    "Id": "service234",
    "Name": "zeneventd",
    "Startup": "/opt/zenoss/bin/zeneventd",
    "Description": "",
    "Instances": 0,
    "ImageId": "",
    "PoolId": "default",
    "DesiredState": 0,
    "Endpoints": null,
    "ParentServiceId": "fakeId1"
};

function fake_services() {
    return [
        fake1, service234
    ];
}

function fake_services_tree() {
    fake1.children = [ fake1Child ];
    var tree = {};
    fake_services().map(function(e) {
        tree[e.Id] = e;
    });
    return tree;
}

